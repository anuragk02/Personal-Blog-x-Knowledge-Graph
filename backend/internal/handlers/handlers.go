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

func (h *Handler) UpdateConcept(c *gin.Context) {
	id := c.Param("id")
	var concept models.Concept
	if err := c.ShouldBindJSON(&concept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (c:Concept {id: $id})
			  SET c.name = $name, c.summary = $summary, c.mastery_level = $mastery_level, c.last_reviewed = $last_reviewed
			  RETURN c.id, c.name, c.summary, c.mastery_level, c.last_reviewed`
	params := map[string]interface{}{
		"id":            id,
		"name":          concept.Name,
		"summary":       concept.Summary,
		"mastery_level": concept.MasteryLevel,
		"last_reviewed": time.Now().Format(time.RFC3339),
	}

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
	updatedConcept := models.Concept{
		ID:           record["c.id"].(string),
		Name:         record["c.name"].(string),
		Summary:      getStringValue(record, "c.summary"),
		MasteryLevel: getIntValue(record, "c.mastery_level"),
	}

	if lastReviewedStr := getStringValue(record, "c.last_reviewed"); lastReviewedStr != "" {
		if lastReviewed, err := time.Parse(time.RFC3339, lastReviewedStr); err == nil {
			updatedConcept.LastReviewed = lastReviewed
		}
	}

	c.JSON(http.StatusOK, updatedConcept)
}

func (h *Handler) DeleteConcept(c *gin.Context) {
	id := c.Param("id")

	// First check if concept exists
	checkQuery := `MATCH (c:Concept {id: $id}) RETURN c.id`
	checkParams := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), checkQuery, checkParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	// Delete concept and all its relationships
	deleteQuery := `MATCH (c:Concept {id: $id}) DETACH DELETE c`
	deleteParams := map[string]interface{}{"id": id}

	_, err = h.db.ExecuteQuery(context.Background(), deleteQuery, deleteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Concept deleted successfully"})
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

func (h *Handler) UpdateEssay(c *gin.Context) {
	id := c.Param("id")
	var essay models.Essay
	if err := c.ShouldBindJSON(&essay); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (e:Essay {id: $id})
			  SET e.title = $title, e.content = $content
			  RETURN e.id, e.title, e.content, e.created_at`
	params := map[string]interface{}{
		"id":      id,
		"title":   essay.Title,
		"content": essay.Content,
	}

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
	updatedEssay := models.Essay{
		ID:      record["e.id"].(string),
		Title:   record["e.title"].(string),
		Content: record["e.content"].(string),
	}

	if createdAtStr := getStringValue(record, "e.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedEssay.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedEssay)
}

func (h *Handler) DeleteEssay(c *gin.Context) {
	id := c.Param("id")

	// First check if essay exists
	checkQuery := `MATCH (e:Essay {id: $id}) RETURN e.id`
	checkParams := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), checkQuery, checkParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Essay not found"})
		return
	}

	// Delete essay and all its relationships
	deleteQuery := `MATCH (e:Essay {id: $id}) DETACH DELETE e`
	deleteParams := map[string]interface{}{"id": id}

	_, err = h.db.ExecuteQuery(context.Background(), deleteQuery, deleteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Essay deleted successfully"})
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

func (h *Handler) UpdateClaim(c *gin.Context) {
	id := c.Param("id")
	var claim models.Claim
	if err := c.ShouldBindJSON(&claim); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (cl:Claim {id: $id})
			  SET cl.text = $text, cl.confidence_score = $confidence_score, cl.is_verified = $is_verified
			  RETURN cl.id, cl.text, cl.confidence_score, cl.is_verified`
	params := map[string]interface{}{
		"id":               id,
		"text":             claim.Text,
		"confidence_score": claim.ConfidenceScore,
		"is_verified":      claim.IsVerified,
	}

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
	updatedClaim := models.Claim{
		ID:              record["cl.id"].(string),
		Text:            record["cl.text"].(string),
		ConfidenceScore: getIntValue(record, "cl.confidence_score"),
		IsVerified:      getBoolValue(record, "cl.is_verified"),
	}

	c.JSON(http.StatusOK, updatedClaim)
}

func (h *Handler) DeleteClaim(c *gin.Context) {
	id := c.Param("id")

	// First check if claim exists
	checkQuery := `MATCH (cl:Claim {id: $id}) RETURN cl.id`
	checkParams := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), checkQuery, checkParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Claim not found"})
		return
	}

	// Delete claim and all its relationships
	deleteQuery := `MATCH (cl:Claim {id: $id}) DETACH DELETE cl`
	deleteParams := map[string]interface{}{"id": id}

	_, err = h.db.ExecuteQuery(context.Background(), deleteQuery, deleteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Claim deleted successfully"})
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

func (h *Handler) UpdateSource(c *gin.Context) {
	id := c.Param("id")
	var source models.Source
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (s:Source {id: $id})
			  SET s.type = $type, s.title = $title, s.author = $author, s.url = $url
			  RETURN s.id, s.type, s.title, s.author, s.url, s.date_added`
	params := map[string]interface{}{
		"id":     id,
		"type":   source.Type,
		"title":  source.Title,
		"author": source.Author,
		"url":    source.URL,
	}

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
	updatedSource := models.Source{
		ID:     record["s.id"].(string),
		Type:   record["s.type"].(string),
		Title:  record["s.title"].(string),
		Author: getStringValue(record, "s.author"),
		URL:    getStringValue(record, "s.url"),
	}

	if dateStr := getStringValue(record, "s.date_added"); dateStr != "" {
		if dateAdded, err := time.Parse(time.RFC3339, dateStr); err == nil {
			updatedSource.DateAdded = dateAdded
		}
	}

	c.JSON(http.StatusOK, updatedSource)
}

func (h *Handler) DeleteSource(c *gin.Context) {
	id := c.Param("id")

	// First check if source exists
	checkQuery := `MATCH (s:Source {id: $id}) RETURN s.id`
	checkParams := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), checkQuery, checkParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	}

	// Delete source and all its relationships
	deleteQuery := `MATCH (s:Source {id: $id}) DETACH DELETE s`
	deleteParams := map[string]interface{}{"id": id}

	_, err = h.db.ExecuteQuery(context.Background(), deleteQuery, deleteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source deleted successfully"})
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

func (h *Handler) UpdateQuestion(c *gin.Context) {
	id := c.Param("id")
	var question models.Question
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (q:Question {id: $id})
			  SET q.text = $text, q.priority = $priority, q.status = $status
			  RETURN q.id, q.text, q.priority, q.status`
	params := map[string]interface{}{
		"id":       id,
		"text":     question.Text,
		"priority": question.Priority,
		"status":   question.Status,
	}

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
	updatedQuestion := models.Question{
		ID:       record["q.id"].(string),
		Text:     record["q.text"].(string),
		Priority: getIntValue(record, "q.priority"),
		Status:   getStringValue(record, "q.status"),
	}

	c.JSON(http.StatusOK, updatedQuestion)
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	id := c.Param("id")

	// First check if question exists
	checkQuery := `MATCH (q:Question {id: $id}) RETURN q.id`
	checkParams := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), checkQuery, checkParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	// Delete question and all its relationships
	deleteQuery := `MATCH (q:Question {id: $id}) DETACH DELETE q`
	deleteParams := map[string]interface{}{"id": id}

	_, err = h.db.ExecuteQuery(context.Background(), deleteQuery, deleteParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted successfully"})
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

// ======================
// ANALYTICAL ENDPOINTS FOR AGENTIC SYSTEMS
// ======================

// Search across all node types
func (h *Handler) SearchKnowledge(c *gin.Context) {
	searchTerm := c.Query("q")
	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search term 'q' is required"})
		return
	}

	query := `
		MATCH (n) 
		WHERE n.name CONTAINS $term 
		   OR n.title CONTAINS $term 
		   OR n.text CONTAINS $term 
		   OR n.content CONTAINS $term
		   OR n.summary CONTAINS $term
		RETURN labels(n)[0] as type, n.id as id, 
		       COALESCE(n.name, n.title, n.text, 'Unknown') as title,
		       COALESCE(n.summary, n.content, '') as content
		LIMIT 50`
	params := map[string]interface{}{"term": searchTerm}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var results []map[string]interface{}
	for _, record := range records {
		results = append(results, map[string]interface{}{
			"type":    record["type"],
			"id":      record["id"],
			"title":   record["title"],
			"content": record["content"],
		})
	}

	// Ensure results is never null in JSON response
	if results == nil {
		results = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

// Get knowledge graph statistics
func (h *Handler) GetKnowledgeStats(c *gin.Context) {
	statsQuery := `
		MATCH (n) 
		WITH labels(n)[0] as nodeType, count(n) as nodeCount
		RETURN nodeType, nodeCount
		UNION ALL
		MATCH ()-[r]->()
		WITH type(r) as relType, count(r) as relCount
		RETURN relType as nodeType, relCount as nodeCount`

	records, err := h.db.ExecuteRead(context.Background(), statsQuery, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats := map[string]interface{}{
		"nodes":         make(map[string]interface{}),
		"relationships": make(map[string]interface{}),
	}

	for _, record := range records {
		nodeType := record["nodeType"].(string)
		count := record["nodeCount"]

		// Distinguish between node types and relationship types
		if nodeType == "Concept" || nodeType == "Essay" || nodeType == "Claim" || nodeType == "Source" || nodeType == "Question" {
			stats["nodes"].(map[string]interface{})[nodeType] = count
		} else {
			stats["relationships"].(map[string]interface{})[nodeType] = count
		}
	}

	c.JSON(http.StatusOK, stats)
}

// Find paths between two nodes
func (h *Handler) FindPath(c *gin.Context) {
	fromId := c.Query("from")
	toId := c.Query("to")
	maxDepth := c.DefaultQuery("depth", "3")

	if fromId == "" || toId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both 'from' and 'to' parameters are required"})
		return
	}

	query := `
		MATCH path = shortestPath((from {id: $from})-[*1..` + maxDepth + `]-(to {id: $to}))
		RETURN [node in nodes(path) | {id: node.id, type: labels(node)[0], name: COALESCE(node.name, node.title, node.text)}] as nodes,
		       [rel in relationships(path) | type(rel)] as relationships,
		       length(path) as pathLength`
	params := map[string]interface{}{
		"from": fromId,
		"to":   toId,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusOK, gin.H{"path": nil, "message": "No path found"})
		return
	}

	record := records[0]
	c.JSON(http.StatusOK, gin.H{
		"nodes":         record["nodes"],
		"relationships": record["relationships"],
		"pathLength":    record["pathLength"],
	})
}

// Get node with its immediate neighborhood
func (h *Handler) GetNodeNeighborhood(c *gin.Context) {
	nodeId := c.Param("id")
	depth := c.DefaultQuery("depth", "1")

	query := `
		MATCH (center {id: $nodeId})
		OPTIONAL MATCH path = (center)-[*1..` + depth + `]-(neighbor)
		WITH center, collect(DISTINCT neighbor) as neighbors, 
		     collect(DISTINCT [rel in relationships(path) | {type: type(rel), from: startNode(rel).id, to: endNode(rel).id}]) as pathRels
		RETURN {
			id: center.id, 
			type: labels(center)[0], 
			name: COALESCE(center.name, center.title, center.text),
			properties: properties(center)
		} as centerNode,
		[n in neighbors | {
			id: n.id, 
			type: labels(n)[0], 
			name: COALESCE(n.name, n.title, n.text)
		}] as neighbors,
		REDUCE(rels = [], relList in pathRels | rels + relList) as relationships`
	params := map[string]interface{}{"nodeId": nodeId}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	record := records[0]
	c.JSON(http.StatusOK, gin.H{
		"center":        record["centerNode"],
		"neighbors":     record["neighbors"],
		"relationships": record["relationships"],
	})
}

// Discover patterns and insights
func (h *Handler) GetKnowledgeInsights(c *gin.Context) {
	insights := make(map[string]interface{})

	// Most connected concepts
	connectedQuery := `
		MATCH (c:Concept)-[r]-()
		WITH c, count(r) as connections
		ORDER BY connections DESC
		LIMIT 5
		RETURN collect({id: c.id, name: c.name, connections: connections}) as mostConnected`

	records, err := h.db.ExecuteRead(context.Background(), connectedQuery, nil)
	if err == nil && len(records) > 0 {
		insights["mostConnectedConcepts"] = records[0]["mostConnected"]
	}

	// Unverified claims
	unverifiedQuery := `
		MATCH (cl:Claim {is_verified: false})
		RETURN count(cl) as unverifiedCount`

	records, err = h.db.ExecuteRead(context.Background(), unverifiedQuery, nil)
	if err == nil && len(records) > 0 {
		insights["unverifiedClaims"] = records[0]["unverifiedCount"]
	}

	// Open questions
	openQuestionsQuery := `
		MATCH (q:Question)
		WHERE q.status = 'open' OR q.status = ''
		RETURN count(q) as openQuestions`

	records, err = h.db.ExecuteRead(context.Background(), openQuestionsQuery, nil)
	if err == nil && len(records) > 0 {
		insights["openQuestions"] = records[0]["openQuestions"]
	}

	// Knowledge gaps (concepts without sources)
	gapsQuery := `
		MATCH (c:Concept)
		WHERE NOT (c)-[:DERIVED_FROM]->(:Source)
		RETURN count(c) as conceptsWithoutSources`

	records, err = h.db.ExecuteRead(context.Background(), gapsQuery, nil)
	if err == nil && len(records) > 0 {
		insights["conceptsWithoutSources"] = records[0]["conceptsWithoutSources"]
	}

	c.JSON(http.StatusOK, insights)
}
