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
	"golang.org/x/crypto/bcrypt"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// CreateCompanyAdmin creates a new user and assigns them as admin for a specific company (Super Admin only)
func CreateCompanyAdmin(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can create company admins"})
		return
	}

	var req models.CreateCompanyAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[COMPANY_ADMIN] Create company admin request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate company ID
	companyID, err := primitive.ObjectIDFromHex(req.CompanyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// Verify company exists and belongs to the organization
	companiesCollection := database.DB.Collection("companies")
	var company models.Company
	err = companiesCollection.FindOne(context.TODO(), bson.M{
		"_id":             companyID,
		"organization_id": user.OrganizationID,
	}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		log.Printf("[COMPANY_ADMIN] Failed to verify company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify company"})
		return
	}

	// Check if user already exists (by username or email)
	usersCollection := database.DB.Collection("users")
	var targetUser models.User
	
	err = usersCollection.FindOne(context.TODO(), bson.M{
		"$or": []bson.M{
			{"username": req.Username},
			{"email": req.Email},
		},
		"organization_id": user.OrganizationID,
	}).Decode(&targetUser)
	
	// If user exists, check if they're already a company admin for this company
	if err == nil {
		companyAdminDB := database.NewCompanyAdminDB()
		exists, checkErr := companyAdminDB.ExistsByCompanyAndUser(companyID, targetUser.ID)
		if checkErr != nil {
			log.Printf("[COMPANY_ADMIN] Failed to check existing company admin: %v", checkErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify company admin status"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "User is already a company admin for this company"})
			return
		}
	}
	
	if err == nil {
		// User exists, we'll use the existing user
		log.Printf("[COMPANY_ADMIN] Using existing user: %s", targetUser.ID.Hex())
	} else if err == mongo.ErrNoDocuments {
		// User doesn't exist, create new user
		
		// Hash password for new user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[COMPANY_ADMIN] Failed to hash password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}
		
		// Create new user
		targetUser = models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: user.OrganizationID,
			Username:       req.Username,
			FullName:       req.FullName,
			Email:          req.Email,
			Password:       string(hashedPassword),
			Role:           req.Role,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		// Insert user into database
		userResult, err := usersCollection.InsertOne(context.TODO(), targetUser)
		if err != nil {
			log.Printf("[COMPANY_ADMIN] Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		
		targetUser.ID = userResult.InsertedID.(primitive.ObjectID)
		log.Printf("[COMPANY_ADMIN] Created new user: %s", targetUser.ID.Hex())
	} else {
		// Database error
		log.Printf("[COMPANY_ADMIN] Database error checking for existing user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing user"})
		return
	}
	
	// Check if this user is already assigned as admin for this specific company
	companyAdminsCollection := database.DB.Collection("company_admins")
	var existingAssignment models.CompanyAdmin
	err = companyAdminsCollection.FindOne(context.TODO(), bson.M{
		"company_id": companyID,
		"user_id":    targetUser.ID,
		"is_active":  true,
	}).Decode(&existingAssignment)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already assigned as admin for this company"})
		return
	}

	// Create company admin assignment
	companyAdmin := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID,
		UserID:         targetUser.ID,
		OrganizationID: user.OrganizationID,
		Role:           req.Role,
		Permissions:    req.Permissions,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      user.ID,
		UpdatedBy:      user.ID,
	}

	// Insert company admin assignment into database
	adminResult, err := companyAdminsCollection.InsertOne(context.TODO(), companyAdmin)
	if err != nil {
		log.Printf("[COMPANY_ADMIN] Failed to assign company admin: %v", err)
		// Try to clean up the created user (only if we created a new user)
		if err == mongo.ErrNoDocuments {
			usersCollection.DeleteOne(context.TODO(), bson.M{"_id": targetUser.ID})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign company admin"})
		return
	}

	companyAdmin.ID = adminResult.InsertedID.(primitive.ObjectID)

	// Create response with detailed information
	response := models.CompanyAdminResponse{
		ID:          companyAdmin.ID,
		CompanyID:   companyID,
		CompanyName: company.Name,
		UserID:      targetUser.ID,
		Username:    targetUser.Username,
		UserEmail:   targetUser.Email,
		Role:        companyAdmin.Role,
		Permissions: companyAdmin.Permissions,
		IsActive:    companyAdmin.IsActive,
		CreatedAt:   companyAdmin.CreatedAt,
		UpdatedAt:   companyAdmin.UpdatedAt,
	}

	log.Printf("[COMPANY_ADMIN] Company admin created successfully: %s for company %s", targetUser.ID.Hex(), companyID.Hex())
	c.JSON(http.StatusCreated, gin.H{"message": "Company admin created successfully", "companyAdmin": response})
}

