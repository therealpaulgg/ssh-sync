package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/mlkem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

// DecryptMLKEM decrypts data that was encrypted with EncryptMLKEM.
// Input format: [1088 bytes ML-KEM ciphertext][12 bytes nonce][AES-GCM ciphertext + tag]
func DecryptMLKEM(data []byte, dk *mlkem.DecapsulationKey768) ([]byte, error) {
	if len(data) < mlkem.CiphertextSize768 {
		return nil, fmt.Errorf("data too short for ML-KEM ciphertext")
	}

	// Extract ML-KEM ciphertext
	kemCiphertext := data[:mlkem.CiphertextSize768]
	remainder := data[mlkem.CiphertextSize768:]

	// Decapsulate to recover shared secret
	sharedKey, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// Use shared secret as AES-256 key to decrypt remainder
	blockCipher, err := aes.NewCipher(sharedKey)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	if len(remainder) < gcm.NonceSize() {
		return nil, fmt.Errorf("data too short for nonce")
	}

	nonce := remainder[:gcm.NonceSize()]
	aesCiphertext := remainder[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, aesCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decryption failed: %w", err)
	}
	return plaintext, nil
}

// Decrypt decrypts data using the local key. Auto-detects key format:
//   - Legacy EC: JWE with ECDH-ES+A256KW
//   - Post-quantum: ML-KEM-768 decapsulation + AES-256-GCM
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

	default: // FormatLegacyEC
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
	decryptedBuf := bytes.NewBuffer(nil)
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	data, err := gcm.Open(nil, b[:gcm.NonceSize()], b[gcm.NonceSize():], nil)
	if err != nil {
		return nil, err
	}
	if _, err := decryptedBuf.Write(data); err != nil {
		return nil, err
	}
	plaintext := decryptedBuf.Bytes()
	return plaintext, nil
}
