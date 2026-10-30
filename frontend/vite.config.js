import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const rootDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': path.resolve(rootDir, 'src') }
  },
  build: {
    // 产物直接落到 web/dist，由 Go 侧 //go:embed all:dist 打包进二进制
    outDir: path.resolve(rootDir, '../web/dist'),
    emptyOutDir: true,
    target: 'es2022',
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['vue', 'vue-router', 'pinia', 'axios'],
          element: ['element-plus', '@element-plus/icons-vue'],
          pdf: ['pdfjs-dist']
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: { '/api': 'http://localhost:8080' }
  }
})
