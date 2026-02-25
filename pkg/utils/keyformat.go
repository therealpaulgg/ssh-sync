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
	// FormatEC indicates keys use classical ECDSA P-256 / ECDH-ES.
	FormatEC KeyFormat = iota
	// FormatPostQuantum indicates keys use ML-DSA-65 + ML-KEM-768.
	FormatPostQuantum
)

// DetectKeyFormat reads the private key file and determines the key format
// based on PEM block types.
//   - "EC PRIVATE KEY" → FormatEC
//   - "SSHSYNC PQ MASTER SEED" → FormatPostQuantum
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
		case "SSHSYNC PQ MASTER SEED":
			return FormatPostQuantum
		case "EC PRIVATE KEY":
			return FormatEC
		}
	}
	// Default to EC if unrecognized (shouldn't happen with valid files)
	return FormatEC
}
