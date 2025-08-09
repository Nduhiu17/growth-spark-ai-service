package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Organization represents an organization in the system
type Organization struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID              string             `bson:"org_id" json:"orgId"`
	Name               string             `bson:"name" json:"name" binding:"required"`
	Description        string             `bson:"description" json:"description" binding:"required"`
	ContactPersonName  string             `bson:"contact_person_name" json:"contactPersonName" binding:"required"`
	ContactPersonPhone string             `bson:"contact_person_phone" json:"contactPersonPhone" binding:"required"`
	ContactPersonEmail string             `bson:"contact_person_email" json:"contactPersonEmail" binding:"required,email"`
	CreatedAt          time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updatedAt"`
}
