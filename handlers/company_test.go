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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

func setupCompanyTestDB(t *testing.T) {
	// Initialize test database
	err := database.Connect()
	require.NoError(t, err)
	
	// Clean up test data
	ctx := context.TODO()
	database.DB.Collection("companies").DeleteMany(ctx, bson.M{})
	database.DB.Collection("users").DeleteMany(ctx, bson.M{})
}

func createTestSuperAdmin(t *testing.T) models.User {
	orgID := primitive.NewObjectID()
	user := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: orgID,
		Username:       "superadmin",
		FullName:       "Super Admin",
		Email:          "superadmin@test.com",
		Password:       "hashedpassword",
		Role:           "super_admin",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Insert user into database
	_, err := database.DB.Collection("users").InsertOne(context.TODO(), user)
	require.NoError(t, err)
	
	return user
}

func createTestRegularUser(t *testing.T) models.User {
	orgID := primitive.NewObjectID()
	user := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: orgID,
		Username:       "regularuser",
		FullName:       "Regular User",
		Email:          "regular@test.com",
		Password:       "hashedpassword",
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Insert user into database
	_, err := database.DB.Collection("users").InsertOne(context.TODO(), user)
	require.NoError(t, err)
	
	return user
}

func TestCreateCompany_Success(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create request
	req := models.CreateCompanyRequest{
		Name: "Test Company",
	}
	reqBody, _ := json.Marshal(req)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/companies", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	
	// Call handler
	CreateCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Company created successfully", response["message"])
	assert.NotNil(t, response["company"])
	
	company := response["company"].(map[string]interface{})
	assert.Equal(t, "Test Company", company["name"])
	assert.Equal(t, superAdmin.OrganizationID.Hex(), company["organization_id"])
}

func TestCreateCompany_NoUserInContext(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create request
	req := models.CreateCompanyRequest{
		Name: "Test Company",
	}
	reqBody, _ := json.Marshal(req)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/companies", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context without user
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	
	// Call handler
	CreateCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "User not found in context", response["error"])
}

func TestCreateCompany_NotSuperAdmin(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test regular user
	regularUser := createTestRegularUser(t)
	
	// Create request
	req := models.CreateCompanyRequest{
		Name: "Test Company",
	}
	reqBody, _ := json.Marshal(req)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/companies", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", regularUser)
	
	// Call handler
	CreateCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Only super admins can create companies", response["error"])
}

func TestCreateCompany_InvalidRequest(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create invalid request (missing name)
	reqBody := []byte(`{"invalid": "data"}`)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/companies", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	
	// Call handler
	CreateCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Invalid request format", response["error"])
}

func TestGetCompany_Success(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create test company
	companyID := primitive.NewObjectID()
	company := models.Company{
		ID:             companyID,
		OrganizationID: superAdmin.OrganizationID,
		Name:           "Test Company",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	
	_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
	require.NoError(t, err)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/companies/"+companyID.Hex(), nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	c.Params = []gin.Param{{Key: "id", Value: companyID.Hex()}}
	
	// Call handler
	GetCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.Company
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Test Company", response.Name)
	assert.Equal(t, companyID, response.ID)
}

func TestGetCompany_NotFound(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Use non-existent company ID
	companyID := primitive.NewObjectID()
	
	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/companies/"+companyID.Hex(), nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	c.Params = []gin.Param{{Key: "id", Value: companyID.Hex()}}
	
	// Call handler
	GetCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Company not found", response["error"])
}

func TestGetAllCompanies_Success(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create test companies
	companies := []models.Company{
		{
			ID:             primitive.NewObjectID(),
			OrganizationID: superAdmin.OrganizationID,
			Name:           "Company 1",
			Status:         "active",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      superAdmin.ID,
			UpdatedBy:      superAdmin.ID,
		},
		{
			ID:             primitive.NewObjectID(),
			OrganizationID: superAdmin.OrganizationID,
			Name:           "Company 2",
			Status:         "active",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      superAdmin.ID,
			UpdatedBy:      superAdmin.ID,
		},
	}
	
	for _, company := range companies {
		_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
		require.NoError(t, err)
	}
	
	// Create HTTP request
	httpReq := httptest.NewRequest("GET", "/companies", nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	
	// Call handler
	GetAllCompanies(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response []models.Company
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Len(t, response, 2)
	assert.Equal(t, "Company 1", response[0].Name)
	assert.Equal(t, "Company 2", response[1].Name)
}

func TestUpdateCompany_Success(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create test company
	companyID := primitive.NewObjectID()
	company := models.Company{
		ID:             companyID,
		OrganizationID: superAdmin.OrganizationID,
		Name:           "Original Company",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	
	_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
	require.NoError(t, err)
	
	// Create update request
	updateReq := models.UpdateCompanyRequest{
		Name: "Updated Company",
	}
	reqBody, _ := json.Marshal(updateReq)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("PUT", "/companies/"+companyID.Hex(), bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	c.Params = []gin.Param{{Key: "id", Value: companyID.Hex()}}
	
	// Call handler
	UpdateCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Company updated successfully", response["message"])
}

func TestDeleteCompany_Success(t *testing.T) {
	setupCompanyTestDB(t)
	gin.SetMode(gin.TestMode)
	
	// Create test super admin
	superAdmin := createTestSuperAdmin(t)
	
	// Create test company
	companyID := primitive.NewObjectID()
	company := models.Company{
		ID:             companyID,
		OrganizationID: superAdmin.OrganizationID,
		Name:           "Test Company",
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      superAdmin.ID,
		UpdatedBy:      superAdmin.ID,
	}
	
	_, err := database.DB.Collection("companies").InsertOne(context.TODO(), company)
	require.NoError(t, err)
	
	// Create HTTP request
	httpReq := httptest.NewRequest("DELETE", "/companies/"+companyID.Hex(), nil)
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Create Gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = httpReq
	c.Set("user", superAdmin)
	c.Params = []gin.Param{{Key: "id", Value: companyID.Hex()}}
	
	// Call handler
	DeleteCompany(c)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "Company deleted successfully", response["message"])
}
