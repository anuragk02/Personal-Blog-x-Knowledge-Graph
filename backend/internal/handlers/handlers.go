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

func getFloat32Value(record map[string]interface{}, key string) float32 {
	if val, ok := record[key]; ok && val != nil {
		if f, ok := val.(float64); ok {
			return float32(f)
		}
		if f, ok := val.(float32); ok {
			return f
		}
	}
	return 0.0
}

// =============================================================================
// NARRATIVE CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateNarrative(c *gin.Context) {
	var narrative models.Narrative
	if err := c.ShouldBindJSON(&narrative); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID and set timestamps
	narrative.ID = fmt.Sprintf("narrative_%d", time.Now().Unix())
	now := time.Now()
	narrative.CreatedAt = now
	narrative.UpdatedAt = now

	query := `CREATE (n:Narrative {id: $id, title: $title, content: $content, created_at: $created_at, updated_at: $updated_at})`
	params := map[string]interface{}{
		"id":         narrative.ID,
		"title":      narrative.Title,
		"content":    narrative.Content,
		"created_at": narrative.CreatedAt.Format(time.RFC3339),
		"updated_at": narrative.UpdatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, narrative)
}

func (h *Handler) GetNarrative(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (n:Narrative {id: $id}) RETURN n.id, n.title, n.content, n.created_at, n.updated_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Narrative not found"})
		return
	}

	record := records[0]
	narrative := models.Narrative{
		ID:      record["n.id"].(string),
		Title:   record["n.title"].(string),
		Content: record["n.content"].(string),
	}

	if createdAtStr := getStringValue(record, "n.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			narrative.CreatedAt = createdAt
		}
	}

	if updatedAtStr := getStringValue(record, "n.updated_at"); updatedAtStr != "" {
		if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			narrative.UpdatedAt = updatedAt
		}
	}

	c.JSON(http.StatusOK, narrative)
}

func (h *Handler) GetNarratives(c *gin.Context) {
	query := `MATCH (n:Narrative) RETURN n.id, n.title, n.created_at, n.updated_at ORDER BY n.updated_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var narratives []map[string]interface{}
	for _, record := range records {
		narratives = append(narratives, map[string]interface{}{
			"id":         record["n.id"],
			"title":      record["n.title"],
			"created_at": record["n.created_at"],
			"updated_at": record["n.updated_at"],
		})
	}

	c.JSON(http.StatusOK, narratives)
}

func (h *Handler) UpdateNarrative(c *gin.Context) {
	id := c.Param("id")
	var narrative models.Narrative
	if err := c.ShouldBindJSON(&narrative); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedAt := time.Now()

	query := `MATCH (n:Narrative {id: $id})
			  SET n.title = $title, n.content = $content, n.updated_at = $updated_at
			  RETURN n.id, n.title, n.content, n.created_at, n.updated_at`
	params := map[string]interface{}{
		"id":         id,
		"title":      narrative.Title,
		"content":    narrative.Content,
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Narrative not found"})
		return
	}

	record := records[0]
	updatedNarrative := models.Narrative{
		ID:        record["n.id"].(string),
		Title:     record["n.title"].(string),
		Content:   record["n.content"].(string),
		UpdatedAt: updatedAt,
	}

	if createdAtStr := getStringValue(record, "n.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedNarrative.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedNarrative)
}

func (h *Handler) DeleteNarrative(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (n:Narrative {id: $id}) DELETE n`
	params := map[string]interface{}{"id": id}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Narrative deleted successfully"})
}

// =============================================================================
// SYSTEM CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateSystem(c *gin.Context) {
	var system models.System
	if err := c.ShouldBindJSON(&system); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	system.ID = fmt.Sprintf("system_%d", time.Now().Unix())
	system.CreatedAt = time.Now()

	query := `CREATE (s:System {id: $id, name: $name, boundary_description: $boundary_description, type: $type, created_at: $created_at})`
	params := map[string]interface{}{
		"id":                   system.ID,
		"name":                 system.Name,
		"boundary_description": system.BoundaryDescription,
		"type":                 system.Type,
		"created_at":           system.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, system)
}

