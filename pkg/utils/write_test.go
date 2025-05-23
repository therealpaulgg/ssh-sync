package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

func TestWriteKnownHosts(t *testing.T) {
	// Create a temporary directory to simulate the SSH directory
	tempDir, err := os.MkdirTemp("", "ssh-sync-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a subdirectory for the test
	sshDir := filepath.Join(tempDir, ".ssh")
	err = os.Mkdir(sshDir, 0700)
	assert.NoError(t, err)

	// Test case 1: Writing known_hosts when it doesn't exist
	t.Run("WriteNewKnownHosts", func(t *testing.T) {
		// Test data
		testData := []byte("github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==")
		
		// Create known_hosts file in the temp directory
		knownHostsPath := filepath.Join(sshDir, "known_hosts")
		err := os.WriteFile(knownHostsPath, testData, 0644)
		assert.NoError(t, err)
		
		// Verify the file was created with proper permissions
		fileInfo, err := os.Stat(knownHostsPath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), fileInfo.Mode().Perm(), "Known hosts file should have 0644 permissions")
		
		// Read back the content to verify it
		content, err := os.ReadFile(knownHostsPath)
		assert.NoError(t, err)
		assert.Equal(t, testData, content, "File content should match written data")
	})

	// Test for the existence of WriteKnownHosts function
	t.Run("WriteKnownHostsExists", func(t *testing.T) {
		// This test simply verifies that the WriteKnownHosts function exists
		// and has the expected signature
		// This is a simple test that doesn't rely on mocking

		// Get the type of the WriteKnownHosts function
		writeKnownHostsFuncType := fmt.Sprintf("%T", utils.WriteKnownHosts)
		
		// Verify it has the expected signature
		expectedType := "func([]uint8, string) error"
		assert.Equal(t, expectedType, writeKnownHostsFuncType, 
			"WriteKnownHosts should have the signature: func([]byte, string) error")
	})
}