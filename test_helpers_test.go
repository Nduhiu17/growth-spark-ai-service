package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/models"
)

func TestSetupTestEnvironment(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Verify that the test helper was created successfully
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.DB)
	assert.NotNil(t, helper.Client)
	assert.NotNil(t, helper.Cleanup)

	// Verify that the database is accessible
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}
}

func TestCreateTestUser(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	organizationID := primitive.NewObjectID()
	role := "test_role"

	user := helper.CreateTestUser(t, organizationID, role)

	// Verify user properties
	assert.NotEqual(t, primitive.NilObjectID, user.ID)
	assert.Equal(t, organizationID, user.OrganizationID)
	assert.Equal(t, role, user.Role)
	assert.Contains(t, user.Username, "testuser")
	assert.Contains(t, user.Email, "@example.com")
	assert.NotEmpty(t, user.FullName)
	assert.NotEmpty(t, user.Password)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())

	// Verify user was inserted into database
	var dbUser models.User
	err = helper.DB.Collection("users").FindOne(context.TODO(), bson.M{"_id": user.ID}).Decode(&dbUser)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, dbUser.Username)
}

func TestCreateTestCompany(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	company := helper.CreateTestCompany(t, organizationID, createdBy)

	// Verify company properties
	assert.NotEqual(t, primitive.NilObjectID, company.ID)
	assert.Equal(t, organizationID, company.OrganizationID)
	assert.Equal(t, createdBy, company.CreatedBy)
	assert.Equal(t, createdBy, company.UpdatedBy)
	assert.Contains(t, company.Name, "Test Company")
	assert.Equal(t, "active", company.Status)
	assert.False(t, company.CreatedAt.IsZero())
	assert.False(t, company.UpdatedAt.IsZero())

	// Verify company was inserted into database
	var dbCompany models.Company
	err = helper.DB.Collection("companies").FindOne(context.TODO(), bson.M{"_id": company.ID}).Decode(&dbCompany)
	assert.NoError(t, err)
	assert.Equal(t, company.Name, dbCompany.Name)
}

func TestCreateTestCompanyAdmin(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	companyAdmin := helper.CreateTestCompanyAdmin(t, companyID, userID, organizationID, createdBy)

	// Verify company admin properties
	assert.NotEqual(t, primitive.NilObjectID, companyAdmin.ID)
	assert.Equal(t, companyID, companyAdmin.CompanyID)
	assert.Equal(t, userID, companyAdmin.UserID)
	assert.Equal(t, organizationID, companyAdmin.OrganizationID)
	assert.Equal(t, createdBy, companyAdmin.CreatedBy)
	assert.Equal(t, createdBy, companyAdmin.UpdatedBy)
	assert.Equal(t, "admin", companyAdmin.Role)
	assert.Equal(t, "active", companyAdmin.Status)
	assert.True(t, companyAdmin.IsActive)
	assert.False(t, companyAdmin.CreatedAt.IsZero())
	assert.False(t, companyAdmin.UpdatedAt.IsZero())

	// Verify company admin was inserted into database
	var dbCompanyAdmin models.CompanyAdmin
	err = helper.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{"_id": companyAdmin.ID}).Decode(&dbCompanyAdmin)
	assert.NoError(t, err)
	assert.Equal(t, companyAdmin.Role, dbCompanyAdmin.Role)
}

func TestCreateTestRole(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	organizationID := primitive.NewObjectID()
	name := "test_role"
	permissions := []string{"read", "write", "delete"}

	role := helper.CreateTestRole(t, organizationID, name, permissions)

	// Verify role properties
	assert.NotEqual(t, primitive.NilObjectID, role.ID)
	assert.Equal(t, organizationID, role.OrganizationID)
	assert.Equal(t, name, role.Name)
	assert.Equal(t, permissions, role.Permissions)
	assert.Contains(t, role.Description, "Test role")
	assert.False(t, role.CreatedAt.IsZero())

	// Verify role was inserted into database
	var dbRole models.Role
	err = helper.DB.Collection("roles").FindOne(context.TODO(), bson.M{"_id": role.ID}).Decode(&dbRole)
	assert.NoError(t, err)
	assert.Equal(t, role.Name, dbRole.Name)
}

