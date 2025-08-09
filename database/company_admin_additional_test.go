package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/models"
)

func setupAdditionalTestDB(t *testing.T) {
	// Connect to test database
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	DB = client.Database("growth_spark_test_additional")
	
	// Clean up function
	t.Cleanup(func() {
		DB.Drop(context.TODO())
		client.Disconnect(context.TODO())
	})
}

func TestGetByOrganization(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Create test data
	orgID := primitive.NewObjectID()
	companyID1 := primitive.NewObjectID()
	companyID2 := primitive.NewObjectID()
	userID1 := primitive.NewObjectID()
	userID2 := primitive.NewObjectID()
	
	// Create company admins for the organization
	companyAdmin1 := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID1,
		UserID:         userID1,
		OrganizationID: orgID,
		Role:           "admin",
		Status:         "active",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	companyAdmin2 := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID2,
		UserID:         userID2,
		OrganizationID: orgID,
		Role:           "manager",
		Status:         "active",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Create company admin for different organization
	otherOrgAdmin := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(), // Different org
		Role:           "admin",
		Status:         "active",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	err := db.Create(&companyAdmin1)
	require.NoError(t, err)
	err = db.Create(&companyAdmin2)
	require.NoError(t, err)
	err = db.Create(&otherOrgAdmin)
	require.NoError(t, err)
	
	// Test GetByOrganization
	admins, err := db.GetByOrganization(orgID)
	assert.NoError(t, err)
	assert.Len(t, admins, 2)
	
	// Verify the returned admins belong to the correct organization
	for _, admin := range admins {
		assert.Equal(t, orgID, admin.OrganizationID)
	}
}

func TestGetWithUserDetails(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Create test data
	orgID := primitive.NewObjectID()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	// Create a user first
	user := models.User{
		ID:             userID,
		OrganizationID: orgID,
		Username:       "testuser",
		FullName:       "Test User",
		Email:          "test@example.com",
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	usersCollection := DB.Collection("users")
	_, err := usersCollection.InsertOne(context.TODO(), user)
	require.NoError(t, err)
	
	// Create a company record (required for aggregation pipeline)
	company := map[string]interface{}{
		"_id":  companyID,
		"name": "Test Company",
	}
	companiesCollection := DB.Collection("companies")
	_, err = companiesCollection.InsertOne(context.TODO(), company)
	require.NoError(t, err)
	
	// Create company admin
	companyAdmin := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID,
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "admin",
		Status:         "active",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	err = db.Create(&companyAdmin)
	require.NoError(t, err)
	
	// Test GetWithUserDetails
	results, err := db.GetWithUserDetails(companyID)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	
	result := results[0]
	assert.Equal(t, companyAdmin.ID, result.ID)
	assert.Equal(t, companyID, result.CompanyID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "testuser", result.Username)
	assert.Equal(t, "test@example.com", result.UserEmail)
}

func TestUpdateByCompanyAndUser(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Create test data
	orgID := primitive.NewObjectID()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	// Create company admin
	companyAdmin := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID,
		UserID:         userID,
		OrganizationID: orgID,
		Role:           "admin",
		Status:         "active",
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	err := db.Create(&companyAdmin)
	require.NoError(t, err)
	
	// Test UpdateByCompanyAndUser
	updateData := map[string]interface{}{
		"role":       "manager",
		"status":     "pending",
		"updated_at": time.Now(),
	}
	
	err = db.UpdateByCompanyAndUser(companyID, userID, updateData)
	assert.NoError(t, err)
	
	// Verify the update
	updatedAdmin, err := db.GetByCompanyAndUser(companyID, userID)
	assert.NoError(t, err)
	assert.Equal(t, "manager", updatedAdmin.Role)
	assert.Equal(t, "pending", updatedAdmin.Status)
}

func TestUpdateByCompanyAndUser_NotFound(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Test updating non-existent company admin
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	updateData := map[string]interface{}{
		"role": "manager",
	}
	
	err := db.UpdateByCompanyAndUser(companyID, userID, updateData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no documents updated")
}

func TestGetByOrganization_EmptyResult(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Test with organization that has no company admins
	orgID := primitive.NewObjectID()
	
	admins, err := db.GetByOrganization(orgID)
	assert.NoError(t, err)
	assert.Len(t, admins, 0)
}

func TestGetWithUserDetails_EmptyResult(t *testing.T) {
	setupAdditionalTestDB(t)
	
	db := NewCompanyAdminDB()
	
	// Test with company that has no admins
	companyID := primitive.NewObjectID()
	
	results, err := db.GetWithUserDetails(companyID)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestGetByOrganization_DatabaseError(t *testing.T) {
	// Don't set up database to simulate connection error
	DB = nil
	
	db := NewCompanyAdminDB()
	orgID := primitive.NewObjectID()
	
	admins, err := db.GetByOrganization(orgID)
	assert.Error(t, err)
	assert.Nil(t, admins)
}

func TestGetWithUserDetails_DatabaseError(t *testing.T) {
	// Don't set up database to simulate connection error
	DB = nil
	
	db := NewCompanyAdminDB()
	companyID := primitive.NewObjectID()
	
	results, err := db.GetWithUserDetails(companyID)
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestUpdateByCompanyAndUser_DatabaseError(t *testing.T) {
	// Don't set up database to simulate connection error
	DB = nil
	
	db := NewCompanyAdminDB()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	updateData := map[string]interface{}{
		"role": "manager",
	}
	
	err := db.UpdateByCompanyAndUser(companyID, userID, updateData)
	assert.Error(t, err)
}
