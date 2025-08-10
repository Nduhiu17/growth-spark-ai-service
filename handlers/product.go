package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"growth-spark-ai-service/database"
	"growth-spark-ai-service/models"
)

// CreateProduct creates a new product for a company (System Manager only)
func CreateProduct(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can create products"})
		return
	}

	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PRODUCT] Create product request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate company ID
	companyID, err := primitive.ObjectIDFromHex(req.CompanyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// If user is system manager (not super admin), verify they are manager of the target company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only create products for companies you manage"})
			return
		}
	}

	// Verify the company exists
	companyCollection := database.DB.Collection("companies")
	var company models.Company
	err = companyCollection.FindOne(context.TODO(), bson.M{"_id": companyID}).Decode(&company)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		log.Printf("[PRODUCT] Error finding company: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify company"})
		return
	}

	// Check if product with this SKU already exists in the company
	productDB := database.NewProductDB()
	exists, err = productDB.ExistsBySKU(companyID, req.SKU)
	if err != nil {
		log.Printf("[PRODUCT] Error checking SKU uniqueness: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check SKU uniqueness"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Product with this SKU already exists in the company"})
		return
	}

	// Validate product fields
	product := &models.Product{
		CompanyID:      companyID,
		OrganizationID: company.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Price:          req.Price,
		Currency:       req.Currency,
		SKU:            req.SKU,
		Status:         req.Status,

		Tags:           req.Tags,
		Images:         req.Images,
		Specifications: req.Specifications,
		CreatedBy:      user.ID,
		UpdatedBy:      user.ID,
	}

	// Set default status if not provided
	if product.Status == "" {
		product.Status = "active"
	}

	// Validate product fields
	if !product.IsValidStatus() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product status", "valid_statuses": models.ValidProductStatuses})
		return
	}

	if !product.IsValidCategory() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product category", "valid_categories": models.ValidProductCategories})
		return
	}

	if !product.IsValidCurrency() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency", "valid_currencies": models.ValidCurrencies})
		return
	}



	// Create the product
	err = productDB.Create(product)
	if err != nil {
		log.Printf("[PRODUCT] Error creating product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	log.Printf("[PRODUCT] Product created successfully: %s for company %s by user %s", product.ID.Hex(), companyID.Hex(), user.ID.Hex())

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Product created successfully",
		"product_id": product.ID,
		"company_id": companyID,
		"sku":        product.SKU,
		"name":       product.Name,
		"status":     product.Status,
	})
}

// GetProducts retrieves all products for a specific company (System Manager only)
func GetProducts(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can view products"})
		return
	}

	// Get company ID from URL parameter
	companyIDParam := c.Param("companyId")
	companyID, err := primitive.ObjectIDFromHex(companyIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// If user is system manager or company admin (not super admin), verify they are authorized for the target company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view products for companies you manage"})
			return
		}
	}

	// Get query parameters
	activeOnly := c.DefaultQuery("active_only", "false") == "true"

	// Get products with company details
	productDB := database.NewProductDB()
	products, err := productDB.GetWithCompanyDetails(companyID, activeOnly)
	if err != nil {
		log.Printf("[PRODUCT] Error retrieving products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"count":    len(products),
	})
}

// GetProduct retrieves a specific product by ID (System Manager only)
func GetProduct(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can view products"})
		return
	}

	// Get product ID from URL parameter
	productIDParam := c.Param("productId")
	productID, err := primitive.ObjectIDFromHex(productIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Get the product
	productDB := database.NewProductDB()
	product, err := productDB.GetByID(productID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		log.Printf("[PRODUCT] Error retrieving product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		return
	}

	// If user is system manager or company admin (not super admin), verify they are authorized for the product's company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(product.CompanyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view products for companies you manage"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"product": product,
	})
}

