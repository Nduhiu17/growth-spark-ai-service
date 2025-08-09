package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"growth-spark-ai-service/auth"
	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the user registration request payload
type RegisterRequest struct {
	OrganizationName string `json:"organization_name" binding:"required"`
	Username         string `json:"username" binding:"required"`
	FullName         string `json:"full_name" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	Password         string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Login handles user authentication
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[AUTH] Login request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	log.Printf("[AUTH] Login attempt for email: %s", req.Email)

	// Find user by email
	usersCollection := database.DB.Collection("users")
	var user models.User
	err := usersCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[AUTH] Login failed - user not found: %s", req.Email)
		} else {
			log.Printf("[AUTH] Database error during login for %s: %v", req.Email, err)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		log.Printf("[AUTH] Password verification failed for user: %s", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.OrganizationID, user.Username, user.Role)
	if err != nil {
		log.Printf("[AUTH] JWT token generation failed for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service temporarily unavailable"})
		return
	}

	log.Printf("[AUTH] Successful login for user: %s (ID: %s)", user.Email, user.ID.Hex())

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

// Register handles user registration
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[AUTH] Registration request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	log.Printf("[AUTH] Registration attempt for email: %s, organization: %s", req.Email, req.OrganizationName)

	// Check if user already exists
	usersCollection := database.DB.Collection("users")
	var existingUser models.User
	err := usersCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		log.Printf("[AUTH] Registration failed - user already exists: %s", req.Email)
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	} else if err != mongo.ErrNoDocuments {
		log.Printf("[AUTH] Database error checking existing user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration service temporarily unavailable"})
		return
	}

	// Create organization automatically
	log.Printf("[AUTH] Creating organization: %s", req.OrganizationName)
	orgsCollection := database.DB.Collection("organizations")
	org := models.Organization{
		Name:               req.OrganizationName,
		Description:        fmt.Sprintf("Auto-created organization for %s", req.OrganizationName),
		ContactPersonName:  req.Username,
		ContactPersonEmail: req.Email,
		ContactPersonPhone: req.PhoneNumber,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	orgResult, err := orgsCollection.InsertOne(context.TODO(), org)
	if err != nil {
		log.Printf("[AUTH] Failed to create organization %s: %v", req.OrganizationName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	// Get the organization ID from the insertion result
	orgID := orgResult.InsertedID.(primitive.ObjectID)
	org.ID = orgID
	log.Printf("[AUTH] Successfully created organization: %s (ID: %s)", req.OrganizationName, orgID.Hex())

	// Hash password
	log.Printf("[AUTH] Hashing password for user: %s", req.Email)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[AUTH] Password hashing failed for user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration service temporarily unavailable"})
		return
	}

	// Create new user with super admin role
	log.Printf("[AUTH] Creating user: %s with super_admin role", req.Email)
	user := models.User{
		OrganizationID: orgID,
		Username:       req.Username,
		FullName:       req.FullName,
		Email:          req.Email,
		Password:       string(hashedPassword),
		Role:           "super_admin", // Automatically assign super admin role
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	userResult, err := usersCollection.InsertOne(context.TODO(), user)
	if err != nil {
		log.Printf("[AUTH] Failed to create user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Set the ID from the insertion result
	user.ID = userResult.InsertedID.(primitive.ObjectID)
	log.Printf("[AUTH] Successfully created user: %s (ID: %s)", req.Email, user.ID.Hex())

	// Create default roles for the organization
	log.Printf("[AUTH] Creating default roles for organization: %s", orgID.Hex())
	err = createDefaultRolesForOrganization(orgID)
	if err != nil {
		log.Printf("[AUTH] Warning: Failed to create default roles for organization %s: %v", orgID.Hex(), err)
		// Log error but don't fail registration
		// The user can still use the system, roles can be created later
	}

	// Generate JWT token
	log.Printf("[AUTH] Generating JWT token for user: %s", user.Email)
	token, err := auth.GenerateToken(user.ID, user.OrganizationID, user.Username, user.Role)
	if err != nil {
		log.Printf("[AUTH] JWT token generation failed for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration completed but authentication service temporarily unavailable"})
		return
	}

	log.Printf("[AUTH] Registration completed successfully for user: %s (ID: %s), organization: %s (ID: %s)", 
		user.Email, user.ID.Hex(), req.OrganizationName, orgID.Hex())

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusCreated, LoginResponse{
		Token: token,
		User:  user,
	})
}

// GetProfile returns the current user's profile
func GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		log.Printf("[AUTH] GetProfile failed - user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	log.Printf("[AUTH] GetProfile request for user ID: %s", userID.(string))

	// Convert userID to ObjectID
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		log.Printf("[AUTH] GetProfile failed - invalid user ID format: %s, error: %v", userID.(string), err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Find user by ID
	usersCollection := database.DB.Collection("users")
	var user models.User
	err = usersCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("[AUTH] GetProfile failed - user not found: %s", objID.Hex())
		} else {
			log.Printf("[AUTH] Database error during GetProfile for user %s: %v", objID.Hex(), err)
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	log.Printf("[AUTH] Successfully retrieved profile for user: %s (%s)", user.Email, user.ID.Hex())

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// createDefaultRolesForOrganization creates default roles and permissions for a new organization
func createDefaultRolesForOrganization(organizationID primitive.ObjectID) error {
	log.Printf("[AUTH] Setting up default roles and permissions for organization: %s", organizationID.Hex())
	
	rolesCollection := database.DB.Collection("roles")
	permissionsCollection := database.DB.Collection("permissions")

	// Create default permissions first
	log.Printf("[AUTH] Creating %d default permissions", len(models.DefaultPermissions))
	var permissionsToInsert []interface{}
	for _, perm := range models.DefaultPermissions {
		permission := models.Permission{
			Name:        perm.Name,
			Description: perm.Description,
			Resource:    perm.Resource,
			Action:      perm.Action,
			CreatedAt:   time.Now(),
		}
		permissionsToInsert = append(permissionsToInsert, permission)
	}

	// Insert permissions
	if len(permissionsToInsert) > 0 {
		_, err := permissionsCollection.InsertMany(context.TODO(), permissionsToInsert)
		if err != nil {
			log.Printf("[AUTH] Failed to create default permissions for organization %s: %v", organizationID.Hex(), err)
			return fmt.Errorf("failed to create default permissions: %w", err)
		}
		log.Printf("[AUTH] Successfully created %d default permissions", len(permissionsToInsert))
	}

	// Create default roles for the organization
	log.Printf("[AUTH] Creating %d default roles for organization: %s", len(models.DefaultRoles), organizationID.Hex())
	var rolesToInsert []interface{}
	for _, role := range models.DefaultRoles {
		newRole := models.Role{
			OrganizationID: organizationID,
			Name:           role.Name,
			Description:    role.Description,
			Permissions:    role.Permissions,
			IsSystemRole:   role.IsSystemRole,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		rolesToInsert = append(rolesToInsert, newRole)
	}

	// Insert roles
	if len(rolesToInsert) > 0 {
		_, err := rolesCollection.InsertMany(context.TODO(), rolesToInsert)
		if err != nil {
			log.Printf("[AUTH] Failed to create default roles for organization %s: %v", organizationID.Hex(), err)
			return fmt.Errorf("failed to create default roles: %w", err)
		}
		log.Printf("[AUTH] Successfully created %d default roles for organization: %s", len(rolesToInsert), organizationID.Hex())
	}

	log.Printf("[AUTH] Completed setup of default roles and permissions for organization: %s", organizationID.Hex())
	return nil
}
