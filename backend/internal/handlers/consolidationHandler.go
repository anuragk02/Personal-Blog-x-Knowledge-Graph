package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anuragk02/jna-nuh-yoh-guh/internal/models"
	"github.com/gin-gonic/gin"
)

// ConsolidateGraph - Main consolidation workflow handler
// Implements the 6-step consolidation process from phase2plan.txt
func (h *Handler) ConsolidateGraph(c *gin.Context) {
	ctx := c.Request.Context()

	log.Println("Starting graph consolidation workflow...")

	// Step 1: Fetch All Nodes
	unconsolidatedNodes, consolidatedNodes, err := h.fetchNodesForConsolidation(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch nodes: " + err.Error()})
		return
	}

	log.Printf("Found %d unconsolidated nodes and %d consolidated nodes", len(unconsolidatedNodes), len(consolidatedNodes))

	// Step 2: Find Node Matches
	nodeMatches, err := h.findNodeMatches(ctx, unconsolidatedNodes, consolidatedNodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find node matches: " + err.Error()})
		return
	}

	log.Printf("Found %d node matches for consolidation", len(nodeMatches))

	// Step 3: Synthesize New Names & Descriptions
	err = h.synthesizeNamesAndDescriptions(ctx, nodeMatches)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to synthesize names: " + err.Error()})
		return
	}

	// Step 4: Consolidate Nodes (Transaction 1)
	err = h.consolidateNodes(ctx, nodeMatches, unconsolidatedNodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consolidate nodes: " + err.Error()})
		return
	}

	// Step 5: Consolidate Relationships (Transaction 2)
	err = h.consolidateRelationships(ctx, nodeMatches)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consolidate relationships: " + err.Error()})
		return
	}

	// Step 6: Cleanup (Transaction 3)
	err = h.cleanupUnconsolidatedNodes(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup: " + err.Error()})
		return
	}

	log.Println("Graph consolidation workflow completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":                  "Graph consolidation completed successfully",
		"consolidations_performed": len(nodeMatches),
	})
}

// Step 1: Fetch all nodes separated by consolidation status
func (h *Handler) fetchNodesForConsolidation(ctx context.Context) (map[string][]interface{}, map[string][]interface{}, error) {
	unconsolidated := make(map[string][]interface{})
	consolidated := make(map[string][]interface{})

	// Fetch Systems
	systemQuery := `MATCH (s:System) WHERE s.embedded = true RETURN s.id as id, s.name as name, s.boundary_description as boundary_description, s.embedding as embedding, s.consolidated as consolidated, s.consolidation_score as consolidation_score`
	systemRecords, err := h.db.ExecuteRead(ctx, systemQuery, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch systems: %v", err)
	}

	for _, record := range systemRecords {
		system := map[string]interface{}{
			"id":                   record["id"],
			"name":                 record["name"],
			"boundary_description": record["boundary_description"],
			"embedding":            record["embedding"],
			"consolidated":         record["consolidated"],
			"consolidation_score":  record["consolidation_score"],
		}
		if system["consolidated"].(bool) {
			consolidated["system"] = append(consolidated["system"], system)
		} else {
			unconsolidated["system"] = append(unconsolidated["system"], system)
		}
	}

	// Fetch Stocks
	stockQuery := `MATCH (st:Stock) WHERE st.embedded = true RETURN st.id as id, st.name as name, st.description as description, st.embedding as embedding, st.consolidated as consolidated, st.consolidation_score as consolidation_score`
	stockRecords, err := h.db.ExecuteRead(ctx, stockQuery, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch stocks: %v", err)
	}

	for _, record := range stockRecords {
		stock := map[string]interface{}{
			"id":                  record["id"],
			"name":                record["name"],
			"description":         record["description"],
			"embedding":           record["embedding"],
			"consolidated":        record["consolidated"],
			"consolidation_score": record["consolidation_score"],
		}
		if stock["consolidated"].(bool) {
			consolidated["stock"] = append(consolidated["stock"], stock)
		} else {
			unconsolidated["stock"] = append(unconsolidated["stock"], stock)
		}
	}

	// Fetch Flows
	flowQuery := `MATCH (f:Flow) WHERE f.embedded = true RETURN f.id as id, f.name as name, f.description as description, f.embedding as embedding, f.consolidated as consolidated, f.consolidation_score as consolidation_score`
	flowRecords, err := h.db.ExecuteRead(ctx, flowQuery, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch flows: %v", err)
	}

	for _, record := range flowRecords {
		flow := map[string]interface{}{
			"id":                  record["id"],
			"name":                record["name"],
			"description":         record["description"],
			"embedding":           record["embedding"],
			"consolidated":        record["consolidated"],
			"consolidation_score": record["consolidation_score"],
		}
		if flow["consolidated"].(bool) {
			consolidated["flow"] = append(consolidated["flow"], flow)
		} else {
			unconsolidated["flow"] = append(unconsolidated["flow"], flow)
		}
	}

	return unconsolidated, consolidated, nil
}

