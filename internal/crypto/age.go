package crypto

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
)

// AgeDecryptor handles age-encrypted backup decryption.
type AgeDecryptor struct {
	identities []age.Identity
}

// NewAgeDecryptor creates a decryptor from a private key file path.
func NewAgeDecryptor(privateKeyPath string) (*AgeDecryptor, error) {
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read age private key from %s: %w", privateKeyPath, err)
	}

	identities, err := age.ParseIdentities(bytes.NewReader(keyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identities: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found in %s", privateKeyPath)
	}

	return &AgeDecryptor{identities: identities}, nil
}

// NewAgeDecryptorFromEnv creates a decryptor using a private key from an environment variable.
func NewAgeDecryptorFromEnv(envVar string) (*AgeDecryptor, error) {
	keyData := os.Getenv(envVar)
	if keyData == "" {
		return nil, fmt.Errorf("age private key environment variable %s is not set", envVar)
	}

	identities, err := age.ParseIdentities(strings.NewReader(keyData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse age identities from env: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found in environment variable %s", envVar)
	}

	return &AgeDecryptor{identities: identities}, nil
}

// Decrypt wraps the reader with age decryption.
// The returned reader must be fully consumed and closed.
func (d *AgeDecryptor) Decrypt(r io.Reader) (io.Reader, error) {
	decrypted, err := age.Decrypt(r, d.identities...)
	if err != nil {
		return nil, fmt.Errorf("age decryption failed: %w", err)
	}
	return decrypted, nil
}

// DecryptReadCloser wraps a ReadCloser with decryption, preserving the Close method.
type DecryptReadCloser struct {
	decrypted io.Reader
	original  io.ReadCloser
}

// NewDecryptReadCloser creates a decrypting ReadCloser.
func (d *AgeDecryptor) NewDecryptReadCloser(rc io.ReadCloser) (*DecryptReadCloser, error) {
	decrypted, err := d.Decrypt(rc)
	if err != nil {
		return nil, err
	}
	return &DecryptReadCloser{
		decrypted: decrypted,
		original:  rc,
	}, nil
}

func (d *DecryptReadCloser) Read(p []byte) (n int, err error) {
	return d.decrypted.Read(p)
}

func (d *DecryptReadCloser) Close() error {
	return d.original.Close()
}
