import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';

describe('useKeyboardShortcuts', () => {
    const actions = {
        onNewDownload: vi.fn(),
        onPauseResume: vi.fn(),
        onDelete: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('fires onNewDownload on Ctrl+N', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'n', ctrlKey: true }));
        expect(actions.onNewDownload).toHaveBeenCalledTimes(1);
    });

    it('fires onNewDownload on Meta+N (Mac)', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'n', metaKey: true }));
        expect(actions.onNewDownload).toHaveBeenCalledTimes(1);
    });

    it('fires onPauseResume on Space', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: ' ' }));
        expect(actions.onPauseResume).toHaveBeenCalledTimes(1);
    });

    it('fires onDelete on Delete key', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Delete' }));
        expect(actions.onDelete).toHaveBeenCalledTimes(1);
    });

    it('suppresses Space when target is INPUT', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        const input = document.createElement('input');
        document.body.appendChild(input);
        const event = new KeyboardEvent('keydown', { key: ' ', bubbles: true });
        Object.defineProperty(event, 'target', { value: input });
        window.dispatchEvent(event);
        expect(actions.onPauseResume).not.toHaveBeenCalled();
        document.body.removeChild(input);
    });

    it('suppresses Delete when target is TEXTAREA', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        const textarea = document.createElement('textarea');
        document.body.appendChild(textarea);
        const event = new KeyboardEvent('keydown', { key: 'Delete', bubbles: true });
        Object.defineProperty(event, 'target', { value: textarea });
        window.dispatchEvent(event);
        expect(actions.onDelete).not.toHaveBeenCalled();
        document.body.removeChild(textarea);
    });

    it('Ctrl+N works even inside input', () => {
        renderHook(() => useKeyboardShortcuts(actions));
        const input = document.createElement('input');
        document.body.appendChild(input);
        const event = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, bubbles: true });
        Object.defineProperty(event, 'target', { value: input });
        window.dispatchEvent(event);
        expect(actions.onNewDownload).toHaveBeenCalledTimes(1);
        document.body.removeChild(input);
    });

    it('cleans up event listener on unmount', () => {
        const spy = vi.spyOn(window, 'removeEventListener');
        const { unmount } = renderHook(() => useKeyboardShortcuts(actions));
        unmount();
        expect(spy).toHaveBeenCalledWith('keydown', expect.any(Function));
        spy.mockRestore();
    });
});
