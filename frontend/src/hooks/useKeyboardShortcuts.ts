import { useEffect } from 'react';

interface ShortcutActions {
    onNewDownload: () => void;
    onPauseResume: () => void;
    onDelete: () => void;
}

/**
 * Global keyboard shortcuts:
 * - Ctrl+N / Cmd+N  → New download modal
 * - Space           → Pause/Resume first active download
 * - Delete          → Delete selected/first active download
 *
 * Shortcuts are suppressed when the user is typing in an input or textarea.
 */
export function useKeyboardShortcuts({ onNewDownload, onPauseResume, onDelete }: ShortcutActions) {
    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            const tag = (e.target as HTMLElement)?.tagName;
            const isTyping = tag === 'INPUT' || tag === 'TEXTAREA' || (e.target as HTMLElement)?.isContentEditable;

            // Ctrl+N always works (even in inputs)
            if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
                e.preventDefault();
                onNewDownload();
                return;
            }

            // Don't handle Space/Delete while typing
            if (isTyping) return;

            if (e.key === ' ') {
                e.preventDefault();
                onPauseResume();
                return;
            }

            if (e.key === 'Delete') {
                e.preventDefault();
                onDelete();
                return;
            }
        };

        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, [onNewDownload, onPauseResume, onDelete]);
}
