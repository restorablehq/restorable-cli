package signing

import (
	"crypto/ed25519"
	"crypto/rand"
)

// GenerateSigningKeyPair creates a new Ed25519 key pair for signing reports.
func GenerateSigningKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
