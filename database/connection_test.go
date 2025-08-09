package database

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestConnect_Success(t *testing.T) {
	// Save original DB state
	originalDB := DB
	defer func() {
		DB = originalDB
	}()

	// Test connection
	err := Connect()
	assert.NoError(t, err)
	assert.NotNil(t, DB)

	// Verify we can ping the database
	err = DB.Client().Ping(context.TODO(), nil)
	assert.NoError(t, err)
}

func TestConnect_WithCustomURI(t *testing.T) {
	// Save original DB state and environment
	originalDB := DB
	originalURI := os.Getenv("MONGODB_URI")
	defer func() {
		DB = originalDB
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		} else {
			os.Unsetenv("MONGODB_URI")
		}
	}()

	// Set custom URI
	customURI := "mongodb://localhost:27017"
	os.Setenv("MONGODB_URI", customURI)

	err := Connect()
	assert.NoError(t, err)
	assert.NotNil(t, DB)
}

func TestConnect_InvalidURI(t *testing.T) {
	// Save original DB state and environment
	originalDB := DB
	originalURI := os.Getenv("MONGODB_URI")
	defer func() {
		DB = originalDB
		if originalURI != "" {
			os.Setenv("MONGODB_URI", originalURI)
		} else {
			os.Unsetenv("MONGODB_URI")
		}
	}()

	// Set invalid URI
	os.Setenv("MONGO_URI", "invalid://uri")

	err := Connect()
	assert.Error(t, err)
}

func TestDisconnect_Success(t *testing.T) {
	// Create a test connection
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	// Set up test database
	originalDB := DB
	DB = client.Database("test_disconnect")
	defer func() {
		DB = originalDB
	}()

	// Test disconnect
	err = Disconnect()
	assert.NoError(t, err)
}

func TestDisconnect_NilDB(t *testing.T) {
	// Save original DB state
	originalDB := DB
	defer func() {
		DB = originalDB
	}()

	// Set DB to nil
	DB = nil

	// Should not panic when DB is nil, but may return error if client is disconnected
	err := Disconnect()
	// Either no error or "client is disconnected" error is acceptable
	if err != nil {
		assert.Contains(t, err.Error(), "client is disconnected")
	}
}
