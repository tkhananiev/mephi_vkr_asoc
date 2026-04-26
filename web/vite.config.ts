import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Прокси обходит отсутствие CORS у Go-сервисов. Пути совпадают с бэкендом.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // С другой машины в сети / по IP хоста открывай URL из строки «Network» в консоли Vite, не localhost.
    host: true,
    proxy: {
      '^/api/v1/scans': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '^/api/v1/sync': {
        target: 'http://127.0.0.1:8081',
        changeOrigin: true,
      },
      '^/api/v1/findings': {
        target: 'http://127.0.0.1:8082',
        changeOrigin: true,
      },
      '^/api/v1/groups': {
        target: 'http://127.0.0.1:8082',
        changeOrigin: true,
      },
      '^/api/v1/tickets': {
        target: 'http://127.0.0.1:8083',
        changeOrigin: true,
      },
    },
  },
})
