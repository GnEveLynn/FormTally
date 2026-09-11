import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/v1': process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080',
    },
  },
})
