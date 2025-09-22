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
		// Narrative routes
		api.POST("/narratives", h.CreateNarrative)
		api.GET("/narratives", h.GetNarratives)
		api.GET("/narratives/:id", h.GetNarrative)
		api.PUT("/narratives/:id", h.UpdateNarrative)
		api.DELETE("/narratives/:id", h.DeleteNarrative)

		// System routes
		api.POST("/systems", h.CreateSystem)
		api.GET("/systems", h.GetSystems)
		api.GET("/systems/:id", h.GetSystem)
		api.PUT("/systems/:id", h.UpdateSystem)
		api.DELETE("/systems/:id", h.DeleteSystem)

		// Stock routes
		api.POST("/stocks", h.CreateStock)
		api.GET("/stocks", h.GetStocks)
		api.GET("/stocks/:id", h.GetStock)
		api.PUT("/stocks/:id", h.UpdateStock)
		api.DELETE("/stocks/:id", h.DeleteStock)

		// Flow routes
		api.POST("/flows", h.CreateFlow)
		api.GET("/flows", h.GetFlows)
		api.GET("/flows/:id", h.GetFlow)
		api.PUT("/flows/:id", h.UpdateFlow)
		api.DELETE("/flows/:id", h.DeleteFlow)

		// QuestionData routes
		api.POST("/questions", h.CreateQuestionData)
		api.GET("/questions", h.GetQuestionDataList)
		api.GET("/questions/:id", h.GetQuestionData)
		api.PUT("/questions/:id", h.UpdateQuestionData)
		api.DELETE("/questions/:id", h.DeleteQuestionData)

		// CausalLink routes
		api.POST("/causal-links", h.CreateCausalLink)
		api.GET("/causal-links", h.GetCausalLinks)
		api.GET("/causal-links/:from_id/:to_id", h.GetCausalLink)
		api.PUT("/causal-links/:from_id/:to_id", h.UpdateCausalLink)
		api.DELETE("/causal-links/:from_id/:to_id", h.DeleteCausalLink)

		// Relationship routes
		api.POST("/relationships/describes", h.CreateDescribesRelationship)
		api.POST("/relationships/constitutes", h.CreateConstitutesRelationship)
		api.POST("/relationships/describes-static", h.CreateDescribesStaticRelationship)
		api.POST("/relationships/describes-dynamic", h.CreateDescribesDynamicRelationship)
		api.POST("/relationships/changes", h.CreateChangesRelationship)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
