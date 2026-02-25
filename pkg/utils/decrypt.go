package utils

import (
	"crypto/mlkem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

// DecryptMLKEM decrypts data encrypted with EncryptMLKEM.
// Input format: [1088 bytes ML-KEM ct][12 bytes nonce][AES-GCM ct+tag]
func DecryptMLKEM(data []byte, dk *mlkem.DecapsulationKey768) ([]byte, error) {
	if len(data) < mlkemCtSize {
		return nil, fmt.Errorf("data too short for ML-KEM ciphertext")
	}

	// 1. Parse ML-KEM ciphertext
	kemCiphertext := data[:mlkemCtSize]
	remainder := data[mlkemCtSize:]

	// 2. ML-KEM-768 decapsulation → shared secret (32 bytes, used directly as AES-256 key)
	sharedSecret, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// 3. AES-256-GCM decrypt
	plaintext, err := aesGCMDecrypt(sharedSecret, remainder)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decryption failed: %w", err)
	}
	return plaintext, nil
}

// Decrypt decrypts data using the local key. Auto-detects key format:
//   - Classical EC: JWE with ECDH-ES+A256KW
//   - Post-quantum: ML-KEM-768 + AES-256-GCM
func Decrypt(b []byte) ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		return DecryptMLKEM(b, dk)
	// FormatEC
	default:
		key, err := RetrievePrivateKey()
		if err != nil {
			return nil, err
		}
		plaintext, err := jwe.Decrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, key))
		if err != nil {
			return nil, err
		}
		return plaintext, nil
	}
}

// DecryptWithMasterKey decrypts data using AES-256-GCM with the given master key.
// Input format: [nonce (12 bytes)][ciphertext + GCM tag]
func DecryptWithMasterKey(b []byte, key []byte) ([]byte, error) {
	return aesGCMDecrypt(key, b)
}
