import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

function normalizeBase(raw) {
  let value = String(raw ?? '').trim()
  if (!value || value === '/') return '/'
  if (!value.startsWith('/')) value = `/${value}`
  if (!value.endsWith('/')) value = `${value}/`
  return value
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export default defineConfig(({ mode }) => {
  const fileEnv = loadEnv(mode, process.cwd(), '')
  const base = normalizeBase(process.env.VITE_BASE_PATH || fileEnv.VITE_BASE_PATH)
  const apiTarget = process.env.VITE_API_PROXY || fileEnv.VITE_API_PROXY || 'http://127.0.0.1:8180'
  const apiPath = `${base === '/' ? '' : base.slice(0, -1)}/api`

  return {
    base,
    plugins: [vue()],
    server: {
      host: true,
      allowedHosts: true,
      port: 5273,
      proxy: {
        [apiPath]: {
          target: apiTarget,
          changeOrigin: true,
          rewrite: (path) => {
            if (base === '/') return path
            const prefix = base.slice(0, -1)
            return path.replace(new RegExp(`^${escapeRegExp(prefix)}`), '') || '/'
          },
        },
      },
    },
    build: {
      outDir: 'dist',
    },
  }
})
