const STORAGE_KEY = 'eino_api_base'

function normalizeApiBase(raw) {
  let value = String(raw ?? '').trim()
  if (!value) return ''
  value = value.replace(/\/+$/, '')
  return value
}

export function isDesktopShell() {
  if (typeof window === 'undefined') return false
  return Boolean(window.__TAURI_INTERNALS__ || window.__TAURI__)
}

export function getStoredApiBase() {
  if (typeof localStorage === 'undefined') return ''
  try {
    return normalizeApiBase(localStorage.getItem(STORAGE_KEY))
  } catch {
    return ''
  }
}

export function setStoredApiBase(value) {
  const next = normalizeApiBase(value)
  if (typeof localStorage === 'undefined') return next
  try {
    if (next) localStorage.setItem(STORAGE_KEY, next)
    else localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore quota / private mode */
  }
  return next
}

/** Absolute origin for API (no trailing slash), or empty for same-origin relative paths. */
export function resolveApiOrigin() {
  const stored = getStoredApiBase()
  if (stored) return stored
  const fromEnv = normalizeApiBase(import.meta.env.VITE_API_BASE)
  if (fromEnv) return fromEnv
  if (isDesktopShell()) return 'http://127.0.0.1:8180'
  return ''
}

/** Prefix for /api/... paths (absolute origin or empty). */
export function apiOrigin() {
  return resolveApiOrigin()
}

export function apiUrl(path) {
  const suffix = String(path || '')
  const normalized = suffix.startsWith('/') ? suffix : `/${suffix}`
  const origin = resolveApiOrigin()
  if (!origin) return normalized
  return `${origin}${normalized}`
}
