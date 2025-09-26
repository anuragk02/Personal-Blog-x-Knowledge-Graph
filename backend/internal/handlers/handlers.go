package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anuragk02/jna-nuh-yoh-guh/internal/database"
	"github.com/anuragk02/jna-nuh-yoh-guh/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
func (h *Handler) CreateNarrativeNode(c *gin.Context) {
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
func (h *Handler) GetNarrativeByID(c *gin.Context) {
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

// Get Narratives - Reads all narratives
func (h *Handler) GetNarratives(c *gin.Context) {
	query := `MATCH (n:Narrative) 
			  RETURN n.id, n.title, n.content, n.created_at, n.updated_at`
	params := map[string]interface{}{}

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var narratives []models.Narrative
	for _, record := range records {
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

		narratives = append(narratives, narrative)
	}

	c.JSON(http.StatusOK, narratives)
}

// UpdateNarrative - Updates an existing narrative
func (h *Handler) UpdateNarrativeNode(c *gin.Context) {
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
func (h *Handler) DeleteNarrativeNode(c *gin.Context) {
	id := c.Param("id")
	query := `MATCH (n:Narrative {id: $id}) DETACH DELETE n`
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
func (h *Handler) CreateSystemNode(c *gin.Context) {
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

func (h *Handler) CreateCausalLink(c *gin.Context) {
	var req models.CausalLink
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// ... (add validation for FromType/ToType)

	// Build the query to MATCH two nodes and CREATE a relationship between them
	query := `
        MATCH (a), (b)
        WHERE a.id = $from_id AND b.id = $to_id
        CREATE (a)-[r:CAUSAL_LINK {
            question: $question,
            curiosity_score: $curiosity_score,
            created_at: $created_at
        }]->(b)
    `
	params := map[string]interface{}{
		"from_id":         req.FromID,
		"to_id":           req.ToID,
		"question":        req.Question,
		"curiosity_score": req.CuriosityScore,
		"created_at":      time.Now().Format(time.RFC3339),
	}

	_, err := h.db.ExecuteQuery(context.Background(), query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req) // Return the request data as confirmation
}

// AnalyzeNarrative takes a narrative ID in the request body, sends its content to an LLM for analysis,
// and executes the returned plan to build out the knowledge graph.
func (h *Handler) AnalyzeNarrative(c *gin.Context) {
	var req models.AnalyzeNarrativeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.NarrativeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Narrative ID is required in the request body"})
		return
	}

	// --- Step 1: Get API Key and Narrative Content ---
	geminiApiKey := os.Getenv("GEMINI_API_KEY")
	if geminiApiKey == "" {
		log.Println("ERROR: GEMINI_API_KEY environment variable not set.")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server configuration error: missing API key"})
		return
	}

	narrative, err := h.getNarrativeByIDFromDB(c.Request.Context(), req.NarrativeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Narrative with ID '%s' not found", req.NarrativeID)})
		return
	}

	// --- Step 2: Build and Send Request to Gemini API ---
	llmApiUrl := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"
	prompt := buildLLMPrompt(narrative.Content)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]string{"response_mime_type": "application/json"},
	}
	llmReqBody, _ := json.Marshal(payload)

	httpRequest, err := http.NewRequestWithContext(c.Request.Context(), "POST", llmApiUrl, bytes.NewBuffer(llmReqBody))
	if err != nil {
		log.Printf("ERROR: Failed to create Gemini request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request to LLM service"})
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-goog-api-key", geminiApiKey)

	client := &http.Client{}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		log.Printf("ERROR: Gemini API request failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Could not connect to the LLM service"})
		return
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		log.Printf("ERROR: Gemini API returned non-200 status: %d", httpResponse.StatusCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("LLM service returned status code %d", httpResponse.StatusCode)})
		return
	}

	// --- Step 3: Parse Gemini API Response ---
	var geminiAPIResponse struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(httpResponse.Body).Decode(&geminiAPIResponse); err != nil {
		log.Printf("ERROR: Failed to decode Gemini API response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response from LLM service"})
		return
	}
	if len(geminiAPIResponse.Candidates) == 0 || len(geminiAPIResponse.Candidates[0].Content.Parts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM service returned no content"})
		return
	}

	llmPlanJSON := geminiAPIResponse.Candidates[0].Content.Parts[0].Text
	var llmPlan models.LLMResponse
	if err := json.Unmarshal([]byte(llmPlanJSON), &llmPlan); err != nil {
		log.Printf("ERROR: Failed to unmarshal LLM plan from content string: %v. Content was: %s", err, llmPlanJSON)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse LLM's structured plan"})
		return
	}

	// --- Step 4 & 5: Execute the Plan (Two-Pass Orchestration) ---
	systemIDs, stockIDs, flowIDs := make(map[string]string), make(map[string]string), make(map[string]string)

	// PASS 1: Create All Nodes
	for _, action := range llmPlan.Actions {
		params := action.Parameters
		switch action.FunctionName {
		case "CreateSystemNode":
			name, ok1 := params["name"].(string)
			desc, ok2 := params["boundaryDescription"].(string)
			if !ok1 || !ok2 {
				log.Printf("Warning: Skipping CreateSystemNode due to malformed parameters: %+v", params)
				continue
			}
			system, err := h.createSystemInDB(c.Request.Context(), models.SystemRequest{Name: name, BoundaryDescription: desc})
			if err != nil {
				log.Printf("Error creating system '%s': %v", name, err)
				continue
			}
			systemIDs[name] = system.ID
		case "CreateStockNode":
			name, ok1 := params["name"].(string)
			desc, ok2 := params["description"].(string)
			stockType, ok3 := params["type"].(string)
			if !ok1 || !ok2 || !ok3 {
				log.Printf("Warning: Skipping CreateStockNode due to malformed parameters: %+v", params)
				continue
			}
			stock, err := h.createStockInDB(c.Request.Context(), models.StockRequest{Name: name, Description: desc, Type: stockType})
			if err != nil {
				log.Printf("Error creating stock '%s': %v", name, err)
				continue
			}
			stockIDs[name] = stock.ID
		case "CreateFlowNode":
			name, ok1 := params["name"].(string)
			desc, ok2 := params["description"].(string)
			if !ok1 || !ok2 {
				log.Printf("Warning: Skipping CreateFlowNode due to malformed parameters: %+v", params)
				continue
			}
			flow, err := h.createFlowInDB(c.Request.Context(), models.FlowRequest{Name: name, Description: desc})
			if err != nil {
				log.Printf("Error creating flow '%s': %v", name, err)
				continue
			}
			flowIDs[name] = flow.ID
		}
	}

	// PASS 2: Create All Relationships
	for _, action := range llmPlan.Actions {
		params := action.Parameters
		switch action.FunctionName {
		case "CreateDescribesRelationship":
			systemName, ok := params["systemName"].(string)
			if !ok {
				continue
			}
			if systemID, ok := systemIDs[systemName]; ok {
				h.createDescribesRelationshipInDB(c.Request.Context(), req.NarrativeID, systemID)
			}
		case "CreateConstitutesRelationship":
			subsystemName, ok1 := params["subsystemName"].(string)
			systemName, ok2 := params["systemName"].(string)
			if !ok1 || !ok2 {
				continue
			}
			if subsystemID, ok1 := systemIDs[subsystemName]; ok1 {
				if systemID, ok2 := systemIDs[systemName]; ok2 {
					h.createConstitutesRelationshipInDB(c.Request.Context(), subsystemID, systemID)
				}
			}
		case "CreateDescribesStaticRelationship":
			stockName, ok1 := params["stockName"].(string)
			systemName, ok2 := params["systemName"].(string)
			if !ok1 || !ok2 {
				continue
			}
			if stockID, ok1 := stockIDs[stockName]; ok1 {
				if systemID, ok2 := systemIDs[systemName]; ok2 {
					h.createDescribesStaticRelationshipInDB(c.Request.Context(), stockID, systemID)
				}
			}
		case "CreateChangesRelationship":
			flowName, ok1 := params["flowName"].(string)
			stockName, ok2 := params["stockName"].(string)
			polarity, ok3 := params["polarity"].(float64)
			if !ok1 || !ok2 || !ok3 {
				continue
			}
			if flowID, ok1 := flowIDs[flowName]; ok1 {
				if stockID, ok2 := stockIDs[stockName]; ok2 {
					h.createChangesRelationshipInDB(c.Request.Context(), flowID, stockID, float32(polarity))
				}
			}
		case "CreateCausalLinkRelationship":
			fromName, ok1 := params["fromName"].(string)
			fromType, ok2 := params["fromType"].(string)
			toName, ok3 := params["toName"].(string)
			toType, ok4 := params["toType"].(string)
			question, ok5 := params["curiosity"].(string)
			score, ok6 := params["curiosityScore"].(float64)
			if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
				continue
			}
			fromID, toID := getIDFromNameAndType(fromName, fromType, stockIDs, flowIDs), getIDFromNameAndType(toName, toType, stockIDs, flowIDs)
			if fromID != "" && toID != "" {
				linkReq := models.CausalLink{FromID: fromID, FromType: fromType, ToID: toID, ToType: toType, Question: question, CuriosityScore: float32(score)}
				h.createCausalLinkInDB(c.Request.Context(), linkReq)
			}
		}
	}

	// --- Step 6: Final Response ---
	c.JSON(http.StatusOK, gin.H{
		"message":         "Narrative analysis completed successfully",
		"narrativeId":     req.NarrativeID,
		"systems_created": len(systemIDs),
		"stocks_created":  len(stockIDs),
		"flows_created":   len(flowIDs),
	})
}

