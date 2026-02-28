package actions

import (
	"os"
	"path/filepath"
	"testing"

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
