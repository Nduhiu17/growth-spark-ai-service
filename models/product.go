package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product represents a product that belongs to a company
type Product struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID      primitive.ObjectID `bson:"company_id" json:"company_id" binding:"required"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id" binding:"required"`
	Name           string             `bson:"name" json:"name" binding:"required"`
	Description    string             `bson:"description" json:"description"`
	Category       string             `bson:"category" json:"category" binding:"required"`
	Price          float64            `bson:"price" json:"price" binding:"required,min=0"`
	Currency       string             `bson:"currency" json:"currency" binding:"required"`
	SKU            string             `bson:"sku" json:"sku" binding:"required"`
	Status         string             `bson:"status" json:"status"` // active, inactive, discontinued
	IsActive       bool               `bson:"is_active" json:"is_active"`
	Stock          int                `bson:"stock" json:"stock" binding:"min=0"`
	MinStock       int                `bson:"min_stock" json:"min_stock" binding:"min=0"`
	MaxStock       int                `bson:"max_stock" json:"max_stock" binding:"min=0"`
	Tags           []string           `bson:"tags" json:"tags"`
	Images         []string           `bson:"images" json:"images"`
	Specifications map[string]string  `bson:"specifications" json:"specifications"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedBy      primitive.ObjectID `bson:"created_by" json:"created_by"`
	UpdatedBy      primitive.ObjectID `bson:"updated_by" json:"updated_by"`
}

// ValidStatuses defines the valid statuses for products
var ValidProductStatuses = []string{"active", "inactive", "discontinued"}

// ValidCategories defines common product categories
var ValidProductCategories = []string{
	"electronics", "clothing", "books", "home", "sports", "automotive",
	"health", "beauty", "toys", "food", "software", "services", "other",
}

// ValidCurrencies defines supported currencies
var ValidCurrencies = []string{"USD", "EUR", "GBP", "KES", "NGN", "ZAR"}

// IsValidStatus checks if the given status is valid
func (p *Product) IsValidStatus() bool {
	for _, status := range ValidProductStatuses {
		if p.Status == status {
			return true
		}
	}
	return false
}

// IsValidCategory checks if the given category is valid
func (p *Product) IsValidCategory() bool {
	for _, category := range ValidProductCategories {
		if p.Category == category {
			return true
		}
	}
	return false
}

// IsValidCurrency checks if the given currency is valid
func (p *Product) IsValidCurrency() bool {
	for _, currency := range ValidCurrencies {
		if p.Currency == currency {
			return true
		}
	}
	return false
}

// IsLowStock checks if the product stock is below minimum threshold
func (p *Product) IsLowStock() bool {
	return p.Stock <= p.MinStock
}

// IsOverStock checks if the product stock is above maximum threshold
func (p *Product) IsOverStock() bool {
	return p.MaxStock > 0 && p.Stock >= p.MaxStock
}

// CreateProductRequest represents the request payload for creating a new product
type CreateProductRequest struct {
	CompanyID      string            `json:"company_id" binding:"required"`
	Name           string            `json:"name" binding:"required"`
	Description    string            `json:"description"`
	Category       string            `json:"category" binding:"required"`
	Price          float64           `json:"price" binding:"required,min=0"`
	Currency       string            `json:"currency" binding:"required"`
	SKU            string            `json:"sku" binding:"required"`
	Status         string            `json:"status"`
	Stock          int               `json:"stock" binding:"min=0"`
	MinStock       int               `json:"min_stock" binding:"min=0"`
	MaxStock       int               `json:"max_stock" binding:"min=0"`
	Tags           []string          `json:"tags"`
	Images         []string          `json:"images"`
	Specifications map[string]string `json:"specifications"`
}

// UpdateProductRequest represents the request payload for updating a product
type UpdateProductRequest struct {
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Category       string            `json:"category"`
	Price          *float64          `json:"price" binding:"omitempty,min=0"`
	Currency       string            `json:"currency"`
	SKU            string            `json:"sku"`
	Status         string            `json:"status"`
	IsActive       *bool             `json:"is_active"`
	Stock          *int              `json:"stock" binding:"omitempty,min=0"`
	MinStock       *int              `json:"min_stock" binding:"omitempty,min=0"`
	MaxStock       *int              `json:"max_stock" binding:"omitempty,min=0"`
	Tags           []string          `json:"tags"`
	Images         []string          `json:"images"`
	Specifications map[string]string `json:"specifications"`
}

// ProductResponse represents the response with product details
type ProductResponse struct {
	ID             primitive.ObjectID `json:"id"`
	CompanyID      primitive.ObjectID `json:"company_id"`
	CompanyName    string             `json:"company_name"`
	OrganizationID primitive.ObjectID `json:"organization_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	Category       string             `json:"category"`
	Price          float64            `json:"price"`
	Currency       string             `json:"currency"`
	SKU            string             `json:"sku"`
	Status         string             `json:"status"`
	IsActive       bool               `json:"is_active"`
	Stock          int                `json:"stock"`
	MinStock       int                `json:"min_stock"`
	MaxStock       int                `json:"max_stock"`
	Tags           []string           `json:"tags"`
	Images         []string           `json:"images"`
	Specifications map[string]string  `json:"specifications"`
	IsLowStock     bool               `json:"is_low_stock"`
	IsOverStock    bool               `json:"is_over_stock"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	CreatedBy      primitive.ObjectID `json:"created_by"`
	UpdatedBy      primitive.ObjectID `json:"updated_by"`
}
