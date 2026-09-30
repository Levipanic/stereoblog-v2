<script setup lang="ts">
const api = usePublicApi()
const { data: health, error } = await useAsyncData('api-health', (_app, { signal }) => api.health({ signal }))

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
