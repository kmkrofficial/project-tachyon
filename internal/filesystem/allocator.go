package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v3/disk"
)

// Allocator handles file pre-allocation and disk space checks
type Allocator struct{}

func NewAllocator() *Allocator {
	return &Allocator{}
}

// AllocateFile reserves disk space for the download
func (a *Allocator) AllocateFile(path string, size int64) error {
	// 1. Check Disk Space
	if err := a.checkDiskSpace(path, size); err != nil {
		return err
	}

	// 2. Truncate (Pre-allocate)
	// Truncate ensures the OS reserves the blocks (sparse on some, allocated on others)
	// It prevents fragmentation and ensures we don't fail late.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("failed to open file for allocation: %w", err)
	}
	defer f.Close()

	if err := f.Truncate(size); err != nil {
		return fmt.Errorf("failed to pre-allocate space: %w", err)
	}

	return nil
}

func (a *Allocator) checkDiskSpace(path string, required int64) error {
	dir := filepath.Dir(path)

	// Get volume usage
	usage, err := disk.Usage(dir)
	if err != nil {
		// Fallback: If path doesn't exist yet, we might check volume of root?
		// But disk.Usage works on directories.
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Add a buffer of 100MB for system stability
	const buffer = 100 * 1024 * 1024

	if int64(usage.Free) < (required + buffer) {
		return fmt.Errorf("disk full: required %d bytes, available %d bytes", required, usage.Free)
	}

	return nil
}
