package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/anuragk02/jna-nuh-yoh-guh/internal/database"
	"github.com/anuragk02/jna-nuh-yoh-guh/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// Helper functions for type conversion
func getStringValue(record map[string]interface{}, key string) string {
	if val, ok := record[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntValue(record map[string]interface{}, key string) int {
	if val, ok := record[key]; ok && val != nil {
		if i, ok := val.(int64); ok {
			return int(i)
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

// Node CRUD operations
func (h *Handler) CreateConcept(c *gin.Context) {
	var concept models.Concept
	if err := c.ShouldBindJSON(&concept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate a simple ID
	concept.ID = fmt.Sprintf("concept_%d", time.Now().Unix())

	query := `CREATE (c:Concept {id: $id, name: $name}) RETURN c.id as id`
	params := map[string]interface{}{
		"id":   concept.ID,
		"name": concept.Name,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, concept)
}

func (h *Handler) GetConcept(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (c:Concept {id: $id}) RETURN c.id, c.name, c.summary, c.mastery_level`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	record := records[0]
	concept := models.Concept{
		ID:           record["c.id"].(string),
		Name:         record["c.name"].(string),
		Summary:      getStringValue(record, "c.summary"),
		MasteryLevel: getIntValue(record, "c.mastery_level"),
	}

	c.JSON(http.StatusOK, concept)
}

// Relationship operations
func (h *Handler) CreateRelationship(c *gin.Context) {
	var rel models.Relationship
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (a), (b) WHERE elementId(a) = $from AND elementId(b) = $to 
			  CREATE (a)-[r:` + rel.Type + `]->(b) RETURN elementId(r) as id`
	params := map[string]interface{}{
		"from": rel.From,
		"to":   rel.To,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Relationship created"})
}

func (h *Handler) GetConnections(c *gin.Context) {
	nodeId := c.Query("node_id")
	query := `MATCH (n)-[r]-(m) WHERE elementId(n) = $node_id 
			  RETURN elementId(n) as from_id, type(r) as rel_type, elementId(m) as to_id`
	params := map[string]interface{}{"node_id": nodeId}

	result, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var connections []map[string]interface{}
	for result.Next(context.Background()) {
		record := result.Record()
		connections = append(connections, map[string]interface{}{
			"from":         record.Values[0],
			"relationship": record.Values[1],
			"to":           record.Values[2],
		})
	}

	c.JSON(http.StatusOK, connections)
}

// Essay handlers
func (h *Handler) CreateEssay(c *gin.Context) {
	var essay models.Essay
	if err := c.ShouldBindJSON(&essay); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID and set creation time
	essay.ID = fmt.Sprintf("essay_%d", time.Now().Unix())
	essay.CreatedAt = time.Now()

	query := `CREATE (e:Essay {id: $id, title: $title, content: $content, created_at: $created_at})`
	params := map[string]interface{}{
		"id":         essay.ID,
		"title":      essay.Title,
		"content":    essay.Content,
		"created_at": essay.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, essay)
}

func (h *Handler) GetEssay(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (e:Essay {id: $id}) RETURN e.id, e.title, e.content, e.created_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Essay not found"})
		return
	}

	record := records[0]
	essay := models.Essay{
		ID:      record["e.id"].(string),
		Title:   record["e.title"].(string),
		Content: record["e.content"].(string),
	}

	// Handle created_at parsing
	if createdAtStr := getStringValue(record, "e.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			essay.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, essay)
}

func (h *Handler) GetEssays(c *gin.Context) {
	query := `MATCH (e:Essay) RETURN e.id, e.title, e.created_at ORDER BY e.created_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var essays []map[string]interface{}
	for _, record := range records {
		essays = append(essays, map[string]interface{}{
			"id":         record["e.id"],
			"title":      record["e.title"],
			"created_at": record["e.created_at"],
		})
	}

	c.JSON(http.StatusOK, essays)
}

// POINTS_TO relationship handler
func (h *Handler) CreatePointsTo(c *gin.Context) {
	var rel models.Relationship
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set relationship type to POINTS_TO
	rel.Type = "POINTS_TO"

	query := `MATCH (from), (to) 
			  WHERE from.id = $from AND to.id = $to 
			  CREATE (from)-[:POINTS_TO]->(to)`
	params := map[string]interface{}{
		"from": rel.From,
		"to":   rel.To,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "POINTS_TO relationship created", "from": rel.From, "to": rel.To})
}
