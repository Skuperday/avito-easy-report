// Хранилище авторизации
export const useAuth = () => {
  const token = useState<string | null>('token', () => null)
  const user = useState<{ userId: number; username: string; role: string } | null>('user', () => null)

  const config = useRuntimeConfig()

  if (import.meta.client) {
    const savedToken = localStorage.getItem('token')
    const savedUser = localStorage.getItem('user')
    if (savedToken && !token.value) token.value = savedToken
    if (savedUser && !user.value) { try { user.value = JSON.parse(savedUser) } catch {} }
  }

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  const apiFetch = async (path: string, options: RequestInit = {}) => {
    const headers: Record<string, string> = {
      ...(options.headers as Record<string, string> || {}),
    }
    if (!(options.body instanceof FormData)) {
      headers['Content-Type'] = 'application/json'
    }
    if (token.value) {
      headers['Authorization'] = `Bearer ${token.value}`
    }
    const res = await fetch(`${config.public.apiBase}${path}`, { ...options, headers })
    if (res.status === 401) {
      logout()
      throw new Error('Сессия истекла')
    }
    return res
  }

  const login = async (username: string, password: string) => {
    const res = await apiFetch('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Ошибка входа')
    token.value = data.token
    user.value = { userId: data.userId, username: data.username, role: data.role }
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(user.value))
    return data
  }

  const register = async (username: string, password: string) => {
    const res = await apiFetch('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Ошибка регистрации')
    return data
  }

  const logout = () => {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    navigateTo('/login')
  }

  const fetchMe = async () => {
    const res = await apiFetch('/auth/me')
    if (!res.ok) { logout(); return null }
    const data = await res.json()
    user.value = { userId: data.userId, username: data.username, role: data.role }
    localStorage.setItem('user', JSON.stringify(user.value))
    return data
  }

  return { token, user, isAuthenticated, isAdmin, apiFetch, login, register, logout, fetchMe }
}
