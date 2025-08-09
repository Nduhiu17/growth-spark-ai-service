package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	"growth-spark-ai-service/models"
)

func TestCreateSystemManager(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Success - Company Admin Creates System Manager", func(t *testing.T) {
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
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		adminUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "companyadmin",
			FullName:       "Company Admin",
			Email:          "admin@company.com",
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
			Permissions:    []string{"users:*", "roles:*"},
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

		// Setup router with middleware
		router.POST("/system-managers/create", func(c *gin.Context) {
			c.Set("user", adminUser)
			CreateSystemManager(c)
		})

		// Create request payload
		requestBody := models.CreateSystemManagerRequest{
			CompanyID:   company.ID.Hex(),
			Username:    "systemmanager",
			FullName:    "System Manager",
			Email:       "manager@company.com",
			Password:    "password123",
			Permissions: []string{"users:read", "users:create", "roles:read"},
			Notes:       "Test system manager",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System manager created successfully", response["message"])
		assert.NotNil(t, response["system_manager_id"])
		assert.NotNil(t, response["user_id"])
		assert.Equal(t, "manager", response["role"])

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
	})

	t.Run("Success - Super Admin Creates System Manager", func(t *testing.T) {
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
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		superAdmin := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "superadmin",
			FullName:       "Super Admin",
			Email:          "super@admin.com",
			Password:       string(hashedPassword),
			Role:           "super_admin",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), superAdmin)
		assert.NoError(t, err)

		// Setup router with middleware
		router.POST("/system-managers/create-super", func(c *gin.Context) {
			c.Set("user", superAdmin)
			CreateSystemManager(c)
		})

		// Create request payload
		requestBody := models.CreateSystemManagerRequest{
			CompanyID: company.ID.Hex(),
			Username:  "systemmanager2",
			FullName:  "System Manager 2",
			Email:     "manager2@company.com",
			Password:  "password123",
			Notes:     "Test system manager by super admin",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create-super", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System manager created successfully", response["message"])
	})

	t.Run("Error - Unauthorized User", func(t *testing.T) {
		// Create regular user
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.POST("/system-managers/create-unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			CreateSystemManager(c)
		})

		requestBody := models.CreateSystemManagerRequest{
			CompanyID: primitive.NewObjectID().Hex(),
			Username:  "manager",
			FullName:  "Manager",
			Email:     "manager@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create-unauthorized", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only company admins can create system managers", response["error"])
	})

	t.Run("Error - Company Admin Not Admin of Target Company", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organization
		orgID := primitive.NewObjectID()

		// Create test companies
		company1 := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Company 1",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		company2 := models.Company{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Name:           "Company 2",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err := database.DB.Collection("companies").InsertMany(context.TODO(), []interface{}{company1, company2})
		assert.NoError(t, err)

		// Create company admin user (admin of company1 only)
		adminUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "companyadmin",
			Role:           "admin",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), adminUser)
		assert.NoError(t, err)

		// Create company admin assignment for company1 only
		companyAdmin := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company1.ID,
			UserID:    adminUser.ID,
			Role:      "admin",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), companyAdmin)
		assert.NoError(t, err)

		router.POST("/system-managers/create-wrong-company", func(c *gin.Context) {
			c.Set("user", adminUser)
			CreateSystemManager(c)
		})

		// Try to create system manager for company2 (not admin of this company)
		requestBody := models.CreateSystemManagerRequest{
			CompanyID: company2.ID.Hex(),
			Username:  "manager",
			FullName:  "Manager",
			Email:     "manager@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create-wrong-company", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "You can only create system managers for companies you administer", response["error"])
	})

	t.Run("Error - Duplicate System Manager", func(t *testing.T) {
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

		// Create existing user that will be assigned as system manager
		existingUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "existinguser",
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

		// Create existing system manager assignment
		existingManager := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			UserID:    existingUser.ID,
			Role:      "manager",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), existingManager)
		assert.NoError(t, err)

		router.POST("/system-managers/create-duplicate", func(c *gin.Context) {
			c.Set("user", adminUser)
			CreateSystemManager(c)
		})

		// Try to create system manager for existing user
		requestBody := models.CreateSystemManagerRequest{
			CompanyID: company.ID.Hex(),
			Username:  "existinguser",
			FullName:  "Existing User",
			Email:     "existing@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create-duplicate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "User is already assigned to this company", response["error"])
	})

	t.Run("Error - Invalid Company ID", func(t *testing.T) {
		adminUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "companyadmin",
			Role:     "admin",
		}

		router.POST("/system-managers/create-invalid-company", func(c *gin.Context) {
			c.Set("user", adminUser)
			CreateSystemManager(c)
		})

		requestBody := models.CreateSystemManagerRequest{
			CompanyID: "invalid-id",
			Username:  "manager",
			FullName:  "Manager",
			Email:     "manager@test.com",
			Password:  "password123",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/system-managers/create-invalid-company", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid company ID", response["error"])
	})
}

func TestGetSystemManagers(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Success - Company Admin Gets System Managers", func(t *testing.T) {
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

		// Create system manager user
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Email:          "manager@test.com",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
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

		// Create system manager assignment
		systemManager := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			UserID:    managerUser.ID,
			Role:      "manager",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), systemManager)
		assert.NoError(t, err)

		router.GET("/system-managers/company/:companyId", func(c *gin.Context) {
			c.Set("user", adminUser)
			GetSystemManagers(c)
		})

		req, _ := http.NewRequest("GET", "/system-managers/company/"+company.ID.Hex(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["system_managers"])
		assert.Equal(t, float64(1), response["count"])
	})

	t.Run("Error - Unauthorized User", func(t *testing.T) {
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.GET("/system-managers/company/:companyId/unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			GetSystemManagers(c)
		})

		req, _ := http.NewRequest("GET", "/system-managers/company/"+primitive.NewObjectID().Hex()+"/unauthorized", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only company admins can view system managers", response["error"])
	})
}

func TestRemoveSystemManager(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Success - Company Admin Removes System Manager", func(t *testing.T) {
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

		// Create system manager user
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
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

		// Create system manager assignment
		systemManager := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			UserID:    managerUser.ID,
			Role:      "manager",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), systemManager)
		assert.NoError(t, err)

		router.DELETE("/system-managers/company/:companyId/user/:userId", func(c *gin.Context) {
			c.Set("user", adminUser)
			RemoveSystemManager(c)
		})

		req, _ := http.NewRequest("DELETE", "/system-managers/company/"+company.ID.Hex()+"/user/"+managerUser.ID.Hex(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "System manager removed successfully", response["message"])

		// Verify system manager was soft deleted
		var deletedManager models.CompanyAdmin
		err = database.DB.Collection("company_admins").FindOne(context.TODO(), bson.M{
			"company_id": company.ID,
			"user_id":    managerUser.ID,
		}).Decode(&deletedManager)
		assert.NoError(t, err)
		assert.False(t, deletedManager.IsActive) // Should be soft deleted
	})

	t.Run("Error - Unauthorized User", func(t *testing.T) {
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.DELETE("/system-managers/company/:companyId/user/:userId/unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			RemoveSystemManager(c)
		})

		req, _ := http.NewRequest("DELETE", "/system-managers/company/"+primitive.NewObjectID().Hex()+"/user/"+primitive.NewObjectID().Hex()+"/unauthorized", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only company admins can remove system managers", response["error"])
	})
}
