package utils

import (
	"crypto/mlkem"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/crypto/hkdf"
)

// ML-KEM-768 ciphertext size.
const mlkemCtSize = 1088

// EncryptWithMasterKey encrypts data using AES-256-GCM with the given master key.
// Output format: [nonce (12 bytes)][ciphertext + GCM tag]
// AES-256 is already quantum-resistant (128-bit security against Grover's algorithm).
func EncryptWithMasterKey(plaintext []byte, key []byte) ([]byte, error) {
	return aesGCMEncrypt(key, plaintext)
}

// Encrypt encrypts data using the local key. Auto-detects key format:
//   - Classical EC: JWE with ECDH-ES+A256KW
//   - Post-quantum: ML-KEM-768 + AES-256-GCM
func Encrypt(b []byte) ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		ek, err := RetrieveEncapsulationKey()
		if err != nil {
			return nil, err
		}
		return EncryptMLKEM(b, ek)
	// FormatEC
	default:
		key, err := RetrievePublicKey()
		if err != nil {
			return nil, err
		}
		ciphertext, err := jwe.Encrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, key))
		if err != nil {
			return nil, err
		}
		return ciphertext, nil
	}
}

// EncryptMLKEM encrypts data using ML-KEM-768 key encapsulation + AES-256-GCM.
// Output format: [1088 bytes ML-KEM ct][32 bytes HKDF salt][12 bytes nonce][AES-GCM ct+tag]
func EncryptMLKEM(plaintext []byte, ek *mlkem.EncapsulationKey768) ([]byte, error) {
	// 1. ML-KEM-768 encapsulation → shared secret
	sharedSecret, kemCiphertext := ek.Encapsulate()

	// 2. Generate random HKDF salt (HashLen bytes per RFC 5869 §3.1)
	// https://datatracker.ietf.org/doc/html/rfc5869#section-3.1
	salt := make([]byte, sha256.Size)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating HKDF salt: %w", err)
	}

	// 3. Derive AES-256 key via HKDF
	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, []byte("ssh-sync-pq-v1"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("deriving AES key: %w", err)
	}

	// 4. AES-256-GCM encrypt; nonceAndCiphertext = [nonce][ciphertext+tag]
	nonceAndCiphertext, err := aesGCMEncrypt(aesKey, plaintext)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM encryption: %w", err)
	}

	// 5. Assemble output: [ML-KEM ct][HKDF salt][nonce][AES-GCM ct+tag]
	result := make([]byte, 0, len(kemCiphertext)+len(salt)+len(nonceAndCiphertext))
	result = append(result, kemCiphertext...)
	result = append(result, salt...)
	result = append(result, nonceAndCiphertext...)
	return result, nil
}

// EncryptWithPQPublicKey encrypts data using a PEM-encoded ML-KEM-768 encapsulation key.
// Used during challenge-response when Machine A encrypts the master key for Machine B.
func EncryptWithPQPublicKey(b []byte, ekPEM []byte) ([]byte, error) {
	ekBlock, _ := pem.Decode(ekPEM)
	if ekBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block for encapsulation key")
	}
	if ekBlock.Type != "MLKEM768 ENCAPSULATION KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", ekBlock.Type)
	}
	ek, err := mlkem.NewEncapsulationKey768(ekBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing ML-KEM-768 encapsulation key: %w", err)
	}

	return EncryptMLKEM(b, ek)
}

// EncryptWithECPublicKey encrypts data using a PEM-encoded EC public key via JWE.
func EncryptWithECPublicKey(b []byte, key []byte) ([]byte, error) {
	pubKey, err := jwk.ParseKey(key, jwk.WithPEM(true))
	if err != nil {
		return nil, err
	}
	ciphertext, err := jwe.Encrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, pubKey))
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}