// buildLLMPrompt is a helper to construct the full prompt for the LLM.
func buildLLMPrompt(narrativeContent string) string {
	const promptTemplate = `
	**1. Your Role and Mission**
	You are a Cognitive Scientist specializing in knowledge modeling. Your mission is to help a friend understand their own thinking by modeling their beliefs, as revealed through their writing, into a Network of Nodes and Relationships. Your analytical framework is Systems Thinking. You see the world as a network of nested Systems, which are described by their Stocks (state variables) and dynamic interactions (Flows). Your goal is to map the writer's understanding of these systems, their parts, and their interactions into a graph database. You must remain detached and abstract. All beliefs about specific people should be generalized to describe the "Human System." You are mapping their knowledge of the world, not their personal diary.
	**2. Core Principles of Analysis**
	- **Principle of Abstraction**: Generalize specific anecdotes into models of broader systems (e.g., a story about a specific job becomes an analysis of the "Work-Life Balance System").
	- **Identify the 'Container' (The System)**: Find the nouns representing a collection of interacting parts (e.g., "economy," "ecosystem," "team").
	- **Identify 'State Variables' (The Stocks)**: Find the nouns representing accumulations or qualities that describe the system's state at a point in time (e.g., "Trust Level," "Bank Balance," "Motivation"). Classify them as 'quantitative' or 'qualitative'.
	- **Identify 'Activities' (The Flows)**: Find the verbs or actions that cause stocks to change over time (e.g., "earning income," "building trust," "spending energy").
	- **Pinpoint Curiosity (The Causal Link)**: This is crucial. You must identify and score the writer's curiosity about causal relationships between Stocks and/or Flows. Score 2.0 for direct questions ("I wonder if...", "?"). Score 1.0 for uncertainty ("It seems like...", "Perhaps..."). Score 0.0 for assertions without explanation.
	**3. Function API**
	You will call these functions to build the graph:
	- ` + "`CreateSystemNode(name: string, boundaryDescription: string)`" + `
	- ` + "`CreateStockNode(name: string, description: string, type: string)`" + `
	- ` + "`CreateFlowNode(name: string, description: string)`" + `
	- ` + "`CreateDescribesRelationship(systemName: string)`" + `
	- ` + "`CreateConstitutesRelationship(subsystemName: string, systemName: string)`" + `
	- ` + "`CreateDescribesStaticRelationship(stockName: string, systemName: string)`" + `
	- ` + "`CreateChangesRelationship(flowName: string, stockName: string, polarity: float)`" + `
	- ` + "`CreateCausalLinkRelationship(fromType: string, fromName: string, toType: string, toName: string, curiosity: string, curiosityScore: float)`" + `
	**4. Your Task**
	Your output must be a single JSON object with a key named "actions". The value should be an array of objects, where each object represents a function call with "function_name" and "parameters" keys. Do not provide any other explanatory text. Analyze the following narrative:
	--- NARRATIVE START ---
	%s
	--- NARRATIVE END ---
	`
	return fmt.Sprintf(promptTemplate, narrativeContent)
}

