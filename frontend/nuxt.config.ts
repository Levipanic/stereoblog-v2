import { validateRuntimeConfig } from './config/validate'

const publicApiBase = process.env.NUXT_PUBLIC_API_BASE || '/api/v1'
const runtimeConfig = {
  internalApiBase: process.env.NUXT_INTERNAL_API_BASE || 'http://127.0.0.1:8080/api/v1',
  apiBase: publicApiBase,
  siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',
  siteName: process.env.NUXT_PUBLIC_SITE_NAME || 'StereoDamage',
  siteHandle: process.env.NUXT_PUBLIC_SITE_HANDLE || '@stereodamage',
}
validateRuntimeConfig(runtimeConfig)

export default defineNuxtConfig({
  compatibilityDate: '2026-09-08',
  devtools: { enabled: false },
  runtimeConfig: {
    internalApiBase: runtimeConfig.internalApiBase,
    public: {
      apiBase: runtimeConfig.apiBase,
      siteUrl: runtimeConfig.siteUrl,
      siteName: runtimeConfig.siteName,
      siteHandle: runtimeConfig.siteHandle,
    },
  },
})