func (h *Handler) GetSystem(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (s:System {id: $id}) RETURN s.id, s.name, s.boundary_description, s.type, s.created_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "System not found"})
		return
	}

	record := records[0]
	system := models.System{
		ID:                  record["s.id"].(string),
		Name:                record["s.name"].(string),
		BoundaryDescription: getStringValue(record, "s.boundary_description"),
		Type:                getStringValue(record, "s.type"),
	}

	if createdAtStr := getStringValue(record, "s.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			system.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, system)
}

func (h *Handler) GetSystems(c *gin.Context) {
	query := `MATCH (s:System) RETURN s.id, s.name, s.type, s.created_at ORDER BY s.created_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var systems []map[string]interface{}
	for _, record := range records {
		systems = append(systems, map[string]interface{}{
			"id":         record["s.id"],
			"name":       record["s.name"],
			"type":       record["s.type"],
			"created_at": record["s.created_at"],
		})
	}

	c.JSON(http.StatusOK, systems)
}

func (h *Handler) UpdateSystem(c *gin.Context) {
	id := c.Param("id")
	var system models.System
	if err := c.ShouldBindJSON(&system); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (s:System {id: $id})
			  SET s.name = $name, s.boundary_description = $boundary_description, s.type = $type
			  RETURN s.id, s.name, s.boundary_description, s.type, s.created_at`
	params := map[string]interface{}{
		"id":                   id,
		"name":                 system.Name,
		"boundary_description": system.BoundaryDescription,
		"type":                 system.Type,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "System not found"})
		return
	}

	record := records[0]
	updatedSystem := models.System{
		ID:                  record["s.id"].(string),
		Name:                record["s.name"].(string),
		BoundaryDescription: getStringValue(record, "s.boundary_description"),
		Type:                getStringValue(record, "s.type"),
	}

	if createdAtStr := getStringValue(record, "s.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedSystem.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedSystem)
}

func (h *Handler) DeleteSystem(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (s:System {id: $id}) DELETE s`
	params := map[string]interface{}{"id": id}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "System deleted successfully"})
}

// =============================================================================
// STOCK CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateStock(c *gin.Context) {
	var stock models.Stock
	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stock.ID = fmt.Sprintf("stock_%d", time.Now().Unix())
	stock.CreatedAt = time.Now()

	query := `CREATE (st:Stock {id: $id, name: $name, description: $description, created_at: $created_at})`
	params := map[string]interface{}{
		"id":          stock.ID,
		"name":        stock.Name,
		"description": stock.Description,
		"created_at":  stock.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, stock)
}

func (h *Handler) GetStock(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (st:Stock {id: $id}) RETURN st.id, st.name, st.description, st.created_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	record := records[0]
	stock := models.Stock{
		ID:          record["st.id"].(string),
		Name:        record["st.name"].(string),
		Description: getStringValue(record, "st.description"),
	}

	if createdAtStr := getStringValue(record, "st.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			stock.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, stock)
}

func (h *Handler) GetStocks(c *gin.Context) {
	query := `MATCH (st:Stock) RETURN st.id, st.name, st.created_at ORDER BY st.created_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var stocks []map[string]interface{}
	for _, record := range records {
		stocks = append(stocks, map[string]interface{}{
			"id":         record["st.id"],
			"name":       record["st.name"],
			"created_at": record["st.created_at"],
		})
	}

	c.JSON(http.StatusOK, stocks)
}

func (h *Handler) UpdateStock(c *gin.Context) {
	id := c.Param("id")
	var stock models.Stock
	if err := c.ShouldBindJSON(&stock); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (st:Stock {id: $id})
			  SET st.name = $name, st.description = $description
			  RETURN st.id, st.name, st.description, st.created_at`
	params := map[string]interface{}{
		"id":          id,
		"name":        stock.Name,
		"description": stock.Description,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stock not found"})
		return
	}

	record := records[0]
	updatedStock := models.Stock{
		ID:          record["st.id"].(string),
		Name:        record["st.name"].(string),
		Description: getStringValue(record, "st.description"),
	}

	if createdAtStr := getStringValue(record, "st.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedStock.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedStock)
}

func (h *Handler) DeleteStock(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (st:Stock {id: $id}) DELETE st`
	params := map[string]interface{}{"id": id}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock deleted successfully"})
}

// =============================================================================
// FLOW CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateFlow(c *gin.Context) {
	var flow models.Flow
	if err := c.ShouldBindJSON(&flow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow.ID = fmt.Sprintf("flow_%d", time.Now().Unix())
	flow.CreatedAt = time.Now()

	query := `CREATE (f:Flow {id: $id, name: $name, description: $description, created_at: $created_at})`
	params := map[string]interface{}{
		"id":          flow.ID,
		"name":        flow.Name,
		"description": flow.Description,
		"created_at":  flow.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, flow)
}

