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
	// FormatPostQuantum indicates keys use ML-DSA-65 / ML-KEM-768.
	FormatPostQuantum
)

// DetectKeyFormat reads the private key file and determines the key format
// based on PEM block types.
//   - "EC PRIVATE KEY" → FormatLegacyEC
//   - "MLDSA65 PRIVATE KEY" → FormatPostQuantum
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
		case "MLDSA65 PRIVATE KEY", "MLKEM768 DECAPSULATION KEY SEED":
			return FormatPostQuantum
		case "EC PRIVATE KEY":
			return FormatLegacyEC
		}
	}
	// Default to legacy if unrecognized (shouldn't happen with valid files)
	return FormatLegacyEC
}

// DetectPublicKeyFormat reads the public key file and determines the format.
func DetectPublicKeyFormat() (KeyFormat, error) {
	u, err := user.Current()
	if err != nil {
		return 0, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return detectPublicKeyFormatFromBytes(data), nil
}

func detectPublicKeyFormatFromBytes(data []byte) KeyFormat {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "MLDSA65 PUBLIC KEY", "MLKEM768 ENCAPSULATION KEY":
			return FormatPostQuantum
		case "PUBLIC KEY":
			return FormatLegacyEC
		}
	}
	return FormatLegacyEC
}

// DetectPEMKeyFormat determines the key format from raw PEM bytes (e.g. received
// from the server during challenge-response). This is used when we receive a
// public key and need to determine how to encrypt with it.
func DetectPEMKeyFormat(data []byte) KeyFormat {
	return detectPublicKeyFormatFromBytes(data)
}
