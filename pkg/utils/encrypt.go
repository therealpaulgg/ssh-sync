package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/pem"
	"fmt"
)

// EncryptWithMasterKey encrypts data using AES-256-GCM with the given master key.
// Output format: [nonce (12 bytes)][ciphertext + GCM tag]
// AES-256 is already quantum-resistant (128-bit security against Grover's algorithm).
func EncryptWithMasterKey(plaintext []byte, key []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if n, err := rand.Read(nonce); err != nil || n != len(nonce) {
		return nil, err
	}
	outBuf := gcm.Seal(nonce, nonce, plaintext, nil)
	return outBuf, nil
}

// Encrypt encrypts data using the local ML-KEM-768 encapsulation key.
// Uses ML-KEM-768 key encapsulation to derive a shared secret, then
// encrypts with AES-256-GCM.
// Output format: [1088 bytes ML-KEM ciphertext][12 bytes nonce][AES-GCM ciphertext + tag]
func Encrypt(b []byte) ([]byte, error) {
	ek, err := RetrieveEncapsulationKey()
	if err != nil {
		return nil, err
	}
	return EncryptMLKEM(b, ek)
}

// EncryptMLKEM encrypts data using an ML-KEM-768 encapsulation key.
// Output format: [1088 bytes ML-KEM ciphertext][12 bytes nonce][AES-GCM ciphertext + tag]
func EncryptMLKEM(plaintext []byte, ek *mlkem.EncapsulationKey768) ([]byte, error) {
	// Encapsulate to produce shared secret and ciphertext
	sharedKey, kemCiphertext := ek.Encapsulate()

	// Use shared secret as AES-256 key
	blockCipher, err := aes.NewCipher(sharedKey)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	aesCiphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Output: [KEM ciphertext][nonce][AES-GCM ciphertext]
	result := make([]byte, 0, len(kemCiphertext)+len(nonce)+len(aesCiphertext))
	result = append(result, kemCiphertext...)
	result = append(result, nonce...)
	result = append(result, aesCiphertext...)
	return result, nil
}

// EncryptWithPublicKey encrypts data using a PEM-encoded ML-KEM-768 encapsulation key.
// Used during challenge-response to encrypt master key with the server's public key.
func EncryptWithPublicKey(b []byte, key []byte) ([]byte, error) {
	// Parse the PEM-encoded encapsulation key
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for encapsulation key")
	}
	if block.Type != "MLKEM768 ENCAPSULATION KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}
	ek, err := mlkem.NewEncapsulationKey768(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing ML-KEM-768 encapsulation key: %w", err)
	}
	return EncryptMLKEM(b, ek)
}
