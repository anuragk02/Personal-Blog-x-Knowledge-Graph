package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const geminiEmbeddingModel = "models/text-embedding-004"

// generateEmbedding takes a single string and returns its vector embedding using the genai client library.
func generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	// Create a new client with your API key.
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		log.Printf("Failed to create genai client: %v", err)
		return nil, err
	}
	defer client.Close()

	// Access the embedding model.
	em := client.EmbeddingModel(geminiEmbeddingModel)

	// Call the model to embed the content.
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		log.Printf("Failed to generate embedding: %v", err)
		return nil, err
	}

	// Check for a valid response and return the embedding.
	if res.Embedding == nil || len(res.Embedding.Values) == 0 {
		return nil, fmt.Errorf("received an empty embedding from the API")
	}

	return res.Embedding.Values, nil
}

// generateEmbeddingsInBatch takes a slice of strings and returns their vector embeddings in a single API call.
func generateEmbeddingsInBatch(ctx context.Context, texts []string) ([][]float32, error) {
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		log.Printf("Failed to create genai client: %v", err)
		return nil, err
	}
	defer client.Close()

	em := client.EmbeddingModel(geminiEmbeddingModel)

	// Create a batch request from the slice of strings.
	batch := em.NewBatch()
	for _, text := range texts {
		batch.AddContent(genai.Text(text))
	}

	// Call the model with the batch of content.
	res, err := em.BatchEmbedContents(ctx, batch)
	if err != nil {
		log.Printf("Failed to generate batch embeddings: %v", err)
		return nil, err
	}

	// Check for a valid response.
	if res == nil || res.Embeddings == nil {
		return nil, fmt.Errorf("received a nil response from the batch embedding API")
	}

	// Extract the float slices from the response.
	var embeddings [][]float32
	for _, e := range res.Embeddings {
		if e != nil && len(e.Values) > 0 {
			embeddings = append(embeddings, e.Values)
		} else {
			// Add a nil or empty slice to maintain order if one text fails.
			embeddings = append(embeddings, nil)
		}
	}

	return embeddings, nil
}

// cosineSimilarity calculates the similarity between two vectors, returning a score between -1 and 1.
func cosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length to calculate similarity")
	}

	var dotProduct, aMagnitude, bMagnitude float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		aMagnitude += float64(a[i] * a[i])
		bMagnitude += float64(b[i] * b[i])
	}

	// Handle the case of zero vectors to avoid division by zero.
	if aMagnitude == 0 || bMagnitude == 0 {
		return 0, nil
	}

	return dotProduct / (math.Sqrt(aMagnitude) * math.Sqrt(bMagnitude)), nil
}

// NodeForEmbedding represents a node that needs an embedding generated
type NodeForEmbedding struct {
	ID          string
	NodeType    string // "system", "stock", "flow"
	Name        string
	Description string
	Text        string // Combined text for embedding
}

// processNodeEmbeddingsInBatch fetches all unconsolidated nodes, generates embeddings, and updates them
func (h *Handler) processNodeEmbeddingsInBatch(ctx context.Context) error {
	// Step 1: Fetch all unconsolidated nodes
	nodes, err := h.fetchUnconsolidatedNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch unconsolidated nodes: %v", err)
	}

	if len(nodes) == 0 {
		log.Println("No unconsolidated nodes found - all embeddings are up to date")
		return nil
	}

	log.Printf("Found %d unconsolidated nodes to process", len(nodes))

	// Step 2: Prepare texts for batch embedding
	texts := make([]string, len(nodes))
	for i, node := range nodes {
		texts[i] = node.Text
	}

	// Step 3: Generate embeddings in batch
	embeddings, err := generateEmbeddingsInBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %v", err)
	}

	// Step 4: Update nodes with embeddings and mark as consolidated
	return h.updateNodesWithEmbeddings(ctx, nodes, embeddings)
}