func (h *Handler) GetFlow(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (f:Flow {id: $id}) RETURN f.id, f.name, f.description, f.created_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	record := records[0]
	flow := models.Flow{
		ID:          record["f.id"].(string),
		Name:        record["f.name"].(string),
		Description: record["f.description"].(string),
	}

	if createdAtStr := getStringValue(record, "f.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			flow.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, flow)
}

func (h *Handler) GetFlows(c *gin.Context) {
	query := `MATCH (f:Flow) RETURN f.id, f.name, f.created_at ORDER BY f.created_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var flows []map[string]interface{}
	for _, record := range records {
		flows = append(flows, map[string]interface{}{
			"id":         record["f.id"],
			"name":       record["f.name"],
			"created_at": record["f.created_at"],
		})
	}

	c.JSON(http.StatusOK, flows)
}

func (h *Handler) UpdateFlow(c *gin.Context) {
	id := c.Param("id")
	var flow models.Flow
	if err := c.ShouldBindJSON(&flow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (f:Flow {id: $id})
			  SET f.name = $name, f.description = $description
			  RETURN f.id, f.name, f.description, f.created_at`
	params := map[string]interface{}{
		"id":          id,
		"name":        flow.Name,
		"description": flow.Description,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Flow not found"})
		return
	}

	record := records[0]
	updatedFlow := models.Flow{
		ID:          record["f.id"].(string),
		Name:        record["f.name"].(string),
		Description: record["f.description"].(string),
	}

	if createdAtStr := getStringValue(record, "f.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedFlow.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedFlow)
}

func (h *Handler) DeleteFlow(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (f:Flow {id: $id}) DELETE f`
	params := map[string]interface{}{"id": id}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Flow deleted successfully"})
}

// =============================================================================
// QUESTION DATA CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateQuestionData(c *gin.Context) {
	var question models.QuestionData
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	question.ID = fmt.Sprintf("question_%d", time.Now().Unix())
	now := time.Now()
	question.CreatedAt = now
	question.UpdatedAt = now

	query := `CREATE (q:QuestionData {id: $id, content: $content, status: $status, type: $type, created_at: $created_at, updated_at: $updated_at})`
	params := map[string]interface{}{
		"id":         question.ID,
		"content":    question.Content,
		"status":     question.Status,
		"type":       question.Type,
		"created_at": question.CreatedAt.Format(time.RFC3339),
		"updated_at": question.UpdatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, question)
}

func (h *Handler) GetQuestionData(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (q:QuestionData {id: $id}) RETURN q.id, q.content, q.status, q.type, q.created_at, q.updated_at`
	params := map[string]interface{}{"id": id}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "QuestionData not found"})
		return
	}

	record := records[0]
	question := models.QuestionData{
		ID:      record["q.id"].(string),
		Content: record["q.content"].(string),
		Status:  record["q.status"].(string),
		Type:    record["q.type"].(string),
	}

	if createdAtStr := getStringValue(record, "q.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			question.CreatedAt = createdAt
		}
	}

	if updatedAtStr := getStringValue(record, "q.updated_at"); updatedAtStr != "" {
		if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			question.UpdatedAt = updatedAt
		}
	}

	c.JSON(http.StatusOK, question)
}

