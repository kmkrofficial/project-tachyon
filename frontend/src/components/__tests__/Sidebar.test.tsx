import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Sidebar } from '../Sidebar';

describe('Sidebar', () => {
    const defaultProps = {
        activeTab: 'all',
        setActiveTab: vi.fn(),
        collapsed: false,
        onToggleCollapse: vi.fn(),
    };

    it('renders all menu items when expanded', () => {
        render(<Sidebar {...defaultProps} />);
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
        expect(screen.getByText('Scheduler')).toBeInTheDocument();
        expect(screen.getByText('Speed Test')).toBeInTheDocument();
        expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('renders brand name when expanded', () => {
        render(<Sidebar {...defaultProps} />);
        expect(screen.getByText('TDM')).toBeInTheDocument();
    });

    it('hides brand name when collapsed', () => {
        render(<Sidebar {...defaultProps} collapsed={true} />);
        const brand = screen.queryByText('TDM');
        expect(brand).toBeInTheDocument();
        expect(brand).toHaveClass('opacity-0');
    });

    it('hides menu labels when collapsed', () => {
        render(<Sidebar {...defaultProps} collapsed={true} />);
        const dashboard = screen.queryByText('Dashboard');
        const scheduler = screen.queryByText('Scheduler');
        expect(dashboard).toBeInTheDocument();
        expect(dashboard).toHaveClass('opacity-0');
        expect(scheduler).toBeInTheDocument();
        expect(scheduler).toHaveClass('opacity-0');
    });

    it('calls setActiveTab when menu item clicked', () => {
        const setActiveTab = vi.fn();
        render(<Sidebar {...defaultProps} setActiveTab={setActiveTab} />);
        fireEvent.click(screen.getByText('Scheduler'));
        expect(setActiveTab).toHaveBeenCalledWith('scheduler');
    });

    it('calls onToggleCollapse when collapse button clicked', () => {
        const onToggle = vi.fn();
        render(<Sidebar {...defaultProps} onToggleCollapse={onToggle} />);
        // Last button is the collapse toggle
        const buttons = screen.getAllByRole('button');
        const collapseBtn = buttons[buttons.length - 1];
        fireEvent.click(collapseBtn);
        expect(onToggle).toHaveBeenCalled();
    });

    it('applies w-64 class when expanded and w-16 when collapsed', () => {
        const { container, rerender } = render(<Sidebar {...defaultProps} />);
        expect(container.querySelector('.w-64')).toBeInTheDocument();

        rerender(<Sidebar {...defaultProps} collapsed={true} />);
        expect(container.querySelector('.w-16')).toBeInTheDocument();
    });
});
