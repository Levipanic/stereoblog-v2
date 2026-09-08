export interface RuntimeConfigValues {
  internalApiBase: string
  apiBase: string
  siteUrl: string
  siteName: string
  siteHandle: string
}

export function validateHttpUrl(name: string, value: string) {
  try {
    const url = new URL(value)
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password) throw new Error()
  } catch {
    throw new Error(`${name} must be an absolute HTTP(S) URL without credentials`)
  }
}

export function validateSameOriginPath(name: string, value: string) {
  const base = new URL('https://config.invalid')
  const url = new URL(value, base)
  if (!value.startsWith('/') || value.startsWith('//') || value.includes('\\') || url.origin !== base.origin) {
    throw new Error(`${name} must be a same-origin absolute path`)
  }
}

export function validateRuntimeConfig(config: RuntimeConfigValues) {
  validateHttpUrl('NUXT_INTERNAL_API_BASE', config.internalApiBase)
  validateSameOriginPath('NUXT_PUBLIC_API_BASE', config.apiBase)
  validateHttpUrl('NUXT_PUBLIC_SITE_URL', config.siteUrl)
  if (!config.siteName.trim()) throw new Error('NUXT_PUBLIC_SITE_NAME must not be empty')
  if (!config.siteHandle.trim()) throw new Error('NUXT_PUBLIC_SITE_HANDLE must not be empty')
}
