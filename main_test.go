package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/handlers"
	"growth-spark-ai-service/middleware"
)

func TestMain(m *testing.M) {
	// Set up test environment
	gin.SetMode(gin.TestMode)
	
	// Set test database URI
	os.Setenv("MONGO_TEST_URI", "mongodb://localhost:27017/saas_marketing_testdb")
	
	// Run tests
	code := m.Run()
	
	// Clean up
	os.Exit(code)
}

// setupTestRouter creates a router with the same configuration as main.go
func setupTestRouter() *gin.Engine {
	router := gin.Default()

	// API routes - same structure as main.go
	api := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		api.POST("/organizations", handlers.CreateOrganization)
		api.POST("/auth/login", handlers.Login)
		api.POST("/auth/register", handlers.Register)

		// Protected routes (authentication required)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/profile", handlers.GetProfile)
			protected.GET("/permissions", middleware.GetUserPermissions)

			// Role management routes (admin only)
			roles := protected.Group("/roles")
			roles.Use(middleware.RequirePermission("roles:read"))
			{
				roles.GET("/", handlers.GetRoles)
				roles.GET("/:id", handlers.GetRole)
				roles.GET("/permissions", handlers.GetPermissions)
			}

			// Role creation, update, delete (requires specific permissions)
			protected.POST("/roles", middleware.RequirePermission("roles:create"), handlers.CreateRole)
			protected.PUT("/roles/:id", middleware.RequirePermission("roles:update"), handlers.UpdateRole)
			protected.DELETE("/roles/:id", middleware.RequirePermission("roles:delete"), handlers.DeleteRole)
			protected.POST("/roles/assign", middleware.RequirePermission("users:update"), handlers.AssignRole)

			// Company management routes (super admin only)
			companies := protected.Group("/companies")
			{
				companies.POST("/", handlers.CreateCompany)
				companies.GET("/", handlers.GetAllCompanies)
				companies.GET("/:id", handlers.GetCompany)
				companies.PUT("/:id", handlers.UpdateCompany)
				companies.DELETE("/:id", handlers.DeleteCompany)
				
				// Company admin management routes (super admin only)
				companies.GET("/:id/admins", handlers.GetCompanyAdmins)
			}

			// Company admin management routes (separate group to avoid path conflicts)
			companyAdmins := protected.Group("/company-admins")
			{
				companyAdmins.POST("/create", handlers.CreateCompanyAdmin)
				companyAdmins.PUT("/update/:adminId", handlers.UpdateCompanyAdmin)
				companyAdmins.DELETE("/delete/:adminId", handlers.RemoveCompanyAdmin)
			}
		}
	}

	return router
}

func TestRouterSetup(t *testing.T) {
	// Skip if MongoDB is not available
	if os.Getenv("MONGO_TEST_URI") == "" && os.Getenv("MONGO_URI") == "" {
		t.Skip("MongoDB not available for testing")
	}

	// Connect to test database
	err := database.Connect()
	require.NoError(t, err)
	defer func() {
		database.Disconnect()
	}()

	router := setupTestRouter()
	assert.NotNil(t, router)

	// Test public routes exist
	t.Run("PublicRoutes", func(t *testing.T) {
		// Test organization creation endpoint exists
		req, _ := http.NewRequest("POST", "/api/v1/organizations", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should not be 404 (route exists)
		assert.NotEqual(t, http.StatusNotFound, w.Code)

		// Test login endpoint exists
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code)

		// Test register endpoint exists
		req, _ = http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("ProtectedRoutes", func(t *testing.T) {
		// Test protected routes require authentication
		req, _ := http.NewRequest("GET", "/api/v1/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// Should be unauthorized, not not found
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Test role routes require authentication
		req, _ = http.NewRequest("GET", "/api/v1/roles/", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Test company routes require authentication
		req, _ = http.NewRequest("GET", "/api/v1/companies/", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestEnvironmentVariableHandling(t *testing.T) {
	t.Run("DefaultPort", func(t *testing.T) {
		// Clear PORT environment variable
		originalPort := os.Getenv("PORT")
		os.Unsetenv("PORT")
		defer func() {
			if originalPort != "" {
				os.Setenv("PORT", originalPort)
			}
		}()

		// Test that default port logic would work
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		assert.Equal(t, "8080", port)
	})

	t.Run("CustomPort", func(t *testing.T) {
		// Set custom PORT environment variable
		originalPort := os.Getenv("PORT")
		os.Setenv("PORT", "9090")
		defer func() {
			if originalPort != "" {
				os.Setenv("PORT", originalPort)
			} else {
				os.Unsetenv("PORT")
			}
		}()

		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		assert.Equal(t, "9090", port)
	})
}

func TestDatabaseConnectionInMain(t *testing.T) {
	// Skip if MongoDB is not available
	if os.Getenv("MONGO_TEST_URI") == "" && os.Getenv("MONGO_URI") == "" {
		t.Skip("MongoDB not available for testing")
	}

	t.Run("DatabaseConnect", func(t *testing.T) {
		// Test database connection (simulating main.go behavior)
		err := database.Connect()
		assert.NoError(t, err)

		// Test database disconnect (simulating defer in main.go)
		err = database.Disconnect()
		assert.NoError(t, err)
	})
}

func TestCompleteAPIWorkflow(t *testing.T) {
	// Skip if MongoDB is not available
	if os.Getenv("MONGO_TEST_URI") == "" && os.Getenv("MONGO_URI") == "" {
		t.Skip("MongoDB not available for testing")
	}

	// Connect to test database
	err := database.Connect()
	require.NoError(t, err)
	defer func() {
		database.Disconnect()
	}()

	router := setupTestRouter()

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// 1. Create organization and user (register)
		registerReq := map[string]interface{}{
			"organizationName": "Test Org Main",
			"fullName":         "Test User Main",
			"email":            "testmain@example.com",
			"username":         "testusermain",
			"phoneNumber":      "+1234567890",
			"password":         "password123",
		}
		
		reqBody, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should successfully register
		assert.Equal(t, http.StatusCreated, w.Code)

		// 2. Login to get token
		loginReq := map[string]interface{}{
			"email":    "testmain@example.com",
			"password": "password123",
		}
		
		reqBody, _ = json.Marshal(loginReq)
		req, _ = http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var loginResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)
		token, ok := loginResp["token"].(string)
		require.True(t, ok, "Token should be a string")

		// 3. Access protected route with token
		req, _ = http.NewRequest("GET", "/api/v1/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRouteGroupsAndMiddleware(t *testing.T) {
	router := setupTestRouter()

	t.Run("APIGroupExists", func(t *testing.T) {
		// Test that API group routes are properly configured
		req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should be 404 for non-existent route, not 500
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("MiddlewareChaining", func(t *testing.T) {
		// Test that middleware is properly applied to protected routes
		req, _ := http.NewRequest("GET", "/api/v1/roles/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should be unauthorized due to AuthMiddleware
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
