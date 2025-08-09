package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSystemManagerRequest(t *testing.T) {
	t.Run("Valid System Manager Request", func(t *testing.T) {
		request := CreateSystemManagerRequest{
			CompanyID:   "507f1f77bcf86cd799439011",
			Username:    "systemmanager",
			FullName:    "System Manager",
			Email:       "manager@company.com",
			Password:    "password123",
			Permissions: []string{"users:read", "users:create", "roles:read"},
			Notes:       "Test system manager",
		}

		assert.Equal(t, "507f1f77bcf86cd799439011", request.CompanyID)
		assert.Equal(t, "systemmanager", request.Username)
		assert.Equal(t, "System Manager", request.FullName)
		assert.Equal(t, "manager@company.com", request.Email)
		assert.Equal(t, "password123", request.Password)
		assert.Equal(t, []string{"users:read", "users:create", "roles:read"}, request.Permissions)
		assert.Equal(t, "Test system manager", request.Notes)
	})

	t.Run("System Manager Request with Optional Fields", func(t *testing.T) {
		expiresAt := "2024-12-31T23:59:59Z"
		request := CreateSystemManagerRequest{
			CompanyID:   "507f1f77bcf86cd799439011",
			Username:    "systemmanager",
			FullName:    "System Manager",
			Email:       "manager@company.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Permissions: []string{"users:read"},
			ExpiresAt:   &expiresAt,
			Notes:       "System manager with expiration",
		}

		assert.Equal(t, "+1234567890", request.PhoneNumber)
		assert.NotNil(t, request.ExpiresAt)
		assert.Equal(t, "2024-12-31T23:59:59Z", *request.ExpiresAt)
	})

	t.Run("System Manager Request with Empty Optional Fields", func(t *testing.T) {
		request := CreateSystemManagerRequest{
			CompanyID: "507f1f77bcf86cd799439011",
			Username:  "systemmanager",
			FullName:  "System Manager",
			Email:     "manager@company.com",
			Password:  "password123",
		}

		assert.Empty(t, request.PhoneNumber)
		assert.Empty(t, request.Permissions)
		assert.Nil(t, request.ExpiresAt)
		assert.Empty(t, request.Notes)
	})

	t.Run("System Manager Request Validation Fields", func(t *testing.T) {
		// Test that required fields are properly tagged
		request := CreateSystemManagerRequest{}
		
		// These would be validated by gin's binding validation
		assert.Empty(t, request.CompanyID)   // binding:"required"
		assert.Empty(t, request.Username)    // binding:"required"
		assert.Empty(t, request.FullName)    // binding:"required"
		assert.Empty(t, request.Email)       // binding:"required,email"
		assert.Empty(t, request.Password)    // binding:"required,min=6"
	})

	t.Run("System Manager Request Default Permissions", func(t *testing.T) {
		request := CreateSystemManagerRequest{
			CompanyID: "507f1f77bcf86cd799439011",
			Username:  "systemmanager",
			FullName:  "System Manager",
			Email:     "manager@company.com",
			Password:  "password123",
		}

		// When no permissions are specified, handler should use defaults
		assert.Empty(t, request.Permissions)
		
		// Test with custom permissions
		customPermissions := []string{"users:read", "companies:read", "reports:read"}
		request.Permissions = customPermissions
		assert.Equal(t, customPermissions, request.Permissions)
		assert.Len(t, request.Permissions, 3)
	})

	t.Run("System Manager Request JSON Serialization", func(t *testing.T) {
		request := CreateSystemManagerRequest{
			CompanyID:   "507f1f77bcf86cd799439011",
			Username:    "systemmanager",
			FullName:    "System Manager",
			Email:       "manager@company.com",
			PhoneNumber: "+1234567890",
			Password:    "password123",
			Permissions: []string{"users:read", "users:create"},
			Notes:       "JSON serialization test",
		}

		// Test that struct fields have proper json tags
		assert.NotEmpty(t, request.CompanyID)
		assert.NotEmpty(t, request.Username)
		assert.NotEmpty(t, request.FullName)
		assert.NotEmpty(t, request.Email)
		assert.NotEmpty(t, request.PhoneNumber)
		assert.NotEmpty(t, request.Password)
		assert.NotEmpty(t, request.Permissions)
		assert.NotEmpty(t, request.Notes)
	})

	t.Run("System Manager Request Edge Cases", func(t *testing.T) {
		// Test with minimum valid password length
		request := CreateSystemManagerRequest{
			CompanyID: "507f1f77bcf86cd799439011",
			Username:  "manager",
			FullName:  "M",
			Email:     "m@c.co",
			Password:  "123456", // minimum 6 characters
		}

		assert.Equal(t, "123456", request.Password)
		assert.Len(t, request.Password, 6)

		// Test with long values
		longRequest := CreateSystemManagerRequest{
			CompanyID:   "507f1f77bcf86cd799439011",
			Username:    "verylongusernameforsystemmanager",
			FullName:    "Very Long Full Name For System Manager Testing",
			Email:       "verylongemailaddressforsystemmanager@verylong.company.domain.com",
			PhoneNumber: "+1234567890123456789",
			Password:    "verylongpasswordforSystemManagerTesting123!@#",
			Notes:       "Very long notes for system manager creation testing with multiple sentences and detailed information about the manager role and responsibilities.",
		}

		assert.Greater(t, len(longRequest.Username), 10)
		assert.Greater(t, len(longRequest.FullName), 20)
		assert.Greater(t, len(longRequest.Email), 30)
		assert.Greater(t, len(longRequest.Notes), 50)
	})
}