// getIDFromNameAndType is a helper to find an ID from the correct map.
func getIDFromNameAndType(name, nodeType string, stockIDs, flowIDs map[string]string) string {
	if strings.EqualFold(nodeType, "Stock") {
		if id, ok := stockIDs[name]; ok {
			return id
		}
	}
	if strings.EqualFold(nodeType, "Flow") {
		if id, ok := flowIDs[name]; ok {
			return id
		}
	}
	log.Printf("Warning: Could not find ID for name '%s' of type '%s'", name, nodeType)
	return ""
}

// ====== DATABASE LOGIC HELPERS ======
// These functions contain the core database logic, making them reusable.

func (h *Handler) getNarrativeByIDFromDB(ctx context.Context, id string) (*models.Narrative, error) {
	query := `MATCH (n:Narrative {id: $id}) RETURN n.id, n.title, n.content, n.created_at`
	params := map[string]interface{}{"id": id}
	records, err := h.db.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("narrative not found")
	}
	record := records[0]
	narrative := &models.Narrative{
		ID:      record["n.id"].(string),
		Title:   record["n.title"].(string),
		Content: record["n.content"].(string),
	}
	return narrative, nil
}

func (h *Handler) createSystemInDB(ctx context.Context, req models.SystemRequest) (*models.System, error) {
	system := &models.System{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		BoundaryDescription: req.BoundaryDescription,
		CreatedAt:           time.Now(),
	}
	query := `CREATE (s:System {id: $id, name: $name, boundary_description: $boundary_description, created_at: $created_at})`
	params := map[string]interface{}{
		"id":                   system.ID,
		"name":                 system.Name,
		"boundary_description": system.BoundaryDescription,
		"created_at":           system.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return system, err
}

func (h *Handler) createStockInDB(ctx context.Context, req models.StockRequest) (*models.Stock, error) {
	stock := &models.Stock{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		CreatedAt:   time.Now(),
	}
	query := `CREATE (st:Stock {id: $id, name: $name, description: $description, type: $type, created_at: $created_at})`
	params := map[string]interface{}{
		"id":          stock.ID,
		"name":        stock.Name,
		"description": stock.Description,
		"type":        stock.Type,
		"created_at":  stock.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return stock, err
}

func (h *Handler) createFlowInDB(ctx context.Context, req models.FlowRequest) (*models.Flow, error) {
	flow := &models.Flow{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	query := `CREATE (f:Flow {id: $id, name: $name, description: $description, created_at: $created_at})`
	params := map[string]interface{}{
		"id":          flow.ID,
		"name":        flow.Name,
		"description": flow.Description,
		"created_at":  flow.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return flow, err
}

func (h *Handler) createDescribesRelationshipInDB(ctx context.Context, narrativeID, systemID string) error {
	query := `MATCH (n:Narrative {id: $narrative_id}), (s:System {id: $system_id}) CREATE (n)-[:DESCRIBES]->(s)`
	params := map[string]interface{}{"narrative_id": narrativeID, "system_id": systemID}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createConstitutesRelationshipInDB(ctx context.Context, subsystemID, systemID string) error {
	query := `MATCH (sub:System {id: $subsystem_id}), (sys:System {id: $system_id}) CREATE (sub)-[:CONSTITUTES]->(sys)`
	params := map[string]interface{}{"subsystem_id": subsystemID, "system_id": systemID}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createDescribesStaticRelationshipInDB(ctx context.Context, stockID, systemID string) error {
	query := `MATCH (st:Stock {id: $stock_id}), (s:System {id: $system_id}) CREATE (st)-[:DESCRIBES_STATIC]->(s)`
	params := map[string]interface{}{"stock_id": stockID, "system_id": systemID}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createChangesRelationshipInDB(ctx context.Context, flowID, stockID string, polarity float32) error {
	query := `MATCH (f:Flow {id: $flow_id}), (st:Stock {id: $stock_id}) CREATE (f)-[:CHANGES {polarity: $polarity}]->(st)`
	params := map[string]interface{}{"flow_id": flowID, "stock_id": stockID, "polarity": polarity}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createCausalLinkInDB(ctx context.Context, req models.CausalLink) error {
	query := `MATCH (a), (b) WHERE a.id = $from_id AND b.id = $to_id CREATE (a)-[r:CAUSAL_LINK {question: $question, curiosity_score: $curiosity_score, created_at: $created_at}]->(b)`
	params := map[string]interface{}{
		"from_id":         req.FromID,
		"to_id":           req.ToID,
		"question":        req.Question,
		"curiosity_score": req.CuriosityScore,
		"created_at":      time.Now().Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

// CleanNonNarrativeData - Deletes all nodes and relationships except for Narratives.
// This is a utility function for resetting the knowledge graph without deleting the source material.
func (h *Handler) CleanNonNarrativeData(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Count nodes to be deleted for reporting purposes.
	countQuery := `
        MATCH (n)
        WHERE NOT n:Narrative
        RETURN count(n) as nodes_to_delete
    `
	records, err := h.db.ExecuteRead(ctx, countQuery, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count nodes for deletion: " + err.Error()})
		return
	}

	var nodesToDelete int64
	if len(records) > 0 {
		if count, ok := records[0]["nodes_to_delete"].(int64); ok {
			nodesToDelete = count
		}
	}

	// 2. Perform the actual deletion.
	// DETACH DELETE removes the nodes and any relationships connected to them atomically.
	deleteQuery := `
        MATCH (n)
        WHERE NOT n:Narrative
        DETACH DELETE n
    `
	if _, err = h.db.ExecuteQuery(ctx, deleteQuery, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete non-narrative nodes: " + err.Error()})
		return
	}

	// 3. Verify the number of remaining Narratives as a final check.
	narrativeCountQuery := `
        MATCH (n:Narrative)
        RETURN count(n) as narratives_remaining
    `
	narrativeRecords, err := h.db.ExecuteRead(ctx, narrativeCountQuery, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count remaining narratives: " + err.Error()})
		return
	}

	var narrativesRemaining int64
	if len(narrativeRecords) > 0 {
		if count, ok := narrativeRecords[0]["narratives_remaining"].(int64); ok {
			narrativesRemaining = count
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "Successfully cleaned all non-narrative data",
		"nodes_deleted":        nodesToDelete,
		"narratives_preserved": narrativesRemaining,
	})
}
