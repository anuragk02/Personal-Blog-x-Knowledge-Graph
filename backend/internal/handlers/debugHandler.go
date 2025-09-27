package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// DebugSimilarity - Test similarity between two specific nodes
func (h *Handler) DebugSimilarity(c *gin.Context) {
	ctx := c.Request.Context()

	// Get node IDs from query parameters
	node1ID := c.Query("node1")
	node2ID := c.Query("node2")

	if node1ID == "" || node2ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both node1 and node2 query parameters are required"})
		return
	}

	// Fetch both nodes
	node1, err := h.fetchNodeForSimilarity(ctx, node1ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch node1: " + err.Error()})
		return
	}

	node2, err := h.fetchNodeForSimilarity(ctx, node2ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch node2: " + err.Error()})
		return
	}

	// Calculate similarity
	similarity, err := cosineSimilarity(node1.Embedding, node2.Embedding)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate similarity: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node1": gin.H{
			"id":               node1.ID,
			"name":             node1.Name,
			"description":      node1.Description,
			"type":             node1.NodeType,
			"consolidated":     node1.Consolidated,
			"embedding_length": len(node1.Embedding),
		},
		"node2": gin.H{
			"id":               node2.ID,
			"name":             node2.Name,
			"description":      node2.Description,
			"type":             node2.NodeType,
			"consolidated":     node2.Consolidated,
			"embedding_length": len(node2.Embedding),
		},
		"similarity_score": similarity,
		"threshold":        0.60, // Updated to match consolidation threshold
		"would_merge":      similarity >= 0.60,
	})
}

type NodeForSimilarity struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	NodeType     string    `json:"nodeType"`
	Consolidated bool      `json:"consolidated"`
	Embedding    []float32 `json:"embedding"`
}

func (h *Handler) fetchNodeForSimilarity(ctx context.Context, nodeID string) (*NodeForSimilarity, error) {
	// Try to find the node in any of the three types
	queries := map[string]string{
		"system": `MATCH (s:System {id: $id}) RETURN s.id as id, s.name as name, s.boundary_description as description, s.consolidated as consolidated, s.embedding as embedding`,
		"stock":  `MATCH (st:Stock {id: $id}) RETURN st.id as id, st.name as name, st.description as description, st.consolidated as consolidated, st.embedding as embedding`,
		"flow":   `MATCH (f:Flow {id: $id}) RETURN f.id as id, f.name as name, f.description as description, f.consolidated as consolidated, f.embedding as embedding`,
	}

	for nodeType, query := range queries {
		records, err := h.db.ExecuteRead(ctx, query, map[string]interface{}{"id": nodeID})
		if err != nil {
			continue
		}

		if len(records) > 0 {
			record := records[0]
			description := ""
			if desc := record["description"]; desc != nil {
				description = desc.(string)
			}

			return &NodeForSimilarity{
				ID:           record["id"].(string),
				Name:         record["name"].(string),
				Description:  description,
				NodeType:     nodeType,
				Consolidated: record["consolidated"].(bool),
				Embedding:    h.convertEmbedding(record["embedding"]),
			}, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}

// DebugNodeRelationships - Check relationships for a specific node
func (h *Handler) DebugNodeRelationships(c *gin.Context) {
	ctx := c.Request.Context()

	nodeID := c.Query("nodeId")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nodeId query parameter is required"})
		return
	}

	// Query to find the node and its relationships
	query := `
		MATCH (n {id: $nodeId})
		OPTIONAL MATCH (n)-[r]-(connected)
		RETURN n.id as nodeId, n.name as nodeName, labels(n) as nodeLabels,
		       count(r) as relationshipCount,
		       collect(DISTINCT type(r)) as relationshipTypes,
		       collect(DISTINCT connected.id) as connectedNodeIds
	`

	records, err := h.db.ExecuteRead(ctx, query, map[string]interface{}{"nodeId": nodeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query relationships: " + err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	record := records[0]
	c.JSON(http.StatusOK, gin.H{
		"node_id":            record["nodeId"],
		"node_name":          record["nodeName"],
		"node_labels":        record["nodeLabels"],
		"relationship_count": record["relationshipCount"],
		"relationship_types": record["relationshipTypes"],
		"connected_node_ids": record["connectedNodeIds"],
	})
}

// DebugSynthesis - Test name synthesis between two specific nodes
func (h *Handler) DebugSynthesis(c *gin.Context) {
	ctx := c.Request.Context()

	node1ID := c.Query("node1")
	node2ID := c.Query("node2")

	if node1ID == "" || node2ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both node1 and node2 query parameters are required"})
		return
	}

	// Fetch both nodes
	node1, err := h.fetchNodeForSimilarity(ctx, node1ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch node1: " + err.Error()})
		return
	}

	node2, err := h.fetchNodeForSimilarity(ctx, node2ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch node2: " + err.Error()})
		return
	}

	// Test synthesis directly
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GEMINI_API_KEY not set"})
		return
	}

	// Create synthesis prompt
	systemPrompt := "You are an expert in systems thinking and knowledge synthesis. Your task is to combine two related concepts into a single, coherent name and description."

	userPrompt := fmt.Sprintf(`Synthesize a new, concise name and a comprehensive description that accurately combines the concepts of these two %s nodes:

Node A - Name: %s, Description: %s
Node B - Name: %s, Description: %s

Please provide the response in this exact JSON format:
{
  "name": "[new synthesized name]",
  "description": "[new synthesized description]"
}`,
		node1.NodeType,
		node1.Name, node1.Description,
		node2.Name, node2.Description)

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request: " + err.Error()})
		return
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-goog-api-key", geminiApiKey)

	client := &http.Client{}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Gemini API call failed: " + err.Error(),
			"prompt": userPrompt,
		})
		return
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  fmt.Sprintf("Gemini API returned status %d", httpResponse.StatusCode),
			"prompt": userPrompt,
		})
		return
	}

	// Parse the response
	var geminiResponse map[string]interface{}
	if err := json.NewDecoder(httpResponse.Body).Decode(&geminiResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to parse response: " + err.Error(),
			"prompt": userPrompt,
		})
		return
	}

	var content string
	var name, description string

	// Extract the synthesized content
	if candidates, ok := geminiResponse["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if contentObj, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := contentObj["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							content = text

							// Parse the JSON response
							var synthesis map[string]string
							if err := json.Unmarshal([]byte(text), &synthesis); err == nil {
								name = synthesis["name"]
								description = synthesis["description"]
							}
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"node1": gin.H{
			"id":          node1.ID,
			"name":        node1.Name,
			"description": node1.Description,
			"type":        node1.NodeType,
		},
		"node2": gin.H{
			"id":          node2.ID,
			"name":        node2.Name,
			"description": node2.Description,
			"type":        node2.NodeType,
		},
		"prompt":                  userPrompt,
		"raw_response":            content,
		"synthesized_name":        name,
		"synthesized_description": description,
	})
}
