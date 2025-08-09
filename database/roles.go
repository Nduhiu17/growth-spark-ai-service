package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/models"
)

// InitializeDefaultRoles creates default roles for an organization
func InitializeDefaultRoles(organizationID primitive.ObjectID) error {
	rolesCollection := DB.Collection("roles")
	permissionsCollection := DB.Collection("permissions")

	// Initialize default permissions if they don't exist
	for _, permission := range models.DefaultPermissions {
		var existingPermission models.Permission
		err := permissionsCollection.FindOne(context.TODO(), bson.M{"name": permission.Name}).Decode(&existingPermission)
		if err != nil {
			// Permission doesn't exist, create it
			permission.ID = primitive.NewObjectID()
			permission.CreatedAt = time.Now()
			_, err := permissionsCollection.InsertOne(context.TODO(), permission)
			if err != nil {
				log.Printf("Failed to create permission %s: %v", permission.Name, err)
			}
		}
	}

	// Initialize default roles for the organization
	for _, role := range models.DefaultRoles {
		var existingRole models.Role
		err := rolesCollection.FindOne(context.TODO(), bson.M{
			"organization_id": organizationID,
			"name":            role.Name,
		}).Decode(&existingRole)
		if err != nil {
			// Role doesn't exist for this organization, create it
			role.ID = primitive.NewObjectID()
			role.OrganizationID = organizationID
			role.CreatedAt = time.Now()
			role.UpdatedAt = time.Now()
			_, err := rolesCollection.InsertOne(context.TODO(), role)
			if err != nil {
				log.Printf("Failed to create role %s for organization %s: %v", role.Name, organizationID.Hex(), err)
				return err
			}
		}
	}

	return nil
}

// GetRoleByName retrieves a role by name for a specific organization
func GetRoleByName(organizationID primitive.ObjectID, roleName string) (*models.Role, error) {
	rolesCollection := DB.Collection("roles")
	var role models.Role
	err := rolesCollection.FindOne(context.TODO(), bson.M{
		"organization_id": organizationID,
		"name":            roleName,
	}).Decode(&role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ValidatePermissions checks if all provided permissions exist
func ValidatePermissions(permissions []string) bool {
	permissionsCollection := DB.Collection("permissions")
	
	for _, permission := range permissions {
		// Skip wildcard permissions
		if permission == "*" {
			continue
		}
		
		var existingPermission models.Permission
		err := permissionsCollection.FindOne(context.TODO(), bson.M{"name": permission}).Decode(&existingPermission)
		if err != nil {
			return false
		}
	}
	
	return true
}
