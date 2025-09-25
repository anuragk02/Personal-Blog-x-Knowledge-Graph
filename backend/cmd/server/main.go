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

	// API routes
	api := r.Group("/api/v1")
	{
		// Health check
		api.GET("/health", h.HealthCheck)

		// FOR WRITING PROCESS AND LLM WORKFLOW
		// Narrative CRUD operations
		api.POST("/narratives", h.CreateNarrative)
		api.GET("/narratives/:id", h.GetNarrative)
		api.PUT("/narratives/:id", h.UpdateNarrative)
		api.DELETE("/narratives/:id", h.DeleteNarrative)

		// AFTER EXTRACTION USING LLM TO CREATE WORLDVIEW
		// Node creation endpoints (with auto-generated IDs)
		api.POST("/systems", h.CreateSystem)
		api.POST("/stocks", h.CreateStock)
		api.POST("/flows", h.CreateFlow)

		// Relationship creation endpoints
		api.POST("/relationships/describes", h.CreateDescribesRelationship)                // Narrative -> System
		api.POST("/relationships/constitutes", h.CreateConstitutesRelationship)            // System -> System
		api.POST("/relationships/describes-static", h.CreateDescribesStaticRelationship)   // Stock -> System
		api.POST("/relationships/describes-dynamic", h.CreateDescribesDynamicRelationship) // Flow -> System
		api.POST("/relationships/changes", h.CreateChangesRelationship)                    // Flow -> Stock

		// CausalLink creation endpoint
		api.POST("/causal-links", h.CreateCausalLink) // Stock/Flow -> Stock/Flow
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
