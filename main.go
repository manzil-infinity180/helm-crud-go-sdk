package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/manzil-infinity180/helm-crud-go-sdk/routes"
)

func main() {
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	routes.SetupRoutes(router)

	if err := router.Run(":4004"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
