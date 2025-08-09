package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// CreateOrganization handles the registration of a new organization and its Super Admin
func CreateOrganization(c *gin.Context) {
	var org models.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Generate a unique organization ID
	org.OrgID = primitive.NewObjectID().Hex()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	// Insert the new organization into the database
	orgsCollection := database.DB.Collection("organizations")
	result, err := orgsCollection.InsertOne(context.TODO(), org)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create organization"})
		return
	}

	// Set the ID from the insertion result
	org.ID = result.InsertedID.(primitive.ObjectID)

	// Initialize default roles for the organization
	if err := database.InitializeDefaultRoles(org.ID); err != nil {
		c.JSON(500, gin.H{"error": "Failed to initialize default roles"})
		return
	}

	// Now, create the Super Admin for this organization
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		// This is a simplified example. In a real app, you would
		// separate the organization and user creation into two steps.
		// For this example, we'll use a single request body.
		// Let's assume the request body also contains the user details.
		c.JSON(400, gin.H{"error": "Invalid user data in request"})
		return
	}

	// Hash the password for security
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = string(hashedPassword)
	user.OrganizationID = org.ID
	user.Role = "super_admin"
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Insert the Super Admin user
	usersCollection := database.DB.Collection("users")
	_, err = usersCollection.InsertOne(context.TODO(), user)
	if err != nil {
		// In a real application, you'd want to roll back the organization creation
		// if user creation fails.
		c.JSON(500, gin.H{"error": "Failed to create super admin user"})
		return
	}

	c.JSON(201, gin.H{"message": "Organization and Super Admin created successfully", "orgId": org.OrgID})
}
