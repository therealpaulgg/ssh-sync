package utils

import (
	"bytes"
	"crypto/mlkem"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"filippo.io/mldsa"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// --- EC key retrieval (ECDSA / ECDH-ES) ---

// RetrievePrivateKey loads a EC private key (JWK) from ~/.ssh-sync/keypair.
func RetrievePrivateKey() (jwk.Key, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	key, err := jwk.ParseKey(file, jwk.WithPEM(true))
	return key, err
}

// RetrievePublicKey loads a EC public key (JWK) from ~/.ssh-sync/keypair.pub.
func RetrievePublicKey() (jwk.Key, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	key, err := jwk.ParseKey(file, jwk.WithPEM(true))
	return key, err
}

// --- Post-quantum key retrieval (ML-DSA-65 + ML-KEM-768) ---

// retrievePQSeed reads the PQ master seed from ~/.ssh-sync/keypair.
// Returns nil if the keypair file doesn't contain a PQ seed PEM block.
func retrievePQSeed() ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "SSHSYNC PQ MASTER SEED" {
			return block.Bytes, nil
		}
	}
	return nil, nil
}

// RetrieveSigningKey loads the ML-DSA-65 private key from the PQ master seed.
func RetrieveSigningKey() (*mldsa.PrivateKey, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	sk, err := DeriveMLDSAKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-DSA-65 key from seed: %w", err)
	}
	return sk, nil
}

// RetrieveDecapsulationKey loads the ML-KEM-768 decapsulation key from the PQ master seed.
func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	dk, err := DeriveMLKEMKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving decapsulation key from seed: %w", err)
	}
	return dk, nil
}

// RetrieveEncapsulationKey derives the ML-KEM-768 encapsulation key from the PQ master seed.
func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	dk, err := RetrieveDecapsulationKey()
	if err != nil {
		return nil, err
	}
	return dk.EncapsulationKey(), nil
}

// BuildMLDSAPublicKeyPEM returns the ML-DSA-65 public key as a PEM-encoded byte slice.
func BuildMLDSAPublicKeyPEM() ([]byte, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	sk, err := DeriveMLDSAKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-DSA key for public key PEM: %w", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "MLDSA PUBLIC KEY", Bytes: sk.PublicKey().Bytes()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildMLKEMEncapsulationKeyPEM returns the ML-KEM-768 encapsulation key as a PEM-encoded byte slice.
func BuildMLKEMEncapsulationKeyPEM() ([]byte, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	dk, err := DeriveMLKEMKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-KEM key for encapsulation key PEM: %w", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "MLKEM768 ENCAPSULATION KEY", Bytes: dk.EncapsulationKey().Bytes()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- Format-aware master key retrieval ---

// RetrieveMasterKey reads and decrypts the master key from ~/.ssh-sync/master_key.
// It auto-detects the key format:
//   - JWE encrypted with ECDH-ES+A256KW
//   - Post-quantum: ML-KEM-768 + AES-256-GCM
func RetrieveMasterKey() ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "master_key")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := DecryptMLKEM(file, dk)
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (PQ): %w", err)
		}
		return masterKey, nil
	// FormatEC
	default:
		privateKey, err := RetrievePrivateKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := jwe.Decrypt(file, jwe.WithKey(jwa.ECDH_ES_A256KW, privateKey))
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (EC): %w", err)
		}
		return masterKey, nil
	}
}
