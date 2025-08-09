package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company represents a company within an organization
type Company struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id" binding:"required"`
	Name           string             `bson:"name" json:"name" binding:"required"`
	Description    string             `bson:"description" json:"description"`
	Industry       string             `bson:"industry" json:"industry"`
	Website        string             `bson:"website" json:"website"`
	Email          string             `bson:"email" json:"email" binding:"omitempty,email"`
	Phone          string             `bson:"phone" json:"phone"`
	Address        CompanyAddress     `bson:"address" json:"address"`
	Status         string             `bson:"status" json:"status"` // active, inactive, suspended
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedBy      primitive.ObjectID `bson:"created_by" json:"created_by"`
	UpdatedBy      primitive.ObjectID `bson:"updated_by" json:"updated_by"`
}

// CompanyAddress represents the address of a company
type CompanyAddress struct {
	Street     string `bson:"street" json:"street"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	PostalCode string `bson:"postal_code" json:"postal_code"`
	Country    string `bson:"country" json:"country"`
}

// CreateCompanyRequest represents the request payload for creating a company
type CreateCompanyRequest struct {
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Industry    string         `json:"industry"`
	Website     string         `json:"website"`
	Email       string         `json:"email" binding:"omitempty,email"`
	Phone       string         `json:"phone"`
	Address     CompanyAddress `json:"address"`
}

// UpdateCompanyRequest represents the request payload for updating a company
type UpdateCompanyRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Industry    string         `json:"industry"`
	Website     string         `json:"website"`
	Email       string         `json:"email" binding:"omitempty,email"`
	Phone       string         `json:"phone"`
	Address     CompanyAddress `json:"address"`
	Status      string         `json:"status"`
}
