package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/handlers"
	"growth-spark-ai-service/middleware"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, loading from system environment.")
	}

	// Connect to MongoDB
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer func() {
		if err := database.Disconnect(); err != nil {
			log.Println("Error disconnecting from database:", err)
		}
	}()

	// Set up the Gin router
	router := gin.Default()

	// API routes
	api := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		api.POST("/organizations", handlers.CreateOrganization)
		api.POST("/auth/login", handlers.Login)
		api.POST("/auth/register", handlers.Register)

		// Protected routes (authentication required)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", handlers.GetProfile)
			protected.GET("/permissions", middleware.GetUserPermissions)

			// Role management routes (admin only)
			roles := protected.Group("/roles")
			roles.Use(middleware.RequirePermission("roles:read"))
			{
				roles.GET("/", handlers.GetRoles)
				roles.GET("/:id", handlers.GetRole)
				roles.GET("/permissions", handlers.GetPermissions)
			}

			// Role creation, update, delete (requires specific permissions)
			protected.POST("/roles", middleware.RequirePermission("roles:create"), handlers.CreateRole)
			protected.PUT("/roles/:id", middleware.RequirePermission("roles:update"), handlers.UpdateRole)
			protected.DELETE("/roles/:id", middleware.RequirePermission("roles:delete"), handlers.DeleteRole)
			protected.POST("/roles/assign", middleware.RequirePermission("users:update"), handlers.AssignRole)
		}
	}

	// Run the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
