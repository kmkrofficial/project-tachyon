/// <reference types="vitest" />
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.ts'],
    globals: true,
    exclude: ['e2e/**', 'node_modules/**'],
    alias: {
      'lucide-react': path.resolve(__dirname, 'src/__mocks__/lucide-react.ts'),
    },
  }
})
