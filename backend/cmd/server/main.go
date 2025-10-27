package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/anuragk02/jna-nuh-yoh-guh/internal/database"
	"github.com/anuragk02/jna-nuh-yoh-guh/internal/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewDB()
	defer db.Close(context.Background())

	h := handlers.NewHandler(db)
	r := gin.Default()

	// Configure CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001", "http://localhost:5174", "http://127.0.0.1:5174", "http://localhost:5173", "http://127.0.0.1:5173"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Simple Neo4j test endpoint
	r.GET("/test-neo4j", func(c *gin.Context) {
		result, err := db.ExecuteQuery(context.Background(), "RETURN 'Hello Neo4j' as message", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if result.Next(context.Background()) {
			record := result.Record()
			c.JSON(http.StatusOK, gin.H{"neo4j": record.Values[0]})
			return
		}

		c.JSON(http.StatusOK, gin.H{"neo4j": "connected but no data"})
	})

	r.POST("/login", h.LoginHandler)

	// API routes (protected by JWT Auth)
	api := r.Group("/api/v1")
	// Apply authentication middleware to all /api/v1 routes
	api.Use(handlers.AuthMiddleware())
	{
		// Health Check
		api.GET("/health", h.HealthCheck)

		// Narrative CRUD endpoints
		narratives := api.Group("/narratives")
		{
			narratives.POST("", h.CreateNarrativeNode)
			narratives.GET("", h.GetNarratives)
			narratives.GET("/:id", h.GetNarrativeByID)
			narratives.PUT("/:id", h.UpdateNarrativeNode)
			narratives.DELETE("/:id", h.DeleteNarrativeNode)
			// LLM Workflow Endpoint - ID provided in request body
			narratives.POST("/analyze", h.AnalyzeNarrative)
		}

		// Utility Endpoint to clean the graph
		api.POST("/clean", h.CleanNonNarrativeData)

		// Utility Endpoint to process embeddings for all unconsolidated nodes
		api.POST("/embeddings", h.ProcessEmbeddings)

		// Consolidation Endpoint - Main workflow for consolidating the graph
		api.POST("/consolidate", h.ConsolidateGraph)

		// Reset Consolidation - Reset all nodes to unconsolidated status
		api.POST("/consolidate/reset", h.ResetConsolidation)

		// Debug Endpoint - Test similarity between two nodes
		api.GET("/debug/similarity", h.DebugSimilarity)

		// Debug Endpoint - Check relationships for a specific node
		api.GET("/debug/relationships", h.DebugNodeRelationships)

		// Debug Endpoint - Test name synthesis between two nodes
		api.GET("/debug/synthesis", h.DebugSynthesis)

		// Debug Endpoint - Check consolidation status of all relationships
		api.GET("/debug/relationship-status", h.DebugRelationshipConsolidationStatus)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
