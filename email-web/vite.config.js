import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vite';

import tailwindcss from '@tailwindcss/vite'; // tailwindcss 插件
import autoImport from 'unplugin-auto-import/vite'; // 自动引入
import compression from 'vite-plugin-compression'; // 代码压缩
// import imageTools from 'vite-plugin-image-tools'; // 图片压缩
import svgr from 'vite-plugin-svgr'; // svg

// https://vite.dev/config/
export default defineConfig({
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
    // imageTools({
    //   quality: 70,
    //   enableWebp: true,
    // }),
    autoImport({
      imports: [
        { 'src/service/api': [['default', 'Http']] },
        { 'src/hooks': [['default', 'Hooks']] },
        { 'src/components/UserPermissions': [['default', 'UserPermissions']] },
      ],
    }),
    compression(),
  ],
  build: {
    rollupOptions: {
      output: {
        entryFileNames: 'assets/js/[name]-[hash].js',
        chunkFileNames: 'assets/js/chunks/[name]-[hash].js',
        assetFileNames: 'assets/[ext]/[name]-[hash].[ext]',
        manualChunks: (id) => {
          // 提前返回非 node_modules 的模块
          if (!id.includes('node_modules')) return

          // 定义分组映射
          const chunkMap = [
            {
              name: 'chunk-vendor',
              deps: ['react', 'react-dom', 'react-router'],
            },
            {
              name: 'chunk-ui',
              deps: ['@arco-design/web-react'],
            },
            {
              name: 'chunk-utils',
              deps: ['crypto-js', 'dayjs', 'axios'],
            },
          ]

          // 查找匹配的分组
          for (const { name, deps } of chunkMap) {
            if (deps.some((dep) => id.includes(dep))) {
              return name
            }
          }
        },
      },
    },
  },
})
