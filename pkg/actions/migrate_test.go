package actions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveLegacyPublicKeyDeletesFile(t *testing.T) {
	dir := t.TempDir()
	pubPath := filepath.Join(dir, "keypair.pub")
	require.NoError(t, os.WriteFile(pubPath, []byte("legacy"), 0600))

	require.NoError(t, removeLegacyPublicKey(dir))

	_, err := os.Stat(pubPath)
	assert.True(t, errors.Is(err, os.ErrNotExist), "keypair.pub should be removed")
}

func TestRemoveLegacyPublicKeyMissingOK(t *testing.T) {
	dir := t.TempDir()

	assert.NoError(t, removeLegacyPublicKey(dir))
}
