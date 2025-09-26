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

		// Node Creation Endpoints (Manual)
		nodes := api.Group("/nodes")
		{
			nodes.POST("/systems", h.CreateSystemNode)
			nodes.POST("/stocks", h.CreateStock)
			nodes.POST("/flows", h.CreateFlow)
		}

		// Relationship Creation Endpoints (Manual)
		relationships := api.Group("/relationships")
		{
			relationships.POST("/describes", h.CreateDescribesRelationship)
			relationships.POST("/constitutes", h.CreateConstitutesRelationship)
			relationships.POST("/describes-static", h.CreateDescribesStaticRelationship)
			relationships.POST("/describes-dynamic", h.CreateDescribesDynamicRelationship)
			relationships.POST("/changes", h.CreateChangesRelationship)
			relationships.POST("/causal-link", h.CreateCausalLink)
		}

		// Utility Endpoint to clean the graph
		api.POST("/clean", h.CleanNonNarrativeData)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
