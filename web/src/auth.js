import { ref } from 'vue'

const TOKEN_KEY = 'eino_jwt'
const USER_KEY = 'eino_user'

function readUser() {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export const currentUser = ref(readUser())

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function getUser() {
  return currentUser.value
}

export function setSession(token, user) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
  if (user) {
    localStorage.setItem(USER_KEY, JSON.stringify(user))
    currentUser.value = user
  } else {
    localStorage.removeItem(USER_KEY)
    currentUser.value = null
  }
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  currentUser.value = null
}

export function isLoggedIn() {
  return !!getToken()
}

export function accountLabel(user) {
  const display = String(user?.display_name || '').trim()
  if (display) return display
  return String(user?.username || '').trim()
}
