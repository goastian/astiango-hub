package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExists(t *testing.T) {
	// Test cases
	pathString := "../config"
	wrongPathString := "test"

	// Test existing path
	res := Exists(pathString)
	assert.True(t, res, "Expected existing path to return true")

	// Test non-existing path
	wrongRes := Exists(wrongPathString)
	assert.False(t, wrongRes, "Expected non-existing path to return false")
}

func TestIsDir(t *testing.T) {
	// Test cases
	pathString := "../config"
	fileString := "../config/config.go"
	wrongString := "test"

	// Test directory path
	res := IsDir(pathString)
	assert.True(t, res, "Expected directory path to return true")

	// Test file path
	fileRes := IsDir(fileString)
	assert.False(t, fileRes, "Expected file path to return false")

	// Test non-existing path
	wrongRes := IsDir(wrongString)
	assert.False(t, wrongRes, "Expected non-existing path to return false")
}

func TestIgnoreFileRegexPattern(t *testing.T) {
	ignoreRegex, err := regexp.Compile(IgnoreFileRegexPattern)
	assert.NoError(t, err, "Regex should compile without error")

	testCases := []struct {
		path     string
		expected bool
		desc     string
	}{
		// Should be ignored (directories)
		{".git/", true, ".git directory should be ignored"},
		{".git/config", true, ".git/config should be ignored"},
		{".git/objects/pack", true, ".git subdirectory should be ignored"},
		{"node_modules/", true, "node_modules directory should be ignored"},
		{"node_modules/package/index.js", true, "node_modules file should be ignored"},
		{"src/__pycache__/", true, "__pycache__ directory should be ignored"},

		// Should be ignored (file extensions)
		{"file.tmp", true, ".tmp files should be ignored"},
		{"file.temp", true, ".temp files should be ignored"},
		{"file.log", true, ".log files should be ignored"},
		{"file.swp", true, ".swp files should be ignored"},
		{"file.pyc", true, ".pyc files should be ignored"},

		// Should NOT be ignored
		{"main.py", false, "Python files should not be ignored"},
		{"src/app.js", false, "JavaScript files should not be ignored"},
		{".gitignore", false, ".gitignore file should not be ignored"},
		{".github/workflows/ci.yml", false, ".github directory should not be ignored"},
		{"config.json", false, "JSON files should not be ignored"},
	}

	for _, tc := range testCases {
		result := ignoreRegex.MatchString(tc.path)
		assert.Equal(t, tc.expected, result, tc.desc)
	}
}
