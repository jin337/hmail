import react from '@vitejs/plugin-react-swc'
import { defineConfig } from 'vite'

import tailwindcss from '@tailwindcss/vite' // tailwindcss 插件
import compression from 'vite-plugin-compression' // 代码压缩
import svgr from 'vite-plugin-svgr' // svg

// https://vite.dev/config/
export default defineConfig({
  base: '/web',
  server: {
    host: '0.0.0.0',
    port: 4055,
    open: true,
    cors: true,
  },
  resolve: {
    alias: {
      src: '/src',
    },
  },
  plugins: [
    react(),
    svgr({
      include: '**/*.svg',
    }),
    tailwindcss(),
    compression(),
  ],
  build: {
    outDir: 'Web',
    rollupOptions: {
      output: {
        entryFileNames: 'assets/js/[name]-[hash].js',
        chunkFileNames: 'assets/js/chunks/[name]-[hash].js',
        assetFileNames: 'assets/[ext]/[name]-[hash].[ext]',
        manualChunks: (id) => {
          if (!id.includes('node_modules')) return
          const chunkMap = [
            { name: 'chunk-vendor', deps: ['react', 'react-dom', 'react-router', 'react-redux', 'axios'] },
            { name: 'chunk-ui', deps: ['@arco-design/web-react'] },
            { name: 'chunk-utils', deps: ['crypto-js', 'dayjs'] },
            { name: 'chunk-editor', deps: ['@wangeditor/editor', '@wangeditor/editor-for-react'] },
            { name: 'chunk-preview', deps: ['jit-viewer'] },
          ]
          for (const { name, deps } of chunkMap) {
            if (deps.some((dep) => id.includes(dep))) return name
          }
        },
      },
      // 忽略第三方警告
      onwarn(warning, warn) {
        if (warning.id && warning.id.includes('node_modules')) return
        warn(warning)
      },
    },
  },
})
