package core

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

// FileVerifier handles streamed integrity checks
type FileVerifier struct{}

func NewFileVerifier() *FileVerifier {
	return &FileVerifier{}
}

// Verify checks the file hash against expected value in a streaming manner
// Support algo: "sha256", "md5"
func (v *FileVerifier) Verify(filePath string, algo string, expectedHash string) error {
	if expectedHash == "" {
		return nil // Nothing to verify
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer f.Close()

	var hasher hash.Hash
	switch algo {
	case "sha256", "": // Default
		hasher = sha256.New()
	case "md5":
		hasher = md5.New()
	default:
		return fmt.Errorf("unsupported hash algorithm: %s", algo)
	}

	// 4MB Buffer for optimal SSD throughput
	buf := make([]byte, 4*1024*1024)
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return fmt.Errorf("hashing failed: %w", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}
