package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// RequirePermission checks if the user has the required permission
func RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		organizationID, exists := c.Get("organization_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
			c.Abort()
			return
		}

		role := userRole.(string)
		orgID := organizationID.(primitive.ObjectID)

		// Check if user has the required permission
		hasPermission, err := checkUserPermission(role, orgID, requiredPermission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has any of the required permissions
func RequireAnyPermission(requiredPermissions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		organizationID, exists := c.Get("organization_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
			c.Abort()
			return
		}

		role := userRole.(string)
		orgID := organizationID.(primitive.ObjectID)

		// Check if user has any of the required permissions
		for _, permission := range requiredPermissions {
			hasPermission, err := checkUserPermission(role, orgID, permission)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
				c.Abort()
				return
			}

			if hasPermission {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// checkUserPermission checks if a user with a specific role has a permission
func checkUserPermission(roleName string, organizationID primitive.ObjectID, requiredPermission string) (bool, error) {
	// Get role from database
	rolesCollection := database.DB.Collection("roles")
	var role models.Role
	err := rolesCollection.FindOne(context.TODO(), bson.M{
		"organization_id": organizationID,
		"name":            roleName,
	}).Decode(&role)
	if err != nil {
		return false, err
	}

	// Check if role has the required permission
	return hasPermission(role.Permissions, requiredPermission), nil
}

// hasPermission checks if a permission list contains the required permission
func hasPermission(permissions []string, requiredPermission string) bool {
	for _, permission := range permissions {
		// Check for exact match
		if permission == requiredPermission {
			return true
		}

		// Check for wildcard permissions
		if permission == "*" {
			return true
		}

		// Check for resource-level wildcards (e.g., "users:*" matches "users:read")
		if strings.Contains(permission, ":*") {
			resource := strings.Split(permission, ":")[0]
			requiredResource := strings.Split(requiredPermission, ":")[0]
			if resource == requiredResource {
				return true
			}
		}
	}

	return false
}

// GetUserPermissions returns all permissions for the current user
func GetUserPermissions(c *gin.Context) {
	userRole, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	role := userRole.(string)
	orgID := organizationID.(primitive.ObjectID)

	// Get role from database
	rolesCollection := database.DB.Collection("roles")
	var roleDoc models.Role
	err := rolesCollection.FindOne(context.TODO(), bson.M{
		"organization_id": orgID,
		"name":            role,
	}).Decode(&roleDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role":        role,
		"permissions": roleDoc.Permissions,
	})
}
