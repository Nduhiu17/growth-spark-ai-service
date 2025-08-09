package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// CreateRoleRequest represents the create role request payload
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest represents the update role request payload
type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// AssignRoleRequest represents the assign role request payload
type AssignRoleRequest struct {
	UserID string `json:"userId" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

// CreateRole creates a new role for the organization
func CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	// Check if role already exists in the organization
	rolesCollection := database.DB.Collection("roles")
	var existingRole models.Role
	err := rolesCollection.FindOne(context.TODO(), bson.M{
		"organization_id": orgID,
		"name":            req.Name,
	}).Decode(&existingRole)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Role already exists"})
		return
	}

	// Create new role
	role := models.Role{
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Permissions:    req.Permissions,
		IsSystemRole:   false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	result, err := rolesCollection.InsertOne(context.TODO(), role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	role.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, role)
}

// GetRoles returns all roles for the organization
func GetRoles(c *gin.Context) {
	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	rolesCollection := database.DB.Collection("roles")
	cursor, err := rolesCollection.Find(context.TODO(), bson.M{"organization_id": orgID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}
	defer cursor.Close(context.TODO())

	var roles []models.Role
	if err := cursor.All(context.TODO(), &roles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode roles"})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetRole returns a specific role by ID
func GetRole(c *gin.Context) {
	roleID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	rolesCollection := database.DB.Collection("roles")
	var role models.Role
	err = rolesCollection.FindOne(context.TODO(), bson.M{
		"_id":             objID,
		"organization_id": orgID,
	}).Decode(&role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, role)
}

// UpdateRole updates an existing role
func UpdateRole(c *gin.Context) {
	roleID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	// Check if role exists and is not a system role
	rolesCollection := database.DB.Collection("roles")
	var role models.Role
	err = rolesCollection.FindOne(context.TODO(), bson.M{
		"_id":             objID,
		"organization_id": orgID,
	}).Decode(&role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	if role.IsSystemRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system roles"})
		return
	}

	// Build update document
	updateDoc := bson.M{"updated_at": time.Now()}
	if req.Name != "" {
		updateDoc["name"] = req.Name
	}
	if req.Description != "" {
		updateDoc["description"] = req.Description
	}
	if req.Permissions != nil {
		updateDoc["permissions"] = req.Permissions
	}

	_, err = rolesCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": objID},
		bson.M{"$set": updateDoc},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	// Fetch updated role
	err = rolesCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated role"})
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role
func DeleteRole(c *gin.Context) {
	roleID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	// Check if role exists and is not a system role
	rolesCollection := database.DB.Collection("roles")
	var role models.Role
	err = rolesCollection.FindOne(context.TODO(), bson.M{
		"_id":             objID,
		"organization_id": orgID,
	}).Decode(&role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	if role.IsSystemRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system roles"})
		return
	}

	// Check if any users have this role
	usersCollection := database.DB.Collection("users")
	count, err := usersCollection.CountDocuments(context.TODO(), bson.M{"role": role.Name})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check role usage"})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete role that is assigned to users"})
		return
	}

	_, err = rolesCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// AssignRole assigns a role to a user
func AssignRole(c *gin.Context) {
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	organizationID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization not found"})
		return
	}

	orgID := organizationID.(primitive.ObjectID)

	// Check if role exists in the organization
	rolesCollection := database.DB.Collection("roles")
	var role models.Role
	err = rolesCollection.FindOne(context.TODO(), bson.M{
		"organization_id": orgID,
		"name":            req.Role,
	}).Decode(&role)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	// Update user's role
	usersCollection := database.DB.Collection("users")
	_, err = usersCollection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":             userObjID,
			"organization_id": orgID,
		},
		bson.M{
			"$set": bson.M{
				"role":       req.Role,
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// GetPermissions returns all available permissions
func GetPermissions(c *gin.Context) {
	c.JSON(http.StatusOK, models.DefaultPermissions)
}
