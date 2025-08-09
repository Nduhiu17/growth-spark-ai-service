package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/handlers"
	"growth-spark-ai-service/models"
)

func TestCompanyAdminIntegration_FullWorkflow(t *testing.T) {
	// Skip if MongoDB is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	th := SetupTestEnvironment(t)
	defer th.Cleanup()

	// Setup test data
	org := th.CreateTestOrganization(t)
	superAdmin := th.CreateTestUser(t, org.ID, "super_admin")
	company := th.CreateTestCompany(t, org.ID, superAdmin.ID)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Add middleware to set user in context
	router.Use(func(c *gin.Context) {
		c.Set("user", superAdmin)
		c.Next()
	})

	// Setup routes
	api := router.Group("/api/v1")
	{
		api.POST("/company-admins/create", handlers.CreateCompanyAdmin)
		api.GET("/companies/:id/admins", handlers.GetCompanyAdmins)
		api.PUT("/company-admins/update/:adminId", handlers.UpdateCompanyAdmin)
		api.DELETE("/company-admins/delete/:adminId", handlers.RemoveCompanyAdmin)
	}

	t.Run("Create Company Admin", func(t *testing.T) {
		requestBody := models.CreateCompanyAdminRequest{
			CompanyID:   company.ID.Hex(),
			Username:    "newadmin",
			FullName:    "New Admin User",
			Email:       "newadmin@example.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Role:        "admin",
			Permissions: []string{"create_user", "edit_user", "delete_user"},
			Status:      "active",
			Notes:       "Created via integration test",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/company-admins/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "message")
		assert.Contains(t, response, "companyAdmin")

		// Verify company admin was created
		companyAdmin := response["companyAdmin"].(map[string]interface{})
		assert.Equal(t, "newadmin", companyAdmin["username"])
		assert.Equal(t, "admin", companyAdmin["role"])
		assert.Equal(t, true, companyAdmin["is_active"])
		assert.NotEmpty(t, companyAdmin["id"])
	})

	t.Run("Get Company Admins", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/companies/"+company.ID.Hex()+"/admins", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "companyAdmins")
		assert.Contains(t, response, "companyName")
		assert.Contains(t, response, "total")
		
		admins := response["companyAdmins"].([]interface{})
		assert.GreaterOrEqual(t, len(admins), 1)
	})

	t.Run("Update Company Admin", func(t *testing.T) {
		// First create a company admin to update
		user := th.CreateTestUser(t, org.ID, "user")
		companyAdmin := th.CreateTestCompanyAdmin(t, company.ID, user.ID, org.ID, superAdmin.ID)

		updateRequest := models.UpdateCompanyAdminRequest{
			Role:        "manager",
			Permissions: []string{"view_users", "edit_users"},
			Status:      "active",
			Notes:       "Updated via integration test",
		}

		jsonBody, _ := json.Marshal(updateRequest)
		req, _ := http.NewRequest("PUT", "/api/v1/company-admins/update/"+companyAdmin.ID.Hex(), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Company admin updated successfully", response["message"])
	})

	t.Run("Remove Company Admin", func(t *testing.T) {
		// Create a company admin to remove
		user := th.CreateTestUser(t, org.ID, "user")
		companyAdmin := th.CreateTestCompanyAdmin(t, company.ID, user.ID, org.ID, superAdmin.ID)

		req, _ := http.NewRequest("DELETE", "/api/v1/company-admins/delete/"+companyAdmin.ID.Hex(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Company admin removed successfully", response["message"])

		// Verify soft delete
		th.AssertCompanyAdminInactive(t, companyAdmin.ID)
	})
}

func TestCompanyAdminIntegration_DuplicatePreventionWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	th := SetupTestEnvironment(t)
	defer th.Cleanup()

	// Setup test data
	org := th.CreateTestOrganization(t)
	superAdmin := th.CreateTestUser(t, org.ID, "super_admin")
	company := th.CreateTestCompany(t, org.ID, superAdmin.ID)
	existingUser := th.CreateTestUser(t, org.ID, "user")

	// Create existing company admin assignment
	th.CreateTestCompanyAdmin(t, company.ID, existingUser.ID, org.ID, superAdmin.ID)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", superAdmin)
		c.Next()
	})
	router.POST("/api/v1/company-admins/create", handlers.CreateCompanyAdmin)

	t.Run("Prevent Duplicate Assignment", func(t *testing.T) {
		// Try to create duplicate assignment using existing user credentials
		requestBody := models.CreateCompanyAdminRequest{
			CompanyID:   company.ID.Hex(),
			Username:    existingUser.Username,
			FullName:    existingUser.FullName,
			Email:       existingUser.Email,
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Role:        "admin",
			Permissions: []string{"create_user"},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/company-admins/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return conflict error
		assert.Equal(t, http.StatusConflict, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User is already a company admin for this company", response["error"])
	})
}

func TestCompanyAdminIntegration_PermissionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	th := SetupTestEnvironment(t)
	defer th.Cleanup()

	// Setup test data
	org := th.CreateTestOrganization(t)
	regularUser := th.CreateTestUser(t, org.ID, "user") // Not super admin
	company := th.CreateTestCompany(t, org.ID, regularUser.ID)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", regularUser) // Set regular user (not super admin)
		c.Next()
	})
	router.POST("/api/v1/company-admins/create", handlers.CreateCompanyAdmin)

	t.Run("Reject Non-Super Admin", func(t *testing.T) {
		requestBody := models.CreateCompanyAdminRequest{
			CompanyID:   company.ID.Hex(),
			Username:    "newadmin",
			FullName:    "New Admin",
			Email:       "newadmin@example.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Role:        "admin",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/company-admins/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return forbidden error
		assert.Equal(t, http.StatusForbidden, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only super admins can create company admins", response["error"])
	})
}

func TestCompanyAdminIntegration_DataConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	th := SetupTestEnvironment(t)
	defer th.Cleanup()

	// Setup test data
	org := th.CreateTestOrganization(t)
	superAdmin := th.CreateTestUser(t, org.ID, "super_admin")
	company1 := th.CreateTestCompany(t, org.ID, superAdmin.ID)
	company2 := th.CreateTestCompany(t, org.ID, superAdmin.ID)
	user := th.CreateTestUser(t, org.ID, "user")

	t.Run("User Can Be Admin of Multiple Companies", func(t *testing.T) {
		// Create company admin for company1
		admin1 := th.CreateTestCompanyAdmin(t, company1.ID, user.ID, org.ID, superAdmin.ID)
		
		// Create company admin for company2
		admin2 := th.CreateTestCompanyAdmin(t, company2.ID, user.ID, org.ID, superAdmin.ID)

		// Verify both assignments exist
		th.AssertCompanyAdminExists(t, company1.ID, user.ID)
		th.AssertCompanyAdminExists(t, company2.ID, user.ID)

		// Verify user has 2 company admin assignments
		count := th.GetUserCompanyAdminCount(t, user.ID)
		assert.Equal(t, 2, count)

		// Verify each company has 1 admin
		count1 := th.GetCompanyAdminCount(t, company1.ID)
		count2 := th.GetCompanyAdminCount(t, company2.ID)
		assert.Equal(t, 1, count1)
		assert.Equal(t, 1, count2)

		// Verify admin IDs are different
		assert.NotEqual(t, admin1.ID, admin2.ID)
	})

	t.Run("Company Can Have Multiple Admins", func(t *testing.T) {
		user1 := th.CreateTestUser(t, org.ID, "user")
		user2 := th.CreateTestUser(t, org.ID, "user")

		// Create multiple admins for the same company
		th.CreateTestCompanyAdmin(t, company1.ID, user1.ID, org.ID, superAdmin.ID)
		th.CreateTestCompanyAdmin(t, company1.ID, user2.ID, org.ID, superAdmin.ID)

		// Verify both assignments exist
		th.AssertCompanyAdminExists(t, company1.ID, user1.ID)
		th.AssertCompanyAdminExists(t, company1.ID, user2.ID)

		// Verify company has multiple admins (including the one from previous test)
		count := th.GetCompanyAdminCount(t, company1.ID)
		assert.GreaterOrEqual(t, count, 3) // At least 3 admins
	})
}

