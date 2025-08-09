package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"growth-spark-ai-service/models"
)

// CompanyAdminDB provides database operations for company admins
type CompanyAdminDB struct {
	collection *mongo.Collection
}

// NewCompanyAdminDB creates a new CompanyAdminDB instance
func NewCompanyAdminDB() *CompanyAdminDB {
	collection := DB.Collection("company_admins")
	
	// Create indexes for better performance
	createCompanyAdminIndexes(collection)
	
	return &CompanyAdminDB{
		collection: collection,
	}
}

// createCompanyAdminIndexes creates necessary indexes for the company_admins collection
func createCompanyAdminIndexes(collection *mongo.Collection) {
	ctx := context.TODO()
	
	// Index for company_id and user_id (compound index for uniqueness)
	companyUserIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "company_id", Value: 1},
			{Key: "user_id", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("company_user_unique"),
	}
	
	// Index for organization_id
	orgIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "organization_id", Value: 1}},
		Options: options.Index().SetName("organization_id_index"),
	}
	
	// Index for user_id
	userIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetName("user_id_index"),
	}
	
	// Index for company_id
	companyIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "company_id", Value: 1}},
		Options: options.Index().SetName("company_id_index"),
	}
	
	// Index for is_active
	activeIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "is_active", Value: 1}},
		Options: options.Index().SetName("is_active_index"),
	}
	
	indexes := []mongo.IndexModel{companyUserIndex, orgIndex, userIndex, companyIndex, activeIndex}
	
	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		log.Printf("Failed to create company admin indexes: %v", err)
	} else {
		log.Println("Company admin indexes created successfully")
	}
}

// Create inserts a new company admin record
func (db *CompanyAdminDB) Create(companyAdmin *models.CompanyAdmin) error {
	companyAdmin.ID = primitive.NewObjectID()
	companyAdmin.CreatedAt = time.Now()
	companyAdmin.UpdatedAt = time.Now()
	companyAdmin.IsActive = true
	
	result, err := db.collection.InsertOne(context.TODO(), companyAdmin)
	if err != nil {
		return err
	}
	
	companyAdmin.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a company admin by ID
func (db *CompanyAdminDB) GetByID(id primitive.ObjectID) (*models.CompanyAdmin, error) {
	var companyAdmin models.CompanyAdmin
	err := db.collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&companyAdmin)
	if err != nil {
		return nil, err
	}
	return &companyAdmin, nil
}

// GetByCompanyAndUser retrieves a company admin by company ID and user ID
func (db *CompanyAdminDB) GetByCompanyAndUser(companyID, userID primitive.ObjectID) (*models.CompanyAdmin, error) {
	var companyAdmin models.CompanyAdmin
	err := db.collection.FindOne(context.TODO(), bson.M{
		"company_id": companyID,
		"user_id":    userID,
		"is_active":  true,
	}).Decode(&companyAdmin)
	if err != nil {
		return nil, err
	}
	return &companyAdmin, nil
}

