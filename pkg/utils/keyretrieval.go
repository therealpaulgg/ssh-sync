package utils

import (
	"bytes"
	"crypto/ecdh"
	"crypto/mlkem"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

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

// --- Hybrid key retrieval (ECDH P-256 + ML-KEM-768) ---

// retrieveHybridSeed reads the hybrid master seed from ~/.ssh-sync/keypair.
// Returns nil if the keypair file doesn't contain a hybrid seed PEM block.
func retrieveHybridSeed() ([]byte, error) {
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
		if block.Type == "SSHSYNC HYBRID SEED" {
			return block.Bytes, nil
		}
	}
	return nil, nil
}

// RetrieveHybridECKey loads the EC P-256 ECDH private key from the hybrid seed.
func RetrieveHybridECKey() (*ecdh.PrivateKey, error) {
	seed, err := retrieveHybridSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("hybrid seed not found in keypair file")
	}
	ecPriv, _, err := DeriveHybridKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving EC key from hybrid seed: %w", err)
	}
	return ecPriv, nil
}

// RetrieveDecapsulationKey loads the ML-KEM-768 decapsulation key from the hybrid seed.
func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
	seed, err := retrieveHybridSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("hybrid seed not found in keypair file")
	}
	_, dk, err := DeriveHybridKeys(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving decapsulation key from hybrid seed: %w", err)
	}
	return dk, nil
}

// RetrieveEncapsulationKey derives the ML-KEM-768 encapsulation key from the hybrid seed.
func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	dk, err := RetrieveDecapsulationKey()
	if err != nil {
		return nil, err
	}
	return dk.EncapsulationKey(), nil
}

// BuildHybridPublicKeys returns the EC P-256 public key and ML-KEM-768
// encapsulation key as separate PEM-encoded byte slices. The caller sends them
// in distinct DTO fields so the server can store the EC key (for JWT auth)
// and relay the encapsulation key independently (for hybrid KEM).
func BuildHybridPublicKeys() (ecPubPEM []byte, ekPEM []byte, err error) {
	seed, err := retrieveHybridSeed()
	if err != nil {
		return nil, nil, err
	}
	if seed == nil {
		return nil, nil, fmt.Errorf("hybrid seed not found in keypair file")
	}
	ecPriv, dk, err := DeriveHybridKeys(seed)
	if err != nil {
		return nil, nil, fmt.Errorf("deriving keys for public key PEM: %w", err)
	}

	// EC public key as PKIX "PUBLIC KEY" PEM
	ecPubBytes, err := x509.MarshalPKIXPublicKey(ecPriv.Public().(*ecdh.PublicKey))
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling EC public key: %w", err)
	}
	var ecBuf bytes.Buffer
	if err := pem.Encode(&ecBuf, &pem.Block{Type: "PUBLIC KEY", Bytes: ecPubBytes}); err != nil {
		return nil, nil, err
	}

	// ML-KEM encapsulation key PEM
	var ekBuf bytes.Buffer
	if err := pem.Encode(&ekBuf, &pem.Block{Type: "MLKEM768 ENCAPSULATION KEY", Bytes: dk.EncapsulationKey().Bytes()}); err != nil {
		return nil, nil, err
	}
	return ecBuf.Bytes(), ekBuf.Bytes(), nil
}

// --- Format-aware master key retrieval ---

// RetrieveMasterKey reads and decrypts the master key from ~/.ssh-sync/master_key.
// It auto-detects the key format:
//   - Legacy: JWE encrypted with ECDH-ES+A256KW
//   - Hybrid: ECDH P-256 + ML-KEM-768 hybrid KEM + AES-256-GCM
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
	case FormatHybrid:
		ecPriv, err := RetrieveHybridECKey()
		if err != nil {
			return nil, err
		}
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := DecryptHybrid(file, ecPriv, dk)
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (hybrid): %w", err)
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
