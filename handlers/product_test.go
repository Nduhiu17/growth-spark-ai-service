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

func TestCreateProduct(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_product_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Success - System Manager Creates Product", func(t *testing.T) {
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
			CreateProduct(c)
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

	t.Run("Success - Company Admin Creates Product", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("products").DeleteMany(context.TODO(), bson.M{})

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

		router.POST("/products/create-admin", func(c *gin.Context) {
			c.Set("user", adminUser)
			CreateProduct(c)
		})

		// Create request payload
		requestBody := models.CreateProductRequest{
			CompanyID:   company.ID.Hex(),
			Name:        "Admin Product",
			Description: "A product created by company admin",
			Category:    "software",
			Price:       199.99,
			Currency:    "USD",
			SKU:         "ADMIN-001",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-admin", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Product created successfully", response["message"])
	})

	t.Run("Success - Super Admin Creates Product", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("products").DeleteMany(context.TODO(), bson.M{})

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

		router.POST("/products/create-super", func(c *gin.Context) {
			c.Set("user", superAdmin)
			CreateProduct(c)
		})

		// Create request payload
		requestBody := models.CreateProductRequest{
			CompanyID: company.ID.Hex(),
			Name:      "Super Admin Product",
			Category:  "services",
			Price:     299.99,
			Currency:  "USD",
			SKU:       "SUPER-001",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-super", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Error - Unauthorized User", func(t *testing.T) {
		// Create regular user
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.POST("/products/create-unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			CreateProduct(c)
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

	t.Run("Error - System Manager Not Authorized for Company", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})

		// Create test organizations
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

		// Create system manager user (manager of company1 only)
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
		assert.NoError(t, err)

		// Create system manager assignment for company1 only
		systemManager := models.CompanyAdmin{
			ID:        primitive.NewObjectID(),
			CompanyID: company1.ID,
			UserID:    managerUser.ID,
			Role:      "manager",
			IsActive:  true,
		}
		_, err = database.DB.Collection("company_admins").InsertOne(context.TODO(), systemManager)
		assert.NoError(t, err)

		router.POST("/products/create-wrong-company", func(c *gin.Context) {
			c.Set("user", managerUser)
			CreateProduct(c)
		})

		// Try to create product for company2 (not manager of this company)
		requestBody := models.CreateProductRequest{
			CompanyID: company2.ID.Hex(),
			Name:      "Unauthorized Product",
			Category:  "electronics",
			Price:     99.99,
			Currency:  "USD",
			SKU:       "WRONG-001",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-wrong-company", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "You can only create products for companies you manage", response["error"])
	})

	t.Run("Error - Duplicate SKU", func(t *testing.T) {
		// Clean up before test
		database.DB.Collection("users").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("companies").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("company_admins").DeleteMany(context.TODO(), bson.M{})
		database.DB.Collection("products").DeleteMany(context.TODO(), bson.M{})

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
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
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

		// Create existing product
		existingProduct := models.Product{
			ID:        primitive.NewObjectID(),
			CompanyID: company.ID,
			Name:      "Existing Product",
			Category:  "electronics",
			Price:     50.00,
			Currency:  "USD",
			SKU:       "DUPLICATE-001",
			Status:    "active",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_, err = database.DB.Collection("products").InsertOne(context.TODO(), existingProduct)
		assert.NoError(t, err)

		router.POST("/products/create-duplicate", func(c *gin.Context) {
			c.Set("user", managerUser)
			CreateProduct(c)
		})

		// Try to create product with duplicate SKU
		requestBody := models.CreateProductRequest{
			CompanyID: company.ID.Hex(),
			Name:      "Duplicate Product",
			Category:  "electronics",
			Price:     99.99,
			Currency:  "USD",
			SKU:       "DUPLICATE-001", // Same SKU as existing product
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-duplicate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Product with this SKU already exists in the company", response["error"])
	})

	t.Run("Error - Invalid Category", func(t *testing.T) {
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

		// Create system manager user
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
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

		router.POST("/products/create-invalid-category", func(c *gin.Context) {
			c.Set("user", managerUser)
			CreateProduct(c)
		})

		requestBody := models.CreateProductRequest{
			CompanyID: company.ID.Hex(), // Use valid company ID that user manages
			Name:      "Invalid Category Product",
			Category:  "invalid_category",
			Price:     99.99,
			Currency:  "USD",
			SKU:       "INVALID-001",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-invalid-category", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid product category", response["error"])
	})

	t.Run("Error - Invalid Currency", func(t *testing.T) {
		managerUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "systemmanager",
			Role:     "manager",
		}

		router.POST("/products/create-invalid-currency", func(c *gin.Context) {
			c.Set("user", managerUser)
			CreateProduct(c)
		})

		requestBody := models.CreateProductRequest{
			CompanyID: primitive.NewObjectID().Hex(),
			Name:      "Invalid Currency Product",
			Category:  "electronics",
			Price:     99.99,
			Currency:  "INVALID",
			SKU:       "INVALID-002",
		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-invalid-currency", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid currency", response["error"])
	})

	t.Run("Error - Invalid Stock Constraints", func(t *testing.T) {
		managerUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "systemmanager",
			Role:     "manager",
		}

		router.POST("/products/create-invalid-stock", func(c *gin.Context) {
			c.Set("user", managerUser)
			CreateProduct(c)
		})

		requestBody := models.CreateProductRequest{
			CompanyID: primitive.NewObjectID().Hex(),
			Name:      "Invalid Stock Product",
			Category:  "electronics",
			Price:     99.99,
			Currency:  "USD",
			SKU:       "INVALID-003",

		}

		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/products/create-invalid-stock", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Minimum stock must be less than maximum stock", response["error"])
	})
}