// GetByCompany retrieves all active company admins for a specific company
func (db *CompanyAdminDB) GetByCompany(companyID primitive.ObjectID) ([]models.CompanyAdmin, error) {
	cursor, err := db.collection.Find(context.TODO(), bson.M{
		"company_id": companyID,
		"is_active":  true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var companyAdmins []models.CompanyAdmin
	if err = cursor.All(context.TODO(), &companyAdmins); err != nil {
		return nil, err
	}
	
	return companyAdmins, nil
}

// GetByUser retrieves all active company admin assignments for a specific user
func (db *CompanyAdminDB) GetByUser(userID primitive.ObjectID) ([]models.CompanyAdmin, error) {
	cursor, err := db.collection.Find(context.TODO(), bson.M{
		"user_id":   userID,
		"is_active": true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var companyAdmins []models.CompanyAdmin
	if err = cursor.All(context.TODO(), &companyAdmins); err != nil {
		return nil, err
	}
	
	return companyAdmins, nil
}

// GetByOrganization retrieves all company admins for an organization
func (db *CompanyAdminDB) GetByOrganization(organizationID primitive.ObjectID) ([]models.CompanyAdmin, error) {
	cursor, err := db.collection.Find(context.TODO(), bson.M{
		"organization_id": organizationID,
		"is_active":       true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var companyAdmins []models.CompanyAdmin
	if err = cursor.All(context.TODO(), &companyAdmins); err != nil {
		return nil, err
	}
	
	return companyAdmins, nil
}

// GetWithUserDetails retrieves company admins with user details using aggregation
func (db *CompanyAdminDB) GetWithUserDetails(companyID primitive.ObjectID) ([]models.CompanyAdminResponse, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"company_id": companyID,
				"is_active":  true,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		{
			"$unwind": "$user",
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
			"$project": bson.M{
				"_id":          1,
				"company_id":   1,
				"user_id":      1,
				"role":         1,
				"permissions":  1,
				"is_active":    1,
				"created_at":   1,
				"updated_at":   1,
				"company_name": "$company.name",
				"username":     "$user.username",
				"user_email":   "$user.email",
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
	
	// Transform results to CompanyAdminResponse
	var companyAdmins []models.CompanyAdminResponse
	for _, result := range results {
		companyAdmin := models.CompanyAdminResponse{
			ID:          result["_id"].(primitive.ObjectID),
			CompanyID:   result["company_id"].(primitive.ObjectID),
			CompanyName: result["company_name"].(string),
			UserID:      result["user_id"].(primitive.ObjectID),
			Username:    result["username"].(string),
			UserEmail:   result["user_email"].(string),
			Role:        result["role"].(string),
			IsActive:    result["is_active"].(bool),
			CreatedAt:   result["created_at"].(time.Time),
			UpdatedAt:   result["updated_at"].(time.Time),
		}
		
		// Handle permissions array (might be nil)
		if permissions, ok := result["permissions"].(bson.A); ok {
			for _, perm := range permissions {
				companyAdmin.Permissions = append(companyAdmin.Permissions, perm.(string))
			}
		}
		
		companyAdmins = append(companyAdmins, companyAdmin)
	}
	
	return companyAdmins, nil
}

// Update updates a company admin record
func (db *CompanyAdminDB) Update(id primitive.ObjectID, updates bson.M) error {
	updates["updated_at"] = time.Now()
	
	_, err := db.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	return err
}

// UpdateByCompanyAndUser updates a company admin record by company and user IDs
func (db *CompanyAdminDB) UpdateByCompanyAndUser(companyID, userID primitive.ObjectID, updates bson.M) error {
	updates["updated_at"] = time.Now()
	
	_, err := db.collection.UpdateOne(
		context.TODO(),
		bson.M{
			"company_id": companyID,
			"user_id":    userID,
		},
		bson.M{"$set": updates},
	)
	return err
}

// SoftDelete performs a soft delete by setting is_active to false
func (db *CompanyAdminDB) SoftDelete(id primitive.ObjectID, deletedBy primitive.ObjectID) error {
	updates := bson.M{
		"is_active":  false,
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

// HardDelete permanently deletes a company admin record
func (db *CompanyAdminDB) HardDelete(id primitive.ObjectID) error {
	_, err := db.collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}

// ExistsByCompanyAndUser checks if a company admin assignment exists
func (db *CompanyAdminDB) ExistsByCompanyAndUser(companyID, userID primitive.ObjectID) (bool, error) {
	count, err := db.collection.CountDocuments(context.TODO(), bson.M{
		"company_id": companyID,
		"user_id":    userID,
		"is_active":  true,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetStats returns statistics about company admins
func (db *CompanyAdminDB) GetStats(organizationID primitive.ObjectID) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"organization_id": organizationID,
				"is_active":       true,
			},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"total_admins": bson.M{"$sum": 1},
				"companies_with_admins": bson.M{
					"$addToSet": "$company_id",
				},
				"users_as_admins": bson.M{
					"$addToSet": "$user_id",
				},
			},
		},
		{
			"$project": bson.M{
				"total_admins":           1,
				"total_companies":        bson.M{"$size": "$companies_with_admins"},
				"total_users_as_admins":  bson.M{"$size": "$users_as_admins"},
			},
		},
	}
	
	cursor, err := db.collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())
	
	var result bson.M
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
	}
	
	// Convert bson.M to map[string]interface{}
	stats := make(map[string]interface{})
	for k, v := range result {
		if k != "_id" {
			stats[k] = v
		}
	}
	
	return stats, nil
}
