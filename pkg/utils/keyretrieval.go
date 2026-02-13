package utils

import (
	"bytes"
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

// retrieveMasterSeed reads the PQ master seed from ~/.ssh-sync/keypair.
// Returns nil if the keypair file doesn't contain a master seed PEM block.
func retrieveMasterSeed() ([]byte, error) {
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

// RetrieveSigningPrivateKey loads the ML-DSA-65 private key from ~/.ssh-sync/keypair
// by deriving it from the PQ master seed via HKDF.
func RetrieveSigningPrivateKey() (*mldsa65.PrivateKey, error) {
	seed, err := retrieveMasterSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	_, sk, _, err := DerivePQKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving signing key from master seed: %w", err)
	}
	return sk, nil
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

// RetrieveDecapsulationKey loads the ML-KEM-768 decapsulation key from ~/.ssh-sync/keypair
// by deriving it from the PQ master seed via HKDF.
func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
	seed, err := retrieveMasterSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	_, _, dk, err := DerivePQKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving decapsulation key from master seed: %w", err)
	}
	return dk, nil
}

// RetrieveEncapsulationKey derives the ML-KEM-768 encapsulation key from the local keypair
// by deriving it from the PQ master seed via HKDF.
func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	seed, err := retrieveMasterSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	_, _, dk, err := DerivePQKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving encapsulation key from master seed: %w", err)
	}
	return dk.EncapsulationKey(), nil
}

// BuildFullPublicKeyPEM returns PEM-encoded bytes containing both ML-DSA-65
// and ML-KEM-768 public keys. This is used for the WebSocket PublicKeyDto
// during existing account setup — the server relays both keys to Machine A
// (which needs ML-KEM to encrypt the master key), but only stores ML-DSA.
func BuildFullPublicKeyPEM() ([]byte, error) {
	seed, err := retrieveMasterSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	sigPub, _, dk, err := DerivePQKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving keys for public key PEM: %w", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "MLDSA65 PUBLIC KEY", Bytes: sigPub.Bytes()}); err != nil {
		return nil, err
	}
	if err := pem.Encode(&buf, &pem.Block{Type: "MLKEM768 ENCAPSULATION KEY", Bytes: dk.EncapsulationKey().Bytes()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
