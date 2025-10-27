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
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

func (h *Handler) LoginHandler(c *gin.Context) {
	var jwtSecretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
	var req models.LoginRequest
	var user models.User

	// 1. Bind the incoming JSON to the LoginRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// --- DEBUGGING: Log the received username ---
	// Check your Go console. Does this match 'anurag' EXACTLY?
	// Any whitespace? Different casing?
	log.Printf("Login attempt for username: '%s'", req.Username)

	// 2. Fetch the user from the database (Neo4j)
	// This query IS case-sensitive.
	query := `MATCH (u:User {username: $username}) 
              RETURN u.uuid, u.username, u.password`
	params := map[string]interface{}{"username": req.Username}

	// ----
	// OPTIONAL: If you want case-INSENSITIVE login, use this query instead:
	// query := `MATCH (u:User) 
	//           WHERE toLower(u.username) = toLower($username)
	//           RETURN u.uuid, u.username, u.password`
	// ----

	records, err := h.db.ExecuteRead(context.Background(), query, params)
	if err != nil {
		log.Printf("Database query error in LoginHandler: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 3. Check if user was found
	if len(records) == 0 {
		// --- DEBUGGING: This is Failure Point 1 ---
		// This means the query returned 0 rows.
		// The username in your DB does not match what was sent.
		log.Printf("Login failed: User '%s' not found.", req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 4. Populate user from database record
	record := records[0]
	user.UUID, _ = record["u.uuid"].(string)
	user.Username, _ = record["u.username"].(string)
	user.Password, _ = record["u.password"].(string)

	log.Printf("Login: Found user '%s', verifying password...", user.Username)
	log.Printf("Password lengths. DB hash: %d. Received password: %d.", len(user.Password), len(req.Password))

	// 5. Compare the stored hashed password with the incoming password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		// --- DEBUGGING: This is Failure Point 2 ---
		// This means the user was FOUND, but the password was WRONG.
		// This confirms your stored hash is incorrect for the password you sent.
		log.Printf("Login failed: Password mismatch for user '%s' '%s' '%s'.", user.Username, user.Password, req.Password)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 6. Generate the JWT token
	log.Printf("Login successful for user: %s", user.Username)

	claims := jwt.MapClaims{
		"userID":   user.UUID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		log.Println("Error signing token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	// 7. Send the token back to the user
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful!",
		"token":   tokenString,
	})
}


// AuthMiddleware creates a gin.HandlerFunc for JWT authentication
func AuthMiddleware() gin.HandlerFunc {
	var jwtSecretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
	return func(c *gin.Context) {
		// 1. Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("Auth failed: No Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// 2. Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Auth failed: Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		// 3. Parse and validate the token
		// We use jwt.Parse to validate the signature and check expiry
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Check the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the secret key (must be the same one used in LoginHandler)
			return jwtSecretKey, nil
		})

		if err != nil {
			log.Printf("Auth failed: Invalid token: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// 4. Check claims and set user ID in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Extract the userID (or whatever you put in the token)
			userID, ok := claims["userID"].(string)
			if !ok {
				log.Println("Auth failed: userID claim missing or invalid")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				return
			}

			// --- SUCCESS ---
			// Set the userID in the context for other handlers to use
			c.Set("userID", userID)
			c.Next() // Continue to the next handler
		} else {
			log.Println("Auth failed: Invalid token claims or token is invalid")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		}
	}
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
	now := time.Now()
	// Generate unique ID using type_timestamp format
	narrative := models.Narrative{
		ID:           fmt.Sprintf("narrative_%d", time.Now().UnixNano()),
		Title:        req.Title,
		Content:      req.Content,
		Extrapolated: false, // Always start as false, only set to true after LLM analysis
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	query := `CREATE (n:Narrative {
		id: $id, 
		title: $title, 
		content: $content,
		extrapolated: $extrapolated, 
		created_at: $created_at, 
		updated_at: $updated_at
	})`
	params := map[string]interface{}{
		"id":           narrative.ID,
		"title":        narrative.Title,
		"content":      narrative.Content,
		"extrapolated": narrative.Extrapolated,
		"created_at":   narrative.CreatedAt.Format(time.RFC3339),
		"updated_at":   narrative.UpdatedAt.Format(time.RFC3339),
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
			  RETURN n.id, n.title, n.content, n.extrapolated, n.created_at, n.updated_at`
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

	// Safely handle boolean conversion for extrapolated field
	if extrapolated, ok := record["n.extrapolated"].(bool); ok {
		narrative.Extrapolated = extrapolated
	} else {
		narrative.Extrapolated = false // default value if not set
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
			  RETURN n.id, n.title, n.content, n.extrapolated, n.created_at, n.updated_at`
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

		// Safely handle boolean conversion for extrapolated field
		if extrapolated, ok := record["n.extrapolated"].(bool); ok {
			narrative.Extrapolated = extrapolated
		} else {
			narrative.Extrapolated = false // default value if not set
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
	userPrompt := fmt.Sprintf(userPromptTemplate, narrative.Title, narrative.Content)

	// The new payload has a dedicated "systemInstruction" field
	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemInstruction},
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

	// Log the LLM response for debugging/analysis
	log.Printf("LLM_RESPONSE [Narrative: %s] [Timestamp: %s]: %s",
		req.NarrativeID,
		time.Now().Format(time.RFC3339),
		llmPlanJSON)

	// --- Step 4 & 5: Execute the Plan (Two-Pass Orchestration) ---
	narrativeIDs, systemIDs, stockIDs, flowIDs := make(map[string]string), make(map[string]string), make(map[string]string), make(map[string]string)
	narrativeIDs[narrative.Title] = narrative.ID // Pre-populate with existing narrative
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
			narrativeName, ok1 := params["narrativeName"].(string)
			systemName, ok2 := params["systemName"].(string)
			if !ok1 || !ok2 {
				continue
			}
			if systemID, ok2 := systemIDs[systemName]; ok2 {
				if narrativeID, ok1 := narrativeIDs[narrativeName]; ok1 {
					h.createDescribesRelationshipInDB(c.Request.Context(), narrativeID, systemID)
				}
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

	// --- Step 6: Mark narrative as extrapolated ---
	// Update the narrative to mark it as extrapolated after successful analysis
	updateQuery := `MATCH (n:Narrative {id: $id}) 
		SET n.extrapolated = true, n.updated_at = $updated_at`
	updateParams := map[string]interface{}{
		"id":         req.NarrativeID,
		"updated_at": time.Now().Format(time.RFC3339),
	}
	_, err = h.db.ExecuteQuery(context.Background(), updateQuery, updateParams)
	if err != nil {
		log.Printf("Warning: Failed to mark narrative as extrapolated: %v", err)
	}

	// --- Step 7: Final Response ---
	c.JSON(http.StatusOK, gin.H{
		"message":         "Narrative analysis completed successfully",
		"narrativeId":     req.NarrativeID,
		"systems_created": len(systemIDs),
		"stocks_created":  len(stockIDs),
		"flows_created":   len(flowIDs),
	})
}

const systemInstruction = `
1. Your Role and Mission
You are a Systems Analyst. Your mission is to analyze unstructured text to reverse-engineer the author's implicit mental model of how a system works. You will formalize their observations, beliefs, and questions into a structured graph of objective, universal components (Systems, Stocks, Flows). You must remain completely detached from the author's personal experience and focus only on the underlying mechanics they are describing.

2. Core Principles of Analysis

Principle of Universalization: Your primary task is to find the universal principle or system behind any specific anecdote. A story about a specific job is evidence for a model of a Workplace Environment. A feeling of sadness after a setback is evidence for a model of Emotional Response Systems.
Strict Naming Convention: All names for Systems, Stocks, and Flows must be objective, formal, and timeless. Avoid subjective or personal framing (e.g., use Cognitive Resource Depletion, not I was tired).
Concise Functional Descriptions: All boundaryDescription and description fields must be under 15 words and describe the component's objective function, not the author's feelings.

3. The Cognitive Workflow
You must follow these guidelines in the exact sequence of analysis:
Deconstruct & Universalize: Break the narrative into key observations. For each, state the universal principle it represents. (e.g., Observation: "I stayed up late and couldn't debug code." -> Principle: "Cognitive effort depletes a finite pool of mental energy, which is restored by rest.")
Identify Formal Systems: Based on the principles, identify the formal systems at play (Software Development Lifecycle, Human Cognitive System, etc.). Create CreateSystemNode actions.
Model System Components: Extract the formal Stocks (Mental Energy) and Flows (Cognitive Exertion, Restorative Sleep) that make up these systems. Create the CreateStockNode and CreateFlowNode actions.
Map Connections: Link components to their systems (CreateDescribesStaticRelationship) and model known mechanisms (CreateChangesRelationship).
Formulate Hypotheses: Identify the author's curiosities about how components interact and create CreateCausalLinkRelationship actions. The curiosity question must be framed as a formal research question.

Overall Follow this framework
Identify Systems: First, read the text to identify the primary containers for the narrative's dynamics. These can be concrete (Business Corporation) or abstract (Workplace Culture). Create CreateSystemNode actions and CreateConstitutesRelationship actions for any nested systems.
Link Narrative: Create a CreateDescribesRelationship action to link the source narrative to each top-level system you identified.
Identify Stocks: Next, identify the state variables that describe each system. These are the accumulations or qualities of the system. Create CreateStockNode actions and CreateDescribesStaticRelationship actions to link them to their parent system.
Identify Flows: Now, identify the processes or activities that cause stocks to change. Create CreateFlowNode actions. For each flow that directly affects a stock, create a CreateChangesRelationship action, specifying the polarity (+1.0 for increase, -1.0 for decrease).
Identify Causal Links: Finally, identify all hypothesized or uncertain connections between any two elements (Stock or Flow). For each, create a CreateCausalLinkRelationship action. You must provide a summarized curiosity question and a curiosityScore based on the following scale:
1.0 (Direct Question): Used for explicit questions (e.g., "I wonder why...", "How does...?").
0.5 (Uncertainty): Used for speculative statements (e.g., "It seems like...", "Perhaps...", "I think...").
0.1 (Assertion without Mechanism): Used for statements of causality where the "how" is not explained (e.g., "X leads to Y.").

4. Function API
You will call these functions to build the graph:

CreateSystemNode(name: string, boundaryDescription: string)
CreateDescribesRelationship(narrativeName: string, systemName: string)
CreateStockNode(name: string, description: string, type: string) (type is 'qualitative' or 'quantitative')
CreateFlowNode(name: string, description: string)
CreateConstitutesRelationship(subsystemName: string, systemName: string)
CreateDescribesStaticRelationship(stockName: string, systemName:string)
CreateChangesRelationship(flowName: string, stockName: string, polarity: float)
CreateCausalLinkRelationship(fromType: string, fromName: string, toType: string, toName: string, curiosity: string, curiosityScore: float)

5. Your Task & Output Format
Your output must be a single, valid JSON object with a key named "actions". The value must be an array of objects, where each object represents a single function call with "function__name" and "parameters" keys. Do not provide any other explanatory text. Ensure that all objects in the 'actions' array are separate and correctly formatted, with no nesting of action objects inside the parameters of other actions. The response will be parsed automatically and must be perfect.
Example valid output:
{
	"actions": [
		{
			"function_name": "CreateSystemNode",
			"parameters": { "name": "System A", "boundaryDescription": "..." }
		},
		{
			"function_name": "CreateStockNode",
			"parameters": { "name": "Stock B", "description": "...", "type": "qualitative" }
		}
	]
}
Analyze the following narrative:	
`

const userPromptTemplate = `
	Narrative Title: %s
	Narrative Content: %s
`

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
	query := `MATCH (n:Narrative {id: $id}) RETURN n.id, n.title, n.content, n.extrapolated, n.created_at`
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

	// Safely handle boolean conversion for extrapolated field
	if extrapolated, ok := record["n.extrapolated"].(bool); ok {
		narrative.Extrapolated = extrapolated
	} else {
		narrative.Extrapolated = false // default value if not set
	}

	return narrative, nil
}

func (h *Handler) createSystemInDB(ctx context.Context, req models.SystemRequest) (*models.System, error) {
	system := &models.System{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		BoundaryDescription: req.BoundaryDescription,
		Embedding:           []float32{}, // Empty embedding initially
		Embedded:            false,       // No embeddings initially
		Consolidated:        false,       // Not consolidated initially
		ConsolidationScore:  0,           // No consolidations yet
		CreatedAt:           time.Now(),
	}
	query := `CREATE (s:System {
		id: $id, 
		name: $name, 
		boundary_description: $boundary_description, 
		embedding: $embedding, 
		embedded: $embedded, 
		consolidated: $consolidated,
		consolidation_score: $consolidation_score,
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"id":                   system.ID,
		"name":                 system.Name,
		"boundary_description": system.BoundaryDescription,
		"embedding":            system.Embedding,
		"embedded":             system.Embedded,
		"consolidated":         system.Consolidated,
		"consolidation_score":  system.ConsolidationScore,
		"created_at":           system.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return system, err
}

func (h *Handler) createStockInDB(ctx context.Context, req models.StockRequest) (*models.Stock, error) {
	stock := &models.Stock{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		Type:               req.Type,
		Embedding:          []float32{}, // Empty embedding initially
		Embedded:           false,       // No embeddings initially
		Consolidated:       false,       // Not consolidated initially
		ConsolidationScore: 0,           // No consolidations yet
		CreatedAt:          time.Now(),
	}
	query := `CREATE (st:Stock {
		id: $id, 
		name: $name, 
		description: $description, 
		type: $type, 
		embedding: $embedding, 
		embedded: $embedded, 
		consolidated: $consolidated,
		consolidation_score: $consolidation_score,
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"id":                  stock.ID,
		"name":                stock.Name,
		"description":         stock.Description,
		"type":                stock.Type,
		"embedding":           stock.Embedding,
		"embedded":            stock.Embedded,
		"consolidated":        stock.Consolidated,
		"consolidation_score": stock.ConsolidationScore,
		"created_at":          stock.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return stock, err
}

func (h *Handler) createFlowInDB(ctx context.Context, req models.FlowRequest) (*models.Flow, error) {
	flow := &models.Flow{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		Embedding:          []float32{}, // Empty embedding initially
		Embedded:           false,       // No embeddings initially
		Consolidated:       false,       // Not consolidated initially
		ConsolidationScore: 0,           // No consolidations yet
		CreatedAt:          time.Now(),
	}
	query := `CREATE (f:Flow {
		id: $id, 
		name: $name, 
		description: $description, 
		embedding: $embedding, 
		embedded: $embedded, 
		consolidated: $consolidated,
		consolidation_score: $consolidation_score,
		created_at: $created_at
	})`
	params := map[string]interface{}{
		"id":                  flow.ID,
		"name":                flow.Name,
		"description":         flow.Description,
		"embedding":           flow.Embedding,
		"embedded":            flow.Embedded,
		"consolidated":        flow.Consolidated,
		"consolidation_score": flow.ConsolidationScore,
		"created_at":          flow.CreatedAt.Format(time.RFC3339),
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return flow, err
}

func (h *Handler) createDescribesRelationshipInDB(ctx context.Context, narrativeID, systemID string) error {
	query := `MATCH (n:Narrative {id: $narrative_id}), (s:System {id: $system_id}) 
		CREATE (n)-[:DESCRIBES {consolidated: $consolidated, consolidation_score: $consolidation_score}]->(s)`
	params := map[string]interface{}{
		"narrative_id":        narrativeID,
		"system_id":           systemID,
		"consolidated":        false,
		"consolidation_score": 0,
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createConstitutesRelationshipInDB(ctx context.Context, subsystemID, systemID string) error {
	query := `MATCH (sub:System {id: $subsystem_id}), (sys:System {id: $system_id}) 
		CREATE (sub)-[:CONSTITUTES {consolidated: $consolidated, consolidation_score: $consolidation_score}]->(sys)`
	params := map[string]interface{}{
		"subsystem_id":        subsystemID,
		"system_id":           systemID,
		"consolidated":        false,
		"consolidation_score": 0,
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createDescribesStaticRelationshipInDB(ctx context.Context, stockID, systemID string) error {
	query := `MATCH (st:Stock {id: $stock_id}), (s:System {id: $system_id}) 
		CREATE (st)-[:DESCRIBES_STATIC {consolidated: $consolidated, consolidation_score: $consolidation_score}]->(s)`
	params := map[string]interface{}{
		"stock_id":            stockID,
		"system_id":           systemID,
		"consolidated":        false,
		"consolidation_score": 0,
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createChangesRelationshipInDB(ctx context.Context, flowID, stockID string, polarity float32) error {
	query := `MATCH (f:Flow {id: $flow_id}), (st:Stock {id: $stock_id}) 
		CREATE (f)-[:CHANGES {polarity: $polarity, consolidated: $consolidated, consolidation_score: $consolidation_score}]->(st)`
	params := map[string]interface{}{
		"flow_id":             flowID,
		"stock_id":            stockID,
		"polarity":            polarity,
		"consolidated":        false,
		"consolidation_score": 0,
	}
	_, err := h.db.ExecuteQuery(ctx, query, params)
	return err
}

func (h *Handler) createCausalLinkInDB(ctx context.Context, req models.CausalLink) error {
	query := `MATCH (a), (b) WHERE a.id = $from_id AND b.id = $to_id 
		CREATE (a)-[r:CAUSAL_LINK {
			question: $question, 
			curiosity_score: $curiosity_score, 
			consolidated: $consolidated,
			consolidation_score: $consolidation_score,
			created_at: $created_at
		}]->(b)`
	params := map[string]interface{}{
		"from_id":             req.FromID,
		"to_id":               req.ToID,
		"question":            req.Question,
		"curiosity_score":     req.CuriosityScore,
		"consolidated":        false,
		"consolidation_score": 0,
		"created_at":          time.Now().Format(time.RFC3339),
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

// ProcessEmbeddings - Processes embeddings for all unconsolidated nodes in batch
func (h *Handler) ProcessEmbeddings(c *gin.Context) {
	err := h.processNodeEmbeddingsInBatch(c.Request.Context())
	if err != nil {
		log.Printf("Error processing embeddings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process embeddings: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully processed embeddings for all unconsolidated nodes"})
}
