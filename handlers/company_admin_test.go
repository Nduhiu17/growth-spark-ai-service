package handlers

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
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// Test setup helpers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func setupTestDatabase(t *testing.T) func() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)

	testDB := client.Database("growth_spark_test_handlers_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB

	return func() {
		testDB.Drop(context.TODO())
		client.Disconnect(context.TODO())
		database.DB = originalDB
	}
}

func createTestUser(t *testing.T, role string) models.User {
	return models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Username:       "testuser",
		FullName:       "Test User",
		Email:          "test@example.com",
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createTestCompany(t *testing.T, organizationID primitive.ObjectID) models.Company {
	return models.Company{
		ID:             primitive.NewObjectID(),
		OrganizationID: organizationID,
		Name:           "Test Company",
		Description:    "Test Description",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestCreateCompanyAdmin_Success(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.POST("/company-admins/create", CreateCompanyAdmin)

	// Create test user (super admin)
	superAdmin := createTestUser(t, "super_admin")
	
	// Create test company
	company := createTestCompany(t, superAdmin.OrganizationID)
	companiesCollection := database.DB.Collection("companies")
	_, err := companiesCollection.InsertOne(context.TODO(), company)
	require.NoError(t, err)

	// Prepare request
	requestBody := models.CreateCompanyAdminRequest{
		CompanyID:   company.ID.Hex(),
		Username:    "newadmin",
		FullName:    "New Admin",
		Email:       "newadmin@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		Role:        "admin",
		Permissions: []string{"create_user", "edit_user"},
		Status:      "active",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/company-admins/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()
	
	// Set user in context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user", superAdmin)

	// Call handler
	CreateCompanyAdmin(c)

	// Assert response
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "companyAdmin")
	
	// Verify company admin details
	companyAdmin := response["companyAdmin"].(map[string]interface{})
	assert.Equal(t, "newadmin", companyAdmin["username"])
	assert.Equal(t, "admin", companyAdmin["role"])
	assert.Equal(t, true, companyAdmin["is_active"])
}

func TestCreateCompanyAdmin_UnauthorizedUser(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.POST("/company-admins/create", CreateCompanyAdmin)

	// Create test user (not super admin)
	regularUser := createTestUser(t, "user")

	// Prepare request
	requestBody := models.CreateCompanyAdminRequest{
		CompanyID:   primitive.NewObjectID().Hex(),
		Username:    "newadmin",
		FullName:    "New Admin",
		Email:       "newadmin@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		Role:        "admin",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/company-admins/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user", regularUser)

	CreateCompanyAdmin(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Only super admins can create company admins", response["error"])
}

func TestCreateCompanyAdmin_InvalidRequest(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.POST("/company-admins/create", CreateCompanyAdmin)

	superAdmin := createTestUser(t, "super_admin")

	// Invalid request (missing required fields)
	requestBody := map[string]interface{}{
		"username": "newadmin",
		// Missing other required fields
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/company-admins/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user", superAdmin)

	CreateCompanyAdmin(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid request format", response["error"])
}

func TestCreateCompanyAdmin_CompanyNotFound(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.POST("/company-admins/create", CreateCompanyAdmin)

	superAdmin := createTestUser(t, "super_admin")

	// Request with non-existent company ID
	requestBody := models.CreateCompanyAdminRequest{
		CompanyID:   primitive.NewObjectID().Hex(), // Non-existent company
		Username:    "newadmin",
		FullName:    "New Admin",
		Email:       "newadmin@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		Role:        "admin",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/company-admins/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user", superAdmin)

	CreateCompanyAdmin(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Company not found", response["error"])
}

func TestCreateCompanyAdmin_DuplicateAssignment(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.POST("/company-admins/create", CreateCompanyAdmin)

	superAdmin := createTestUser(t, "super_admin")
	company := createTestCompany(t, superAdmin.OrganizationID)

	// Insert company
	companiesCollection := database.DB.Collection("companies")
	_, err := companiesCollection.InsertOne(context.TODO(), company)
	require.NoError(t, err)

	// Create existing user
	existingUser := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: superAdmin.OrganizationID,
		Username:       "existinguser",
		FullName:       "Existing User",
		Email:          "existing@example.com",
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	usersCollection := database.DB.Collection("users")
	_, err = usersCollection.InsertOne(context.TODO(), existingUser)
	require.NoError(t, err)

	// Create existing company admin assignment
	companyAdminDB := database.NewCompanyAdminDB()
	existingAdmin := &models.CompanyAdmin{
		CompanyID:      company.ID,
		UserID:         existingUser.ID,
		OrganizationID: superAdmin.OrganizationID,
		Role:           "admin",
		IsActive:       true,
		Status:         "active",
		AssignedAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	err = companyAdminDB.Create(existingAdmin)
	require.NoError(t, err)

	// Try to create duplicate assignment
	requestBody := models.CreateCompanyAdminRequest{
		CompanyID:   company.ID.Hex(),
		Username:    "existinguser",
		FullName:    "Existing User",
		Email:       "existing@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		Role:        "admin",
	}

	jsonBody, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", "/company-admins/create", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user", superAdmin)

	CreateCompanyAdmin(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User is already a company admin for this company", response["error"])
}

func TestGetCompanyAdmins_Success(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.GET("/companies/:id/admins", GetCompanyAdmins)

	superAdmin := createTestUser(t, "super_admin")
	company := createTestCompany(t, superAdmin.OrganizationID)

	// Insert company
	companiesCollection := database.DB.Collection("companies")
	_, err := companiesCollection.InsertOne(context.TODO(), company)
	require.NoError(t, err)

	// Create test users and company admins
	companyAdminDB := database.NewCompanyAdminDB()
	usersCollection := database.DB.Collection("users")

	for i := 0; i < 2; i++ {
		user := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: superAdmin.OrganizationID,
			Username:       "admin" + string(rune(i+'1')),
			FullName:       "Admin " + string(rune(i+'1')),
			Email:          "admin" + string(rune(i+'1')) + "@example.com",
			Role:           "user",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = usersCollection.InsertOne(context.TODO(), user)
		require.NoError(t, err)

		companyAdmin := &models.CompanyAdmin{
			CompanyID:      company.ID,
			UserID:         user.ID,
			OrganizationID: superAdmin.OrganizationID,
			Role:           "admin",
			IsActive:       true,
			Status:         "active",
			AssignedAt:     time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      superAdmin.ID,
			UpdatedBy:      superAdmin.ID,
		}
		err = companyAdminDB.Create(companyAdmin)
		require.NoError(t, err)
	}

	req, _ := http.NewRequest("GET", "/companies/"+company.ID.Hex()+"/admins", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "id", Value: company.ID.Hex()}}
	c.Set("user", superAdmin)

	GetCompanyAdmins(c)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "companyAdmins")
	assert.Contains(t, response, "companyName")
	assert.Contains(t, response, "total")
	
	admins := response["companyAdmins"].([]interface{})
	assert.Len(t, admins, 2)
	assert.Equal(t, "Test Company", response["companyName"])
	assert.Equal(t, float64(2), response["total"])
}

func TestUpdateCompanyAdmin_Success(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.PUT("/company-admins/update/:adminId", UpdateCompanyAdmin)

	superAdmin := createTestUser(t, "super_admin")
	company := createTestCompany(t, superAdmin.OrganizationID)

	// Create test company admin
	companyAdminDB := database.NewCompanyAdminDB()
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      company.ID,
		UserID:         primitive.NewObjectID(),
		OrganizationID: superAdmin.OrganizationID,
		Role:           "admin",
		Permissions:    []string{"create_user"},
		IsActive:       true,
		Status:         "active",
		AssignedAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	err := companyAdminDB.Create(companyAdmin)
	require.NoError(t, err)

	// Update request
	updateRequest := models.UpdateCompanyAdminRequest{
		Role:        "manager",
		Permissions: []string{"view_users"},
		Status:      "active",
		Notes:       "Updated role",
	}

	jsonBody, _ := json.Marshal(updateRequest)
	req, _ := http.NewRequest("PUT", "/company-admins/update/"+companyAdmin.ID.Hex(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "adminId", Value: companyAdmin.ID.Hex()}}
	c.Set("user", superAdmin)

	UpdateCompanyAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Company admin updated successfully", response["message"])
}

func TestRemoveCompanyAdmin_Success(t *testing.T) {
	cleanup := setupTestDatabase(t)
	defer cleanup()

	router := setupTestRouter()
	router.DELETE("/company-admins/delete/:adminId", RemoveCompanyAdmin)

	superAdmin := createTestUser(t, "super_admin")
	company := createTestCompany(t, superAdmin.OrganizationID)

	// Create test company admin
	companyAdminDB := database.NewCompanyAdminDB()
	companyAdmin := &models.CompanyAdmin{
		CompanyID:      company.ID,
		UserID:         primitive.NewObjectID(),
		OrganizationID: superAdmin.OrganizationID,
		Role:           "admin",
		IsActive:       true,
		Status:         "active",
		AssignedAt:     time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	err := companyAdminDB.Create(companyAdmin)
	require.NoError(t, err)

	req, _ := http.NewRequest("DELETE", "/company-admins/delete/"+companyAdmin.ID.Hex(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = []gin.Param{{Key: "adminId", Value: companyAdmin.ID.Hex()}}
	c.Set("user", superAdmin)

	RemoveCompanyAdmin(c)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Company admin removed successfully", response["message"])

	// Verify soft delete
	deleted, err := companyAdminDB.GetByID(companyAdmin.ID)
	assert.NoError(t, err)
	assert.False(t, deleted.IsActive)
}
