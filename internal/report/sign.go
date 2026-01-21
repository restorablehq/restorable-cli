package report

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

// Sign signs the report using Ed25519 and stores the signature in the report.
func Sign(report *Report, privateKey ed25519.PrivateKey) error {
	// Clear existing signature before signing
	report.Signature = ""

	// Serialize report for signing
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report for signing: %w", err)
	}

	// Sign the data
	signature := ed25519.Sign(privateKey, data)

	// Store base64-encoded signature
	report.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// Verify verifies the report signature using the public key.
func Verify(report *Report, publicKey ed25519.PublicKey) (bool, error) {
	if report.Signature == "" {
		return false, fmt.Errorf("report has no signature")
	}

	// Decode the signature
	signature, err := base64.StdEncoding.DecodeString(report.Signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Create a copy without the signature for verification
	reportCopy := *report
	reportCopy.Signature = ""

	data, err := json.Marshal(&reportCopy)
	if err != nil {
		return false, fmt.Errorf("failed to marshal report for verification: %w", err)
	}

	// Verify the signature
	return ed25519.Verify(publicKey, data, signature), nil
}

// LoadPrivateKey loads an Ed25519 private key from a file.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// The key file contains the raw 64-byte private key
	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d bytes, got %d", ed25519.PrivateKeySize, len(data))
	}

	return ed25519.PrivateKey(data), nil
}

// LoadPublicKey loads an Ed25519 public key from a file.
func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	// The key file contains the raw 32-byte public key
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d bytes, got %d", ed25519.PublicKeySize, len(data))
	}

	return ed25519.PublicKey(data), nil
}
