// Simple Random Joke Microservice using Gin
// Routes:
//
//	GET /healthz         -> health check
//	GET /api/v1/joke     -> returns a random joke
package main

import (
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Hardcoded jokes
var jokes = []string{
	"Why do programmers hate nature? It has too many bugs.",
	"I told my computer I needed a break, and it said 'No problem — I’ll go to sleep.'",
	"Debugging is like being the detective in a crime movie where you are also the murderer.",
	"Why do Java developers wear glasses? Because they don't C#.",
	"To understand recursion, you must first understand recursion.",
}

func main() {
	r := gin.Default()

	// Health check (used by Docker/K8s)
	r.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
		c.JSON(http.StatusOK, gin.H{"message": "OK", "timestamp": time.Now().Format(time.RFC3339)})
	})

	// Random joke endpoint
	r.GET("/api/v1/joke", func(c *gin.Context) {
		rand.Seed(time.Now().UnixNano())
		joke := jokes[rand.Intn(len(jokes))]
		c.JSON(http.StatusOK, gin.H{"joke": joke})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
