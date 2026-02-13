package utils

import (
	"crypto/mlkem"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// --- Legacy EC key retrieval (ECDSA / ECDH-ES) ---

// RetrievePrivateKey loads a legacy EC private key (JWK) from ~/.ssh-sync/keypair.
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

// RetrievePublicKey loads a legacy EC public key (JWK) from ~/.ssh-sync/keypair.pub.
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

// --- Post-quantum key retrieval (ML-DSA-65 / ML-KEM-768) ---

// RetrieveSigningPrivateKey loads the ML-DSA-65 private key from ~/.ssh-sync/keypair.
func RetrieveSigningPrivateKey() (*mldsa65.PrivateKey, error) {
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
		if block.Type == "MLDSA65 PRIVATE KEY" {
			var sk mldsa65.PrivateKey
			if err := sk.UnmarshalBinary(block.Bytes); err != nil {
				return nil, fmt.Errorf("unmarshaling ML-DSA-65 private key: %w", err)
			}
			return &sk, nil
		}
	}
	return nil, fmt.Errorf("ML-DSA-65 private key not found in %s", p)
}

// RetrieveSigningPublicKey loads the ML-DSA-65 public key from ~/.ssh-sync/keypair.pub.
func RetrieveSigningPublicKey() (*mldsa65.PublicKey, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
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
		if block.Type == "MLDSA65 PUBLIC KEY" {
			var pk mldsa65.PublicKey
			if err := pk.UnmarshalBinary(block.Bytes); err != nil {
				return nil, fmt.Errorf("unmarshaling ML-DSA-65 public key: %w", err)
			}
			return &pk, nil
		}
	}
	return nil, fmt.Errorf("ML-DSA-65 public key not found in %s", p)
}

// RetrieveDecapsulationKey loads the ML-KEM-768 decapsulation key from ~/.ssh-sync/keypair.
func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
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
		if block.Type == "MLKEM768 DECAPSULATION KEY SEED" {
			dk, err := mlkem.NewDecapsulationKey768(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("creating ML-KEM-768 decapsulation key: %w", err)
			}
			return dk, nil
		}
	}
	return nil, fmt.Errorf("ML-KEM-768 decapsulation key not found in %s", p)
}

// RetrieveEncapsulationKey loads the ML-KEM-768 encapsulation key from ~/.ssh-sync/keypair.pub.
func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
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
		if block.Type == "MLKEM768 ENCAPSULATION KEY" {
			ek, err := mlkem.NewEncapsulationKey768(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("creating ML-KEM-768 encapsulation key: %w", err)
			}
			return ek, nil
		}
	}
	return nil, fmt.Errorf("ML-KEM-768 encapsulation key not found in %s", p)
}

// --- Format-aware master key retrieval ---

// RetrieveMasterKey reads and decrypts the master key from ~/.ssh-sync/master_key.
// It auto-detects the key format:
//   - Legacy: JWE encrypted with ECDH-ES+A256KW
//   - Post-quantum: ML-KEM-768 encapsulation + AES-256-GCM
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
			return nil, fmt.Errorf("decrypting master key (post-quantum): %w", err)
		}
		return masterKey, nil

	default: // FormatLegacyEC
		privateKey, err := RetrievePrivateKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := jwe.Decrypt(file, jwe.WithKey(jwa.ECDH_ES_A256KW, privateKey))
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (legacy EC): %w", err)
		}
		return masterKey, nil
	}
}
