package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organizationId"`
	Username       string             `bson:"username" json:"username" binding:"required"`
	FullName       string             `bson:"full_name" json:"fullName" binding:"required"`
	Email          string             `bson:"email" json:"email" binding:"required,email"`
	Password       string             `bson:"password" json:"-" binding:"required"`
	Role           string             `bson:"role" json:"role"`
	CreatedAt      time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updatedAt"`
}
