package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/mlkem"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/crypto/hkdf"
)

// Hybrid KEM ciphertext layout sizes.
const (
	ecdhPubKeySize  = 65   // uncompressed P-256 point (0x04 || x || y)
	mlkemCtSize     = 1088 // ML-KEM-768 ciphertext size
	hybridHeaderLen = ecdhPubKeySize + mlkemCtSize
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

// Encrypt encrypts data using the local key. Auto-detects key format:
//   - Legacy EC: JWE with ECDH-ES+A256KW
//   - Hybrid: ECDH P-256 + ML-KEM-768 + AES-256-GCM
func Encrypt(b []byte) ([]byte, error) {
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
		ek, err := RetrieveEncapsulationKey()
		if err != nil {
			return nil, err
		}
		return EncryptHybrid(b, ecPriv.PublicKey(), ek)

	default: // FormatLegacyEC
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

// EncryptHybrid encrypts data using a hybrid ECDH P-256 + ML-KEM-768 KEM.
// Output format: [65 bytes eph EC pub][1088 bytes ML-KEM ct][12 bytes nonce][AES-GCM ct+tag]
func EncryptHybrid(plaintext []byte, ecPub *ecdh.PublicKey, ek *mlkem.EncapsulationKey768) ([]byte, error) {
	// 1. Generate ephemeral EC P-256 keypair and perform ECDH
	ephPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating ephemeral EC key: %w", err)
	}
	sharedEC, err := ephPriv.ECDH(ecPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// 2. ML-KEM-768 encapsulation
	sharedKEM, kemCiphertext := ek.Encapsulate()

	// 3. Combine shared secrets via HKDF
	combined := make([]byte, 0, len(sharedEC)+len(sharedKEM))
	combined = append(combined, sharedEC...)
	combined = append(combined, sharedKEM...)
	hkdfReader := hkdf.New(sha256.New, combined, nil, []byte("ssh-sync-hybrid-v1"))
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, aesKey); err != nil {
		return nil, fmt.Errorf("deriving AES key: %w", err)
	}

	// 4. AES-256-GCM encrypt
	blockCipher, err := aes.NewCipher(aesKey)
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

	// 5. Assemble output: [eph EC pub][ML-KEM ct][nonce][AES-GCM ct]
	ephPubBytes := ephPriv.PublicKey().Bytes()
	result := make([]byte, 0, len(ephPubBytes)+len(kemCiphertext)+len(nonce)+len(aesCiphertext))
	result = append(result, ephPubBytes...)
	result = append(result, kemCiphertext...)
	result = append(result, nonce...)
	result = append(result, aesCiphertext...)
	return result, nil
}

// EncryptWithHybridPublicKey encrypts data using PEM-encoded EC + ML-KEM public keys.
// Used during challenge-response when Machine A encrypts the master key for Machine B.
func EncryptWithHybridPublicKey(b []byte, ecPubPEM []byte, ekPEM []byte) ([]byte, error) {
	// Parse EC public key
	ecBlock, _ := pem.Decode(ecPubPEM)
	if ecBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block for EC public key")
	}
	genericKey, err := x509.ParsePKIXPublicKey(ecBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing EC public key: %w", err)
	}
	ecdsaPub, ok := genericKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected *ecdsa.PublicKey, got %T", genericKey)
	}
	ecdhPub, err := ecdsaPub.ECDH()
	if err != nil {
		return nil, fmt.Errorf("converting EC public key to ECDH: %w", err)
	}

	// Parse ML-KEM encapsulation key
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

	return EncryptHybrid(b, ecdhPub, ek)
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
