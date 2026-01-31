// Package integrity provides file verification and hash calculation
package integrity

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// FileVerifier handles file integrity checks
type FileVerifier struct{}

func NewFileVerifier() *FileVerifier {
	return &FileVerifier{}
}

// Verify checks if the file at path matches the expected hash
func (v *FileVerifier) Verify(path string, algo string, expected string) error {
	actual, err := CalculateHash(path, algo)
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// CalculateHash computes the hash of a file
// algorithm should be "sha256" or "md5"
func CalculateHash(filePath string, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var hash string
	if algorithm == "sha256" {
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(hasher.Sum(nil))
	} else if algorithm == "md5" {
		hasher := md5.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return "", err
		}
		hash = hex.EncodeToString(hasher.Sum(nil))
	} else {
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	return hash, nil
}
