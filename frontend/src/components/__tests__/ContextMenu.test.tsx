import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ContextMenu } from '../ContextMenu';

describe('ContextMenu', () => {
    const defaultProps = {
        x: 100,
        y: 200,
        visible: true,
        onClose: vi.fn(),
        onOpen: vi.fn(),
        onShowInFolder: vi.fn(),
        onCopyLink: vi.fn(),
        onDelete: vi.fn(),
        onRetry: vi.fn(),
        onSetPriority: vi.fn(),
        onPause: vi.fn(),
        onResume: vi.fn(),
        onStop: vi.fn(),
        status: 'downloading',
    };

    it('renders when visible is true', () => {
        render(<ContextMenu {...defaultProps} />);
        expect(screen.getByText('Open File')).toBeInTheDocument();
        expect(screen.getByText('Show in Folder')).toBeInTheDocument();
        expect(screen.getByText('Copy Link')).toBeInTheDocument();
        expect(screen.getByText('Delete')).toBeInTheDocument();
    });

    it('does not render when visible is false', () => {
        render(<ContextMenu {...defaultProps} visible={false} />);
        expect(screen.queryByText('Open File')).not.toBeInTheDocument();
    });

    it('calls onOpen and onClose when Open File clicked', () => {
        const onOpen = vi.fn();
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onOpen={onOpen} onClose={onClose} />);
        fireEvent.click(screen.getByText('Open File'));
        expect(onOpen).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    it('calls onShowInFolder and onClose when Show in Folder clicked', () => {
        const onShow = vi.fn();
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onShowInFolder={onShow} onClose={onClose} />);
        fireEvent.click(screen.getByText('Show in Folder'));
        expect(onShow).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    it('calls onCopyLink and onClose when Copy Link clicked', () => {
        const onCopy = vi.fn();
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onCopyLink={onCopy} onClose={onClose} />);
        fireEvent.click(screen.getByText('Copy Link'));
        expect(onCopy).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    it('calls onDelete and onClose when Delete clicked', () => {
        const onDelete = vi.fn();
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onDelete={onDelete} onClose={onClose} />);
        fireEvent.click(screen.getByText('Delete'));
        expect(onDelete).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    it('calls onRetry and onClose when Re-download clicked', () => {
        const onRetry = vi.fn();
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onRetry={onRetry} onClose={onClose} />);
        fireEvent.click(screen.getByText('Re-download'));
        expect(onRetry).toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });

    it('closes menu on outside click', () => {
        const onClose = vi.fn();
        render(<ContextMenu {...defaultProps} onClose={onClose} />);
        fireEvent.mouseDown(document.body);
        expect(onClose).toHaveBeenCalled();
    });

    it('positions at given x,y coordinates', () => {
        const { container } = render(<ContextMenu {...defaultProps} x={150} y={250} />);
        const menu = container.querySelector('.fixed.z-50');
        expect(menu).toHaveStyle({ top: '250px', left: '150px' });
    });
});
