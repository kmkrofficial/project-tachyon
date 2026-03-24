import { useEffect } from 'react';
import { useSettingsStore } from '../store';

/** Syncs the `<html>` class list with the persisted theme preference. */
export function useTheme() {
    const theme = useSettingsStore(s => s.theme);
    const setTheme = useSettingsStore(s => s.setTheme);

    useEffect(() => {
        const root = document.documentElement;

        const applyDark = (dark: boolean) => {
            if (dark) root.classList.add('dark');
            else root.classList.remove('dark');
        };

        if (theme === 'system') {
            const mq = window.matchMedia('(prefers-color-scheme: dark)');
            applyDark(mq.matches);
            const handler = (e: MediaQueryListEvent) => applyDark(e.matches);
            mq.addEventListener('change', handler);
            return () => mq.removeEventListener('change', handler);
        }

        applyDark(theme === 'dark');
    }, [theme]);

    return { theme, setTheme } as const;
}
