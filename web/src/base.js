function normalizeBase(raw) {
  let value = String(raw ?? '').trim()
  if (!value || value === '/') return '/'
  if (!value.startsWith('/')) value = `/${value}`
  if (!value.endsWith('/')) value = `${value}/`
  return value
}

export const appBase = normalizeBase(import.meta.env.BASE_URL)

export function withBase(path) {
  const suffix = String(path || '').replace(/^\//, '')
  if (!suffix) return appBase
  return `${appBase}${suffix}`
}

export function stripBase(pathname) {
  const path = pathname || '/'
  if (appBase === '/') return path
  const prefix = appBase.slice(0, -1)
  if (path === prefix || path.startsWith(`${prefix}/`)) {
    const rest = path.slice(prefix.length)
    return rest || '/'
  }
  return path
}
