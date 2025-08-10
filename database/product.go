package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/models"
)

// ProductDB handles product database operations
type ProductDB struct {
	collection *mongo.Collection
}

// NewProductDB creates a new ProductDB instance
func NewProductDB() *ProductDB {
	collection := DB.Collection("products")
	
	// Create indexes for better performance
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "company_id", Value: 1},
				{Key: "sku", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "company_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "organization_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}
	
	_, err := collection.Indexes().CreateMany(context.TODO(), indexModels)
	if err != nil {
		// Log error but don't fail - indexes might already exist
		// log.Printf("Product indexes creation warning: %v", err)
	}
	
	return &ProductDB{collection: collection}
}

// Create inserts a new product
func (db *ProductDB) Create(product *models.Product) error {
	product.ID = primitive.NewObjectID()
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.IsActive = true
	
	if product.Status == "" {
		product.Status = "active"
	}
	
	result, err := db.collection.InsertOne(context.TODO(), product)
	if err != nil {
		return err
	}
	
	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a product by its ID
func (db *ProductDB) GetByID(id primitive.ObjectID) (*models.Product, error) {
	var product models.Product
	err := db.collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetBySKU retrieves a product by its SKU within a company
func (db *ProductDB) GetBySKU(companyID primitive.ObjectID, sku string) (*models.Product, error) {
	var product models.Product
	err := db.collection.FindOne(context.TODO(), bson.M{
		"company_id": companyID,
		"sku":        sku,
	}).Decode(&product)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// GetByCompany retrieves all products for a specific company
func (db *ProductDB) GetByCompany(companyID primitive.ObjectID, activeOnly bool) ([]models.Product, error) {
	filter := bson.M{"company_id": companyID}
	if activeOnly {
		filter["is_active"] = true
	}
	
	cursor, err := db.collection.Find(context.TODO(), filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var products []models.Product
	if err = cursor.All(context.TODO(), &products); err != nil {
		return nil, err
	}
	
	return products, nil
}

// GetWithCompanyDetails retrieves products with company details using aggregation
func (db *ProductDB) GetWithCompanyDetails(companyID primitive.ObjectID, activeOnly bool) ([]models.ProductResponse, error) {
	matchStage := bson.M{"company_id": companyID}
	if activeOnly {
		matchStage["is_active"] = true
	}
	
	pipeline := []bson.M{
		{
			"$match": matchStage,
		},
		{
			"$lookup": bson.M{
				"from":         "companies",
				"localField":   "company_id",
				"foreignField": "_id",
				"as":           "company",
			},
		},
		{
			"$unwind": "$company",
		},
		{
			"$sort": bson.M{"created_at": -1},
		},
	}
	
	cursor, err := db.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	
	// Transform results to ProductResponse
	var products []models.ProductResponse
	for _, result := range results {
		product := models.ProductResponse{
			ID:             result["_id"].(primitive.ObjectID),
			CompanyID:      result["company_id"].(primitive.ObjectID),
			CompanyName:    result["company"].(bson.M)["name"].(string),
			OrganizationID: result["organization_id"].(primitive.ObjectID),
			Name:           result["name"].(string),
			Description:    getStringValue(result, "description"),
			Category:       result["category"].(string),
			Price:          result["price"].(float64),
			Currency:       result["currency"].(string),
			SKU:            result["sku"].(string),
			Status:         result["status"].(string),
			IsActive:       result["is_active"].(bool),

			CreatedAt:      result["created_at"].(time.Time),
			UpdatedAt:      result["updated_at"].(time.Time),
			CreatedBy:      result["created_by"].(primitive.ObjectID),
			UpdatedBy:      result["updated_by"].(primitive.ObjectID),
		}
		
		// Handle optional arrays
		if tags, ok := result["tags"].(bson.A); ok {
			for _, tag := range tags {
				product.Tags = append(product.Tags, tag.(string))
			}
		}
		
		if images, ok := result["images"].(bson.A); ok {
			for _, image := range images {
				product.Images = append(product.Images, image.(string))
			}
		}
		
		// Handle specifications map
		if specs, ok := result["specifications"].(bson.M); ok {
			product.Specifications = make(map[string]string)
			for key, value := range specs {
				product.Specifications[key] = value.(string)
			}
		}
		

		
		products = append(products, product)
	}
	
	return products, nil
}

// Update updates a product
func (db *ProductDB) Update(id primitive.ObjectID, updates bson.M, updatedBy primitive.ObjectID) error {
	updates["updated_at"] = time.Now()
	updates["updated_by"] = updatedBy
	
	_, err := db.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	return err
}

// SoftDelete performs a soft delete by setting is_active to false
func (db *ProductDB) SoftDelete(id primitive.ObjectID, deletedBy primitive.ObjectID) error {
	updates := bson.M{
		"is_active":  false,
		"status":     "discontinued",
		"updated_at": time.Now(),
		"updated_by": deletedBy,
	}
	
	_, err := db.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	return err
}

// ExistsBySKU checks if a product with the given SKU exists in the company
func (db *ProductDB) ExistsBySKU(companyID primitive.ObjectID, sku string) (bool, error) {
	count, err := db.collection.CountDocuments(context.TODO(), bson.M{
		"company_id": companyID,
		"sku":        sku,
		"is_active":  true,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetStats returns product statistics for a company
func (db *ProductDB) GetStats(companyID primitive.ObjectID) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{"company_id": companyID},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"total_products": bson.M{"$sum": 1},
				"active_products": bson.M{
					"$sum": bson.M{
						"$cond": bson.M{
							"if":   "$is_active",
							"then": 1,
							"else": 0,
						},
					},
				},
				"total_stock": bson.M{"$sum": "$stock"},
				"avg_price":   bson.M{"$avg": "$price"},
				"categories": bson.M{
					"$addToSet": "$category",
				},
			},
		},
	}
	
	cursor, err := db.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return map[string]interface{}{
			"total_products":  0,
			"active_products": 0,
			"total_stock":     0,
			"avg_price":       0,
			"categories":      []string{},
		}, nil
	}
	
	stats := results[0]
	delete(stats, "_id")
	return stats, nil
}

// GetLowStockProducts returns products with low stock for a company
func (db *ProductDB) GetLowStockProducts(companyID primitive.ObjectID) ([]models.Product, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"company_id": companyID,
				"is_active":  true,
			},
		},
		{
			"$match": bson.M{
				"$expr": bson.M{
					"$lte": []interface{}{"$stock", "$min_stock"},
				},
			},
		},
		{
			"$sort": bson.M{"stock": 1},
		},
	}
	
	cursor, err := db.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var products []models.Product
	if err = cursor.All(context.TODO(), &products); err != nil {
		return nil, err
	}
	
	return products, nil
}

// Helper function to safely get string values from bson.M
func getStringValue(data bson.M, key string) string {
	if value, ok := data[key]; ok && value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
