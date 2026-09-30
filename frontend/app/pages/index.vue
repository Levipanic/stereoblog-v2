<script setup lang="ts">
const api = usePublicApi()
const config = useRuntimeConfig()
const { t } = useReaderSettings()
const { data: health, error } = await useAsyncData('api-health', (_app, { signal }) => api.health({ signal }))

if (error.value) {
  throw createError({ statusCode: 502, statusMessage: 'API unavailable' })
}

useHead(() => ({ title: `${config.public.siteName} — ${t('posts')}` }))
</script>

<template>
  <main id="main-content" class="site-shell" tabindex="-1">
    <section class="timeline-panel" aria-labelledby="feed-title">
      <header class="timeline-head">
        <div>
          <h1 id="feed-title">{{ t('posts') }}</h1>
          <p>{{ t('chronological') }}</p>
        </div>
        <span class="site-handle">{{ config.public.siteHandle }}</span>
      </header>
      <p class="feed-status">{{ t('preparingFeed') }}</p>
      <p class="system-status">v2 / {{ health?.status }}</p>
    </section>
  </main>
</template>
