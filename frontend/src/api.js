import { reactive } from 'vue'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api'
const ACCESS_KEY = 'aurum_access_token'
const REFRESH_KEY = 'aurum_refresh_token'
let refreshPromise = null
const tokens = reactive({
  access: localStorage.getItem(ACCESS_KEY),
  refresh: localStorage.getItem(REFRESH_KEY)
})

export const session = {
  get access() { return tokens.access },
  get refresh() { return tokens.refresh },
  get authenticated() { return Boolean(this.refresh) },
  save(data) {
    tokens.access = data.access_token
    tokens.refresh = data.refresh_token
    localStorage.setItem(ACCESS_KEY, data.access_token)
    localStorage.setItem(REFRESH_KEY, data.refresh_token)
  },
  clear() {
    tokens.access = null
    tokens.refresh = null
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  }
}

async function refreshAccessToken() {
  if (!session.refresh) throw new Error('Your session has expired.')
  if (!refreshPromise) {
    refreshPromise = fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: session.refresh })
    }).then(async response => {
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || 'Your session has expired.')
      session.save(data)
      return data.access_token
    }).finally(() => { refreshPromise = null })
  }
  return refreshPromise
}

export async function api(path, options = {}, retry = true) {
  const headers = { 'Content-Type': 'application/json', ...options.headers }
  if (session.access) headers.Authorization = `Bearer ${session.access}`
  const response = await fetch(`${API_URL}${path}`, { ...options, headers })

  if (response.status === 401 && retry && session.refresh && path !== '/auth/refresh') {
    try {
      await refreshAccessToken()
      return api(path, options, false)
    } catch (error) {
      session.clear()
      window.location.assign('/login')
      throw error
    }
  }

  const data = response.status === 204 ? null : await response.json().catch(() => null)
  if (!response.ok) throw new Error(data?.error || 'Something went wrong.')
  return data
}

export async function logout() {
  const refreshToken = session.refresh
  try {
    if (refreshToken) await api('/auth/logout', { method: 'POST', body: JSON.stringify({ refresh_token: refreshToken }) }, false)
  } finally {
    session.clear()
  }
}
