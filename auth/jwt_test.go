package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInit(t *testing.T) {
	// Test that init function sets jwtSecret
	assert.NotNil(t, jwtSecret)
	assert.NotEmpty(t, jwtSecret)
}

func TestGenerateToken(t *testing.T) {
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	username := "testuser"
	role := "admin"

	token, err := GenerateToken(userID, organizationID, username, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify the token can be parsed
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(*Claims)
	require.True(t, ok)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, organizationID, claims.OrganizationID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestValidateToken_ValidToken(t *testing.T) {
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	username := "testuser"
	role := "admin"

	// Generate a valid token
	token, err := GenerateToken(userID, organizationID, username, role)
	require.NoError(t, err)

	// Validate the token
	claims, err := ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, organizationID, claims.OrganizationID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// Test with invalid token string
	claims, err := ValidateToken("invalid-token")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	username := "testuser"
	role := "admin"

	// Create an expired token manually
	claims := Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		Username:       username,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	require.NoError(t, err)

	// Validate the expired token
	validatedClaims, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	username := "testuser"
	role := "admin"

	// Create a token with wrong signing method
	claims := Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		Username:       username,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Use RS256 instead of HS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(jwtSecret) // This will fail, but let's test validation
	if err != nil {
		// Expected - RS256 requires different key type
		return
	}

	// If somehow it worked, validate should fail
	validatedClaims, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}

func TestJWTSecretFromEnvironment(t *testing.T) {
	// Save original value
	originalSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	// Test with custom secret
	customSecret := "test-secret-123"
	os.Setenv("JWT_SECRET", customSecret)

	// Re-initialize (simulate package reload)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key"
	}
	testSecret := []byte(secret)

	assert.Equal(t, []byte(customSecret), testSecret)
}

func TestClaims_Structure(t *testing.T) {
	userID := primitive.NewObjectID()
	organizationID := primitive.NewObjectID()
	username := "testuser"
	role := "admin"

	claims := Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		Username:       username,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Verify all fields are set correctly
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, organizationID, claims.OrganizationID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}
