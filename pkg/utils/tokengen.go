package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// GetToken generates a JWT signed with ML-DSA-65 (post-quantum digital signature).
// The token uses the "MLDSA65" algorithm identifier per the JOSE PQ draft standard.
func GetToken() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	sk, err := RetrieveSigningPrivateKey()
	if err != nil {
		return "", err
	}

	// Build JWT header
	header := map[string]string{
		"alg": "MLDSA65",
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT header: %w", err)
	}

	// Build JWT payload
	now := time.Now()
	payload := map[string]interface{}{
		"iss":      "github.com/therealpaulgg/ssh-sync",
		"iat":      now.Add(-1 * time.Minute).Unix(),
		"exp":      now.Add(2 * time.Minute).Unix(),
		"username": profile.Username,
		"machine":  profile.MachineName,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT payload: %w", err)
	}

	// Encode header and payload
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	// Sign with ML-DSA-65
	sig := make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(sk, []byte(signingInput), nil, false, sig); err != nil {
		return "", fmt.Errorf("ML-DSA-65 signing failed: %w", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}
