package utils

import (
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"golang.org/x/crypto/hkdf"
)

// hkdfSaltSize is the HKDF salt length, equal to HashLen (SHA-256 output size) per RFC 5869 §3.1.
// https://datatracker.ietf.org/doc/html/rfc5869#section-3.1
const hkdfSaltSize = sha256.Size

// DecryptMLKEM decrypts data encrypted with EncryptMLKEM.
// Input format: [1088 bytes ML-KEM ct][32 bytes HKDF salt][12 bytes nonce][AES-GCM ct+tag]
func DecryptMLKEM(data []byte, dk *mlkem.DecapsulationKey768) ([]byte, error) {
	if len(data) < mlkemCtSize+hkdfSaltSize {
		return nil, fmt.Errorf("data too short for ML-KEM ciphertext")
	}

	// 1. Parse ML-KEM ciphertext and HKDF salt
	kemCiphertext := data[:mlkemCtSize]
	salt := data[mlkemCtSize : mlkemCtSize+hkdfSaltSize]
	remainder := data[mlkemCtSize+hkdfSaltSize:]

	// 2. ML-KEM-768 decapsulation → shared secret
	sharedSecret, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// 3. Derive AES-256 key via HKDF (same as encrypt)
	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, []byte("ssh-sync-pq-v1"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("deriving AES key: %w", err)
	}

	// 4. AES-256-GCM decrypt
	plaintext, err := aesGCMDecrypt(aesKey, remainder)
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
