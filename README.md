# Tachyon Download Manager ⚡

<div align="center">

![Tachyon Logo](frontend/src/assets/images/logo-universal.png)

**A blazing-fast, enterprise-grade download manager built with Go + React**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Wails](https://img.shields.io/badge/Wails-v2.11-EE4E4E?style=flat)](https://wails.io)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)](https://react.dev)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)

[Features](#features) • [Installation](#installation) • [Development](#development) • [Building](#building) • [Comparison](#comparison)

</div>

---

## Why Tachyon?

Tachyon is a modern download manager that combines the performance of native desktop applications with the elegance of a web-based UI. Unlike typical download managers, Tachyon is built from the ground up with enterprise-grade features like AIMD congestion control, real-time antivirus scanning, and an MCP (Model Context Protocol) AI interface.

---

## Features

### 🚀 Core Engine
| Feature | Description |
|---------|-------------|
| **Multi-threaded Downloads** | Up to 32 parallel connections per file using HTTP Range requests |
| **Smart Chunk Allocation** | Dynamic chunk sizing based on file size and server capabilities |
| **Resume Support** | Seamlessly resume interrupted downloads - survives app crashes and restarts |
| **Queue with Priority** | Priority-based download queue with intelligent scheduling |
| **Bandwidth Throttling** | Global and per-host speed limiting |

### 🔒 Security
| Feature | Description |
|---------|-------------|
| **Windows Defender Integration** | Automatic file scanning on download completion |
| **ClamAV Support** | Optional ClamAV daemon integration for enterprise environments |
| **Audit Logging** | Complete security audit trail for all operations |
| **Checksum Verification** | SHA-256 and MD5 hash verification for downloaded files |

### 🌐 Network Intelligence
| Feature | Description |
|---------|-------------|
| **AIMD Congestion Control** | TCP-inspired additive increase/multiplicative decrease per host |
| **Connection Speed Testing** | Built-in speed test to measure actual throughput |
| **Smart Retry Logic** | Exponential backoff with jitter for failed requests |
| **Custom User-Agent** | Configurable headers to bypass download restrictions |

### 🎨 Modern UI
| Feature | Description |
|---------|-------------|
| **Dark Mode Interface** | Premium glassmorphic design with smooth animations |
| **Real-time Progress** | Live speed, ETA, and progress updates via WebSocket events |
| **Drag & Drop Reordering** | Reorder queue items with intuitive drag-and-drop |
| **Context Menus** | Right-click actions for quick operations |
| **Analytics Dashboard** | Historical download stats, disk usage, and daily trends |
| **Network Health Indicator** | Visual congestion level (green/yellow/red) in header |

### 🤖 AI Interface (MCP Server)
| Feature | Description |
|---------|-------------|
| **Model Context Protocol** | RESTful API for AI assistants to control downloads |
| **Token Authentication** | Secure API access with configurable tokens |
| **Remote Control** | Add/pause/resume/cancel downloads programmatically |

### 💾 Persistence
| Feature | Description |
|---------|-------------|
| **SQLite Database** | Reliable storage with GORM ORM (no CGO dependencies) |
| **Checkpoint System** | Progress saved every few seconds for crash recovery |
| **Download History** | Complete history with search and filtering |

### 🔌 Browser Integration
| Feature | Description |
|---------|-------------|
| **Chrome Extension** | Intercept browser downloads automatically |
| **Context Menu** | Right-click "Download with Tachyon" on any link |
| **Link Detection** | Smart detection of downloadable content |

---

## Comparison with Alternatives

### vs Commercial Applications

| Feature | Tachyon | IDM | JDownloader | Motrix |
|---------|:-------:|:---:|:-----------:|:------:|
| **Free & Open Source** | ✅ | ❌ ($30) | ✅ | ✅ |
| **Cross-Platform** | ✅ | ❌ (Windows) | ✅ | ✅ |
| **Multi-threaded** | ✅ (32) | ✅ (32) | ✅ (varies) | ✅ (16) |
| **Modern UI** | ✅ React | ❌ Win32 | ❌ Java Swing | ✅ Electron |
| **Native Performance** | ✅ Go | ✅ C++ | ❌ JVM | ❌ Electron |
| **AV Integration** | ✅ | ✅ | ❌ | ❌ |
| **AI Interface** | ✅ MCP | ❌ | ❌ | ❌ |
| **Congestion Control** | ✅ AIMD | ❌ | ❌ | ❌ |
| **Memory Footprint** | ~50 MB | ~30 MB | ~300 MB | ~200 MB |

### vs Open Source Alternatives

| Feature | Tachyon | aria2 | wget | youtube-dl |
|---------|:-------:|:-----:|:----:|:----------:|
| **GUI** | ✅ Native | ❌ CLI | ❌ CLI | ❌ CLI |
| **Multi-threaded** | ✅ | ✅ | ❌ | Limited |
| **Resume Support** | ✅ | ✅ | ✅ | ✅ |
| **Real-time Dashboard** | ✅ | ❌ | ❌ | ❌ |
| **Queue Management** | ✅ | ✅ | ❌ | Limited |
| **Security Scanning** | ✅ | ❌ | ❌ | ❌ |
| **Browser Extension** | ✅ | Requires frontend | ❌ | ❌ |

### Key Advantages

1. **No Java/Electron bloat** - Native Go binary with React UI via Wails
2. **Enterprise Security** - Built-in AV scanning and audit logging
3. **AI-Ready** - MCP interface for automation and AI assistants
4. **Modern Architecture** - Clean package structure, easily extensible

---

## Requirements

### Runtime Requirements
- **OS**: Windows 10/11, macOS 10.15+, Linux (Ubuntu 20.04+)
- **Memory**: 50 MB minimum, 100 MB recommended
- **Disk**: 10 MB for application + space for downloads

### Development Requirements
- **Go**: 1.22 or higher
- **Node.js**: 18.x or higher
- **npm**: 9.x or higher
- **Wails CLI**: v2.11+

#### Windows-specific
- NSIS (for installer generation)
- WebView2 Runtime (usually pre-installed on Windows 10/11)

#### macOS-specific
- Xcode Command Line Tools

#### Linux-specific
- `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`

---

## Installation

### Pre-built Binaries

1. Download the latest release from [Releases](https://github.com/yourrepo/tachyon/releases)
2. Run the installer:
   - **Windows**: `Tachyon-Setup-v1.0.0.exe`
   - **macOS**: Extract `Tachyon-v1.0.0-macos-universal.zip` and move to Applications
   - **Linux**: Make executable and run `./Tachyon-v1.0.0-linux-amd64`

### Browser Extension

1. Open Chrome → `chrome://extensions`
2. Enable "Developer Mode" (top right)
3. Click "Load Unpacked" → Select the `extension/` folder
4. Right-click any link → "Download with Tachyon"

---

## Development

### Quick Start

```bash
# Clone the repository
git clone https://github.com/yourrepo/tachyon.git
cd tachyon

# Install Wails CLI (if not installed)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode (with hot reload)
wails dev
```

### Project Structure

```
project-tachyon/
├── cmd/builder/        # Unified build system
├── frontend/           # React + TypeScript UI
│   ├── src/
│   │   ├── components/ # Reusable UI components
│   │   ├── pages/      # Settings pages
│   │   └── hooks/      # Custom React hooks
│   └── wailsjs/        # Auto-generated Wails bindings
├── internal/           # Go backend packages
│   ├── app/            # Wails bridge layer
│   ├── engine/         # Download engine core
│   ├── storage/        # SQLite persistence
│   ├── queue/          # Priority queue & scheduler
│   ├── network/        # HTTP client & congestion
│   ├── security/       # AV scanning & audit
│   ├── analytics/      # Stats tracking
│   └── ...
├── extension/          # Chrome extension
└── docs/               # Documentation
```

---

## Testing

### Run All Tests

```bash
# Run all Go tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
# Engine tests
go test -v ./internal/engine/...

# Storage tests
go test -v ./internal/storage/...

# Queue tests  
go test -v ./internal/queue/...
```

### Frontend Tests

```bash
cd frontend
npm test
```

---

## Building

### Using the Build System

Tachyon includes a unified Go-based build system:

```bash
# Check all required tools are installed
go run cmd/builder/main.go check

# Build for current platform
go run cmd/builder/main.go build

# Build release packages for all supported platforms
go run cmd/builder/main.go release

# Build Docker image (for server mode)
go run cmd/builder/main.go docker
```

### Manual Build

```bash
# Development build
wails build

# Production build with Windows installer
wails build -nsis

# Cross-compile (limited GUI support)
wails build -platform windows/amd64
wails build -platform darwin/universal
wails build -platform linux/amd64
```

### Build Output

Binaries are placed in `build/bin/`:
- Windows: `Tachyon.exe` and `Tachyon-amd64-installer.exe`
- macOS: `Tachyon.app`
- Linux: `Tachyon`

---

## Configuration

Configuration is stored at:
- **Windows**: `%APPDATA%\Tachyon\config.json`
- **macOS**: `~/Library/Application Support/Tachyon/config.json`
- **Linux**: `~/.config/Tachyon/config.json`

### Key Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `maxConcurrent` | 3 | Max simultaneous downloads |
| `globalLimit` | 0 | Global bandwidth limit (0 = unlimited) |
| `downloadPath` | Downloads folder | Default save location |
| `enableAI` | false | Enable MCP AI interface |
| `aiPort` | 8765 | MCP server port |

---

## API Reference

See [docs/api.md](docs/api.md) for the complete MCP API documentation.

---

## Security

See [docs/security.md](docs/security.md) for security architecture details.

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

MIT License - See [LICENSE](LICENSE) for details.

---

<div align="center">

**Built with ❤️ by Keerthi Raajan K M**

© 2026 All Rights Reserved

</div>
