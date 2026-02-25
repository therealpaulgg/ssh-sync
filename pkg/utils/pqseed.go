package utils

import (
	"crypto/mlkem"
	"fmt"

	"filippo.io/mldsa"
)

// MasterSeedSize is the size in bytes of the PQ master seed.
const MasterSeedSize = 64

// DeriveMLDSAKey deterministically derives an MLDSA keypair from a single master seed
func DeriveMLDSAKey(masterSeed []byte) (*mldsa.PrivateKey, error) {
	if len(masterSeed) != MasterSeedSize {
		return nil, fmt.Errorf("master seed must be %d bytes, got %d", MasterSeedSize, len(masterSeed))
	}

	sk, err := mldsa.NewPrivateKey(mldsa.MLDSA65(), masterSeed[:mldsa.PrivateKeySize])
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

	dk, err := mlkem.NewDecapsulationKey768(masterSeed)
	if err != nil {
		return nil, fmt.Errorf("creating ML-KEM-768 key from seed: %w", err)
	}

	return dk, nil
}
