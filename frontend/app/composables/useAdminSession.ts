import { ApiError, createApiRequest } from '~/utils/api'

interface AdminSession { authenticated: boolean, csrf_token?: string }

export function useAdminSession() {
  const config = useRuntimeConfig()
  const session = useState<AdminSession>('admin-session', () => ({ authenticated: false }))
  const request = createApiRequest(config.public.apiBase)

  async function write<T>(path: string, body: object = {}) {
    try {
      return await request<T>(`/admin${path}`, { headers: { 'X-CSRF-Token': session.value.csrf_token ?? '' } }, body)
    }
    catch (error) {
      if (error instanceof ApiError && error.status === 401) session.value = { authenticated: false }
      throw error
    }
  }
  async function check() {
    session.value = { authenticated: false }
    session.value = await request<AdminSession>('/admin/session')
  }
  async function login(secret: string) {
    session.value = await request<AdminSession>('/admin/login', {}, { secret })
  }
  async function logout() {
    await write('/logout')
    session.value = { authenticated: false }
  }
  return { session: readonly(session), check, login, logout, write }
}
