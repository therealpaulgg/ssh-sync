package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// GetToken generates a signed JWT for authenticating with the server.
// Auto-detects key format:
//   - Legacy EC: ES512 signed via lestrrat-go/jwx (existing server compat)
//   - Hybrid: ES256 signed via lestrrat-go/jwx (EC P-256 derived from hybrid seed)
func GetToken() (string, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return "", err
	}

	switch format {
	case FormatHybrid:
		return getTokenHybrid()
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

// getTokenHybrid generates a JWT signed with ES256 (ECDSA P-256).
// The EC key is derived from the hybrid master seed.
func getTokenHybrid() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	ecdhKey, err := RetrieveHybridECKey()
	if err != nil {
		return "", err
	}

	// Convert ecdh.PrivateKey → ecdsa.PrivateKey (same P-256 scalar)
	ecdsaKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
		},
		D: new(big.Int).SetBytes(ecdhKey.Bytes()),
	}
	ecdsaKey.PublicKey.X, ecdsaKey.PublicKey.Y = elliptic.P256().ScalarBaseMult(ecdhKey.Bytes())

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
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES256, ecdsaKey))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}
