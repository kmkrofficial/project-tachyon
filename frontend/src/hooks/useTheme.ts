import { useEffect } from 'react';
import { useSettingsStore } from '../store';

/** Syncs the `<html>` class list with the persisted theme preference. */
export function useTheme() {
    const theme = useSettingsStore(s => s.theme);
    const setTheme = useSettingsStore(s => s.setTheme);

    useEffect(() => {
        const root = document.documentElement;

        const applyTheme = (mode: 'light' | 'dark' | 'black') => {
            root.classList.remove('dark', 'black');
            if (mode === 'dark') root.classList.add('dark');
            else if (mode === 'black') root.classList.add('black');
        };

        if (theme === 'system') {
            const mq = window.matchMedia('(prefers-color-scheme: dark)');
            applyTheme(mq.matches ? 'dark' : 'light');
            const handler = (e: MediaQueryListEvent) => applyTheme(e.matches ? 'dark' : 'light');
            mq.addEventListener('change', handler);
            return () => mq.removeEventListener('change', handler);
        }

        applyTheme(theme);
    }, [theme]);

    return { theme, setTheme } as const;
}
