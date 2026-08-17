package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var testLogDir string

func setupFileDriverTest() {
	var err error
	testLogDir, err = os.MkdirTemp("", "astiango-hub-test-logs")
	if err != nil {
		panic(err)
	}
	// Set the log path in viper configuration
	viper.Set("log.path", testLogDir)
}

func cleanupFileDriverTest() {
	_ = os.RemoveAll(testLogDir)
	// Reset the log path in viper configuration
	viper.Set("log.path", "")
}

func TestFileDriver_WriteLine(t *testing.T) {
	setupFileDriverTest()
	t.Cleanup(cleanupFileDriverTest)

	d := newFileLogDriver()
	defer d.Close()

	id := primitive.NewObjectID()

	err := d.WriteLine(id.Hex(), "it works")
	require.Nil(t, err)

	logFilePath := filepath.Join(testLogDir, id.Hex(), "log.txt")
	require.FileExists(t, logFilePath)
	text, err := os.ReadFile(logFilePath)
	require.Nil(t, err)
	require.Equal(t, "it works\n", string(text))
}

func TestFileDriver_WriteLines(t *testing.T) {
	setupFileDriverTest()
	t.Cleanup(cleanupFileDriverTest)

	d := newFileLogDriver()
	defer d.Close()

	id := primitive.NewObjectID()

	for i := 0; i < 100; i++ {
		err := d.WriteLine(id.Hex(), "it works")
		require.Nil(t, err)
	}

	logFilePath := filepath.Join(testLogDir, id.Hex(), "log.txt")
	require.FileExists(t, logFilePath)
	text, err := os.ReadFile(logFilePath)
	require.Nil(t, err)
	require.Contains(t, string(text), "it works\n")
	lines := strings.Split(string(text), "\n")
	require.Equal(t, 101, len(lines))
}

func TestFileDriver_Find(t *testing.T) {
	setupFileDriverTest()
	t.Cleanup(cleanupFileDriverTest)

	d := newFileLogDriver()
	defer d.Close()

	id := primitive.NewObjectID()

	batch := 1000
	var lines []string
	for i := 0; i < 10; i++ {
		for j := 0; j < batch; j++ {
			line := fmt.Sprintf("line: %d", i*batch+j+1)
			lines = append(lines, line)
		}
		err := d.WriteLines(id.Hex(), lines)
		require.Nil(t, err)
		lines = []string{}
	}

	driver := d

	t.Run("Normal Mode (tail=false)", func(t *testing.T) {
		// Test reading first 10 lines
		lines, err := driver.Find(id.Hex(), "", 0, 10, false)
		require.Nil(t, err)
		assert.Equal(t, 10, len(lines))
		assert.Equal(t, "line: 1", lines[0])
		assert.Equal(t, "line: 10", lines[9])

		// Test reading with skip
		lines, err = driver.Find(id.Hex(), "", 5, 5, false)
		require.Nil(t, err)
		assert.Equal(t, 5, len(lines))
		assert.Equal(t, "line: 6", lines[0])
		assert.Equal(t, "line: 10", lines[4])

		// Test reading with no limit (should read to end)
		lines, err = driver.Find(id.Hex(), "", 9995, 0, false)
		require.Nil(t, err)
		assert.Equal(t, 5, len(lines))
		assert.Equal(t, "line: 9996", lines[0])
		assert.Equal(t, "line: 10000", lines[4])

		// Test reading past end
		lines, err = driver.Find(id.Hex(), "", 10000, 10, false)
		require.Nil(t, err)
		assert.Equal(t, 0, len(lines))
	})

	t.Run("Tail Mode (tail=true)", func(t *testing.T) {
		// Test reading last 10 lines
		lines, err := driver.Find(id.Hex(), "", 0, 10, true)
		require.Nil(t, err)
		assert.Equal(t, 10, len(lines))
		assert.Equal(t, "line: 9991", lines[0])
		assert.Equal(t, "line: 10000", lines[9])

		// Test reading with skip from end
		lines, err = driver.Find(id.Hex(), "", 5, 10, true)
		require.Nil(t, err)
		assert.Equal(t, 10, len(lines))
		assert.Equal(t, "line: 9986", lines[0])
		assert.Equal(t, "line: 9995", lines[9])

		// Test reading with no limit (should read all lines after skip)
		lines, err = driver.Find(id.Hex(), "", 5, 0, true)
		require.Nil(t, err)
		assert.Equal(t, 9995, len(lines))
		assert.Equal(t, "line: 1", lines[0])
		assert.Equal(t, "line: 9995", lines[9994])

		// Test reading with large skip
		lines, err = driver.Find(id.Hex(), "", 9995, 10, true)
		require.Nil(t, err)
		assert.Equal(t, 5, len(lines))
		assert.Equal(t, "line: 1", lines[0])
		assert.Equal(t, "line: 5", lines[4])

		// Test reading with skip larger than file
		lines, err = driver.Find(id.Hex(), "", 10001, 10, true)
		require.Nil(t, err)
		assert.Equal(t, 0, len(lines))
	})

	cleanupFileDriverTest()
}
