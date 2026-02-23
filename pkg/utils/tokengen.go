package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"filippo.io/mldsa"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// GetToken generates a signed JWT for authenticating with the server.
// Auto-detects key format:
//   - Legacy EC: ES512 signed via lestrrat-go/jwx (existing server compat)
//   - Post-quantum: ML-DSA-65 signed JWT (custom signing)
func GetToken() (string, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return "", err
	}

	switch format {
	case FormatPostQuantum:
		return getTokenPQ()
	default:
		return getTokenLegacy()
	}
}

// getTokenLegacy generates a JWT signed with ES512 (ECDSA + SHA-512).
func getTokenLegacy() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	key, err := RetrievePrivateKey()
	if err != nil {
		return "", err
	}
	builder := jwt.NewBuilder()
	builder.Issuer("github.com/therealpaulgg/ssh-sync")
	builder.IssuedAt(time.Now().Add(-1 * time.Minute))
	builder.Expiration(time.Now().Add(2 * time.Minute))
	builder.Claim("username", profile.Username)
	builder.Claim("machine", profile.MachineName)
	tok, err := builder.Build()
	if err != nil {
		return "", err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES512, key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

// getTokenPQ generates a JWT signed with ML-DSA-65 (FIPS 204).
// Uses a custom "MLDSA65" algorithm header since JWS doesn't have a
// standard algorithm identifier for ML-DSA yet.
func getTokenPQ() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	sk, err := RetrieveSigningKey()
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

	// Encode header.payload
	b64Header := base64.RawURLEncoding.EncodeToString(headerJSON)
	b64Payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := b64Header + "." + b64Payload

	// Sign with ML-DSA-65
	sig, err := sk.Sign(rand.Reader, []byte(signingInput), nil)
	if err != nil {
		return "", fmt.Errorf("ML-DSA-65 signing: %w", err)
	}
	b64Sig := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + b64Sig, nil
}

// VerifyMLDSA65JWT verifies a JWT signed with ML-DSA-65.
// This is provided for completeness; the server performs verification.
func VerifyMLDSA65JWT(tokenStr string, pk *mldsa.PublicKey65) (bool, error) {
	// Split into header.payload.signature
	parts := splitJWT(tokenStr)
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid JWT format")
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("decoding signature: %w", err)
	}
	return mldsa.Verify65(pk, []byte(signingInput), sig), nil
}

func splitJWT(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