// UpdateProduct updates a product (System Manager only)
func UpdateProduct(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can update products"})
		return
	}

	// Get product ID from URL parameter
	productIDParam := c.Param("productId")
	productID, err := primitive.ObjectIDFromHex(productIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req models.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[PRODUCT] Update product request validation failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get the existing product to verify authorization
	productDB := database.NewProductDB()
	existingProduct, err := productDB.GetByID(productID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		log.Printf("[PRODUCT] Error retrieving product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		return
	}

	// If user is system manager or company admin (not super admin), verify they are authorized for the product's company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(existingProduct.CompanyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only update products for companies you manage"})
			return
		}
	}

	// Build update document
	updates := bson.M{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		// Validate category
		tempProduct := &models.Product{Category: req.Category}
		if !tempProduct.IsValidCategory() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product category", "valid_categories": models.ValidProductCategories})
			return
		}
		updates["category"] = req.Category
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}
	if req.Currency != "" {
		// Validate currency
		tempProduct := &models.Product{Currency: req.Currency}
		if !tempProduct.IsValidCurrency() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid currency", "valid_currencies": models.ValidCurrencies})
			return
		}
		updates["currency"] = req.Currency
	}
	if req.SKU != "" {
		// Check if new SKU conflicts with existing products (excluding current product)
		exists, err := productDB.ExistsBySKU(existingProduct.CompanyID, req.SKU)
		if err != nil {
			log.Printf("[PRODUCT] Error checking SKU uniqueness: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check SKU uniqueness"})
			return
		}
		if exists && req.SKU != existingProduct.SKU {
			c.JSON(http.StatusConflict, gin.H{"error": "Product with this SKU already exists in the company"})
			return
		}
		updates["sku"] = req.SKU
	}
	if req.Status != "" {
		// Validate status
		tempProduct := &models.Product{Status: req.Status}
		if !tempProduct.IsValidStatus() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product status", "valid_statuses": models.ValidProductStatuses})
			return
		}
		updates["status"] = req.Status
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if req.Tags != nil {
		updates["tags"] = req.Tags
	}
	if req.Images != nil {
		updates["images"] = req.Images
	}
	if req.Specifications != nil {
		updates["specifications"] = req.Specifications
	}



	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	// Update the product
	err = productDB.Update(productID, updates, user.ID)
	if err != nil {
		log.Printf("[PRODUCT] Error updating product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	log.Printf("[PRODUCT] Product updated successfully: %s by user %s", productID.Hex(), user.ID.Hex())

	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
	})
}

// DeleteProduct soft deletes a product (System Manager only)
func DeleteProduct(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can delete products"})
		return
	}

	// Get product ID from URL parameter
	productIDParam := c.Param("productId")
	productID, err := primitive.ObjectIDFromHex(productIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Get the existing product to verify authorization
	productDB := database.NewProductDB()
	existingProduct, err := productDB.GetByID(productID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		log.Printf("[PRODUCT] Error retrieving product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product"})
		return
	}

	// If user is system manager or company admin (not super admin), verify they are authorized for the product's company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(existingProduct.CompanyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete products for companies you manage"})
			return
		}
	}

	// Soft delete the product
	err = productDB.SoftDelete(productID, user.ID)
	if err != nil {
		log.Printf("[PRODUCT] Error deleting product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	log.Printf("[PRODUCT] Product deleted successfully: %s by user %s", productID.Hex(), user.ID.Hex())

	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}

// GetProductStats retrieves product statistics for a company (System Manager only)
func GetProductStats(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userInterface.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	// Check if user is system manager, company admin, or super admin
	if user.Role != "manager" && user.Role != "admin" && user.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only system managers can view product statistics"})
		return
	}

	// Get company ID from URL parameter
	companyIDParam := c.Param("companyId")
	companyID, err := primitive.ObjectIDFromHex(companyIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid company ID"})
		return
	}

	// If user is system manager or company admin (not super admin), verify they are authorized for the target company
	if user.Role == "manager" || user.Role == "admin" {
		companyAdminDB := database.NewCompanyAdminDB()
		isAuthorized, err := companyAdminDB.ExistsByCompanyAndUser(companyID, user.ID)
		if err != nil {
			log.Printf("[PRODUCT] Error checking authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify authorization"})
			return
		}
		if !isAuthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only view statistics for companies you manage"})
			return
		}
	}

	// Get product statistics
	productDB := database.NewProductDB()
	stats, err := productDB.GetStats(companyID)
	if err != nil {
		log.Printf("[PRODUCT] Error retrieving product stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve product statistics"})
		return
	}

	// Get low stock products
	lowStockProducts, err := productDB.GetLowStockProducts(companyID)
	if err != nil {
		log.Printf("[PRODUCT] Error retrieving low stock products: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve low stock products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":              stats,
		"low_stock_products": lowStockProducts,
		"low_stock_count":    len(lowStockProducts),
	})
}
