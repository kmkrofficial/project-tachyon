// Package main implements a unified build system for Project Tachyon.
// Usage: go run cmd/builder/main.go [build|release|docker|check]
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appName    = "Tachyon"
	appVersion = "1.0.0" // TODO: Read from version file
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "check":
		runCheck()
	case "build":
		runBuild()
	case "release":
		runRelease()
	case "docker":
		runDocker()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
Tachyon Build System
=====================

Usage: go run cmd/builder/main.go <command>

Commands:
  check     Verify all required tools are installed
  build     Build for current platform
  release   Build release packages for all platforms
  docker    Build Docker image for server mode
  help      Show this help message

Examples:
  go run cmd/builder/main.go check
  go run cmd/builder/main.go build
  go run cmd/builder/main.go release
`)
}

// runCheck verifies all required tools are installed
func runCheck() {
	fmt.Println("🔍 Checking required tools...")

	tools := []struct {
		name  string
		check string
		args  []string
	}{
		{"go", "go", []string{"version"}},
		{"wails", "wails", []string{"version"}},
		{"node", "node", []string{"--version"}},
		{"npm", "npm", []string{"--version"}},
	}

	allFound := true
	for _, tool := range tools {
		cmd := exec.Command(tool.check, tool.args...)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("❌ CRITICAL: %s is missing or not in PATH\n", tool.name)
			allFound = false
		} else {
			version := strings.TrimSpace(string(output))
			if len(version) > 50 {
				version = version[:50] + "..."
			}
			fmt.Printf("✅ %s: %s\n", tool.name, version)
		}
	}

	if !allFound {
		fmt.Println("\n⚠️  Some required tools are missing. Please install them and try again.")
		os.Exit(1)
	}

	fmt.Println("\n✅ All tools verified!")
}

// runBuild builds for the current platform
func runBuild() {
	runCheck()

	fmt.Printf("\n🔨 Building for %s/%s...\n", runtime.GOOS, runtime.GOARCH)

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	args := []string{"build", "-platform", platform}

	// Add NSIS for Windows installer
	if runtime.GOOS == "windows" {
		args = append(args, "-nsis")
	}

	cmd := exec.Command("wails", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ Build completed successfully!")
	printBuildArtifacts()
}

// runRelease builds release packages for all platforms
func runRelease() {
	runCheck()

	fmt.Println("\n📦 Building release packages...")

	// Define platforms to build
	platforms := []struct {
		goos   string
		goarch string
		nsis   bool
	}{
		{"windows", "amd64", true},
		{"darwin", "universal", false},
		{"linux", "amd64", false},
	}

	buildDir := "build/release"
	os.MkdirAll(buildDir, 0755)

	for _, p := range platforms {
		if runtime.GOOS != p.goos && p.goos != "darwin" {
			fmt.Printf("⚠️  Skipping %s/%s (cross-compile not supported for GUI apps)\n", p.goos, p.goarch)
			continue
		}

		fmt.Printf("\n🔨 Building for %s/%s...\n", p.goos, p.goarch)

		args := []string{"build", "-platform", fmt.Sprintf("%s/%s", p.goos, p.goarch)}

		if p.nsis {
			args = append(args, "-nsis")
		}

		cmd := exec.Command("wails", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Build failed for %s/%s: %v\n", p.goos, p.goarch, err)
			continue
		}

		// Post-process: rename and package
		if err := postProcessBuild(p.goos, p.goarch, buildDir); err != nil {
			fmt.Printf("⚠️  Post-processing failed: %v\n", err)
		}
	}

	fmt.Println("\n✅ Release build completed!")
	fmt.Printf("📁 Artifacts in: %s\n", buildDir)
}

// postProcessBuild renames and packages build artifacts
func postProcessBuild(goos, goarch, buildDir string) error {
	wailsBuildDir := "build/bin"

	switch goos {
	case "windows":
		// Look for NSIS installer
		installerPattern := filepath.Join(wailsBuildDir, "*-amd64-installer.exe")
		matches, _ := filepath.Glob(installerPattern)
		if len(matches) > 0 {
			dest := filepath.Join(buildDir, fmt.Sprintf("%s-Setup-v%s.exe", appName, appVersion))
			return copyFile(matches[0], dest)
		}
		// Fallback to regular exe
		exePattern := filepath.Join(wailsBuildDir, "*.exe")
		matches, _ = filepath.Glob(exePattern)
		if len(matches) > 0 {
			dest := filepath.Join(buildDir, fmt.Sprintf("%s-v%s-windows-amd64.exe", appName, appVersion))
			return copyFile(matches[0], dest)
		}

	case "darwin":
		// Zip the .app bundle
		appPattern := filepath.Join(wailsBuildDir, "*.app")
		matches, _ := filepath.Glob(appPattern)
		if len(matches) > 0 {
			zipDest := filepath.Join(buildDir, fmt.Sprintf("%s-v%s-macos-universal.zip", appName, appVersion))
			return zipDirectory(matches[0], zipDest)
		}

	case "linux":
		// Copy binary
		binPattern := filepath.Join(wailsBuildDir, appName)
		if _, err := os.Stat(binPattern); err == nil {
			dest := filepath.Join(buildDir, fmt.Sprintf("%s-v%s-linux-amd64", appName, appVersion))
			return copyFile(binPattern, dest)
		}
	}

	return nil
}

// runDocker builds the Docker image
func runDocker() {
	fmt.Println("🐳 Building Docker image...")

	// Check if Dockerfile exists
	if _, err := os.Stat("Dockerfile"); os.IsNotExist(err) {
		fmt.Println("❌ Dockerfile not found in project root")
		fmt.Println("   Create a Dockerfile for server mode first.")
		os.Exit(1)
	}

	imageName := fmt.Sprintf("tachyon-server:v%s", appVersion)

	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Docker build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Docker image built: %s\n", imageName)
}

// printBuildArtifacts lists files in build/bin
func printBuildArtifacts() {
	fmt.Println("\n📁 Build artifacts:")
	filepath.Walk("build/bin", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size := float64(info.Size()) / (1024 * 1024)
			fmt.Printf("   %s (%.1f MB)\n", path, size)
		}
		return nil
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// zipDirectory creates a zip archive of a directory
func zipDirectory(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(filepath.Dir(source), path)
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