func TestCompanyAdminIntegration_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	th := SetupTestEnvironment(t)
	defer th.Cleanup()

	// Setup test data
	org := th.CreateTestOrganization(t)
	superAdmin := th.CreateTestUser(t, org.ID, "super_admin")
	company := th.CreateTestCompany(t, org.ID, superAdmin.ID)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", superAdmin)
		c.Next()
	})
	router.POST("/api/v1/company-admins/create", handlers.CreateCompanyAdmin)

	t.Run("Invalid Company ID", func(t *testing.T) {
		requestBody := models.CreateCompanyAdminRequest{
			CompanyID:   "invalid-id",
			Username:    "newadmin",
			FullName:    "New Admin",
			Email:       "newadmin@example.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Role:        "admin",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/company-admins/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid company ID", response["error"])
	})

	t.Run("Non-existent Company", func(t *testing.T) {
		requestBody := models.CreateCompanyAdminRequest{
			CompanyID:   primitive.NewObjectID().Hex(), // Valid format but non-existent
			Username:    "newadmin",
			FullName:    "New Admin",
			Email:       "newadmin@example.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Role:        "admin",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/api/v1/company-admins/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Company not found", response["error"])
	})

	t.Run("Expired Company Admin Assignment", func(t *testing.T) {
		user := th.CreateTestUser(t, org.ID, "user")
		
		// Create expired company admin
		expiredTime := time.Now().Add(-24 * time.Hour)
		companyAdmin := models.CompanyAdmin{
			CompanyID:      company.ID,
			UserID:         user.ID,
			OrganizationID: org.ID,
			Role:           "admin",
			Permissions:    []string{"create_user"},
			IsActive:       true,
			Status:         "expired",
			AssignedAt:     time.Now().Add(-48 * time.Hour),
			ExpiresAt:      &expiredTime,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      superAdmin.ID,
			UpdatedBy:      superAdmin.ID,
		}

		// Test expiration check
		assert.True(t, companyAdmin.IsExpired())
		assert.Equal(t, "expired", companyAdmin.Status)
	})
}
