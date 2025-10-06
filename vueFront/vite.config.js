import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://172.31.45.255:8080' // Forward /api calls to backend
      //'/api': 'http://localhost:8080' 
    }
  }
})
