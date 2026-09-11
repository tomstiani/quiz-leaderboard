import stylex from '@stylexjs/unplugin'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [stylex.vite(), react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