// GetCompanyAdmins retrieves all admins for a specific company (Super Admin only)
func GetCompanyAdmins(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can view company admins"})
		return
	}

	// Get company ID from URL parameter
	companyIDStr := c.Param("id")
	companyID, err := primitive.ObjectIDFromHex(companyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// Verify company exists and belongs to the organization
	companiesCollection := database.DB.Collection("companies")
	var company models.Company
	err = companiesCollection.FindOne(context.TODO(), bson.M{
		"_id":             companyID,
		"organization_id": user.OrganizationID,
	}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		log.Printf("[COMPANY_ADMIN] Failed to verify company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify company"})
		return
	}

	// Get company admins with user details using aggregation
	companyAdminsCollection := database.DB.Collection("company_admins")
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"company_id":      companyID,
				"organization_id": user.OrganizationID,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{
			"$unwind": "$user",
		},
	}

	cursor, err := companyAdminsCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		log.Printf("[COMPANY_ADMIN] Failed to get company admins: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve company admins"})
		return
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Printf("[COMPANY_ADMIN] Failed to decode company admins: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process company admins"})
		return
	}

	// Transform results to response format
	var companyAdmins []models.CompanyAdminResponse
	for _, result := range results {
		userDoc := result["user"].(bson.M)
		
		companyAdmin := models.CompanyAdminResponse{
			ID:          result["_id"].(primitive.ObjectID),
			CompanyID:   companyID,
			CompanyName: company.Name,
			UserID:      result["user_id"].(primitive.ObjectID),
			Username:    userDoc["username"].(string),
			UserEmail:   userDoc["email"].(string),
			Role:        result["role"].(string),
			IsActive:    result["is_active"].(bool),
			CreatedAt:   result["created_at"].(time.Time),
			UpdatedAt:   result["updated_at"].(time.Time),
		}

		// Handle permissions array (might be nil)
		if permissions, ok := result["permissions"].(bson.A); ok {
			for _, perm := range permissions {
				companyAdmin.Permissions = append(companyAdmin.Permissions, perm.(string))
			}
		}

		companyAdmins = append(companyAdmins, companyAdmin)
	}

	c.JSON(http.StatusOK, gin.H{
		"companyAdmins": companyAdmins,
		"companyName":   company.Name,
		"total":         len(companyAdmins),
	})
}

// RemoveCompanyAdmin removes an admin from a company (Super Admin only)
func RemoveCompanyAdmin(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can remove company admins"})
		return
	}

	// Get company admin ID from URL parameter
	adminIDStr := c.Param("adminId")
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	// Remove company admin (soft delete by setting is_active to false)
	companyAdminsCollection := database.DB.Collection("company_admins")
	result, err := companyAdminsCollection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":             adminID,
			"organization_id": user.OrganizationID,
		},
		bson.M{
			"$set": bson.M{
				"is_active":  false,
				"updated_at": time.Now(),
				"updated_by": user.ID,
			},
		},
	)

	if err != nil {
		log.Printf("[COMPANY_ADMIN] Failed to remove company admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove company admin"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company admin not found"})
		return
	}

	log.Printf("[COMPANY_ADMIN] Company admin removed successfully: %s", adminID.Hex())
	c.JSON(http.StatusOK, gin.H{"message": "Company admin removed successfully"})
}

// UpdateCompanyAdmin updates a company admin's role or permissions (Super Admin only)
func UpdateCompanyAdmin(c *gin.Context) {
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can update company admins"})
		return
	}

	// Get company admin ID from URL parameter
	adminIDStr := c.Param("adminId")
	adminID, err := primitive.ObjectIDFromHex(adminIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid admin ID"})
		return
	}

	var req models.UpdateCompanyAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[COMPANY_ADMIN] Update company admin request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Build update document
	updateDoc := bson.M{
		"updated_at": time.Now(),
		"updated_by": user.ID,
	}

	if req.Role != "" {
		updateDoc["role"] = req.Role
	}
	if req.Permissions != nil {
		updateDoc["permissions"] = req.Permissions
	}
	if req.IsActive != nil {
		updateDoc["is_active"] = *req.IsActive
	}

	// Update company admin
	companyAdminsCollection := database.DB.Collection("company_admins")
	result, err := companyAdminsCollection.UpdateOne(
		context.TODO(),
		bson.M{
			"_id":             adminID,
			"organization_id": user.OrganizationID,
		},
		bson.M{"$set": updateDoc},
	)

	if err != nil {
		log.Printf("[COMPANY_ADMIN] Failed to update company admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update company admin"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company admin not found"})
		return
	}

	log.Printf("[COMPANY_ADMIN] Company admin updated successfully: %s", adminID.Hex())
	c.JSON(http.StatusOK, gin.H{"message": "Company admin updated successfully"})
}
