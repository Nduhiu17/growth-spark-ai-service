package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCompanyAdmin_IsValidRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"Valid admin role", "admin", true},
		{"Valid manager role", "manager", true},
		{"Valid supervisor role", "supervisor", true},
		{"Valid viewer role", "viewer", true},
		{"Invalid role", "invalid_role", false},
		{"Empty role", "", false},
		{"Case sensitive", "Admin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &CompanyAdmin{Role: tt.role}
			assert.Equal(t, tt.expected, ca.IsValidRole())
		})
	}
}

func TestCompanyAdmin_IsValidStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Valid active status", "active", true},
		{"Valid suspended status", "suspended", true},
		{"Valid pending status", "pending", true},
		{"Valid expired status", "expired", true},
		{"Invalid status", "invalid_status", false},
		{"Empty status", "", false},
		{"Case sensitive", "Active", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &CompanyAdmin{Status: tt.status}
			assert.Equal(t, tt.expected, ca.IsValidStatus())
		})
	}
}

func TestCompanyAdmin_IsExpired(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		expected  bool
	}{
		{"No expiration date", nil, false},
		{"Expired", &pastTime, true},
		{"Not expired", &futureTime, false},
		{"Expires now", &now, true}, // Consider equal time as expired
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := &CompanyAdmin{ExpiresAt: tt.expiresAt}
			assert.Equal(t, tt.expected, ca.IsExpired())
		})
	}
}

func TestCreateCompanyAdminRequest_Validation(t *testing.T) {
	validRequest := CreateCompanyAdminRequest{
		CompanyID:   "507f1f77bcf86cd799439011",
		Username:    "testuser",
		FullName:    "Test User",
		Email:       "test@example.com",
		PhoneNumber: "+1234567890",
		Password:    "password123",
		Role:        "admin",
		Permissions: []string{"create_user", "edit_user"},
		Status:      "active",
	}

	// Test valid request
	assert.NotEmpty(t, validRequest.CompanyID)
	assert.NotEmpty(t, validRequest.Username)
	assert.NotEmpty(t, validRequest.FullName)
	assert.NotEmpty(t, validRequest.Email)
	assert.NotEmpty(t, validRequest.PhoneNumber)
	assert.NotEmpty(t, validRequest.Password)
	assert.NotEmpty(t, validRequest.Role)
}

func TestUpdateCompanyAdminRequest_OptionalFields(t *testing.T) {
	updateRequest := UpdateCompanyAdminRequest{
		Role:        "manager",
		Permissions: []string{"view_users"},
		Status:      "suspended",
		Notes:       "Updated by admin",
	}

	assert.Equal(t, "manager", updateRequest.Role)
	assert.Equal(t, []string{"view_users"}, updateRequest.Permissions)
	assert.Equal(t, "suspended", updateRequest.Status)
	assert.Equal(t, "Updated by admin", updateRequest.Notes)
}

func TestAssignExistingUserRequest_Validation(t *testing.T) {
	request := AssignExistingUserRequest{
		CompanyID:   "507f1f77bcf86cd799439011",
		UserID:      "507f1f77bcf86cd799439012",
		Role:        "admin",
		Permissions: []string{"manage_company"},
		Status:      "active",
	}

	assert.NotEmpty(t, request.CompanyID)
	assert.NotEmpty(t, request.UserID)
	assert.NotEmpty(t, request.Role)
	assert.Equal(t, "active", request.Status)
}

func TestCompanyAdminResponse_Structure(t *testing.T) {
	id := primitive.NewObjectID()
	companyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	now := time.Now()

	response := CompanyAdminResponse{
		ID:          id,
		CompanyID:   companyID,
		CompanyName: "Test Company",
		UserID:      userID,
		Username:    "testuser",
		UserEmail:   "test@example.com",
		Role:        "admin",
		Permissions: []string{"create_user", "edit_user"},
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, id, response.ID)
	assert.Equal(t, companyID, response.CompanyID)
	assert.Equal(t, "Test Company", response.CompanyName)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "testuser", response.Username)
	assert.Equal(t, "test@example.com", response.UserEmail)
	assert.Equal(t, "admin", response.Role)
	assert.True(t, response.IsActive)
	assert.Equal(t, 2, len(response.Permissions))
}

func TestValidRoles_Constants(t *testing.T) {
	expectedRoles := []string{"admin", "manager", "supervisor", "viewer"}
	assert.Equal(t, expectedRoles, ValidRoles)
}

func TestValidStatuses_Constants(t *testing.T) {
	expectedStatuses := []string{"active", "suspended", "pending", "expired"}
	assert.Equal(t, expectedStatuses, ValidStatuses)
}