// fetchUnconsolidatedNodes retrieves all nodes that don't have embeddings yet
func (h *Handler) fetchUnconsolidatedNodes(ctx context.Context) ([]NodeForEmbedding, error) {
	var nodes []NodeForEmbedding

	// Query for systems without embeddings
	systemQuery := `MATCH (s:System) WHERE s.embedded = false OR s.embedded IS NULL 
		RETURN s.id, s.name, COALESCE(s.boundary_description, '') as description`
	systemRecords, err := h.db.ExecuteRead(ctx, systemQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %v", err)
	}

	for _, record := range systemRecords {
		id := record["s.id"].(string)
		name := record["s.name"].(string)
		description := record["description"].(string)

		// Combine name and description for embedding
		text := name
		if description != "" {
			text += ": " + description
		}

		nodes = append(nodes, NodeForEmbedding{
			ID:          id,
			NodeType:    "system",
			Name:        name,
			Description: description,
			Text:        text,
		})
	}

	// Query for stocks without embeddings
	stockQuery := `MATCH (st:Stock) WHERE st.embedded = false OR st.embedded IS NULL 
		RETURN st.id, st.name, COALESCE(st.description, '') as description`
	stockRecords, err := h.db.ExecuteRead(ctx, stockQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query stocks: %v", err)
	}

	for _, record := range stockRecords {
		id := record["st.id"].(string)
		name := record["st.name"].(string)
		description := record["description"].(string)

		// Combine name and description for embedding
		text := name
		if description != "" {
			text += ": " + description
		}

		nodes = append(nodes, NodeForEmbedding{
			ID:          id,
			NodeType:    "stock",
			Name:        name,
			Description: description,
			Text:        text,
		})
	}

	// Query for flows without embeddings
	flowQuery := `MATCH (f:Flow) WHERE f.embedded = false OR f.embedded IS NULL 
		RETURN f.id, f.name, COALESCE(f.description, '') as description`
	flowRecords, err := h.db.ExecuteRead(ctx, flowQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query flows: %v", err)
	}

	for _, record := range flowRecords {
		id := record["f.id"].(string)
		name := record["f.name"].(string)
		description := record["description"].(string)

		// Combine name and description for embedding
		text := name
		if description != "" {
			text += ": " + description
		}

		nodes = append(nodes, NodeForEmbedding{
			ID:          id,
			NodeType:    "flow",
			Name:        name,
			Description: description,
			Text:        text,
		})
	}

	return nodes, nil
}

// updateNodesWithEmbeddings updates nodes in the database with their embeddings and marks them as consolidated
func (h *Handler) updateNodesWithEmbeddings(ctx context.Context, nodes []NodeForEmbedding, embeddings [][]float32) error {
	if len(nodes) != len(embeddings) {
		return fmt.Errorf("mismatch between nodes count (%d) and embeddings count (%d)", len(nodes), len(embeddings))
	}

	for i, node := range nodes {
		if embeddings[i] == nil {
			log.Printf("Warning: No embedding generated for %s node '%s', skipping", node.NodeType, node.Name)
			continue
		}

		var query string
		switch node.NodeType {
		case "system":
			query = `MATCH (s:System {id: $id}) 
				SET s.embedding = $embedding, s.embedded = true`
		case "stock":
			query = `MATCH (st:Stock {id: $id}) 
				SET st.embedding = $embedding, st.embedded = true`
		case "flow":
			query = `MATCH (f:Flow {id: $id}) 
				SET f.embedding = $embedding, f.embedded = true`
		default:
			log.Printf("Warning: Unknown node type '%s' for node '%s', skipping", node.NodeType, node.Name)
			continue
		}

		params := map[string]interface{}{
			"id":        node.ID,
			"embedding": embeddings[i],
		}

		_, err := h.db.ExecuteQuery(ctx, query, params)
		if err != nil {
			log.Printf("Error updating %s node '%s' with embedding: %v", node.NodeType, node.Name, err)
			continue
		}

		log.Printf("Successfully updated %s node '%s' with embedding", node.NodeType, node.Name)
	}

	return nil
}
