package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"golang.org/x/crypto/hkdf"
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

	// 2. ML-KEM-768 decapsulation → shared secret
	sharedSecret, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// 3. Derive AES-256 key via HKDF (same as encrypt)
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, []byte("ssh-sync-pq-v1"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("deriving AES key: %w", err)
	}

	// 4. AES-256-GCM decrypt
	blockCipher, err := aes.NewCipher(aesKey)
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
