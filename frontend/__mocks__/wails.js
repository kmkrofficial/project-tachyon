// Mock Wails Runtime
window.runtime = {
    EventsOn: vi.fn(),
    EventsOff: vi.fn(),
    BrowserOpenURL: vi.fn(),
};

// Mock Go Bindings
window.go = {
    main: {
        App: {
            AddDownload: vi.fn(),
            GetTasks: vi.fn().mockResolvedValue([]),
            OpenFolder: vi.fn(),
        },
    },
};
