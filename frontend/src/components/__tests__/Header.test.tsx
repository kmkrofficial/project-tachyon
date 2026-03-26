import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Header } from '../Header';

describe('Header', () => {
    const defaultProps = {
        activeTab: 'all',
        onAddDownload: vi.fn(),
        onPauseAll: vi.fn(),
        onResumeAll: vi.fn(),
        sidebarCollapsed: false,
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders page title for dashboard', () => {
        render(<Header {...defaultProps} />);
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
    });

    it('renders page title for other tabs', () => {
        render(<Header {...defaultProps} activeTab="analytics" />);
        expect(screen.getByText('Analytics')).toBeInTheDocument();
    });

    it('renders action buttons on dashboard', () => {
        render(<Header {...defaultProps} />);
        expect(screen.getByText('Add Download')).toBeInTheDocument();
        expect(screen.getByTitle('Pause All')).toBeInTheDocument();
        expect(screen.getByTitle('Resume All')).toBeInTheDocument();
    });

    it('hides action buttons on non-dashboard tabs', () => {
        render(<Header {...defaultProps} activeTab="analytics" />);
        expect(screen.queryByText('Add Download')).not.toBeInTheDocument();
        expect(screen.queryByTitle('Pause All')).not.toBeInTheDocument();
        expect(screen.queryByTitle('Resume All')).not.toBeInTheDocument();
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

    it('adjusts left offset based on sidebar collapse state', () => {
        const { container, rerender } = render(<Header {...defaultProps} sidebarCollapsed={false} />);
        expect(container.querySelector('.left-64')).toBeInTheDocument();

        rerender(<Header {...defaultProps} sidebarCollapsed={true} />);
        expect(container.querySelector('.left-16')).toBeInTheDocument();
    });
});
