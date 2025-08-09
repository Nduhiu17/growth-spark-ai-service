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
	"golang.org/x/crypto/bcrypt"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

func setupAuthTestDB(t *testing.T) {
	// Connect to test database
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	
	database.DB = client.Database("growth_spark_test_auth_handlers")
	
	// Clean up function
	t.Cleanup(func() {
		database.DB.Drop(context.TODO())
		client.Disconnect(context.TODO())
	})
}

func TestLogin_InvalidRequestFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", Login)

	// Test with invalid JSON
	req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

func TestLogin_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", Login)

	// Test with missing email
	loginReq := LoginRequest{
		Password: "password123",
	}
	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

func TestLogin_UserNotFound(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", Login)

	loginReq := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
}

func TestLogin_InvalidPassword(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", Login)

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	testUser := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Username:       "testuser",
		FullName:       "Test User",
		Email:          "test@example.com",
		Password:       string(hashedPassword),
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	collection := database.DB.Collection("users")
	_, err := collection.InsertOne(context.TODO(), testUser)
	require.NoError(t, err)

	// Try to login with wrong password
	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
}

func TestLogin_Success(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", Login)

	// Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	testUser := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Username:       "testuser",
		FullName:       "Test User",
		Email:          "test@example.com",
		Password:       string(hashedPassword),
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	collection := database.DB.Collection("users")
	_, err := collection.InsertOne(context.TODO(), testUser)
	require.NoError(t, err)

	// Login with correct credentials
	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "correctpassword",
	}
	reqBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response LoginResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, testUser.Email, response.User.Email)
	assert.Equal(t, testUser.Username, response.User.Username)
}

func TestRegister_InvalidRequestFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", Register)

	// Test with invalid JSON
	req, _ := http.NewRequest("POST", "/register", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

func TestRegister_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", Register)

	// Test with missing required fields
	registerReq := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		// Missing other required fields
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", Register)

	// Create an existing user
	existingUser := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Email:          "existing@example.com",
		Username:       "existinguser",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	collection := database.DB.Collection("users")
	_, err := collection.InsertOne(context.TODO(), existingUser)
	require.NoError(t, err)

	// Try to register with the same email
	registerReq := RegisterRequest{
		OrganizationName: "Test Org",
		Username:         "newuser",
		FullName:         "New User",
		Email:            "existing@example.com",
		PhoneNumber:      "+1234567890",
		Password:         "password123",
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "User already exists")
}

func TestRegister_Success(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", Register)

	registerReq := RegisterRequest{
		OrganizationName: "Test Organization",
		Username:         "testuser",
		FullName:         "Test User",
		Email:            "test@example.com",
		PhoneNumber:      "+1234567890",
		Password:         "password123",
	}
	reqBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, registerReq.Email, response.User.Email)
	assert.Equal(t, registerReq.Username, response.User.Username)
	assert.Equal(t, "super_admin", response.User.Role) // First user becomes super_admin
}

func TestGetProfile_MissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/profile", GetProfile)

	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "User not authenticated")
}

func TestGetProfile_Success(t *testing.T) {
	setupAuthTestDB(t)
	
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Create a test user in the database
	testUser := models.User{
		ID:             primitive.NewObjectID(),
		OrganizationID: primitive.NewObjectID(),
		Username:       "testuser",
		FullName:       "Test User",
		Email:          "test@example.com",
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	collection := database.DB.Collection("users")
	_, err := collection.InsertOne(context.TODO(), testUser)
	require.NoError(t, err)
	
	// Mock authentication middleware that sets userID
	router.Use(func(c *gin.Context) {
		c.Set("userID", testUser.ID.Hex())
		c.Next()
	})
	
	router.GET("/profile", GetProfile)

	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	userData := response["user"].(map[string]interface{})
	assert.Equal(t, "testuser", userData["username"])
	assert.Equal(t, "test@example.com", userData["email"])
	assert.Empty(t, userData["password"]) // Password should be removed from response
}
