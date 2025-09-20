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
	query := `MATCH (c:Concept) WHERE elementId(c) = $id RETURN c`
	params := map[string]interface{}{"id": id}

	result, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	record, err := result.Single(context.Background())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	node, _ := record.Get("c")
	c.JSON(http.StatusOK, node)
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
