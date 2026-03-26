/** @type {import('tailwindcss').Config} */
export default {
    darkMode: 'class',
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            colors: {
                th: {
                    base:       'rgb(var(--th-base) / <alpha-value>)',
                    surface:    'rgb(var(--th-surface) / <alpha-value>)',
                    raised:     'rgb(var(--th-raised) / <alpha-value>)',
                    overlay:    'rgb(var(--th-overlay) / <alpha-value>)',
                    border:     'rgb(var(--th-border) / <alpha-value>)',
                    'border-s': 'rgb(var(--th-border-s) / <alpha-value>)',
                    text:       'rgb(var(--th-text) / <alpha-value>)',
                    'text-s':   'rgb(var(--th-text-s) / <alpha-value>)',
                    'text-m':   'rgb(var(--th-text-m) / <alpha-value>)',
                    accent:     'rgb(var(--th-accent) / <alpha-value>)',
                    'accent-h': 'rgb(var(--th-accent-h) / <alpha-value>)',
                    'accent-t': 'rgb(var(--th-accent-t) / <alpha-value>)',
                },
            },
        },
    },
    plugins: [],
}
