package core

import (
	"testing"
)

// TestCompletedPartsToBitfield tests bitfield creation from map
func TestCompletedPartsToBitfield(t *testing.T) {
	tests := []struct {
		name           string
		completedParts map[int]bool
		numParts       int
		expectedBits   []int // Which bits should be set
	}{
		{
			name:           "empty map",
			completedParts: map[int]bool{},
			numParts:       10,
			expectedBits:   []int{},
		},
		{
			name:           "all complete",
			completedParts: map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true},
			numParts:       8,
			expectedBits:   []int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			name:           "sparse completion",
			completedParts: map[int]bool{0: true, 5: true, 10: true},
			numParts:       16,
			expectedBits:   []int{0, 5, 10},
		},
		{
			name:           "large number of parts",
			completedParts: map[int]bool{0: true, 99: true, 999: true},
			numParts:       1000,
			expectedBits:   []int{0, 99, 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bitfield := CompletedPartsToBitfield(tt.completedParts, tt.numParts)

			// Verify length
			expectedBytes := (tt.numParts + 7) / 8
			if len(bitfield) != expectedBytes {
				t.Errorf("Expected %d bytes, got %d", expectedBytes, len(bitfield))
			}

			// Verify bits are set correctly
			for _, bit := range tt.expectedBits {
				byteIdx := bit / 8
				bitIdx := uint(bit % 8)
				if (bitfield[byteIdx] & (1 << bitIdx)) == 0 {
					t.Errorf("Expected bit %d to be set", bit)
				}
			}
		})
	}
}

// TestBitfieldToCompletedParts tests bitfield decoding
func TestBitfieldToCompletedParts(t *testing.T) {
	// Create a bitfield with some bits set
	// Byte 0: bits 0, 2, 5 set = 0b00100101 = 37
	// Byte 1: bit 8 (idx 0 of byte 1) set = 0b00000001 = 1
	bitfield := []byte{37, 1}
	numParts := 16

	result := BitfieldToCompletedParts(bitfield, numParts)

	expected := map[int]bool{0: true, 2: true, 5: true, 8: true}
	for id := range expected {
		if !result[id] {
			t.Errorf("Expected part %d to be marked complete", id)
		}
	}

	// Verify no extra parts
	if len(result) != len(expected) {
		t.Errorf("Expected %d parts, got %d", len(expected), len(result))
	}
}

// TestBitfieldRoundTrip tests conversion both ways
func TestBitfieldRoundTrip(t *testing.T) {
	numParts := 50000 // Large number for the spec

	// Create random-ish completion map
	original := make(map[int]bool)
	for i := 0; i < numParts; i += 3 {
		original[i] = true
	}
	for i := 1; i < numParts; i += 7 {
		original[i] = true
	}

	// Convert to bitfield
	bitfield := CompletedPartsToBitfield(original, numParts)

	// Convert back
	result := BitfieldToCompletedParts(bitfield, numParts)

	// Verify equality
	if len(result) != len(original) {
		t.Errorf("Length mismatch: original %d, result %d", len(original), len(result))
	}

	for id := range original {
		if !result[id] {
			t.Errorf("Part %d missing from result", id)
		}
	}
}

// TestCountCompletedParts tests bit counting
func TestCountCompletedParts(t *testing.T) {
	tests := []struct {
		name     string
		bitfield []byte
		expected int
	}{
		{"empty", []byte{}, 0},
		{"all zeros", []byte{0, 0, 0, 0}, 0},
		{"all ones (1 byte)", []byte{255}, 8},
		{"mixed", []byte{37, 1}, 4},            // 37 = 0b00100101 (3 bits), 1 = 0b00000001 (1 bit) = 4 bits
		{"alternating", []byte{0xAA, 0x55}, 8}, // 0xAA = 10101010 (4 bits), 0x55 = 01010101 (4 bits)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := CountCompletedParts(tt.bitfield)
			if count != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, count)
			}
		})
	}
}

// TestBitfieldStorageSize verifies space savings
func TestBitfieldStorageSize(t *testing.T) {
	numParts := 50000

	// Full map would be at least 50000 * (int + bool) = at least 450KB with overhead
	// Bitfield should be ceil(50000/8) = 6250 bytes

	completedParts := make(map[int]bool)
	for i := 0; i < numParts; i++ {
		completedParts[i] = true
	}

	bitfield := CompletedPartsToBitfield(completedParts, numParts)

	expectedBytes := 6250
	if len(bitfield) != expectedBytes {
		t.Errorf("Expected %d bytes, got %d", expectedBytes, len(bitfield))
	}

	// Verify all bits are set
	count := CountCompletedParts(bitfield)
	if count != numParts {
		t.Errorf("Expected %d completed parts, got %d", numParts, count)
	}
}

// TestZeroNumParts tests edge case
func TestZeroNumParts(t *testing.T) {
	bitfield := CompletedPartsToBitfield(map[int]bool{0: true}, 0)
	if bitfield != nil {
		t.Error("Expected nil for zero numParts")
	}

	result := BitfieldToCompletedParts([]byte{255}, 0)
	if len(result) != 0 {
		t.Error("Expected empty map for zero numParts")
	}
}
