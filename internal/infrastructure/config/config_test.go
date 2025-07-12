package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("JWT_SECRET", "test_secret")
	os.Setenv("GOOGLE_CLIENT_ID", "test_client_id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test_client_secret")

	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("GOOGLE_CLIENT_ID")
		os.Unsetenv("GOOGLE_CLIENT_SECRET")
	}()

	config, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "debug", config.Server.Mode)

	// Test JWT configuration
	assert.Equal(t, "test_secret", config.JWT.Secret)
	assert.Equal(t, 24*time.Hour, config.JWT.Expiration)

	// Test Google OAuth configuration
	assert.Equal(t, "test_client_id", config.Google.ClientID)
	assert.Equal(t, "test_client_secret", config.Google.ClientSecret)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"JWT_SECRET":           "test_secret",
				"GOOGLE_CLIENT_ID":     "test_client_id",
				"GOOGLE_CLIENT_SECRET": "test_client_secret",
			},
			expectError: false,
		},
		{
			name: "missing JWT secret",
			envVars: map[string]string{
				"GOOGLE_CLIENT_ID":     "test_client_id",
				"GOOGLE_CLIENT_SECRET": "test_client_secret",
			},
			expectError: true,
		},
		{
			name: "missing Google credentials",
			envVars: map[string]string{
				"JWT_SECRET": "test_secret",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			_, err := LoadConfig()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetDatabaseDSN(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
			SSLMode:  "disable",
		},
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	assert.Equal(t, expected, config.GetDatabaseDSN())
}

func TestGetRedisAddr(t *testing.T) {
	config := &Config{
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	expected := "localhost:6379"
	assert.Equal(t, expected, config.GetRedisAddr())
}