func (h *Handler) GetQuestionDataList(c *gin.Context) {
	query := `MATCH (q:QuestionData) RETURN q.id, q.content, q.status, q.type, q.created_at, q.updated_at ORDER BY q.updated_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var questions []map[string]interface{}
	for _, record := range records {
		questions = append(questions, map[string]interface{}{
			"id":         record["q.id"],
			"content":    record["q.content"],
			"status":     record["q.status"],
			"type":       record["q.type"],
			"created_at": record["q.created_at"],
			"updated_at": record["q.updated_at"],
		})
	}

	c.JSON(http.StatusOK, questions)
}

func (h *Handler) UpdateQuestionData(c *gin.Context) {
	id := c.Param("id")
	var question models.QuestionData
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedAt := time.Now()

	query := `MATCH (q:QuestionData {id: $id})
			  SET q.content = $content, q.status = $status, q.type = $type, q.updated_at = $updated_at
			  RETURN q.id, q.content, q.status, q.type, q.created_at, q.updated_at`
	params := map[string]interface{}{
		"id":         id,
		"content":    question.Content,
		"status":     question.Status,
		"type":       question.Type,
		"updated_at": updatedAt.Format(time.RFC3339),
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "QuestionData not found"})
		return
	}

	record := records[0]
	updatedQuestion := models.QuestionData{
		ID:        record["q.id"].(string),
		Content:   record["q.content"].(string),
		Status:    record["q.status"].(string),
		Type:      record["q.type"].(string),
		UpdatedAt: updatedAt,
	}

	if createdAtStr := getStringValue(record, "q.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedQuestion.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedQuestion)
}

func (h *Handler) DeleteQuestionData(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (q:QuestionData {id: $id}) DELETE q`
	params := map[string]interface{}{"id": id}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "QuestionData deleted successfully"})
}

// =============================================================================
// CAUSAL LINK CRUD OPERATIONS
// =============================================================================

func (h *Handler) CreateCausalLink(c *gin.Context) {
	var causalLink models.CausalLink
	if err := c.ShouldBindJSON(&causalLink); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	causalLink.CreatedAt = time.Now()

	query := `CREATE (cl:CausalLink {
		from_id: $from_id, 
		from_type: $from_type, 
		to_id: $to_id, 
		to_type: $to_type, 
		polarity: $polarity, 
		confidence: $confidence, 
		stock_count: $stock_count, 
		flow_count: $flow_count, 
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"from_id":     causalLink.FromID,
		"from_type":   causalLink.FromType,
		"to_id":       causalLink.ToID,
		"to_type":     causalLink.ToType,
		"polarity":    causalLink.Polarity,
		"confidence":  causalLink.Confidence,
		"stock_count": causalLink.StockCount,
		"flow_count":  causalLink.FlowCount,
		"created_at":  causalLink.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, causalLink)
}

func (h *Handler) GetCausalLink(c *gin.Context) {
	fromID := c.Param("from_id")
	toID := c.Param("to_id")
	
	query := `MATCH (cl:CausalLink {from_id: $from_id, to_id: $to_id}) 
			  RETURN cl.from_id, cl.from_type, cl.to_id, cl.to_type, cl.polarity, cl.confidence, cl.stock_count, cl.flow_count, cl.created_at`
	params := map[string]interface{}{
		"from_id": fromID,
		"to_id":   toID,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "CausalLink not found"})
		return
	}

	record := records[0]
	causalLink := models.CausalLink{
		FromID:     record["cl.from_id"].(string),
		FromType:   record["cl.from_type"].(string),
		ToID:       record["cl.to_id"].(string),
		ToType:     record["cl.to_type"].(string),
		Polarity:   getFloat32Value(record, "cl.polarity"),
		Confidence: getFloat32Value(record, "cl.confidence"),
		StockCount: getIntValue(record, "cl.stock_count"),
		FlowCount:  getIntValue(record, "cl.flow_count"),
	}

	if createdAtStr := getStringValue(record, "cl.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			causalLink.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, causalLink)
}

func (h *Handler) GetCausalLinks(c *gin.Context) {
	query := `MATCH (cl:CausalLink) 
			  RETURN cl.from_id, cl.from_type, cl.to_id, cl.to_type, cl.polarity, cl.confidence, cl.created_at 
			  ORDER BY cl.created_at DESC`

	records, err := h.db.ExecuteRead(context.Background(), query, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var causalLinks []map[string]interface{}
	for _, record := range records {
		causalLinks = append(causalLinks, map[string]interface{}{
			"from_id":    record["cl.from_id"],
			"from_type":  record["cl.from_type"],
			"to_id":      record["cl.to_id"],
			"to_type":    record["cl.to_type"],
			"polarity":   record["cl.polarity"],
			"confidence": record["cl.confidence"],
			"created_at": record["cl.created_at"],
		})
	}

	c.JSON(http.StatusOK, causalLinks)
}

func (h *Handler) UpdateCausalLink(c *gin.Context) {
	fromID := c.Param("from_id")
	toID := c.Param("to_id")
	
	var causalLink models.CausalLink
	if err := c.ShouldBindJSON(&causalLink); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (cl:CausalLink {from_id: $from_id, to_id: $to_id})
			  SET cl.polarity = $polarity, cl.confidence = $confidence, cl.stock_count = $stock_count, cl.flow_count = $flow_count
			  RETURN cl.from_id, cl.from_type, cl.to_id, cl.to_type, cl.polarity, cl.confidence, cl.stock_count, cl.flow_count, cl.created_at`
	params := map[string]interface{}{
		"from_id":     fromID,
		"to_id":       toID,
		"polarity":    causalLink.Polarity,
		"confidence":  causalLink.Confidence,
		"stock_count": causalLink.StockCount,
		"flow_count":  causalLink.FlowCount,
	}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "CausalLink not found"})
		return
	}

	record := records[0]
	updatedCausalLink := models.CausalLink{
		FromID:     record["cl.from_id"].(string),
		FromType:   record["cl.from_type"].(string),
		ToID:       record["cl.to_id"].(string),
		ToType:     record["cl.to_type"].(string),
		Polarity:   getFloat32Value(record, "cl.polarity"),
		Confidence: getFloat32Value(record, "cl.confidence"),
		StockCount: getIntValue(record, "cl.stock_count"),
		FlowCount:  getIntValue(record, "cl.flow_count"),
	}

	if createdAtStr := getStringValue(record, "cl.created_at"); createdAtStr != "" {
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			updatedCausalLink.CreatedAt = createdAt
		}
	}

	c.JSON(http.StatusOK, updatedCausalLink)
}

