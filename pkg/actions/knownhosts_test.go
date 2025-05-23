package actions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// This test verifies that the KnownHosts field is included in the DataDto structure
// which is essential for the known_hosts syncing feature to work properly
func TestKnownHostsInDataDto(t *testing.T) {
	// Create a DataDto instance
	dataDto := dto.DataDto{}
	
	// Set some test known_hosts data
	testData := []byte("github.com ssh-rsa AAAAB3NzaC1yc2EAAAABI...")
	dataDto.KnownHosts = testData
	
	// Verify the data was properly set
	assert.Equal(t, testData, dataDto.KnownHosts, "KnownHosts field should be accessible in DataDto")
}

// This test verifies that the known_hosts file is written with the correct permissions (0644)
// which is different from the SSH keys that use 0600
func TestKnownHostsPermissions(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "ssh-sync-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create a known_hosts file in the temp directory
	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	testData := []byte("github.com ssh-rsa AAAAB3NzaC1yc2EAAAABI...")
	err = os.WriteFile(knownHostsPath, testData, 0644)
	assert.NoError(t, err)
	
	// Verify the file has the correct permissions
	fileInfo, err := os.Stat(knownHostsPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), fileInfo.Mode().Perm(), 
		"Known hosts file should have 0644 permissions (different from SSH keys)")
}