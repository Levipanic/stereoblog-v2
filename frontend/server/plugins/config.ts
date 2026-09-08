import { validateRuntimeConfig } from '../../config/validate'

export default defineNitroPlugin(() => {
  const config = useRuntimeConfig()
  validateRuntimeConfig({
    internalApiBase: String(config.internalApiBase),
    apiBase: String(config.public.apiBase),
    siteUrl: String(config.public.siteUrl),
    siteName: String(config.public.siteName),
    siteHandle: String(config.public.siteHandle),
  })
})