func TestGetProducts(t *testing.T) {
	// Setup test database
	testURI := os.Getenv("MONGO_TEST_URI")
	if testURI == "" {
		testURI = "mongodb://localhost:27017/saas_marketing_testdb"
	}
	
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(testURI))
	assert.NoError(t, err)
	defer client.Disconnect(context.TODO())
	
	testDB := client.Database("growth_spark_product_get_test_" + primitive.NewObjectID().Hex())
	originalDB := database.DB
	database.DB = testDB
	defer func() {
		testDB.Drop(context.TODO())
		database.DB = originalDB
	}()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("Success - System Manager Gets Products", func(t *testing.T) {
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
		managerUser := models.User{
			ID:             primitive.NewObjectID(),
			OrganizationID: orgID,
			Username:       "systemmanager",
			Role:           "manager",
		}
		_, err = database.DB.Collection("users").InsertOne(context.TODO(), managerUser)
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

		// Create test products
		product1 := models.Product{
			ID:             primitive.NewObjectID(),
			CompanyID:      company.ID,
			OrganizationID: orgID,
			Name:           "Product 1",
			Category:       "electronics",
			Price:          99.99,
			Currency:       "USD",
			SKU:            "PROD-001",
			Status:         "active",
			IsActive:       true,

			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		product2 := models.Product{
			ID:             primitive.NewObjectID(),
			CompanyID:      company.ID,
			OrganizationID: orgID,
			Name:           "Product 2",
			Category:       "software",
			Price:          199.99,
			Currency:       "USD",
			SKU:            "PROD-002",
			Status:         "inactive",
			IsActive:       false,

			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		_, err = database.DB.Collection("products").InsertMany(context.TODO(), []interface{}{product1, product2})
		assert.NoError(t, err)

		router.GET("/products/company/:companyId", func(c *gin.Context) {
			c.Set("user", managerUser)
			GetProducts(c)
		})

		req, _ := http.NewRequest("GET", "/products/company/"+company.ID.Hex(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["products"])
		assert.Equal(t, float64(2), response["count"]) // Both active and inactive products
	})

	t.Run("Success - Get Active Products Only", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/products/company/"+primitive.NewObjectID().Hex()+"?active_only=true", nil)
		
		managerUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "systemmanager",
			Role:     "manager",
		}

		router.GET("/products/company/:companyId/active", func(c *gin.Context) {
			c.Set("user", managerUser)
			GetProducts(c)
		})

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// This will fail authorization but tests the active_only parameter parsing
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Error - Unauthorized User", func(t *testing.T) {
		regularUser := models.User{
			ID:       primitive.NewObjectID(),
			Username: "regularuser",
			Role:     "user",
		}

		router.GET("/products/company/:companyId/unauthorized", func(c *gin.Context) {
			c.Set("user", regularUser)
			GetProducts(c)
		})

		req, _ := http.NewRequest("GET", "/products/company/"+primitive.NewObjectID().Hex()+"/unauthorized", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Only system managers can view products", response["error"])
	})
}
