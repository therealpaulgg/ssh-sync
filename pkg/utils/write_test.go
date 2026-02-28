package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteKeyNonInteractiveSkipsDiff(t *testing.T) {
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, "ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("failed to create ssh dir: %v", err)
	}
	target := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(target, []byte("old-key"), 0600); err != nil {
		t.Fatalf("failed to seed key: %v", err)
	}

	if err := WriteKey([]byte("new-key"), "id_rsa", sshDir, true); err != nil {
		t.Fatalf("WriteKey returned error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed to read key after write: %v", err)
	}
	assert.Equal(t, "old-key", string(data), "non-interactive mode should skip overwriting differing keys")
}
