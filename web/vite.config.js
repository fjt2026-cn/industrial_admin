import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Vite 配置：开发期用代理解决跨域，生产期由 Nginx 反向代理同一路径
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    // 代理规则：浏览器请求 /api/xxx，开发服务器转发到后端，避免浏览器跨域
    proxy: {
      '/api': {
        // 后端地址；默认 8080，可用环境变量 PROXY_TARGET 覆盖（如后端改端口时）
        target: process.env.PROXY_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: false
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false
  }
})
