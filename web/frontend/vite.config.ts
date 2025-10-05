import { defineConfig } from 'vite'
import viteReact from '@vitejs/plugin-react'
import viteTsConfigPaths from 'vite-tsconfig-paths'
import tailwindcss from '@tailwindcss/vite'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'

const config = defineConfig({
  plugins: [
    // Path aliases support
    viteTsConfigPaths({
      projects: ['./tsconfig.json'],
    }),
    // Tailwind CSS
    tailwindcss(),
    // TanStack Router plugin for file-based routing
    TanStackRouterVite(),
    // React plugin
    viteReact(),
  ],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
          'router-vendor': ['@tanstack/react-router', '@tanstack/react-query'],
        },
      },
    },
  },
  server: {
    port: 3000,
  },
})

export default config
