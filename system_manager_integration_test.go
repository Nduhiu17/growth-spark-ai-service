package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/handlers"
	"growth-spark-ai-service/models"
)

func TestSystemManagerIntegration(t *testing.T) {
	// Setup test database
	testURI := "mongodb://localhost:27017/saas_marketing_testdb"
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_integration_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)

	t.Run("Complete System Manager Workflow", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organization
		orgID := primitive.NewObjectID()

		// Create test company
		company := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Integration Test Company",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
		assert.NoError(t, err)

		// Create company admin user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		adminUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "companyadmin",
			FullName:       "Company Admin",
			Email:          "admin@integration.com",
			Password:       string(hashedPassword),
			Role:           "admin",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), adminUser)
		assert.NoError(t, err)

		// Create company admin assignment
		companyAdmin := models.CompanyAdmin{
			ID:             primitive.NewObjectID(),
			CompanyID:      company.ID,
			UserID:         adminUser.ID,
			OrganizationID: orgID,
			Role:           "admin",
			Permissions:    []string{"users:*", "roles:*", "companies:read"},
			IsActive:       true,
			Status:         "active",
			AssignedAt:     time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      adminUser.ID,
			UpdatedBy:      adminUser.ID,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), companyAdmin)
		assert.NoError(t, err)

		// Setup router with authentication middleware
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user", adminUser)
			c.Next()
		})

		// Add system manager routes
		systemManagers := router.Group("/system-managers")
		{
			systemManagers.POST("/create", handlers.CreateSystemManager)
			systemManagers.GET("/company/:companyId", handlers.GetSystemManagers)
			systemManagers.DELETE("/company/:companyId/user/:userId", handlers.RemoveSystemManager)
		}

		// Step 1: Create system manager
		createRequest := models.CreateSystemManagerRequest{
			CompanyID:   company.ID.Hex(),
			Username:    "systemmanager",
			FullName:    "System Manager",
			Email:       "manager@integration.com",
			PhoneNumber: "+1234567890",
			Password:    "manager123",
			Permissions: []string{"users:read", "users:create", "roles:read", "companies:read"},
			Notes:       "Integration test system manager",
		}

		jsonBody, _ := json.Marshal(createRequest)
		req, _ := http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createResponse)
		assert.NoError(t, err)
		assert.Equal(t, "System manager created successfully", createResponse["message"])
		assert.NotNil(t, createResponse["system_manager_id"])
		assert.NotNil(t, createResponse["user_id"])
		assert.Equal(t, "manager", createResponse["role"])

		// Verify system manager was created in database
		var createdManager models.CompanyAdmin
		err = database.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{
			"company_id": company.ID,
			"role":       "manager",
		}).Decode(&createdManager)
		assert.NoError(t, err)
		assert.Equal(t, "manager", createdManager.Role)
		assert.Equal(t, company.ID, createdManager.CompanyID)
		assert.True(t, createdManager.IsActive)
		assert.Equal(t, "active", createdManager.Status)
		assert.Equal(t, []string{"users:read", "users:create", "roles:read", "companies:read"}, createdManager.Permissions)

		// Verify user was created with manager role
		var createdUser models.User
		err = database.DB.Collection("users").FindOne(context.TODO(), bson.M{
			"email": "manager@integration.com",
		}).Decode(&createdUser)
		assert.NoError(t, err)
		assert.Equal(t, "manager", createdUser.Role)
		assert.Equal(t, "systemmanager", createdUser.Username)
		assert.Equal(t, "System Manager", createdUser.FullName)

		// Step 2: Get system managers for the company
		req, _ = http.NewRequest("GET", "/system-managers/company/"+company.ID.Hex(), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var getResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &getResponse)
		assert.NoError(t, err)
		assert.NotNil(t, getResponse["system_managers"])
		assert.Equal(t, float64(1), getResponse["count"])

		// Verify the returned system manager data
		managersList := getResponse["system_managers"].([]interface{})
		assert.Len(t, managersList, 1)
		
		manager := managersList[0].(map[string]interface{})
		assert.Equal(t, "manager", manager["role"])
		assert.Equal(t, "systemmanager", manager["username"])
		assert.Equal(t, "manager@integration.com", manager["user_email"])

		// Step 3: Remove system manager
		req, _ = http.NewRequest("DELETE", "/system-managers/company/"+company.ID.Hex()+"/user/"+createdUser.ID.Hex(), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var removeResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &removeResponse)
		assert.NoError(t, err)
		assert.Equal(t, "System manager removed successfully", removeResponse["message"])

		// Verify system manager was soft deleted
		var deletedManager models.CompanyAdmin
		err = database.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{
			"company_id": company.ID,
			"user_id":    createdUser.ID,
			"role":       "manager",
		}).Decode(&deletedManager)
		assert.NoError(t, err)
		assert.False(t, deletedManager.IsActive) // Should be soft deleted

		// Step 4: Verify system manager is no longer returned in get request
		req, _ = http.NewRequest("GET", "/system-managers/company/"+company.ID.Hex(), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var finalGetResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &finalGetResponse)
		assert.NoError(t, err)
		assert.Equal(t, float64(0), finalGetResponse["count"]) // Should be 0 after removal
	})

	t.Run("System Manager Creation with Existing User", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organization
		orgID := primitive.NewObjectID()

		// Create test company
		company := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Test Company",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
		assert.NoError(t, err)

		// Create company admin user
		adminUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "companyadmin",
			Role:           "admin",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), adminUser)
		assert.NoError(t, err)

		// Create existing user
		existingUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "existinguser",
			FullName:       "Existing User",
			Email:          "existing@test.com",
			Role:           "user",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), existingUser)
		assert.NoError(t, err)

		// Create company admin assignment
		companyAdmin := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			UserID:    adminUser.ID,
			Role:      "admin",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), companyAdmin)
		assert.NoError(t, err)

		// Setup router
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user", adminUser)
			c.Next()
		})
		router.POST("/system-managers/create", handlers.CreateSystemManager)

		// Create system manager using existing user's email
		createRequest := models.CreateSystemManagerRequest{
			CompanyID: company.ID.Hex(),
			Username:  "existinguser",
			FullName:  "Existing User",
			Email:     "existing@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(createRequest)
		req, _ := http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System manager created successfully", response["message"])

		// Verify system manager assignment was created for existing user
		var managerAssignment models.CompanyAdmin
		err = database.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{
			"company_id": company.ID,
			"user_id":    existingUser.ID,
			"role":       "manager",
		}).Decode(&managerAssignment)
		assert.NoError(t, err)
		assert.Equal(t, "manager", managerAssignment.Role)
		assert.True(t, managerAssignment.IsActive)
	})

	t.Run("Super Admin Can Create System Manager for Any Company", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organization
		orgID := primitive.NewObjectID()

		// Create test company
		company := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Test Company",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
		assert.NoError(t, err)

		// Create super admin user
		superAdmin := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "superadmin",
			Role:           "super_admin",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), superAdmin)
		assert.NoError(t, err)

		// Setup router
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user", superAdmin)
			c.Next()
		})
		router.POST("/system-managers/create", handlers.CreateSystemManager)

		// Super admin creates system manager for any company (no need to be admin of that company)
		createRequest := models.CreateSystemManagerRequest{
			CompanyID: company.ID.Hex(),
			Username:  "systemmanager",
			FullName:  "System Manager",
			Email:     "manager@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(createRequest)
		req, _ := http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System manager created successfully", response["message"])
	})

	t.Run("Duplicate Prevention Integration Test", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organization
		orgID := primitive.NewObjectID()

		// Create test company
		company := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Test Company",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
		assert.NoError(t, err)

		// Create company admin user
		adminUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "companyadmin",
			Role:           "admin",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), adminUser)
		assert.NoError(t, err)

		// Create company admin assignment
		companyAdmin := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			UserID:    adminUser.ID,
			Role:      "admin",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), companyAdmin)
		assert.NoError(t, err)

		// Setup router
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("user", adminUser)
			c.Next()
		})
		router.POST("/system-managers/create", handlers.CreateSystemManager)

		// First request - should succeed
		createRequest := models.CreateSystemManagerRequest{
			CompanyID: company.ID.Hex(),
			Username:  "systemmanager",
			FullName:  "System Manager",
			Email:     "manager@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(createRequest)
		req, _ := http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Second request with same email - should fail with conflict
		req, _ = http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User is already assigned to this company", response["error"])
	})
}
