# Tachyon Download Manager ⚡

High-performance, persistent download manager built with Go (Wails) and React.

## Features
- **🚀 32-Thread Engine**: Maximizes bandwidth using parallel connections.
- **🧠 Persistent "Brain"**: Remembers downloads and progress across restarts (BadgerDB).
- **🔌 Browser Integration**: Chrome Extension intercepts downloads automatically.
- **🎨 Modern UI**: Clean React dashboard with Dark Mode.
- **📁 Smart Management**: Open destination folders instantly.

## Installation
1.  Download `Tachyon-Installer.exe` from Releases.
2.  Run the installer.
3.  Launch "Tachyon Download Manager".

## Browser Extension
1.  Go to `chrome://extensions` -> Enable Developer Mode.
2.  Click "Load Unpacked" -> Select the `extension/` folder (or unzip `tachyon_extension.zip`).
3.  Right-click any link -> "Download with Tachyon".

## Development
```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Run Dev Mode
wails dev

# Build for Production
wails build -nsis
```

## Architecture
- **Frontend**: React + Vite + Tailwind + Lucide
- **Backend**: Go + Grab (Engine) + BadgerDB (Storage) + API Server
- **Protocol**: HTTP/1.1 (Standard) + HTTP/2 (Supported by Go)

---
© 2026 Keerthi Raajan K M
