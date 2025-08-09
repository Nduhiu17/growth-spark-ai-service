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

// CreateSystemManager creates a new user and assigns them as system manager for a specific company (Company Admin only)
func CreateSystemManager(c *gin.Context) {
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

	// Check if user is company admin - they should be able to create system managers for their company
	if user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only company admins can create system managers"})
		return
	}

	var req models.CreateSystemManagerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[SYSTEM_MANAGER] Create system manager request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate company ID
	companyID, err := primitive.ObjectIDFromHex(req.CompanyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// If user is company admin (not super admin), verify they are admin of the target company
	if user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAdmin, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error checking company admin status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin status"})
			return
		}
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only create system managers for companies you administer"})
			return
		}
	}

	// Verify the company exists
	companyCollection := database.DB.Collection("companies")
	var company models.Company
	err = companyCollection.FindOne(context.TODO(), bson.M{"_id": companyID}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		log.Printf("[SYSTEM_MANAGER] Error finding company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify company"})
		return
	}

	// Check if user with this email already exists
	userCollection := database.DB.Collection("users")
	var existingUser models.User
	err = userCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&existingUser)
	
	var newUserID primitive.ObjectID
	
	if err == mongo.ErrNoDocuments {
		// User doesn't exist, create new user
		log.Printf("[SYSTEM_MANAGER] Creating new user: %s", req.Email)
		
		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error hashing password: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}

		// Create new user with "manager" role
		newUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: company.OrganizationID,
			Username:       req.Username,
			FullName:       req.FullName,
			Email:          req.Email,
			Password:       string(hashedPassword),
			Role:           "manager", // System manager role
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		_, err = userCollection.InsertOne(context.TODO(), newUser)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error creating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		newUserID = newUser.ID
		log.Printf("[SYSTEM_MANAGER] Created new user: %s", newUserID.Hex())
	} else if err != nil {
		log.Printf("[SYSTEM_MANAGER] Error checking existing user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user"})
		return
	} else {
		// User exists, check if they're already a system manager for this company
		log.Printf("[SYSTEM_MANAGER] Found existing user: %s, checking for duplicate assignment", existingUser.ID.Hex())
		
		companyAdminDB := database.NewCompanyAdminDB()
		exists, err := companyAdminDB.ExistsByCompanyAndUser(companyID, existingUser.ID)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error checking duplicate assignment: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing assignment"})
			return
		}
		
		log.Printf("[SYSTEM_MANAGER] Duplicate check result: exists=%t for company=%s, user=%s", exists, companyID.Hex(), existingUser.ID.Hex())
		
		if exists {
			log.Printf("[SYSTEM_MANAGER] Preventing duplicate assignment for user %s in company %s", existingUser.ID.Hex(), companyID.Hex())
			c.JSON(http.StatusConflict, gin.H{"error": "User is already assigned to this company"})
			return
		}

		newUserID = existingUser.ID
	}

	// Create system manager assignment (using company_admin collection with "manager" role)
	companyAdminDB := database.NewCompanyAdminDB()
	
	// Set default permissions for system manager
	defaultPermissions := []string{"users:read", "users:create", "roles:read", "companies:read"}
	if len(req.Permissions) > 0 {
		defaultPermissions = req.Permissions
	}

	systemManager := &models.CompanyAdmin{
		CompanyID:      companyID,
		UserID:         newUserID,
		OrganizationID: company.OrganizationID,
		Role:           "manager", // System manager role
		Permissions:    defaultPermissions,
		IsActive:       true,
		Status:         "active",
		AssignedAt:     time.Now(),
		Notes:          req.Notes,
		CreatedBy:      user.ID,
		UpdatedBy:      user.ID,
	}

	// Set expiration if provided
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiration date format"})
			return
		}
		systemManager.ExpiresAt = &expiresAt
	}

	err = companyAdminDB.Create(systemManager)
	if err != nil {
		log.Printf("[SYSTEM_MANAGER] Error creating system manager assignment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create system manager assignment"})
		return
	}

	log.Printf("[SYSTEM_MANAGER] System manager created successfully: %s for company %s", newUserID.Hex(), companyID.Hex())

	c.JSON(http.StatusCreated, gin.H{
		"message":           "System manager created successfully",
		"system_manager_id": systemManager.ID,
		"user_id":          newUserID,
		"company_id":       companyID,
		"role":             systemManager.Role,
		"permissions":      systemManager.Permissions,
	})
}

// GetSystemManagers retrieves all system managers for a specific company (Company Admin only)
func GetSystemManagers(c *gin.Context) {
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

	// Check if user is company admin or super admin
	if user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only company admins can view system managers"})
		return
	}

	// Get company ID from URL parameter
	companyIDParam := c.Param("companyId")
	companyID, err := primitive.ObjectIDFromHex(companyIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// If user is company admin (not super admin), verify they are admin of the target company
	if user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAdmin, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error checking company admin status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin status"})
			return
		}
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view system managers for companies you administer"})
			return
		}
	}

	// Get system managers (company admins with "manager" role) for the company
	companyAdminDB := database.NewCompanyAdminDB()
	systemManagers, err := companyAdminDB.GetWithUserDetails(companyID)
	if err != nil {
		log.Printf("[SYSTEM_MANAGER] Error retrieving system managers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve system managers"})
		return
	}

	// Filter only managers
	var managers []models.CompanyAdminResponse
	for _, admin := range systemManagers {
		if admin.Role == "manager" {
			managers = append(managers, admin)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"system_managers": managers,
		"count":          len(managers),
	})
}

// RemoveSystemManager removes a system manager from a company (Company Admin only)
func RemoveSystemManager(c *gin.Context) {
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

	// Check if user is company admin or super admin
	if user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only company admins can remove system managers"})
		return
	}

	// Get company ID and user ID from URL parameters
	companyIDParam := c.Param("companyId")
	userIDParam := c.Param("userId")

	companyID, err := primitive.ObjectIDFromHex(companyIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	managerUserID, err := primitive.ObjectIDFromHex(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// If user is company admin (not super admin), verify they are admin of the target company
	if user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAdmin, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[SYSTEM_MANAGER] Error checking company admin status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin status"})
			return
		}
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only remove system managers from companies you administer"})
			return
		}
	}

	// Find the system manager assignment first
	companyAdminDB := database.NewCompanyAdminDB()
	var systemManagerAssignment models.CompanyAdmin
	err = database.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{
		"company_id": companyID,
		"user_id":    managerUserID,
		"role":       "manager",
		"is_active":  true,
	}).Decode(&systemManagerAssignment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "System manager not found"})
			return
		}
		log.Printf("[SYSTEM_MANAGER] Error finding system manager: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find system manager"})
		return
	}

	// Remove the system manager assignment (soft delete)
	err = companyAdminDB.SoftDelete(systemManagerAssignment.ID, user.ID)
	if err != nil {
		log.Printf("[SYSTEM_MANAGER] Error removing system manager: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove system manager"})
		return
	}

	log.Printf("[SYSTEM_MANAGER] System manager removed successfully: %s from company %s", managerUserID.Hex(), companyID.Hex())

	c.JSON(http.StatusOK, gin.H{
		"message": "System manager removed successfully",
	})
}
