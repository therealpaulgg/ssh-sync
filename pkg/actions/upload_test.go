package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSkippedBinaryUpload(t *testing.T) {
	skipped := []string{"known_hosts", "authorized_keys"}
	for _, name := range skipped {
		assert.True(t, isSkippedBinaryUpload(name), "%q should be skipped from binary upload", name)
	}

	notSkipped := []string{"id_rsa", "id_rsa.pub", "id_ed25519", "id_ed25519.pub", "id_ecdsa", "config"}
	for _, name := range notSkipped {
		assert.False(t, isSkippedBinaryUpload(name), "%q should not be skipped from binary upload", name)
	}
}
