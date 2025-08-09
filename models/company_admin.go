package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CompanyAdmin represents the relationship between a user and a company they administer
// This is stored in the dedicated 'company_admins' collection
type CompanyAdmin struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID      primitive.ObjectID `bson:"company_id" json:"company_id" binding:"required"`
	UserID         primitive.ObjectID `bson:"user_id" json:"user_id" binding:"required"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id" binding:"required"`
	Role           string             `bson:"role" json:"role" binding:"required"` // admin, manager, etc.
	Permissions    []string           `bson:"permissions" json:"permissions"`
	IsActive       bool               `bson:"is_active" json:"is_active"`
	Status         string             `bson:"status" json:"status"` // active, suspended, pending
	AssignedAt     time.Time          `bson:"assigned_at" json:"assigned_at"`
	ExpiresAt      *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Notes          string             `bson:"notes" json:"notes"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedBy      primitive.ObjectID `bson:"created_by" json:"created_by"`
	UpdatedBy      primitive.ObjectID `bson:"updated_by" json:"updated_by"`
}

// ValidRoles defines the valid roles for company admins
var ValidRoles = []string{"admin", "manager", "supervisor", "viewer"}

// ValidStatuses defines the valid statuses for company admin assignments
var ValidStatuses = []string{"active", "suspended", "pending", "expired"}

// IsValidRole checks if the given role is valid
func (ca *CompanyAdmin) IsValidRole() bool {
	for _, role := range ValidRoles {
		if ca.Role == role {
			return true
		}
	}
	return false
}

// IsValidStatus checks if the given status is valid
func (ca *CompanyAdmin) IsValidStatus() bool {
	for _, status := range ValidStatuses {
		if ca.Status == status {
			return true
		}
	}
	return false
}

// IsExpired checks if the company admin assignment has expired
func (ca *CompanyAdmin) IsExpired() bool {
	if ca.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*ca.ExpiresAt)
}

// CreateCompanyAdminRequest represents the request payload for creating a new company admin
type CreateCompanyAdminRequest struct {
	CompanyID   string   `json:"company_id" binding:"required"`
	Username    string   `json:"username" binding:"required"`
	FullName    string   `json:"full_name" binding:"required"`
	Email       string   `json:"email" binding:"required,email"`
	PhoneNumber string   `json:"phone_number" binding:"required"`
	Password    string   `json:"password" binding:"required,min=6"`
	Role        string   `json:"role" binding:"required"` // admin, manager, supervisor, viewer
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"` // active, suspended, pending
	ExpiresAt   *string  `json:"expires_at,omitempty"` // ISO date string, optional
	Notes       string   `json:"notes"`
}

// UpdateCompanyAdminRequest represents the request payload for updating a company admin
type UpdateCompanyAdminRequest struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
	IsActive    *bool    `json:"is_active"`
	ExpiresAt   *string  `json:"expires_at,omitempty"` // ISO date string, optional
	Notes       string   `json:"notes"`
}

// AssignExistingUserRequest represents the request for assigning an existing user as company admin
type AssignExistingUserRequest struct {
	CompanyID   string   `json:"company_id" binding:"required"`
	UserID      string   `json:"user_id" binding:"required"`
	Role        string   `json:"role" binding:"required"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
	ExpiresAt   *string  `json:"expires_at,omitempty"`
	Notes       string   `json:"notes"`
}

// CompanyAdminResponse represents the response with user and company details
type CompanyAdminResponse struct {
	ID          primitive.ObjectID `json:"id"`
	CompanyID   primitive.ObjectID `json:"company_id"`
	CompanyName string             `json:"company_name"`
	UserID      primitive.ObjectID `json:"user_id"`
	Username    string             `json:"username"`
	UserEmail   string             `json:"user_email"`
	Role        string             `json:"role"`
	Permissions []string           `json:"permissions"`
	IsActive    bool               `json:"is_active"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}
