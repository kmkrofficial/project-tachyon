import { render, screen, fireEvent, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ToastContainer, ToastMessage } from '../Toast';

describe('ToastContainer', () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });
    afterEach(() => {
        vi.useRealTimers();
    });

    const mockToasts: ToastMessage[] = [
        { id: '1', type: 'success', title: 'Done', message: 'Download completed' },
        { id: '2', type: 'error', title: 'Failed', message: 'Network error' },
    ];

    it('renders all toasts', () => {
        render(<ToastContainer toasts={mockToasts} removeToast={vi.fn()} />);
        expect(screen.getByText('Done')).toBeInTheDocument();
        expect(screen.getByText('Failed')).toBeInTheDocument();
    });

    it('renders toast title and message', () => {
        render(<ToastContainer toasts={[mockToasts[0]]} removeToast={vi.fn()} />);
        expect(screen.getByText('Done')).toBeInTheDocument();
        expect(screen.getByText('Download completed')).toBeInTheDocument();
    });

    it('calls removeToast on dismiss button click', () => {
        const removeMock = vi.fn();
        render(<ToastContainer toasts={[mockToasts[0]]} removeToast={removeMock} />);

        // Find close button (the X button)
        const buttons = screen.getAllByRole('button');
        fireEvent.click(buttons[0]);

        // After animation delay (300ms)
        act(() => { vi.advanceTimersByTime(300); });
        expect(removeMock).toHaveBeenCalledWith('1');
    });

    it('auto-removes toast after timeout', () => {
        const removeMock = vi.fn();
        render(<ToastContainer toasts={[mockToasts[0]]} removeToast={removeMock} />);

        act(() => { vi.advanceTimersByTime(4100); });
        expect(removeMock).toHaveBeenCalledWith('1');
    });

    it('renders different types with appropriate styles', () => {
        const allTypes: ToastMessage[] = [
            { id: '1', type: 'success', title: 'S', message: 'm' },
            { id: '2', type: 'error', title: 'E', message: 'm' },
            { id: '3', type: 'warning', title: 'W', message: 'm' },
            { id: '4', type: 'info', title: 'I', message: 'm' },
        ];
        render(<ToastContainer toasts={allTypes} removeToast={vi.fn()} />);
        expect(screen.getByText('S')).toBeInTheDocument();
        expect(screen.getByText('E')).toBeInTheDocument();
        expect(screen.getByText('W')).toBeInTheDocument();
        expect(screen.getByText('I')).toBeInTheDocument();
    });

    it('renders nothing when toasts array is empty', () => {
        const { container } = render(<ToastContainer toasts={[]} removeToast={vi.fn()} />);
        // Container div exists but has no toast children
        const toastItems = container.querySelectorAll('.w-80');
        expect(toastItems.length).toBe(0);
    });
});
