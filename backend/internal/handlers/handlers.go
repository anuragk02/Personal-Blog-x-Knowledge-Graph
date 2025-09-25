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

// Health check handler
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// =============================================================================
// NARRATIVE HANDLERS - FOR WRITING PROCESS AND LLM WORKFLOW
// =============================================================================

// CreateNarrative - Creates a new narrative with auto-generated ID
func (h *Handler) CreateNarrative(c *gin.Context) {
	var req models.NarrativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate unique ID using type_timestamp format
	narrative := models.Narrative{
		ID:      fmt.Sprintf("narrative_%d", time.Now().UnixNano()),
		Title:   req.Title,
		Content: req.Content,
	}

	now := time.Now()
	narrative.CreatedAt = now
	narrative.UpdatedAt = now

	query := `CREATE (n:Narrative {
		id: $id, 
		title: $title, 
		content: $content, 
		created_at: $created_at, 
		updated_at: $updated_at
	})`
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

// GetNarrative - Reads a single narrative by ID
func (h *Handler) GetNarrative(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (n:Narrative {id: $id}) 
			  RETURN n.id, n.title, n.content, n.created_at, n.updated_at`
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

// UpdateNarrative - Updates an existing narrative
func (h *Handler) UpdateNarrative(c *gin.Context) {
	id := c.Param("id")
	var req models.NarrativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedAt := time.Now()

	query := `MATCH (n:Narrative {id: $id})
			  SET n.title = $title, n.content = $content, n.updated_at = $updated_at
			  RETURN n.id, n.title, n.content, n.created_at, n.updated_at`
	params := map[string]interface{}{
		"id":         id,
		"title":      req.Title,
		"content":    req.Content,
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

// DeleteNarrative - Deletes a narrative
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
// NODE CREATION HANDLERS - AFTER EXTRACTION USING LLM TO CREATE WORLDVIEW
// =============================================================================

// CreateSystem - Creates a new system node with auto-generated ID
func (h *Handler) CreateSystem(c *gin.Context) {
	var req models.SystemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	system := models.System{
		ID:                  fmt.Sprintf("system_%d", time.Now().UnixNano()),
		Name:                req.Name,
		BoundaryDescription: req.BoundaryDescription,
		CreatedAt:           time.Now(),
	}

	query := `CREATE (s:System {
		id: $id, 
		name: $name, 
		boundary_description: $boundary_description, 
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"id":                   system.ID,
		"name":                 system.Name,
		"boundary_description": system.BoundaryDescription,
		"created_at":           system.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, system)
}

// CreateStock - Creates a new stock node with auto-generated ID
func (h *Handler) CreateStock(c *gin.Context) {
	var req models.StockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Stock.Type
	if req.Type != "qualitative" && req.Type != "quantitative" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stock type must be either 'qualitative' or 'quantitative'"})
		return
	}

	stock := models.Stock{
		ID:          fmt.Sprintf("stock_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		CreatedAt:   time.Now(),
	}

	query := `CREATE (st:Stock {
		id: $id, 
		name: $name, 
		description: $description, 
		type: $type, 
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"id":          stock.ID,
		"name":        stock.Name,
		"description": stock.Description,
		"type":        stock.Type,
		"created_at":  stock.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, stock)
}

// CreateFlow - Creates a new flow node with auto-generated ID
func (h *Handler) CreateFlow(c *gin.Context) {
	var req models.FlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow := models.Flow{
		ID:          fmt.Sprintf("flow_%d", time.Now().UnixNano()),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	query := `CREATE (f:Flow {
		id: $id, 
		name: $name, 
		description: $description, 
		created_at: $created_at
	})`
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

// =============================================================================
// RELATIONSHIP CREATION HANDLERS
// =============================================================================

// CreateDescribesRelationship - Creates Narrative -> System relationship
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

// CreateConstitutesRelationship - Creates System -> System (subsystem) relationship
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

// CreateDescribesStaticRelationship - Creates Stock -> System relationship
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

// CreateDescribesDynamicRelationship - Creates Flow -> System relationship
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

// CreateChangesRelationship - Creates Flow -> Stock relationship with polarity
func (h *Handler) CreateChangesRelationship(c *gin.Context) {
	var changes models.Changes
	if err := c.ShouldBindJSON(&changes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate polarity
	if changes.Polarity != 1.0 && changes.Polarity != -1.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Polarity must be +1 or -1"})
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

// CreateCausalLink - Creates CausalLink relationships between Stock/Flow nodes
func (h *Handler) CreateCausalLink(c *gin.Context) {
	var req models.CausalLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate types
	if (req.FromType != "Stock" && req.FromType != "Flow") ||
		(req.ToType != "Stock" && req.ToType != "Flow") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FromType and ToType must be either 'Stock' or 'Flow'"})
		return
	}

	causalLink := models.CausalLink{
		FromID:    req.FromID,
		FromType:  req.FromType,
		ToID:      req.ToID,
		ToType:    req.ToType,
		Question:  req.Question,
		CreatedAt: time.Now(),
	}

	query := `CREATE (cl:CausalLink {
		from_id: $from_id, 
		from_type: $from_type, 
		to_id: $to_id, 
		to_type: $to_type, 
		question: $question, 
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"from_id":    causalLink.FromID,
		"from_type":  causalLink.FromType,
		"to_id":      causalLink.ToID,
		"to_type":    causalLink.ToType,
		"question":   causalLink.Question,
		"created_at": causalLink.CreatedAt.Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, causalLink)
}