func (h *Handler) DeleteCausalLink(c *gin.Context) {
	fromID := c.Param("from_id")
	toID := c.Param("to_id")
	
	query := `MATCH (cl:CausalLink {from_id: $from_id, to_id: $to_id}) DELETE cl`
	params := map[string]interface{}{
		"from_id": fromID,
		"to_id":   toID,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CausalLink deleted successfully"})
}

// =============================================================================
// RELATIONSHIP CRUD OPERATIONS (Describes, Constitutes, etc.)
// =============================================================================

func (h *Handler) CreateDescribesRelationship(c *gin.Context) {
	var describes models.Describes
	if err := c.ShouldBindJSON(&describes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (n:Narrative {id: $narrative_id}), (s:System {id: $system_id})
			  CREATE (n)-[:DESCRIBES]->(s)`
	params := map[string]interface{}{
		"narrative_id": describes.NarrativeID,
		"system_id":    describes.SystemID,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, describes)
}

func (h *Handler) CreateConstitutesRelationship(c *gin.Context) {
	var constitutes models.Constitutes
	if err := c.ShouldBindJSON(&constitutes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (sub:System {id: $subsystem_id}), (sys:System {id: $system_id})
			  CREATE (sub)-[:CONSTITUTES]->(sys)`
	params := map[string]interface{}{
		"subsystem_id": constitutes.SubsystemID,
		"system_id":    constitutes.SystemID,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, constitutes)
}

func (h *Handler) CreateDescribesStaticRelationship(c *gin.Context) {
	var describesStatic models.DescribesStatic
	if err := c.ShouldBindJSON(&describesStatic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (st:Stock {id: $stock_id}), (s:System {id: $system_id})
			  CREATE (st)-[:DESCRIBES_STATIC]->(s)`
	params := map[string]interface{}{
		"stock_id":  describesStatic.StockID,
		"system_id": describesStatic.SystemID,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, describesStatic)
}

func (h *Handler) CreateDescribesDynamicRelationship(c *gin.Context) {
	var describesDynamic models.DescribesDynamic
	if err := c.ShouldBindJSON(&describesDynamic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (f:Flow {id: $flow_id}), (s:System {id: $system_id})
			  CREATE (f)-[:DESCRIBES_DYNAMIC]->(s)`
	params := map[string]interface{}{
		"flow_id":   describesDynamic.FlowID,
		"system_id": describesDynamic.SystemID,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, describesDynamic)
}

func (h *Handler) CreateChangesRelationship(c *gin.Context) {
	var changes models.Changes
	if err := c.ShouldBindJSON(&changes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `MATCH (f:Flow {id: $flow_id}), (st:Stock {id: $stock_id})
			  CREATE (f)-[:CHANGES {polarity: $polarity}]->(st)`
	params := map[string]interface{}{
		"flow_id":  changes.FlowID,
		"stock_id": changes.StockID,
		"polarity": changes.Polarity,
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, changes)
}