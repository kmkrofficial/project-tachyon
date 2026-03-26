import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Header } from '../Header';

// Mock the store
vi.mock('../../store', () => ({
    useSettingsStore: vi.fn((selector) => {
        const state = { theme: 'dark' as const, setTheme: vi.fn() };
        return selector(state);
    }),
}));

describe('Header', () => {
    const defaultProps = {
        onAddDownload: vi.fn(),
        onPauseAll: vi.fn(),
        onResumeAll: vi.fn(),
        globalSpeed: 5.5,
        sidebarCollapsed: false,
    };

    beforeEach(() => {
        vi.clearAllMocks();
        // Mock window.go for GetNetworkHealth
        (window as any).go = {
            app: {
                App: {
                    GetNetworkHealth: vi.fn().mockResolvedValue({ level: 'normal' }),
                },
            },
        };
    });

    it('renders Dashboard title', () => {
        render(<Header {...defaultProps} />);
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
        expect(screen.getByText('Overview')).toBeInTheDocument();
    });

    it('renders Add Download button', () => {
        render(<Header {...defaultProps} />);
        expect(screen.getByText('Add Download')).toBeInTheDocument();
    });

    it('calls onAddDownload when button clicked', () => {
        const onAdd = vi.fn();
        render(<Header {...defaultProps} onAddDownload={onAdd} />);
        fireEvent.click(screen.getByText('Add Download'));
        expect(onAdd).toHaveBeenCalled();
    });

    it('calls onPauseAll when pause button clicked', () => {
        const onPause = vi.fn();
        render(<Header {...defaultProps} onPauseAll={onPause} />);
        fireEvent.click(screen.getByTitle('Pause All'));
        expect(onPause).toHaveBeenCalled();
    });

    it('calls onResumeAll when resume button clicked', () => {
        const onResume = vi.fn();
        render(<Header {...defaultProps} onResumeAll={onResume} />);
        fireEvent.click(screen.getByTitle('Resume All'));
        expect(onResume).toHaveBeenCalled();
    });

    it('shows theme toggle button', () => {
        render(<Header {...defaultProps} />);
        const themeBtn = screen.getByTitle(/Theme:/);
        expect(themeBtn).toBeInTheDocument();
    });

    it('adjusts left offset based on sidebar collapse state', () => {
        const { container, rerender } = render(<Header {...defaultProps} sidebarCollapsed={false} />);
        expect(container.querySelector('.left-64')).toBeInTheDocument();

        rerender(<Header {...defaultProps} sidebarCollapsed={true} />);
        expect(container.querySelector('.left-16')).toBeInTheDocument();
    });
});
