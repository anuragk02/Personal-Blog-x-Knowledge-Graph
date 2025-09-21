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
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001", "http://localhost:5174", "http://127.0.0.1:5174"}
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
		// Concept routes
		api.POST("/concepts", h.CreateConcept)
		api.GET("/concepts", h.GetConcepts)
		api.GET("/concepts/:id", h.GetConcept)
		api.PUT("/concepts/:id", h.UpdateConcept)
		api.DELETE("/concepts/:id", h.DeleteConcept)

		// Essay routes
		api.POST("/essays", h.CreateEssay)
		api.GET("/essays", h.GetEssays)
		api.GET("/essays/:id", h.GetEssay)
		api.PUT("/essays/:id", h.UpdateEssay)
		api.DELETE("/essays/:id", h.DeleteEssay)

		// Claim routes
		api.POST("/claims", h.CreateClaim)
		api.GET("/claims", h.GetClaims)
		api.GET("/claims/:id", h.GetClaim)
		api.PUT("/claims/:id", h.UpdateClaim)
		api.DELETE("/claims/:id", h.DeleteClaim)

		// Source routes
		api.POST("/sources", h.CreateSource)
		api.GET("/sources", h.GetSources)
		api.GET("/sources/:id", h.GetSource)
		api.PUT("/sources/:id", h.UpdateSource)
		api.DELETE("/sources/:id", h.DeleteSource)

		// Question routes
		api.POST("/questions", h.CreateQuestion)
		api.GET("/questions", h.GetQuestions)
		api.GET("/questions/:id", h.GetQuestion)
		api.PUT("/questions/:id", h.UpdateQuestion)
		api.DELETE("/questions/:id", h.DeleteQuestion)

		// Relationship routes
		api.POST("/relationships", h.CreateRelationship)
		api.GET("/connections", h.GetConnections)
		api.POST("/points-to", h.CreatePointsTo)
		api.POST("/defines", h.CreateDefines)
		api.POST("/influences", h.CreateInfluences)
		api.POST("/supports", h.CreateSupports)
		api.POST("/contradicts", h.CreateContradicts)
		api.POST("/derived-from", h.CreateDerivedFrom)
		api.POST("/raises", h.CreateRaises)

		// Analytical endpoints for agentic systems
		api.GET("/search", h.SearchKnowledge)
		api.GET("/stats", h.GetKnowledgeStats)
		api.GET("/path", h.FindPath)
		api.GET("/neighborhood/:id", h.GetNodeNeighborhood)
		api.GET("/insights", h.GetKnowledgeInsights)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
