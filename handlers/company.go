package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// CreateCompany creates a new company (Super Admin only)
func CreateCompany(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is super admin
	if user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can create companies"})
		return
	}

	var req models.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[COMPANY] Create company request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Create company document
	company := models.Company{
		ID:             primitive.NewObjectID(),
		OrganizationID: user.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Industry:       req.Industry,
		Website:        req.Website,
		Email:          req.Email,
		Phone:          req.Phone,
		Address:        req.Address,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      user.ID,
		UpdatedBy:      user.ID,
	}

	// Insert into database
	collection := database.DB.Collection("companies")
	result, err := collection.InsertOne(context.TODO(), company)
	if err != nil {
		log.Printf("[COMPANY] Failed to create company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create company"})
		return
	}

	company.ID = result.InsertedID.(primitive.ObjectID)
	log.Printf("[COMPANY] Company created successfully: %s", company.ID.Hex())
	c.JSON(http.StatusCreated, gin.H{"message": "Company created successfully", "company": company})
}

// GetCompany retrieves a specific company by ID (Super Admin only)
func GetCompany(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is super admin
	if user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can view companies"})
		return
	}

	// Get company ID from URL parameter
	companyIDStr := c.Param("id")
	companyID, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// Find company in database
	collection := database.DB.Collection("companies")
	var company models.Company
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":             companyID,
		"organization_id": user.OrganizationID,
	}).Decode(&company)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		log.Printf("[COMPANY] Failed to get company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve company"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"company": company})
}

// GetAllCompanies retrieves all companies for the organization (Super Admin only)
func GetAllCompanies(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is super admin
	if user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can view companies"})
		return
	}

	// Get query parameters for pagination and filtering
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	status := c.Query("status")

	// Build filter
	filter := bson.M{"organization_id": user.OrganizationID}
	if status != "" {
		filter["status"] = status
	}

	// Find companies in database
	collection := database.DB.Collection("companies")
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("[COMPANY] Failed to get companies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve companies"})
		return
	}
	defer cursor.Close(context.TODO())

	var companies []models.Company
	if err = cursor.All(context.TODO(), &companies); err != nil {
		log.Printf("[COMPANY] Failed to decode companies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process companies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"companies": companies,
		"total":     len(companies),
		"page":      page,
		"limit":     limit,
	})
}

// UpdateCompany updates an existing company (Super Admin only)
func UpdateCompany(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is super admin
	if user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can update companies"})
		return
	}

	// Get company ID from URL parameter
	companyIDStr := c.Param("id")
	companyID, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	var req models.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[COMPANY] Update company request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Build update document
	updateDoc := bson.M{
		"updated_at": time.Now(),
		"updated_by": user.ID,
	}

	if req.Name != "" {
		updateDoc["name"] = req.Name
	}
	if req.Description != "" {
		updateDoc["description"] = req.Description
	}
	if req.Industry != "" {
		updateDoc["industry"] = req.Industry
	}
	if req.Website != "" {
		updateDoc["website"] = req.Website
	}
	if req.Email != "" {
		updateDoc["email"] = req.Email
	}
	if req.Phone != "" {
		updateDoc["phone"] = req.Phone
	}
	if req.Status != "" {
		updateDoc["status"] = req.Status
	}
	// Update address if provided
	if req.Address.Street != "" || req.Address.City != "" || req.Address.State != "" || req.Address.PostalCode != "" || req.Address.Country != "" {
		updateDoc["address"] = req.Address
	}

	// Update company in database
	collection := database.DB.Collection("companies")
	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":             companyID,
			"organization_id": user.OrganizationID,
		},
		bson.M{"$set": updateDoc},
	)

	if err != nil {
		log.Printf("[COMPANY] Failed to update company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update company"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	// Get updated company
	var updatedCompany models.Company
	err = collection.FindOne(context.TODO(), bson.M{
		"_id":             companyID,
		"organization_id": user.OrganizationID,
	}).Decode(&updatedCompany)

	if err != nil {
		log.Printf("[COMPANY] Failed to get updated company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Company updated but failed to retrieve updated data"})
		return
	}

	log.Printf("[COMPANY] Company updated successfully: %s", companyID.Hex())
	c.JSON(http.StatusOK, gin.H{"message": "Company updated successfully", "company": updatedCompany})
}

// DeleteCompany deletes a company (Super Admin only)
func DeleteCompany(c *gin.Context) {
	// Get user from context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is super admin
	if user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can delete companies"})
		return
	}

	// Get company ID from URL parameter
	companyIDStr := c.Param("id")
	companyID, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// Delete company from database
	collection := database.DB.Collection("companies")
	result, err := collection.DeleteOne(context.TODO(), bson.M{
		"_id":             companyID,
		"organization_id": user.OrganizationID,
	})

	if err != nil {
		log.Printf("[COMPANY] Failed to delete company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete company"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	log.Printf("[COMPANY] Company deleted successfully: %s", companyID.Hex())
	c.JSON(http.StatusOK, gin.H{"message": "Company deleted successfully"})
}
