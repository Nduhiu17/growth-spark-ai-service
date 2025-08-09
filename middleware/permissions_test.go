package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

func setupPermissionsTestDB(t *testing.T) {
	// Connect to test database
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	database.DB = client.Database("growth_spark_test_permissions")
	
	// Clean up function
	t.Cleanup(func() {
		database.DB.Drop(context.TODO())
		client.Disconnect(context.TODO())
	})
}

func TestRequirePermission_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequirePermission("create_user"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User role not found")
}

func TestRequirePermission_MissingOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})
	router.Use(RequirePermission("create_user"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Organization not found")
}

func TestRequirePermission_InsufficientPermissions(t *testing.T) {
	setupPermissionsTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create a test organization and role with limited permissions
	orgID := primitive.NewObjectID()
	role := models.Role{
		ID:             primitive.NewObjectID(),
		OrganizationID: orgID,
		Name:           "viewer",
		Permissions:    []string{"read_user"}, // Only read permission, not create
	}
	
	rolesCollection := database.DB.Collection("roles")
	_, err := rolesCollection.InsertOne(context.TODO(), role)
	require.NoError(t, err)
	
	router.Use(func(c *gin.Context) {
		c.Set("role", "viewer")
		c.Set("organization_id", orgID)
		c.Next()
	})
	router.Use(RequirePermission("create_user"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions")
}

func TestRequireAnyPermission_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireAnyPermission([]string{"create_user", "edit_user"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User role not found")
}

func TestRequireAnyPermission_MissingOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})
	router.Use(RequireAnyPermission([]string{"create_user", "edit_user"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Organization not found")
}

func TestRequireAnyPermission_InsufficientPermissions(t *testing.T) {
	setupPermissionsTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create a test organization and role with limited permissions
	orgID := primitive.NewObjectID()
	role := models.Role{
		ID:             primitive.NewObjectID(),
		OrganizationID: orgID,
		Name:           "viewer",
		Permissions:    []string{"read_user"}, // Only read permission, not create or edit
	}
	
	rolesCollection := database.DB.Collection("roles")
	_, err := rolesCollection.InsertOne(context.TODO(), role)
	require.NoError(t, err)
	
	router.Use(func(c *gin.Context) {
		c.Set("role", "viewer")
		c.Set("organization_id", orgID)
		c.Next()
	})
	router.Use(RequireAnyPermission([]string{"create_user", "edit_user"}))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions")
}

func TestHasPermission(t *testing.T) {
	permissions := []string{"users:create", "users:edit", "users:view"}

	// Test exact match
	assert.True(t, hasPermission(permissions, "users:create"))
	assert.True(t, hasPermission(permissions, "users:edit"))
	assert.True(t, hasPermission(permissions, "users:view"))

	// Test wildcard permissions
	wildcardPermissions := []string{"users:*", "admin:*"}
	assert.True(t, hasPermission(wildcardPermissions, "users:create"))
	assert.True(t, hasPermission(wildcardPermissions, "users:edit"))
	assert.True(t, hasPermission(wildcardPermissions, "admin:manage"))

	// Test universal wildcard
	universalPermissions := []string{"*"}
	assert.True(t, hasPermission(universalPermissions, "users:create"))
	assert.True(t, hasPermission(universalPermissions, "admin:manage"))

	// Test no match
	assert.False(t, hasPermission(permissions, "users:delete"))
	assert.False(t, hasPermission(wildcardPermissions, "roles:create"))

	// Test empty permissions
	assert.False(t, hasPermission([]string{}, "users:create"))

	// Test empty required permission
	assert.False(t, hasPermission(permissions, ""))
}

func TestGetUserPermissions_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/permissions", GetUserPermissions)

	req, _ := http.NewRequest("GET", "/permissions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User role not found")
}

func TestGetUserPermissions_MissingOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Next()
	})
	router.GET("/permissions", GetUserPermissions)

	req, _ := http.NewRequest("GET", "/permissions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Organization not found")
}

// Test helper functions
func TestCheckUserPermission_InvalidRole(t *testing.T) {
	orgID := primitive.NewObjectID()
	hasPermission, err := checkUserPermission("nonexistent_role", orgID, "create_user")
	assert.Error(t, err)
	assert.False(t, hasPermission)
}

func TestCheckUserPermission_DatabaseError(t *testing.T) {
	// Test with invalid ObjectID to trigger database error
	hasPermission, err := checkUserPermission("admin", primitive.NilObjectID, "create_user")
	assert.Error(t, err)
	assert.False(t, hasPermission)
}
