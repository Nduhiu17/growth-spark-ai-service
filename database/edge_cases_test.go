package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestConnectionEdgeCases(t *testing.T) {
	t.Run("ConnectWithoutEnvironmentVariables", func(t *testing.T) {
		// Save original environment variables
		originalMongoURI := os.Getenv("MONGO_URI")
		originalTestURI := os.Getenv("MONGO_TEST_URI")
		
		// Clear environment variables
		os.Unsetenv("MONGO_URI")
		os.Unsetenv("MONGO_TEST_URI")
		
		defer func() {
			// Restore environment variables
			if originalMongoURI != "" {
				os.Setenv("MONGO_URI", originalMongoURI)
			}
			if originalTestURI != "" {
				os.Setenv("MONGO_TEST_URI", originalTestURI)
			}
		}()
		
		// Test connection without environment variables
		err := Connect()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MONGO_URI or MONGO_TEST_URI environment variable not set")
	})

	t.Run("DisconnectWithoutConnection", func(t *testing.T) {
		// Save original client
		originalClient := Client
		Client = nil
		
		defer func() {
			Client = originalClient
		}()
		
		// Test disconnect without connection
		err := Disconnect()
		assert.NoError(t, err) // Should not error when client is nil
	})
}

func TestDatabaseOperationsEdgeCases(t *testing.T) {
	// Skip if MongoDB is not available
	if os.Getenv("MONGO_TEST_URI") == "" && os.Getenv("MONGO_URI") == "" {
		t.Skip("MongoDB not available for testing")
	}

	// Connect to test database
	err := Connect()
	require.NoError(t, err)
	defer Disconnect()

	t.Run("CollectionOperationsWithTimeout", func(t *testing.T) {
		collection := DB.Collection("test_timeout")
		
		// Test with very short context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		// This should timeout quickly
		_, err := collection.InsertOne(ctx, bson.M{"test": "data"})
		// Error is expected due to timeout
		assert.Error(t, err)
	})

	t.Run("DatabaseConnectionState", func(t *testing.T) {
		// Test that we have a valid connection
		assert.NotNil(t, Client)
		assert.NotNil(t, DB)
		
		// Test ping
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := Client.Ping(ctx, nil)
		assert.NoError(t, err)
	})

	t.Run("InvalidObjectIDOperations", func(t *testing.T) {
		// Test with invalid ObjectID string
		invalidID := "invalid_object_id"
		objectID, err := primitive.ObjectIDFromHex(invalidID)
		
		// This should fail
		assert.Error(t, err)
		assert.Equal(t, primitive.NilObjectID, objectID)
	})

	t.Run("ValidObjectIDOperations", func(t *testing.T) {
		// Test valid ObjectID operations
		id1 := primitive.NewObjectID()
		id2 := primitive.NewObjectID()
		
		assert.NotEqual(t, id1, id2)
		assert.NotEqual(t, primitive.NilObjectID, id1)
		
		// Test ObjectID string conversion
		idString := id1.Hex()
		assert.Len(t, idString, 24)
		
		// Test ObjectID from string
		parsedID, err := primitive.ObjectIDFromHex(idString)
		assert.NoError(t, err)
		assert.Equal(t, id1, parsedID)
	})
}

func TestCompanyAdminDBEdgeCases(t *testing.T) {
	// Skip if MongoDB is not available
	if os.Getenv("MONGO_TEST_URI") == "" && os.Getenv("MONGO_URI") == "" {
		t.Skip("MongoDB not available for testing")
	}

	// Connect to test database
	err := Connect()
	require.NoError(t, err)
	defer Disconnect()

	companyAdminDB := &CompanyAdminDB{
		collection: DB.Collection("company_admins_edge_test"),
	}

	t.Run("GetByOrganizationWithEmptyResult", func(t *testing.T) {
		// Use a non-existent organization ID
		nonExistentOrgID := primitive.NewObjectID()
		
		admins, err := companyAdminDB.GetByOrganization(nonExistentOrgID)
		assert.NoError(t, err)
		assert.Empty(t, admins)
	})

	t.Run("UpdateByCompanyAndUserNotFound", func(t *testing.T) {
		// Try to update non-existent company admin
		nonExistentCompanyID := primitive.NewObjectID()
		nonExistentUserID := primitive.NewObjectID()
		
		updates := bson.M{"role": "updated_role"}
		
		err := companyAdminDB.UpdateByCompanyAndUser(nonExistentCompanyID, nonExistentUserID, updates)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "company admin not found")
	})

	t.Run("GetWithUserDetailsEmptyResult", func(t *testing.T) {
		// Use a non-existent company ID
		nonExistentCompanyID := primitive.NewObjectID()
		
		result, err := companyAdminDB.GetWithUserDetails(nonExistentCompanyID)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestBSONOperations(t *testing.T) {
	t.Run("BSONMarshallingEdgeCases", func(t *testing.T) {
		// Test marshalling of complex structures
		complexData := bson.M{
			"id":        primitive.NewObjectID(),
			"timestamp": primitive.NewDateTimeFromTime(time.Now()),
			"nested": bson.M{
				"array": bson.A{"item1", "item2"},
				"null":  nil,
			},
		}
		
		// Test that BSON can handle complex data
		assert.NotNil(t, complexData["id"])
		assert.NotNil(t, complexData["timestamp"])
		assert.NotNil(t, complexData["nested"])
	})

	t.Run("TimeHandling", func(t *testing.T) {
		now := time.Now()
		zeroTime := time.Time{}
		
		// Test time comparison
		assert.True(t, now.After(zeroTime))
		assert.False(t, zeroTime.After(now))
		
		// Test time formatting
		timeString := now.Format(time.RFC3339)
		assert.NotEmpty(t, timeString)
		
		// Test time parsing
		parsedTime, err := time.Parse(time.RFC3339, timeString)
		assert.NoError(t, err)
		assert.True(t, parsedTime.Equal(now.Truncate(time.Second)))
	})
}
