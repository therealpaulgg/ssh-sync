package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// GetToken generates a signed JWT for authenticating with the server.
// Auto-detects key format:
//   - Legacy EC: ES512 signed via lestrrat-go/jwx
//   - Post-quantum: ML-DSA-65 signed with custom JWT implementation
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

// getTokenPQ generates a JWT signed with ML-DSA-65 (post-quantum).
func getTokenPQ() (string, error) {
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
