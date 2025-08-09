package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Role represents a role in the system
type Role struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id"`
	Name           string             `bson:"name" json:"name" binding:"required"`
	Description    string             `bson:"description" json:"description"`
	Permissions    []string           `bson:"permissions" json:"permissions"`
	IsSystemRole   bool               `bson:"is_system_role" json:"is_system_role"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// Permission represents a permission in the system
type Permission struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name" binding:"required"`
	Description string             `bson:"description" json:"description"`
	Resource    string             `bson:"resource" json:"resource"`
	Action      string             `bson:"action" json:"action"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// DefaultRoles defines the system default roles
var DefaultRoles = []Role{
	{
		Name:         "super_admin",
		Description:  "Super Administrator with full system access",
		Permissions:  []string{"*"},
		IsSystemRole: true,
	},
	{
		Name:         "admin",
		Description:  "Administrator with organization-wide access",
		Permissions:  []string{"users:*", "roles:*", "organizations:read", "organizations:update"},
		IsSystemRole: true,
	},
	{
		Name:         "manager",
		Description:  "Manager with limited administrative access",
		Permissions:  []string{"users:read", "users:create", "roles:read"},
		IsSystemRole: true,
	},
	{
		Name:         "user",
		Description:  "Standard user with basic access",
		Permissions:  []string{"profile:read", "profile:update"},
		IsSystemRole: true,
	},
}

// DefaultPermissions defines the system default permissions
var DefaultPermissions = []Permission{
	{Name: "users:create", Description: "Create users", Resource: "users", Action: "create"},
	{Name: "users:read", Description: "Read users", Resource: "users", Action: "read"},
	{Name: "users:update", Description: "Update users", Resource: "users", Action: "update"},
	{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete"},
	{Name: "users:*", Description: "All user permissions", Resource: "users", Action: "*"},
	{Name: "roles:create", Description: "Create roles", Resource: "roles", Action: "create"},
	{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read"},
	{Name: "roles:update", Description: "Update roles", Resource: "roles", Action: "update"},
	{Name: "roles:delete", Description: "Delete roles", Resource: "roles", Action: "delete"},
	{Name: "roles:*", Description: "All role permissions", Resource: "roles", Action: "*"},
	{Name: "organizations:read", Description: "Read organization", Resource: "organizations", Action: "read"},
	{Name: "organizations:update", Description: "Update organization", Resource: "organizations", Action: "update"},
	{Name: "organizations:delete", Description: "Delete organization", Resource: "organizations", Action: "delete"},
	{Name: "profile:read", Description: "Read own profile", Resource: "profile", Action: "read"},
	{Name: "profile:update", Description: "Update own profile", Resource: "profile", Action: "update"},
	{Name: "*", Description: "All permissions", Resource: "*", Action: "*"},
}
