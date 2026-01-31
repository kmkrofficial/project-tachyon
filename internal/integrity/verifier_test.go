package integrity

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestCalculateHash_SHA256(t *testing.T) {
	// Create dummy file
	content := []byte("hello world")
	tmpFile, _ := os.CreateTemp("", "hash_test")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(content)
	tmpFile.Close()

	// Calc expected
	expected := sha256.Sum256(content)
	expectedStr := hex.EncodeToString(expected[:])

	actual, err := CalculateHash(tmpFile.Name(), "sha256")
	if err != nil {
		t.Fatalf("CalculateHash failed: %v", err)
	}

	if actual != expectedStr {
		t.Errorf("Expected %s, got %s", expectedStr, actual)
	}
}

func TestCalculateHash_MD5(t *testing.T) {
	content := []byte("hello world")
	tmpFile, _ := os.CreateTemp("", "hash_test")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(content)
	tmpFile.Close()

	expected := md5.Sum(content)
	expectedStr := hex.EncodeToString(expected[:])

	actual, err := CalculateHash(tmpFile.Name(), "md5")
	if err != nil {
		t.Fatalf("CalculateHash failed: %v", err)
	}

	if actual != expectedStr {
		t.Errorf("Expected %s, got %s", expectedStr, actual)
	}
}

func TestVerifier_MismatchDetection(t *testing.T) {
	content := []byte("hello world")
	tmpFile, _ := os.CreateTemp("", "hash_test")
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(content)
	tmpFile.Close()

	v := NewFileVerifier()

	// Wrong hash
	err := v.Verify(tmpFile.Name(), "md5", "wronghash")
	if err == nil {
		t.Error("Expected error for mismatching hash, got nil")
	}
}
