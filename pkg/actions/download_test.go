package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
)

func TestCheckForDeletedKeysSkipsAuthorizedKeys(t *testing.T) {
	tmpDir := t.TempDir()

	authKeysPath := filepath.Join(tmpDir, "authorized_keys")
	if err := os.WriteFile(authKeysPath, []byte("dummy"), 0600); err != nil {
		t.Fatalf("failed to write authorized_keys: %v", err)
	}

	if err := checkForDeletedKeys([]dto.KeyDto{}, tmpDir); err != nil {
		t.Fatalf("checkForDeletedKeys returned error: %v", err)
	}

	if _, err := os.Stat(authKeysPath); err != nil {
		t.Fatalf("authorized_keys should remain untouched, got error: %v", err)
	}
}

func TestCheckForDeletedKeysSkipsConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(configPath, []byte("Host example\n"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := checkForDeletedKeys([]dto.KeyDto{}, tmpDir); err != nil {
		t.Fatalf("checkForDeletedKeys returned error: %v", err)
	}

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config should remain untouched, got error: %v", err)
	}
}

func TestCheckForDeletedKeysSkipsKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()

	knownHostsPath := filepath.Join(tmpDir, "known_hosts")
	if err := os.WriteFile(knownHostsPath, []byte("github.com ssh-ed25519 AAAA\n"), 0644); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}

	if err := checkForDeletedKeys([]dto.KeyDto{}, tmpDir); err != nil {
		t.Fatalf("checkForDeletedKeys returned error: %v", err)
	}

	if _, err := os.Stat(knownHostsPath); err != nil {
		t.Fatalf("known_hosts should remain untouched, got error: %v", err)
	}
}

func TestIsReservedFilename(t *testing.T) {
	reserved := []string{"known_hosts", "authorized_keys", "config"}
	for _, name := range reserved {
		assert.True(t, isReservedFilename(name), "%q should be reserved", name)
	}

	notReserved := []string{"id_rsa", "id_rsa.pub", "id_ed25519", "id_ed25519.pub", "id_ecdsa"}
	for _, name := range notReserved {
		assert.False(t, isReservedFilename(name), "%q should not be reserved", name)
	}
}
