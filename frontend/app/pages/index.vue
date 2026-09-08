<script setup lang="ts">
interface HealthResponse {
  status: string
}

const config = useRuntimeConfig()
const apiBase = (import.meta.server ? config.internalApiBase : config.public.apiBase).replace(/\/+$/, '')
const { data: health, error } = await useFetch<HealthResponse>(`${apiBase}/health`, { key: 'api-health' })

if (error.value) {
  throw createError({ statusCode: 502, statusMessage: 'API unavailable' })
}
</script>

<template>
  <main class="site-shell">
    <h1>StereoDamage</h1>
    <p class="system-status">v2 / {{ health?.status }}</p>
  </main>
</template>
