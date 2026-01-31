# Project Tachyon Architecture

This document describes the high-level architecture of Project Tachyon, a modern Wails-based download manager.

## Project Structure

```
project-tachyon/
├── cmd/
│   └── builder/       # Unified build system
├── frontend/          # React + TypeScript UI
│   └── src/
│       ├── components/
│       ├── pages/
│       └── hooks/
├── internal/          # Go backend packages
│   ├── app/           # Wails bridge (frontend ↔ backend)
│   ├── engine/        # Download engine core
│   ├── storage/       # SQLite persistence via GORM
│   ├── queue/         # Priority queue & scheduler
│   ├── network/       # HTTP client & congestion control
│   ├── filesystem/    # File allocation & disk checks
│   ├── integrity/     # Hash verification (SHA256/MD5)
│   ├── analytics/     # Stats & disk usage tracking
│   ├── security/      # AV scanning & audit logging
│   ├── config/        # Configuration management
│   ├── logger/        # Structured logging
│   └── speedtest/     # Connection speed testing
└── docs/              # Documentation
```

## Package Responsibilities

### `internal/app` (Wails Bridge)
Connects React frontend to Go backend. Split by domain:
- `app.go` - Core app lifecycle
- `bridge_downloads.go` - Download operations
- `bridge_events.go` - Event emission (security, network)
- `bridge_security.go` - Security settings
- `bridge_settings.go` - General settings

### `internal/engine` (Download Engine)
The heart of Tachyon. Split into focused files:
- `manager.go` - TachyonEngine struct & initialization
- `executor.go` - Queue worker & parallel chunk downloads
- `downloads.go` - Start/pause/resume/cancel operations
- `limiter.go` - Bandwidth throttling
- `probing.go` - URL inspection (size, range support)

### `internal/network`
- `client.go` - Configured HTTP client with timeouts
- `congestion.go` - AIMD congestion control per host

### `internal/queue`
- `queue.go` - Priority queue with condition-based waiting
- `scheduler.go` - Smart task dispatch with cooldowns

### `internal/storage`
- `models.go` - GORM models (DownloadTask, DailyStat)
- `db.go` - SQLite operations & migrations

## Data Flow

```
┌─────────────────┐     WebSocket Events     ┌─────────────────┐
│  React Frontend │ ◄────────────────────── │    Wails App    │
│                 │ ─────────────────────► │   (bridge/*.go)  │
└─────────────────┘     Wails Bindings       └─────────────────┘
                                                      │
                                                      ▼
                                            ┌─────────────────┐
                                            │  TachyonEngine  │
                                            │   (internal/    │
                                            │    engine/)     │
                                            └─────────────────┘
                                                      │
                    ┌─────────────────┬───────────────┼───────────────┬─────────────────┐
                    ▼                 ▼               ▼               ▼                 ▼
           ┌─────────────┐   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
           │   Network   │   │    Queue    │  │   Storage   │  │  Security   │  │  Analytics  │
           │  (HTTP +    │   │ (Priority   │  │  (SQLite)   │  │ (AV +       │  │  (Stats)    │
           │ Congestion) │   │  Scheduler) │  │             │  │  Audit)     │  │             │
           └─────────────┘   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
```

## Key Technologies

| Layer | Technology |
|-------|------------|
| Desktop Framework | Wails v2 |
| Frontend | React 18, TypeScript, TailwindCSS |
| Backend | Go 1.22+ |
| Database | SQLite (via glebarez/sqlite + GORM) |
| HTTP Client | Standard library + configurable client |
| Logging | log/slog (structured) |

## Building

```bash
# Check dependencies
go run cmd/builder/main.go check

# Development build
go run cmd/builder/main.go build

# Release packages
go run cmd/builder/main.go release
```

## Event System

The backend emits events to the frontend via Wails runtime:
- `download:update` - Progress updates
- `security:scan_result` - Antivirus scan results
- `network:congestion_level` - Network health status
- `download:completed` - Download finished
- `download:error` - Download error
