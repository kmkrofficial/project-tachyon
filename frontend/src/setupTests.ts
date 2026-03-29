import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Provide matchMedia stub for jsdom (not available by default)
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        onchange: null,
        dispatchEvent: vi.fn(),
    })),
});

// lucide-react is aliased in vite.config.ts test.alias to avoid resolving 1671 sub-modules

// Mock Wails runtime
vi.mock('../wailsjs/runtime/runtime', () => ({
    EventsOn: vi.fn(() => () => {}),
    EventsOff: vi.fn(),
    EventsEmit: vi.fn(),
    BrowserOpenURL: vi.fn(),
}));

// Mock Wails Go bindings
vi.mock('../wailsjs/go/app/App', () => ({
    GetTasks: vi.fn().mockResolvedValue([]),
    GetDownloadLocations: vi.fn().mockResolvedValue([]),
    ProbeURL: vi.fn().mockResolvedValue({ status: 200, filename: 'test.zip', size: 1024 }),
    CheckHistory: vi.fn().mockResolvedValue(false),
    CheckCollision: vi.fn().mockResolvedValue({ exists: false }),
    AddDownload: vi.fn().mockResolvedValue('test-id'),
    AddDownloadWithFilename: vi.fn().mockResolvedValue('test-id'),
    AddDownloadWithOptions: vi.fn().mockResolvedValue('test-id'),
    AddDownloadWithParams: vi.fn().mockResolvedValue('test-id'),
    PauseDownload: vi.fn(),
    ResumeDownload: vi.fn(),
    StopDownload: vi.fn(),
    DeleteDownload: vi.fn(),
    OpenFile: vi.fn(),
    OpenFolder: vi.fn(),
    ReorderDownload: vi.fn(),
    SetPriority: vi.fn(),
    PauseAllDownloads: vi.fn(),
    ResumeAllDownloads: vi.fn(),
    GetNetworkHealth: vi.fn().mockResolvedValue({ level: 'normal' }),
    RunNetworkSpeedTest: vi.fn().mockResolvedValue({ download_speed: 15.5, upload_speed: 5.2, latency: 25 }),
    GetEnableAI: vi.fn().mockResolvedValue(false),
    SetEnableAI: vi.fn(),
    GetAIPort: vi.fn().mockResolvedValue(4444),
    SetAIPort: vi.fn(),
    GetAIMaxConcurrent: vi.fn().mockResolvedValue(5),
    SetAIMaxConcurrent: vi.fn(),
    GetAIToken: vi.fn().mockResolvedValue('mock-token'),
    SetMaxConcurrentDownloads: vi.fn(),
    SetGlobalSpeedLimit: vi.fn(),
    UpdateSettings: vi.fn(),
    FactoryReset: vi.fn(),
    GetRecentAuditLogs: vi.fn().mockResolvedValue([]),
    GetSpeedTestHistory: vi.fn().mockResolvedValue([]),
    CancelSpeedTest: vi.fn(),
    SetHostLimit: vi.fn(),
}));
