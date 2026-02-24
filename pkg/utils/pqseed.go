package utils

import (
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"filippo.io/mldsa"
	"golang.org/x/crypto/hkdf"
)

// MasterSeedSize is the size in bytes of the PQ master seed.
const MasterSeedSize = 64

// DeriveMLDSAKey deterministically derives an MLDSA keypair from a single master seed
func DeriveMLDSAKey(masterSeed []byte) (*mldsa.PrivateKey, error) {
	if len(masterSeed) != MasterSeedSize {
		return nil, fmt.Errorf("master seed must be %d bytes, got %d", MasterSeedSize, len(masterSeed))
	}

	// Derive ML-DSA-65 seed (32 bytes)
	dsaReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-mldsa-v1"))
	dsaSeed := make([]byte, mldsa.PrivateKeySize)
	if _, err := io.ReadFull(dsaReader, dsaSeed); err != nil {
		return nil, fmt.Errorf("deriving ML-DSA seed: %w", err)
	}
	sk, err := mldsa.NewPrivateKey(mldsa.MLDSA65(), dsaSeed)
	if err != nil {
		return nil, fmt.Errorf("creating ML-DSA private key: %w", err)
	}

	return sk, nil
}

// DeriveMLKEMKey deterministically derives an MLKEM keypair from a single master seed
func DeriveMLKEMKey(masterSeed []byte) (*mlkem.DecapsulationKey768, error) {
	if len(masterSeed) != MasterSeedSize {
		return nil, fmt.Errorf("master seed must be %d bytes, got %d", MasterSeedSize, len(masterSeed))
	}

	// Derive ML-KEM-768 seed (64 bytes)
	kemReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-mlkem768-v1"))
	kemSeed := make([]byte, mlkem.SeedSize)
	if _, err := io.ReadFull(kemReader, kemSeed); err != nil {
		return nil, fmt.Errorf("deriving ML-KEM-768 seed: %w", err)
	}
	dk, err := mlkem.NewDecapsulationKey768(kemSeed)
	if err != nil {
		return nil, fmt.Errorf("creating ML-KEM-768 key from derived seed: %w", err)
	}

	return dk, nil
}
