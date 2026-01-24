import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock Lucide React
vi.mock('lucide-react', () => ({
    File: () => 'FileIcon',
    FileVideo: () => 'FileVideoIcon',
    FileArchive: () => 'FileArchiveIcon',
    FileText: () => 'FileTextIcon',
    Pause: () => 'PauseIcon',
    Play: () => 'PlayIcon',
    X: () => 'XIcon',
    FolderOpen: () => 'FolderOpenIcon',
    LayoutDashboard: () => 'LayoutDashboardIcon',
    Download: () => 'DownloadIcon',
    CheckCircle: () => 'CheckCircleIcon',
    List: () => 'ListIcon',
    Plus: () => 'PlusIcon'
}));
