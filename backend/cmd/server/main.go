package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/anuragk02/jna-nuh-yoh-guh/internal/database"
	"github.com/anuragk02/jna-nuh-yoh-guh/internal/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	db := database.NewDB()
	defer db.Close(context.Background())

	h := handlers.NewHandler(db)
	r := gin.Default()

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
		api.POST("/concepts", h.CreateConcept)
		api.GET("/concepts/:id", h.GetConcept)
		api.POST("/relationships", h.CreateRelationship)
		api.GET("/connections", h.GetConnections)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
