package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// TestHelper provides utilities for testing
type TestHelper struct {
	DB       *mongo.Database
	Client   *mongo.Client
	Cleanup  func()
}

// SetupTestEnvironment creates a test database and returns cleanup function
func SetupTestEnvironment(t *testing.T) *TestHelper {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)

	testDB := client.Database("growth_spark_integration_test_" + primitive.NewObjectID().Hex())
	
	// Set the global DB for testing
	originalDB := database.DB
	database.DB = testDB

	cleanup := func() {
		// Drop test database
		testDB.Drop(context.TODO())
		client.Disconnect(context.TODO())
		// Restore original DB
		database.DB = originalDB
	}

	return &TestHelper{
		DB:      testDB,
		Client:  client,
		Cleanup: cleanup,
	}
}

// CreateTestOrganization creates a test organization
func (th *TestHelper) CreateTestOrganization(t *testing.T) models.Organization {
	org := models.Organization{
		ID:                 primitive.NewObjectID(),
		OrgID:              "test-org-" + primitive.NewObjectID().Hex()[:8],
		Name:               "Test Organization",
		Description:        "Test organization for testing",
		ContactPersonName:  "Test Contact",
		ContactPersonPhone: "+1234567890",
		ContactPersonEmail: "contact@testorg.com",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	collection := th.DB.Collection("organizations")
	_, err := collection.InsertOne(context.TODO(), org)
	require.NoError(t, err)

	return org
}

// CreateTestUser creates a test user
func (th *TestHelper) CreateTestUser(t *testing.T, organizationID primitive.ObjectID, role string) models.User {
	// Generate unique identifiers using test name and timestamp to avoid conflicts
	testName := t.Name()
	timestamp := time.Now().UnixNano()
	uniqueID := primitive.NewObjectID().Hex()[:8]
	
	user := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: organizationID,
		Username:       fmt.Sprintf("testuser_%s_%d_%s", testName, timestamp, uniqueID),
		FullName:       "Test User",
		Email:          fmt.Sprintf("testuser_%s_%d_%s@example.com", testName, timestamp, uniqueID),
		Password:       "$2a$10$hashedpassword", // Pre-hashed password
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	collection := th.DB.Collection("users")
	_, err := collection.InsertOne(context.TODO(), user)
	require.NoError(t, err)

	return user
}

// CreateTestCompany creates a test company
func (th *TestHelper) CreateTestCompany(t *testing.T, organizationID primitive.ObjectID, createdBy primitive.ObjectID) models.Company {
	company := models.Company{
		ID:             primitive.NewObjectID(),
		OrganizationID: organizationID,
		Name:           "Test Company " + primitive.NewObjectID().Hex()[:8],
		Description:    "Test company for testing",
		Industry:       "Technology",
		Email:          "company@example.com",
		Phone:          "+1234567890",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}

	collection := th.DB.Collection("companies")
	_, err := collection.InsertOne(context.TODO(), company)
	require.NoError(t, err)

	return company
}

// CreateTestCompanyAdmin creates a test company admin
func (th *TestHelper) CreateTestCompanyAdmin(t *testing.T, companyID, userID, organizationID, createdBy primitive.ObjectID) models.CompanyAdmin {
	companyAdmin := models.CompanyAdmin{
		ID:             primitive.NewObjectID(),
		CompanyID:      companyID,
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           "admin",
		Permissions:    []string{"create_user", "edit_user", "delete_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}

	db := database.NewCompanyAdminDB()
	err := db.Create(&companyAdmin)
	require.NoError(t, err)

	return companyAdmin
}

// CreateTestRole creates a test role
func (th *TestHelper) CreateTestRole(t *testing.T, organizationID primitive.ObjectID, name string, permissions []string) models.Role {
	role := models.Role{
		ID:             primitive.NewObjectID(),
		OrganizationID: organizationID,
		Name:           name,
		Description:    "Test role: " + name,
		Permissions:    permissions,
		IsSystemRole:   false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	collection := th.DB.Collection("roles")
	_, err := collection.InsertOne(context.TODO(), role)
	require.NoError(t, err)

	return role
}

// AssertCompanyAdminExists verifies that a company admin exists with the given criteria
func (th *TestHelper) AssertCompanyAdminExists(t *testing.T, companyID, userID primitive.ObjectID) {
	db := database.NewCompanyAdminDB()
	exists, err := db.ExistsByCompanyAndUser(companyID, userID)
	require.NoError(t, err)
	require.True(t, exists, "Company admin should exist")
}

// AssertCompanyAdminNotExists verifies that a company admin does not exist with the given criteria
func (th *TestHelper) AssertCompanyAdminNotExists(t *testing.T, companyID, userID primitive.ObjectID) {
	db := database.NewCompanyAdminDB()
	exists, err := db.ExistsByCompanyAndUser(companyID, userID)
	require.NoError(t, err)
	require.False(t, exists, "Company admin should not exist")
}

// AssertCompanyAdminInactive verifies that a company admin is inactive (soft deleted)
func (th *TestHelper) AssertCompanyAdminInactive(t *testing.T, adminID primitive.ObjectID) {
	db := database.NewCompanyAdminDB()
	admin, err := db.GetByID(adminID)
	require.NoError(t, err)
	require.False(t, admin.IsActive, "Company admin should be inactive")
}

// GetCompanyAdminCount returns the count of company admins for a company
func (th *TestHelper) GetCompanyAdminCount(t *testing.T, companyID primitive.ObjectID) int {
	db := database.NewCompanyAdminDB()
	admins, err := db.GetByCompany(companyID)
	require.NoError(t, err)
	return len(admins)
}

// GetUserCompanyAdminCount returns the count of company admin assignments for a user
func (th *TestHelper) GetUserCompanyAdminCount(t *testing.T, userID primitive.ObjectID) int {
	db := database.NewCompanyAdminDB()
	admins, err := db.GetByUser(userID)
	require.NoError(t, err)
	return len(admins)
}
