package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"golang.org/x/crypto/hkdf"
)

// DecryptHybrid decrypts data encrypted with EncryptHybrid.
// Input format: [65 bytes eph EC pub][1088 bytes ML-KEM ct][12 bytes nonce][AES-GCM ct+tag]
func DecryptHybrid(data []byte, ecPriv *ecdh.PrivateKey, dk *mlkem.DecapsulationKey768) ([]byte, error) {
	if len(data) < hybridHeaderLen {
		return nil, fmt.Errorf("data too short for hybrid ciphertext header")
	}

	// 1. Parse components
	ephPubBytes := data[:ecdhPubKeySize]
	kemCiphertext := data[ecdhPubKeySize:hybridHeaderLen]
	remainder := data[hybridHeaderLen:]

	// 2. ECDH key agreement with ephemeral public key
	ephPub, err := ecdh.P256().NewPublicKey(ephPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing ephemeral EC public key: %w", err)
	}
	sharedEC, err := ecPriv.ECDH(ephPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// 3. ML-KEM-768 decapsulation
	sharedKEM, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	// 4. Combine shared secrets via HKDF (same as encrypt)
	combined := make([]byte, 0, len(sharedEC)+len(sharedKEM))
	combined = append(combined, sharedEC...)
	combined = append(combined, sharedKEM...)
	hkdfReader := hkdf.New(sha256.New, combined, nil, []byte("ssh-sync-hybrid-v1"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("deriving AES key: %w", err)
	}

	// 5. AES-256-GCM decrypt
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
//   - Legacy EC: JWE with ECDH-ES+A256KW
//   - Hybrid: ECDH P-256 + ML-KEM-768 + AES-256-GCM
func Decrypt(b []byte) ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatHybrid:
		ecPriv, err := RetrieveHybridECKey()
		if err != nil {
			return nil, err
		}
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		return DecryptHybrid(b, ecPriv, dk)

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