// Step 2: Find matches between unconsolidated and consolidated nodes
func (h *Handler) findNodeMatches(ctx context.Context, unconsolidated, consolidated map[string][]interface{}) ([]models.NodeMatch, error) {
	var nodeMatches []models.NodeMatch
	const similarityThreshold = 0.60 // Lowered to 0.60 to capture more similar nodes

	// Process each node type
	for nodeType := range unconsolidated {
		if len(consolidated[nodeType]) == 0 {
			// FIRST RUN: Find similarities between unconsolidated nodes themselves
			log.Printf("First run for type %s - finding similarities between unconsolidated nodes", nodeType)

			unconsolidatedNodes := unconsolidated[nodeType]
			processed := make(map[string]bool)

			for i, node1 := range unconsolidatedNodes {
				node1Map := node1.(map[string]interface{})
				node1ID := node1Map["id"].(string)

				if processed[node1ID] {
					continue // Already grouped with another node
				}

				node1Embedding := h.convertEmbedding(node1Map["embedding"])
				bestMatchID := node1ID
				bestScore := -1.0

				// Compare with remaining nodes
				for j := i + 1; j < len(unconsolidatedNodes); j++ {
					node2 := unconsolidatedNodes[j]
					node2Map := node2.(map[string]interface{})
					node2ID := node2Map["id"].(string)

					if processed[node2ID] {
						continue
					}

					node2Embedding := h.convertEmbedding(node2Map["embedding"])
					score, err := cosineSimilarity(node1Embedding, node2Embedding)
					if err != nil {
						log.Printf("Warning: Failed to calculate similarity: %v", err)
						continue
					}

					log.Printf("Similarity between %s and %s: %.4f", node1ID, node2ID, score)

					if score >= similarityThreshold && score > bestScore {
						bestScore = score
						bestMatchID = node2ID
					}
				}

				// Create match
				if bestMatchID != node1ID {
					// Found a similar node - consolidate into the first one
					nodeMatches = append(nodeMatches, models.NodeMatch{
						UnconsolidatedID: bestMatchID,
						ConsolidatedID:   node1ID,
						NodeType:         nodeType,
						SimilarityScore:  bestScore,
					})
					processed[bestMatchID] = true
					log.Printf("MATCH: %s -> %s (similarity: %.4f)", bestMatchID, node1ID, bestScore)
				}

				// Mark the consolidated node (first one) as promoted
				nodeMatches = append(nodeMatches, models.NodeMatch{
					UnconsolidatedID: node1ID,
					ConsolidatedID:   node1ID, // Self-promotion to consolidated
					NodeType:         nodeType,
					SimilarityScore:  1.0,
				})
				processed[node1ID] = true
			}
		} else {
			// SUBSEQUENT RUNS: Match unconsolidated with existing consolidated
			for _, unconsolidatedNode := range unconsolidated[nodeType] {
				unconsolidatedMap := unconsolidatedNode.(map[string]interface{})
				unconsolidatedID := unconsolidatedMap["id"].(string)
				unconsolidatedEmbedding := h.convertEmbedding(unconsolidatedMap["embedding"])

				// Find best match among consolidated nodes
				var bestMatch models.NodeMatch
				bestScore := -1.0

				for _, consolidatedNode := range consolidated[nodeType] {
					consolidatedMap := consolidatedNode.(map[string]interface{})
					consolidatedEmbedding := h.convertEmbedding(consolidatedMap["embedding"])

					score, err := cosineSimilarity(unconsolidatedEmbedding, consolidatedEmbedding)
					if err != nil {
						log.Printf("Warning: Failed to calculate similarity: %v", err)
						continue
					}

					if score > bestScore {
						bestScore = score
						bestMatch = models.NodeMatch{
							UnconsolidatedID: unconsolidatedID,
							ConsolidatedID:   consolidatedMap["id"].(string),
							NodeType:         nodeType,
							SimilarityScore:  score,
						}
					}
				}

				// If above threshold, it's a match; otherwise promote to consolidated
				if bestScore >= similarityThreshold {
					nodeMatches = append(nodeMatches, bestMatch)
				} else {
					nodeMatches = append(nodeMatches, models.NodeMatch{
						UnconsolidatedID: unconsolidatedID,
						ConsolidatedID:   unconsolidatedID, // Promote to consolidated
						NodeType:         nodeType,
						SimilarityScore:  1.0,
					})
				}
			}
		}
	}

	return nodeMatches, nil
}

