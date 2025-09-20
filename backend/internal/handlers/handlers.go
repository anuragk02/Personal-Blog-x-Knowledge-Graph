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

func getBoolValue(record map[string]interface{}, key string) bool {
	if val, ok := record[key]; ok && val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
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

func (h *Handler) GetConcepts(c *gin.Context) {
	query := `MATCH (c:Concept) RETURN c.id, c.name, c.summary, c.mastery_level`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var concepts []models.Concept
	for _, record := range records {
		concept := models.Concept{
			ID:           record["c.id"].(string),
			Name:         record["c.name"].(string),
			Summary:      getStringValue(record, "c.summary"),
			MasteryLevel: getIntValue(record, "c.mastery_level"),
		}
		concepts = append(concepts, concept)
	}

	c.JSON(http.StatusOK, concepts)
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

// Claim handlers
func (h *Handler) CreateClaim(c *gin.Context) {
	var claim models.Claim
	if err := c.ShouldBindJSON(&claim); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim.ID = fmt.Sprintf("claim_%d", time.Now().Unix())

	query := `CREATE (cl:Claim {id: $id, text: $text, confidence_score: $confidence_score, is_verified: $is_verified})`
	params := map[string]interface{}{
		"id":               claim.ID,
		"text":             claim.Text,
		"confidence_score": claim.ConfidenceScore,
		"is_verified":      claim.IsVerified,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, claim)
}

func (h *Handler) GetClaim(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (cl:Claim {id: $id}) RETURN cl.id, cl.text, cl.confidence_score, cl.is_verified`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claim not found"})
		return
	}

	record := records[0]
	claim := models.Claim{
		ID:              record["cl.id"].(string),
		Text:            record["cl.text"].(string),
		ConfidenceScore: getIntValue(record, "cl.confidence_score"),
		IsVerified:      getBoolValue(record, "cl.is_verified"),
	}

	c.JSON(http.StatusOK, claim)
}

func (h *Handler) GetClaims(c *gin.Context) {
	query := `MATCH (cl:Claim) RETURN cl.id, cl.text, cl.confidence_score, cl.is_verified`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var claims []models.Claim
	for _, record := range records {
		claim := models.Claim{
			ID:              record["cl.id"].(string),
			Text:            record["cl.text"].(string),
			ConfidenceScore: getIntValue(record, "cl.confidence_score"),
			IsVerified:      getBoolValue(record, "cl.is_verified"),
		}
		claims = append(claims, claim)
	}

	c.JSON(http.StatusOK, claims)
}

// Source handlers
func (h *Handler) CreateSource(c *gin.Context) {
	var source models.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source.ID = fmt.Sprintf("source_%d", time.Now().Unix())
	source.DateAdded = time.Now()

	query := `CREATE (s:Source {id: $id, type: $type, title: $title, author: $author, url: $url, date_added: $date_added})`
	params := map[string]interface{}{
		"id":         source.ID,
		"type":       source.Type,
		"title":      source.Title,
		"author":     source.Author,
		"url":        source.URL,
		"date_added": source.DateAdded.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, source)
}

func (h *Handler) GetSource(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (s:Source {id: $id}) RETURN s.id, s.type, s.title, s.author, s.url, s.date_added`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	}

	record := records[0]
	source := models.Source{
		ID:     record["s.id"].(string),
		Type:   record["s.type"].(string),
		Title:  record["s.title"].(string),
		Author: getStringValue(record, "s.author"),
		URL:    getStringValue(record, "s.url"),
	}

	if dateStr := getStringValue(record, "s.date_added"); dateStr != "" {
		if dateAdded, err := time.Parse(time.RFC3339, dateStr); err == nil {
			source.DateAdded = dateAdded
		}
	}

	c.JSON(http.StatusOK, source)
}

func (h *Handler) GetSources(c *gin.Context) {
	query := `MATCH (s:Source) RETURN s.id, s.type, s.title, s.author, s.url, s.date_added`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sources []models.Source
	for _, record := range records {
		source := models.Source{
			ID:     record["s.id"].(string),
			Type:   record["s.type"].(string),
			Title:  record["s.title"].(string),
			Author: getStringValue(record, "s.author"),
			URL:    getStringValue(record, "s.url"),
		}

		if dateStr := getStringValue(record, "s.date_added"); dateStr != "" {
			if dateAdded, err := time.Parse(time.RFC3339, dateStr); err == nil {
				source.DateAdded = dateAdded
			}
		}

		sources = append(sources, source)
	}

	c.JSON(http.StatusOK, sources)
}

// Question handlers
func (h *Handler) CreateQuestion(c *gin.Context) {
	var question models.Question
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	question.ID = fmt.Sprintf("question_%d", time.Now().Unix())

	query := `CREATE (q:Question {id: $id, text: $text, priority: $priority, status: $status})`
	params := map[string]interface{}{
		"id":       question.ID,
		"text":     question.Text,
		"priority": question.Priority,
		"status":   question.Status,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, question)
}

func (h *Handler) GetQuestion(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (q:Question {id: $id}) RETURN q.id, q.text, q.priority, q.status`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	record := records[0]
	question := models.Question{
		ID:       record["q.id"].(string),
		Text:     record["q.text"].(string),
		Priority: getIntValue(record, "q.priority"),
		Status:   getStringValue(record, "q.status"),
	}

	c.JSON(http.StatusOK, question)
}

func (h *Handler) GetQuestions(c *gin.Context) {
	query := `MATCH (q:Question) RETURN q.id, q.text, q.priority, q.status`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var questions []models.Question
	for _, record := range records {
		question := models.Question{
			ID:       record["q.id"].(string),
			Text:     record["q.text"].(string),
			Priority: getIntValue(record, "q.priority"),
			Status:   getStringValue(record, "q.status"),
		}
		questions = append(questions, question)
	}

	c.JSON(http.StatusOK, questions)
}

// Relationship handlers for schema relationships
func (h *Handler) CreateDefines(c *gin.Context) {
	h.createRelationship(c, "DEFINES")
}

func (h *Handler) CreateInfluences(c *gin.Context) {
	var rel models.Relationship
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// INFLUENCES relationship needs polarity property
	polarity := "positive" // default
	if rel.Data != nil && rel.Data["polarity"] != nil {
		polarity = rel.Data["polarity"].(string)
	}

	query := `MATCH (from), (to) 
			  WHERE from.id = $from AND to.id = $to 
			  CREATE (from)-[:INFLUENCES {polarity: $polarity}]->(to)`
	params := map[string]interface{}{
		"from":     rel.From,
		"to":       rel.To,
		"polarity": polarity,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "INFLUENCES relationship created", "from": rel.From, "to": rel.To, "polarity": polarity})
}

func (h *Handler) CreateSupports(c *gin.Context) {
	h.createRelationship(c, "SUPPORTS")
}

func (h *Handler) CreateContradicts(c *gin.Context) {
	h.createRelationship(c, "CONTRADICTS")
}

func (h *Handler) CreateDerivedFrom(c *gin.Context) {
	h.createRelationship(c, "DERIVED_FROM")
}

func (h *Handler) CreateRaises(c *gin.Context) {
	h.createRelationship(c, "RAISES")
}

// Generic relationship creator
func (h *Handler) createRelationship(c *gin.Context, relType string) {
	var rel models.Relationship
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (from), (to) 
			  WHERE from.id = $from AND to.id = $to 
			  CREATE (from)-[:` + relType + `]->(to)`
	params := map[string]interface{}{
		"from": rel.From,
		"to":   rel.To,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": relType + " relationship created", "from": rel.From, "to": rel.To})
}