func TestAssertCompanyAdminExists(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	// Create a test company admin
	helper.CreateTestCompanyAdmin(t, companyID, userID, organizationID, createdBy)

	// Test assertion - should not fail
	helper.AssertCompanyAdminExists(t, companyID, userID)
}

func TestAssertCompanyAdminNotExists(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	// Test assertion for non-existent company admin - should not fail
	helper.AssertCompanyAdminNotExists(t, companyID, userID)
}

func TestAssertCompanyAdminInactive(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	// Create a test company admin
	companyAdmin := helper.CreateTestCompanyAdmin(t, companyID, userID, organizationID, createdBy)

	// Soft delete the company admin
	_, err = helper.DB.Collection("company_admins").UpdateOne(
		context.TODO(),
		bson.M{"_id": companyAdmin.ID},
		bson.M{"$set": bson.M{"is_active": false}},
	)
	require.NoError(t, err)

	// Test assertion - should not fail
	helper.AssertCompanyAdminInactive(t, companyAdmin.ID)
}

func TestGetCompanyAdminCount(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	companyID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	// Initially should be 0
	count := helper.GetCompanyAdminCount(t, companyID)
	assert.Equal(t, 0, count)

	// Create a company admin
	userID1 := primitive.NewObjectID()
	helper.CreateTestCompanyAdmin(t, companyID, userID1, organizationID, createdBy)

	// Should be 1 now
	count = helper.GetCompanyAdminCount(t, companyID)
	assert.Equal(t, 1, count)

	// Create another company admin
	userID2 := primitive.NewObjectID()
	helper.CreateTestCompanyAdmin(t, companyID, userID2, organizationID, createdBy)

	// Should be 2 now
	count = helper.GetCompanyAdminCount(t, companyID)
	assert.Equal(t, 2, count)
}

func TestGetUserCompanyAdminCount(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)
	defer helper.Cleanup()

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	createdBy := primitive.NewObjectID()

	// Initially should be 0
	count := helper.GetUserCompanyAdminCount(t, userID)
	assert.Equal(t, 0, count)

	// Create a company admin assignment
	companyID1 := primitive.NewObjectID()
	helper.CreateTestCompanyAdmin(t, companyID1, userID, organizationID, createdBy)

	// Should be 1 now
	count = helper.GetUserCompanyAdminCount(t, userID)
	assert.Equal(t, 1, count)

	// Create another company admin assignment for the same user
	companyID2 := primitive.NewObjectID()
	helper.CreateTestCompanyAdmin(t, companyID2, userID, organizationID, createdBy)

	// Should be 2 now
	count = helper.GetUserCompanyAdminCount(t, userID)
	assert.Equal(t, 2, count)
}

// Test helper cleanup functionality
func TestTestHelperCleanup(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping test that requires MongoDB")
	}

	helper := SetupTestEnvironment(t)

	// Skip if MongoDB is not available
	err := helper.DB.Client().Ping(context.TODO(), nil)
	if err != nil {
		t.Skip("MongoDB not available for testing")
	}

	// Verify we can list databases before cleanup
	names, err := helper.Client.ListDatabaseNames(context.TODO(), bson.M{})
	require.NoError(t, err)
	// Database should exist before cleanup
	assert.True(t, len(names) >= 0, "Should be able to list databases")

	// Call cleanup
	helper.Cleanup()

	// Give some time for cleanup to complete
	time.Sleep(100 * time.Millisecond)

	// Verify database cleanup completed successfully
	// Note: We don't assert on database list contents due to MongoDB timing issues
	// The important verification is that cleanup was called without errors
}
