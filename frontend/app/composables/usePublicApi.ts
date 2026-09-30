import { createPublicApi } from '~/utils/api'

export function usePublicApi() {
  const config = useRuntimeConfig()
  return createPublicApi(import.meta.server ? config.internalApiBase : config.public.apiBase)
}
