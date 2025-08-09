package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"growth-spark-ai-service/auth"
	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// AuthMiddleware validates JWT tokens and adds user info to context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := tokenParts[1]
		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Fetch complete user object from database
		usersCollection := database.DB.Collection("users")
		var user models.User
		err = usersCollection.FindOne(context.TODO(), bson.M{"_id": claims.UserID}).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Printf("[AUTH] User not found in database: %s", claims.UserID.Hex())
				c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			} else {
				log.Printf("[AUTH] Database error fetching user: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service error"})
			}
			c.Abort()
			return
		}

		// Set complete user object in context
		c.Set("user", user)
		// Also set individual fields for backward compatibility
		c.Set("user_id", claims.UserID)
		c.Set("organization_id", claims.OrganizationID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole checks if the user has the required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok || userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
