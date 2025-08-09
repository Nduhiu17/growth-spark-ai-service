package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/models"
)

// Test database setup
func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	// Use a test database
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)

	testDB := client.Database("growth_spark_test_" + primitive.NewObjectID().Hex())
	
	// Set the global DB for testing
	originalDB := DB
	DB = testDB

	cleanup := func() {
		// Drop test database
		testDB.Drop(context.TODO())
		client.Disconnect(context.TODO())
		// Restore original DB
		DB = originalDB
	}

	return testDB, cleanup
}

func TestNewCompanyAdminDB(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	assert.NotNil(t, db)
	assert.NotNil(t, db.collection)
	assert.Equal(t, "company_admins", db.collection.Name())
}

func TestCompanyAdminDB_Create(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()

	companyAdmin := &models.CompanyAdmin{
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user", "edit_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	assert.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, companyAdmin.ID)
}

func TestCompanyAdminDB_GetByID(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	require.NoError(t, err)

	// Test GetByID
	retrieved, err := db.GetByID(companyAdmin.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, companyAdmin.ID, retrieved.ID)
	assert.Equal(t, companyAdmin.CompanyID, retrieved.CompanyID)
	assert.Equal(t, companyAdmin.UserID, retrieved.UserID)
	assert.Equal(t, companyAdmin.Role, retrieved.Role)
}

func TestCompanyAdminDB_GetByCompanyAndUser(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      companyID,
		UserID:         userID,
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	require.NoError(t, err)

	// Test GetByCompanyAndUser
	retrieved, err := db.GetByCompanyAndUser(companyID, userID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, companyID, retrieved.CompanyID)
	assert.Equal(t, userID, retrieved.UserID)
}

func TestCompanyAdminDB_GetByCompany(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	companyID := primitive.NewObjectID()

	// Create multiple company admins for the same company
	for i := 0; i < 3; i++ {
		companyAdmin := &models.CompanyAdmin{
			CompanyID:      companyID,
			UserID:         primitive.NewObjectID(),
			OrganizationID: primitive.NewObjectID(),
			Role:           "admin",
			Permissions:    []string{"create_user"},
			IsActive:       true,
			Status:         "active",
			AssignedAt:     now,
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      primitive.NewObjectID(),
			UpdatedBy:      primitive.NewObjectID(),
		}
		err := db.Create(companyAdmin)
		require.NoError(t, err)
	}

	// Test GetByCompany
	admins, err := db.GetByCompany(companyID)
	assert.NoError(t, err)
	assert.Len(t, admins, 3)
	for _, admin := range admins {
		assert.Equal(t, companyID, admin.CompanyID)
		assert.True(t, admin.IsActive)
	}
}

func TestCompanyAdminDB_GetByUser(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	userID := primitive.NewObjectID()

	// Create multiple company admin assignments for the same user
	for i := 0; i < 2; i++ {
		companyAdmin := &models.CompanyAdmin{
			CompanyID:      primitive.NewObjectID(),
			UserID:         userID,
			OrganizationID: primitive.NewObjectID(),
			Role:           "admin",
			Permissions:    []string{"create_user"},
			IsActive:       true,
			Status:         "active",
			AssignedAt:     now,
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      primitive.NewObjectID(),
			UpdatedBy:      primitive.NewObjectID(),
		}
		err := db.Create(companyAdmin)
		require.NoError(t, err)
	}

	// Test GetByUser
	admins, err := db.GetByUser(userID)
	assert.NoError(t, err)
	assert.Len(t, admins, 2)
	for _, admin := range admins {
		assert.Equal(t, userID, admin.UserID)
		assert.True(t, admin.IsActive)
	}
}

func TestCompanyAdminDB_Update(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	require.NoError(t, err)

	// Test Update
	updates := bson.M{
		"role":        "manager",
		"permissions": []string{"view_users"},
		"updated_at":  time.Now(),
	}

	err = db.Update(companyAdmin.ID, updates)
	assert.NoError(t, err)

	// Verify update
	updated, err := db.GetByID(companyAdmin.ID)
	assert.NoError(t, err)
	assert.Equal(t, "manager", updated.Role)
	assert.Equal(t, []string{"view_users"}, updated.Permissions)
}

func TestCompanyAdminDB_SoftDelete(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	deletedBy := primitive.NewObjectID()

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	require.NoError(t, err)

	// Test SoftDelete
	err = db.SoftDelete(companyAdmin.ID, deletedBy)
	assert.NoError(t, err)

	// Verify soft delete
	deleted, err := db.GetByID(companyAdmin.ID)
	assert.NoError(t, err)
	assert.False(t, deleted.IsActive)
	assert.Equal(t, deletedBy, deleted.UpdatedBy)
}

func TestCompanyAdminDB_HardDelete(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      primitive.NewObjectID(),
		UserID:         primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err := db.Create(companyAdmin)
	require.NoError(t, err)

	// Test HardDelete
	err = db.HardDelete(companyAdmin.ID)
	assert.NoError(t, err)

	// Verify hard delete
	_, err = db.GetByID(companyAdmin.ID)
	assert.Error(t, err)
	assert.Equal(t, mongo.ErrNoDocuments, err)
}

func TestCompanyAdminDB_ExistsByCompanyAndUser(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	// Test non-existing
	exists, err := db.ExistsByCompanyAndUser(companyID, userID)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Create test company admin
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      companyID,
		UserID:         userID,
		OrganizationID: primitive.NewObjectID(),
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      primitive.NewObjectID(),
		UpdatedBy:      primitive.NewObjectID(),
	}

	err = db.Create(companyAdmin)
	require.NoError(t, err)

	// Test existing
	exists, err = db.ExistsByCompanyAndUser(companyID, userID)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCompanyAdminDB_GetStats(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	db := NewCompanyAdminDB()
	now := time.Now()
	organizationID := primitive.NewObjectID()

	// Create test data with different statuses
	statuses := []string{"active", "suspended", "pending"}
	for _, status := range statuses {
		for i := 0; i < 2; i++ {
			companyAdmin := &models.CompanyAdmin{
				CompanyID:      primitive.NewObjectID(),
				UserID:         primitive.NewObjectID(),
				OrganizationID: organizationID,
				Role:           "admin",
				Permissions:    []string{"create_user"},
				IsActive:       status == "active",
				Status:         status,
				AssignedAt:     now,
				CreatedAt:      now,
				UpdatedAt:      now,
				CreatedBy:      primitive.NewObjectID(),
				UpdatedBy:      primitive.NewObjectID(),
			}
			err := db.Create(companyAdmin)
			require.NoError(t, err)
		}
	}

	// Test GetStats
	stats, err := db.GetStats(organizationID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	
	// Verify stats structure matches actual implementation
	assert.Contains(t, stats, "total_admins")
	assert.Contains(t, stats, "total_companies")
	assert.Contains(t, stats, "total_users_as_admins")
	
	// Verify stats values
	assert.Equal(t, int32(6), stats["total_admins"]) // 2 admins per status * 3 statuses
	assert.GreaterOrEqual(t, stats["total_companies"], int32(1))
	assert.GreaterOrEqual(t, stats["total_users_as_admins"], int32(1))
}