// Step 3: Synthesize new names and descriptions using Gemini
func (h *Handler) synthesizeNamesAndDescriptions(ctx context.Context, nodeMatches []models.NodeMatch) error {
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	for i := range nodeMatches {
		match := &nodeMatches[i]

		// Skip if it's a promotion (same ID)
		if match.UnconsolidatedID == match.ConsolidatedID {
			continue
		}

		log.Printf("Starting synthesis for nodes %s and %s", match.UnconsolidatedID, match.ConsolidatedID)

		// Fetch both nodes' details
		unconsolidatedNode, err := h.fetchNodeDetails(ctx, match.UnconsolidatedID, match.NodeType)
		if err != nil {
			log.Printf("Warning: Could not fetch unconsolidated node %s: %v", match.UnconsolidatedID, err)
			continue
		}

		consolidatedNode, err := h.fetchNodeDetails(ctx, match.ConsolidatedID, match.NodeType)
		if err != nil {
			log.Printf("Warning: Could not fetch consolidated node %s: %v", match.ConsolidatedID, err)
			continue
		}

		// Create synthesis prompt
		systemPrompt := "You are a Systems Analyst specializing in knowledge model normalization. Your task is to synthesize two similar concepts into a single, more universal concept. You must create a new formal name, a universal formal concept, and a concise, objective description that accurately represents both parent concepts."

		userPrompt := fmt.Sprintf(`Your task is to synthesize the following two similar '%s' nodes into a single, more universal concept that gracefully merges their meaning.

**Node A (Existing Consolidated Node):**
- Name: "%s"
- Description: "%s"

**Node B (New Unconsolidated Node):**
- Name: "%s"
- Description: "%s"

**Instructions:**
1.  **Synthesize Name:** Create a new, objective, and timeless name.
2.  **Synthesize Description:** Create a new description, under 15 words, that defines the component's objective function.

Provide the response in this exact JSON format, with no other text:
{
  "name": "[new synthesized name]",
  "description": "[new synthesized description]"
}`,
			match.NodeType,
			unconsolidatedNode["name"].(string),
			h.getDescription(unconsolidatedNode),
			consolidatedNode["name"].(string),
			h.getDescription(consolidatedNode))

		// Call Gemini API using HTTP
		llmApiUrl := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"

		payload := map[string]interface{}{
			"systemInstruction": map[string]interface{}{
				"parts": []map[string]string{
					{"text": systemPrompt},
				},
			},
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": userPrompt},
					},
				},
			},
			"generationConfig": map[string]string{
				"response_mime_type": "application/json",
			},
		}

		llmReqBody, _ := json.Marshal(payload)
		httpRequest, err := http.NewRequestWithContext(ctx, "POST", llmApiUrl, bytes.NewBuffer(llmReqBody))
		if err != nil {
			log.Printf("Warning: Failed to create synthesis request for nodes %s and %s: %v", match.UnconsolidatedID, match.ConsolidatedID, err)
			continue
		}

		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("X-goog-api-key", geminiApiKey)

		client := &http.Client{}
		httpResponse, err := client.Do(httpRequest)
		if err != nil {
			log.Printf("Warning: Failed to synthesize for nodes %s and %s: %v", match.UnconsolidatedID, match.ConsolidatedID, err)
			continue
		}
		defer httpResponse.Body.Close()

		if httpResponse.StatusCode != http.StatusOK {
			log.Printf("Warning: Synthesis API returned status %d for nodes %s and %s", httpResponse.StatusCode, match.UnconsolidatedID, match.ConsolidatedID)
			continue
		}

		// Parse the response
		var geminiResponse map[string]interface{}
		if err := json.NewDecoder(httpResponse.Body).Decode(&geminiResponse); err != nil {
			log.Printf("Warning: Failed to parse synthesis response for nodes %s and %s: %v", match.UnconsolidatedID, match.ConsolidatedID, err)
			continue
		}

		// Extract the synthesized content
		if candidates, ok := geminiResponse["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								log.Printf("Synthesis response: %s", text)

								// Parse the JSON response
								var synthesis map[string]string
								if err := json.Unmarshal([]byte(text), &synthesis); err == nil {
									match.NewName = synthesis["name"]
									match.NewDescription = synthesis["description"]
									log.Printf("Parsed synthesis - Name: '%s', Description: '%s'", match.NewName, match.NewDescription)
								} else {
									log.Printf("Warning: Failed to parse synthesis JSON for nodes %s and %s: %v", match.UnconsolidatedID, match.ConsolidatedID, err)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// Helper functions
func (h *Handler) convertEmbedding(embeddingInterface interface{}) []float32 {
	if embeddingInterface == nil {
		return []float32{}
	}

	switch v := embeddingInterface.(type) {
	case []float32:
		return v
	case []interface{}:
		result := make([]float32, len(v))
		for i, val := range v {
			if f, ok := val.(float64); ok {
				result[i] = float32(f)
			}
		}
		return result
	default:
		return []float32{}
	}
}

func (h *Handler) fetchNodeDetails(ctx context.Context, nodeID, nodeType string) (map[string]interface{}, error) {
	var query string

	switch nodeType {
	case "system":
		query = `MATCH (s:System {id: $id}) RETURN s.id as id, s.name as name, s.boundary_description as boundary_description`
	case "stock":
		query = `MATCH (st:Stock {id: $id}) RETURN st.id as id, st.name as name, st.description as description`
	case "flow":
		query = `MATCH (f:Flow {id: $id}) RETURN f.id as id, f.name as name, f.description as description`
	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	records, err := h.db.ExecuteRead(ctx, query, map[string]interface{}{"id": nodeID})
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	result := map[string]interface{}{
		"id":   records[0]["id"],
		"name": records[0]["name"],
	}

	if nodeType == "system" {
		result["boundary_description"] = records[0]["boundary_description"]
	} else {
		result["description"] = records[0]["description"]
	}

	return result, nil
}

func (h *Handler) getDescription(node map[string]interface{}) string {
	if desc, exists := node["description"]; exists && desc != nil {
		return desc.(string)
	}
	if desc, exists := node["boundary_description"]; exists && desc != nil {
		return desc.(string)
	}
	return ""
}

// Step 4: Consolidate Nodes (Transaction 1)
func (h *Handler) consolidateNodes(ctx context.Context, nodeMatches []models.NodeMatch, unconsolidatedNodes map[string][]interface{}) error {
	for _, match := range nodeMatches {
		if match.UnconsolidatedID == match.ConsolidatedID {
			// This is a promotion - mark unconsolidated node as consolidated
			err := h.promoteNodeToConsolidated(ctx, match.UnconsolidatedID, match.NodeType)
			if err != nil {
				log.Printf("Warning: Failed to promote node %s: %v", match.UnconsolidatedID, err)
				continue
			}
		} else {
			// This is a merge - consolidate into existing node
			err := h.mergeIntoConsolidatedNode(ctx, match)
			if err != nil {
				log.Printf("Warning: Failed to merge nodes %s -> %s: %v", match.UnconsolidatedID, match.ConsolidatedID, err)
				continue
			}
		}
	}
	return nil
}

// Step 5: Consolidate Relationships (Transaction 2)
func (h *Handler) consolidateRelationships(ctx context.Context, nodeMatches []models.NodeMatch) error {
	// Create a mapping for quick lookup
	nodeMapping := make(map[string]string)
	for _, match := range nodeMatches {
		nodeMapping[match.UnconsolidatedID] = match.ConsolidatedID
	}

	// Fetch all unconsolidated relationships
	relationships, err := h.fetchUnconsolidatedRelationships(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch relationships: %v", err)
	}

	// Process each relationship
	for _, rel := range relationships {
		err := h.processRelationshipConsolidation(ctx, rel, nodeMapping)
		if err != nil {
			log.Printf("Warning: Failed to consolidate relationship: %v", err)
			continue
		}
	}

	return nil
}

// Step 6: Cleanup (Transaction 3)
func (h *Handler) cleanupUnconsolidatedNodes(ctx context.Context) error {
	query := `MATCH (n) WHERE n.consolidated = false DETACH DELETE n`
	_, err := h.db.ExecuteQuery(ctx, query, nil)
	return err
}

// Helper methods for consolidation workflow

func (h *Handler) promoteNodeToConsolidated(ctx context.Context, nodeID, nodeType string) error {
	var query string
	switch nodeType {
	case "system":
		query = `MATCH (s:System {id: $id}) SET s.consolidated = true, s.consolidation_score = 1, s.last_consolidated_at = $timestamp`
	case "stock":
		query = `MATCH (st:Stock {id: $id}) SET st.consolidated = true, st.consolidation_score = 1, st.last_consolidated_at = $timestamp`
	case "flow":
		query = `MATCH (f:Flow {id: $id}) SET f.consolidated = true, f.consolidation_score = 1, f.last_consolidated_at = $timestamp`
	default:
		return fmt.Errorf("unknown node type: %s", nodeType)
	}

	params := map[string]interface{}{
		"id":        nodeID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) mergeIntoConsolidatedNode(ctx context.Context, match models.NodeMatch) error {
	// Get both nodes to calculate weighted average
	unconsolidatedNode, err := h.getNodeEmbeddingAndScore(ctx, match.UnconsolidatedID, match.NodeType)
	if err != nil {
		return err
	}

	consolidatedNode, err := h.getNodeEmbeddingAndScore(ctx, match.ConsolidatedID, match.NodeType)
	if err != nil {
		return err
	}

	// Calculate weighted average embedding
	newEmbedding := h.calculateWeightedAverageEmbedding(
		unconsolidatedNode.Embedding, 1.0,
		consolidatedNode.Embedding, float64(consolidatedNode.ConsolidationScore),
	)

	// Update consolidated node
	var query string
	switch match.NodeType {
	case "system":
		query = `MATCH (s:System {id: $id}) 
			SET s.embedding = $embedding, 
				s.consolidation_score = s.consolidation_score + 1, 
				s.last_consolidated_at = $timestamp`
		if match.NewName != "" {
			query += `, s.name = $name`
		}
		if match.NewDescription != "" {
			query += `, s.boundary_description = $description`
		}
	case "stock":
		query = `MATCH (st:Stock {id: $id}) 
			SET st.embedding = $embedding, 
				st.consolidation_score = st.consolidation_score + 1, 
				st.last_consolidated_at = $timestamp`
		if match.NewName != "" {
			query += `, st.name = $name`
		}
		if match.NewDescription != "" {
			query += `, st.description = $description`
		}
	case "flow":
		query = `MATCH (f:Flow {id: $id}) 
			SET f.embedding = $embedding, 
				f.consolidation_score = f.consolidation_score + 1, 
				f.last_consolidated_at = $timestamp`
		if match.NewName != "" {
			query += `, f.name = $name`
		}
		if match.NewDescription != "" {
			query += `, f.description = $description`
		}
	default:
		return fmt.Errorf("unknown node type: %s", match.NodeType)
	}

	params := map[string]interface{}{
		"id":        match.ConsolidatedID,
		"embedding": newEmbedding,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if match.NewName != "" {
		params["name"] = match.NewName
	}
	if match.NewDescription != "" {
		params["description"] = match.NewDescription
	}

	_, err = h.db.ExecuteQuery(ctx, query, params)
	if err != nil {
		return err
	}

	// Transfer relationships - simple approach using multiple queries
	// First get all relationships from the node to be merged
	relationshipsQuery := `
		MATCH (from {id: $from_id})-[r]-(other)
		RETURN type(r) as rel_type, startNode(r) = from as is_outgoing, other.id as other_id, properties(r) as props
	`

	relRecords, err := h.db.ExecuteRead(ctx, relationshipsQuery, map[string]interface{}{
		"from_id": match.UnconsolidatedID,
	})

	if err != nil {
		log.Printf("Warning: Failed to fetch relationships for transfer: %v", err)
	} else {
		// Transfer each relationship
		for _, relRecord := range relRecords {
			relType := relRecord["rel_type"].(string)
			isOutgoing := relRecord["is_outgoing"].(bool)
			otherID := relRecord["other_id"].(string)

			var createQuery string
			if isOutgoing {
				createQuery = fmt.Sprintf(`
					MATCH (to {id: $to_id}), (other {id: $other_id})
					WHERE NOT (to)-[:%s]->(other)
					CREATE (to)-[r:%s]->(other)
					SET r = $props
				`, relType, relType)
			} else {
				createQuery = fmt.Sprintf(`
					MATCH (to {id: $to_id}), (other {id: $other_id})
					WHERE NOT (other)-[:%s]->(to)
					CREATE (other)-[r:%s]->(to)
					SET r = $props
				`, relType, relType)
			}

			createParams := map[string]interface{}{
				"to_id":    match.ConsolidatedID,
				"other_id": otherID,
				"props":    relRecord["props"],
			}

			_, err = h.db.ExecuteQuery(ctx, createQuery, createParams)
			if err != nil {
				log.Printf("Warning: Failed to create relationship: %v", err)
			}
		}
	}

	// Delete all relationships from the old node and mark for deletion
	deleteQuery := `
		MATCH (n {id: $id})
		DETACH DELETE n
	`
	_, err = h.db.ExecuteQuery(ctx, deleteQuery, map[string]interface{}{"id": match.UnconsolidatedID})

	return err
}

// Additional helper methods for consolidation workflow

type NodeEmbeddingScore struct {
	Embedding          []float32
	ConsolidationScore int
}

func (h *Handler) getNodeEmbeddingAndScore(ctx context.Context, nodeID, nodeType string) (*NodeEmbeddingScore, error) {
	var query string
	switch nodeType {
	case "system":
		query = `MATCH (s:System {id: $id}) RETURN s.embedding as embedding, s.consolidation_score as consolidation_score`
	case "stock":
		query = `MATCH (st:Stock {id: $id}) RETURN st.embedding as embedding, st.consolidation_score as consolidation_score`
	case "flow":
		query = `MATCH (f:Flow {id: $id}) RETURN f.embedding as embedding, f.consolidation_score as consolidation_score`
	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}

	records, err := h.db.ExecuteRead(ctx, query, map[string]interface{}{"id": nodeID})
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	embedding := h.convertEmbedding(records[0]["embedding"])
	consolidationScore := 0
	if score := records[0]["consolidation_score"]; score != nil {
		consolidationScore = int(score.(int64))
	}

	return &NodeEmbeddingScore{
		Embedding:          embedding,
		ConsolidationScore: consolidationScore,
	}, nil
}

func (h *Handler) calculateWeightedAverageEmbedding(embedding1 []float32, weight1 float64, embedding2 []float32, weight2 float64) []float32 {
	if len(embedding1) != len(embedding2) {
		log.Printf("Warning: Embedding lengths don't match (%d vs %d), using first embedding", len(embedding1), len(embedding2))
		return embedding1
	}

	totalWeight := weight1 + weight2
	result := make([]float32, len(embedding1))

	for i := range embedding1 {
		result[i] = float32((float64(embedding1[i])*weight1 + float64(embedding2[i])*weight2) / totalWeight)
	}

	return result
}

func (h *Handler) fetchUnconsolidatedRelationships(ctx context.Context) ([]models.RelationshipConsolidation, error) {
	var relationships []models.RelationshipConsolidation

	// First, dynamically discover all relationship types that need consolidation
	discoveryQuery := `
		MATCH ()-[r]->()
		WHERE r.consolidated = false OR r.consolidated IS NULL
		RETURN DISTINCT type(r) as rel_type
	`

	typeRecords, err := h.db.ExecuteRead(ctx, discoveryQuery, nil)
	if err != nil {
		log.Printf("Warning: Could not discover relationship types dynamically: %v", err)
		// Fallback to actual relationship types in your graph
		typeRecords = []map[string]interface{}{
			{"rel_type": "DESCRIBES"},
			{"rel_type": "DESCRIBES_STATIC"},
			{"rel_type": "CAUSAL_LINK"},
			{"rel_type": "CHANGES"},
		}
	}

	// Process each relationship type found
	for _, typeRecord := range typeRecords {
		relType := typeRecord["rel_type"].(string)

		// Generic query to get all relationships of this type
		query := fmt.Sprintf(`
			MATCH (from)-[r:%s]->(to)
			WHERE r.consolidated = false OR r.consolidated IS NULL
			RETURN '%s' as type, from.id as from_id, to.id as to_id
		`, relType, relType)

		records, err := h.db.ExecuteRead(ctx, query, nil)
		if err != nil {
			log.Printf("Warning: Failed to fetch %s relationships: %v", relType, err)
			continue
		}

		for _, record := range records {
			relationships = append(relationships, models.RelationshipConsolidation{
				RelationType: record["type"].(string),
				FromID:       record["from_id"].(string),
				ToID:         record["to_id"].(string),
			})
		}

		log.Printf("Found %d unconsolidated %s relationships", len(records), relType)
	}

	log.Printf("Total unconsolidated relationships found: %d", len(relationships))
	return relationships, nil
}

func (h *Handler) processRelationshipConsolidation(ctx context.Context, rel models.RelationshipConsolidation, nodeMapping map[string]string) error {
	// Map from/to IDs to consolidated versions (if they exist in mapping)
	consolidatedFrom := rel.FromID
	consolidatedTo := rel.ToID

	// Check if nodes were consolidated (either merged or promoted)
	fromWasConsolidated := false
	toWasConsolidated := false

	if mappedFrom, exists := nodeMapping[rel.FromID]; exists {
		consolidatedFrom = mappedFrom
		fromWasConsolidated = true
	}
	if mappedTo, exists := nodeMapping[rel.ToID]; exists {
		consolidatedTo = mappedTo
		toWasConsolidated = true
	}

	log.Printf("Processing %s relationship: %s -> %s (originally %s -> %s)",
		rel.RelationType, consolidatedFrom, consolidatedTo, rel.FromID, rel.ToID)

	// Case 1: Neither node was consolidated (e.g., both are Narratives, or other non-consolidating types)
	// Just mark the existing relationship as consolidated
	if !fromWasConsolidated && !toWasConsolidated {
		query := fmt.Sprintf(`
			MATCH (from {id: $from_id})-[r:%s]->(to {id: $to_id})
			SET r.consolidated = true, r.consolidation_score = 1
		`, rel.RelationType)

		params := map[string]interface{}{
			"from_id": rel.FromID,
			"to_id":   rel.ToID,
		}

		_, err := h.db.ExecuteQuery(ctx, query, params)
		return err
	}

	// Case 2: At least one node was consolidated
	// Create/update consolidated relationship and delete the old unconsolidated one

	// First, create or update the consolidated relationship
	mergeQuery := fmt.Sprintf(`
		MATCH (from {id: $consolidated_from_id}), (to {id: $consolidated_to_id})
		MERGE (from)-[r:%s]->(to)
		ON CREATE SET r.consolidated = true, r.consolidation_score = 1
		ON MATCH SET r.consolidated = true, r.consolidation_score = COALESCE(r.consolidation_score, 0) + 1
	`, rel.RelationType)

	mergeParams := map[string]interface{}{
		"consolidated_from_id": consolidatedFrom,
		"consolidated_to_id":   consolidatedTo,
	}

	_, err := h.db.ExecuteQuery(ctx, mergeQuery, mergeParams)
	if err != nil {
		log.Printf("Failed to create/update consolidated %s relationship: %v", rel.RelationType, err)
		return err
	}

	// Second, delete the old unconsolidated relationship (only if nodes actually changed)
	if consolidatedFrom != rel.FromID || consolidatedTo != rel.ToID {
		deleteQuery := fmt.Sprintf(`
			MATCH (from {id: $original_from_id})-[r:%s]->(to {id: $original_to_id})
			DELETE r
		`, rel.RelationType)

		deleteParams := map[string]interface{}{
			"original_from_id": rel.FromID,
			"original_to_id":   rel.ToID,
		}

		_, err = h.db.ExecuteQuery(ctx, deleteQuery, deleteParams)
		if err != nil {
			log.Printf("Failed to delete old unconsolidated %s relationship: %v", rel.RelationType, err)
			return err
		}

		log.Printf("Successfully consolidated %s relationship: deleted (%s -> %s), created (%s -> %s)",
			rel.RelationType, rel.FromID, rel.ToID, consolidatedFrom, consolidatedTo)
	} else {
		log.Printf("Updated %s relationship consolidation status: %s -> %s",
			rel.RelationType, rel.FromID, rel.ToID)
	}


	if err != nil {
		log.Printf("Warning: Failed to consolidate %s relationship %s -> %s: %v",
			rel.RelationType, consolidatedFrom, consolidatedTo, err)
		return err
	} else {
		log.Printf("Successfully consolidated %s relationship %s -> %s",
			rel.RelationType, consolidatedFrom, consolidatedTo)
		return nil
	}


}

// ResetConsolidation - Reset all nodes to unconsolidated status for re-consolidation
func (h *Handler) ResetConsolidation(c *gin.Context) {
	ctx := c.Request.Context()

	// Reset all nodes to unconsolidated
	nodeQueries := []string{
		`MATCH (s:System) WHERE s.embedded = true SET s.consolidated = false, s.consolidation_score = 0`,
		`MATCH (st:Stock) WHERE st.embedded = true SET st.consolidated = false, st.consolidation_score = 0`,
		`MATCH (f:Flow) WHERE f.embedded = true SET f.consolidated = false, f.consolidation_score = 0`,
	}

	for _, query := range nodeQueries {
		_, err := h.db.ExecuteQuery(ctx, query, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset node consolidation: " + err.Error()})
			return
		}
	}

	// Reset all relationships to unconsolidated (using actual relationship types)
	relationshipQueries := []string{
		`MATCH ()-[r:DESCRIBES]->() SET r.consolidated = false, r.consolidation_score = 0`,
		`MATCH ()-[r:DESCRIBES_STATIC]->() SET r.consolidated = false, r.consolidation_score = 0`,
		`MATCH ()-[r:CAUSAL_LINK]->() SET r.consolidated = false, r.consolidation_score = 0`,
		`MATCH ()-[r:CHANGES]->() SET r.consolidated = false, r.consolidation_score = 0`,
	}

	for _, query := range relationshipQueries {
		_, err := h.db.ExecuteQuery(ctx, query, nil)
		if err != nil {
			log.Printf("Warning: Failed to reset relationship consolidation: %v", err)
			// Continue with other relationship types
		}
	}

	log.Println("Reset all nodes and relationships to unconsolidated status")

	c.JSON(http.StatusOK, gin.H{
		"message": "All nodes and relationships reset to unconsolidated status - ready for re-consolidation",
	})
}
