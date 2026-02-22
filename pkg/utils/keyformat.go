package utils

import (
	"encoding/pem"
	"os"
	"os/user"
	"path/filepath"
)

// KeyFormat represents the cryptographic format of the stored keys.
type KeyFormat int

const (
	// FormatLegacyEC indicates keys use classical ECDSA P-256 / ECDH-ES.
	FormatLegacyEC KeyFormat = iota
	// FormatHybrid indicates keys use ECDH P-256 + ML-KEM-768 hybrid KEM.
	FormatHybrid
)

// DetectKeyFormat reads the private key file and determines the key format
// based on PEM block types.
//   - "EC PRIVATE KEY" → FormatLegacyEC
//   - "SSHSYNC HYBRID SEED" → FormatHybrid
func DetectKeyFormat() (KeyFormat, error) {
	u, err := user.Current()
	if err != nil {
		return 0, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair")
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return detectKeyFormatFromBytes(data), nil
}

func detectKeyFormatFromBytes(data []byte) KeyFormat {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "SSHSYNC HYBRID SEED":
			return FormatHybrid
		case "EC PRIVATE KEY":
			return FormatLegacyEC
		}
	}
	// Default to legacy if unrecognized (shouldn't happen with valid files)
	return FormatLegacyEC
}
