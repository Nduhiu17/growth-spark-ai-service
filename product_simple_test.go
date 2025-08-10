package main

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
	"growth-spark-ai-service/handlers"
	"growth-spark-ai-service/models"
)

func TestProductCreationSimple(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_product_simple_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("System Manager Creates Product Successfully", func(t *testing.T) {
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

		// Create system manager user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			FullName:       "System Manager",
			Email:          "manager@company.com",
			Password:       string(hashedPassword),
			Role:           "manager",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
		assert.NoError(t, err)

		// Create system manager assignment
		systemManager := models.CompanyAdmin{
			ID:             primitive.NewObjectID(),
			CompanyID:      company.ID,
			UserID:         managerUser.ID,
			OrganizationID: orgID,
			Role:           "manager",
			Permissions:    []string{"users:read", "users:create", "roles:read"},
			IsActive:       true,
			Status:         "active",
			AssignedAt:     time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			CreatedBy:      managerUser.ID,
			UpdatedBy:      managerUser.ID,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), systemManager)
		assert.NoError(t, err)

		// Setup router with middleware
		router.POST("/products/create", func(c *gin.Context) {
			c.Set("user", managerUser)
			handlers.CreateProduct(c)
		})

		// Create request payload
		requestBody := models.CreateProductRequest{
			CompanyID:   company.ID.Hex(),
			Name:        "Test Product",
			Description: "A test product for system manager",
			Category:    "electronics",
			Price:       99.99,
			Currency:    "USD",
			SKU:         "TEST-001",
			Status:      "active",

			Tags:        []string{"test", "electronics"},
			Images:      []string{"image1.jpg", "image2.jpg"},
			Specifications: map[string]string{
				"color":  "black",
				"weight": "1kg",
			},
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Product created successfully", response["message"])
		assert.NotNil(t, response["product_id"])
		assert.Equal(t, "TEST-001", response["sku"])
		assert.Equal(t, "Test Product", response["name"])
		assert.Equal(t, "active", response["status"])

		// Verify product was created in database
		var createdProduct models.Product
		err = database.DB.Collection("products").FindOne(context.TODO(), bson.M{
			"company_id": company.ID,
			"sku":        "TEST-001",
		}).Decode(&createdProduct)
		assert.NoError(t, err)
		assert.Equal(t, "Test Product", createdProduct.Name)
		assert.Equal(t, "electronics", createdProduct.Category)
		assert.Equal(t, 99.99, createdProduct.Price)
		assert.Equal(t, "USD", createdProduct.Currency)

		assert.True(t, createdProduct.IsActive)
	})

	t.Run("Unauthorized User Cannot Create Product", func(t *testing.T) {
		// Create regular user
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.POST("/products/create-unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			handlers.CreateProduct(c)
		})

		requestBody := models.CreateProductRequest{
			CompanyID: primitive.NewObjectID().Hex(),
			Name:      "Unauthorized Product",
			Category:  "electronics",
			Price:     99.99,
			Currency:  "USD",
			SKU:       "UNAUTH-001",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-unauthorized", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only system managers can create products", response["error"])
	})
}
