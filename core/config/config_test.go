package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set test environment variables
	viper.Set("ASTIANGO_TEST_STRING", "test_string_value")
	viper.Set("ASTIANGO_TEST_INT", 42)
	viper.Set("ASTIANGO_TEST_BOOL", true)
	viper.Set("ASTIANGO_TEST_NESTED_KEY", "nested_value")
}

func TestInitConfig(t *testing.T) {
	// Create a new Config instance
	InitConfig()

	// Test default values
	assert.Equal(t, "localhost", viper.GetString("mongo.host"), "Unexpected default value for mongo.host")
	assert.Equal(t, 27017, viper.GetInt("mongo.port"), "Unexpected default value for mongo.port")
	assert.Equal(t, "astiango_hub_test", viper.GetString("mongo.db"), "Unexpected default value for mongo.db")

	// Test environment variable override
	os.Setenv("ASTIANGO_MONGO_HOST", "mongodb.example.com")
	defer os.Unsetenv("ASTIANGO_MONGO_HOST")
	assert.Equal(t, "mongodb.example.com", viper.GetString("mongo.host"), "Environment variable should override default value")

	// Test with a config file
	tempDir, err := os.MkdirTemp("", "astiango-hub-config-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	configContent := []byte(`
edition: global.edition.pro
mongo:
  host: mongodb.custom.com
  port: 27018
server:
  port: 8001
`)
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, configContent, 0644)
	require.NoError(t, err, "Failed to write config file")

	// Remove the environment variable before testing with config file
	os.Unsetenv("ASTIANGO_MONGO_HOST")

	// Create a new Config instance with the config file
	cWithFile := Config{
		Name:   configPath,
		Logger: utils.NewLogger("Config"),
	}
	cWithFile.Init()

	// Test values from config file
	assert.Equal(t, "global.edition.pro", viper.GetString("edition"), "Unexpected value for edition from config file")
	assert.Equal(t, "mongodb.custom.com", viper.GetString("mongo.host"), "Unexpected value for mongo.host from config file")
	assert.Equal(t, 27018, viper.GetInt("mongo.port"), "Unexpected value for mongo.port from config file")
	assert.Equal(t, 8001, viper.GetInt("server.port"), "Unexpected value for server.port from config file")
}
