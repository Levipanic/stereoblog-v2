<script setup lang="ts">
import { ApiError } from '~/utils/api'

// Minimal navigation destination for V2-306. V2-401 supplies the full article renderer.
const route = useRoute()
const router = useRouter()
const api = usePublicApi()
const config = useRuntimeConfig()
const { t } = useReaderSettings()
const slug = String(route.params.slug)
const { data: post, error } = await useAsyncData(`post-shell:${slug}`, async (_app, { signal }) => {
  try { return await api.post(slug, { signal }) }
  catch (error) {
    throw createError({ statusCode: error instanceof ApiError && error.status === 404 ? 404 : 502, statusMessage: 'Post unavailable' })
  }
})
if (error.value) throw createError(error.value)

function back() {
  const previous = router.options.history.state.back
  if (typeof previous === 'string' && previous.split(/[?#]/)[0] === '/') router.back()
  else navigateTo('/')
}

useHead(() => ({ title: `${post.value?.title ?? t('posts')} — ${config.public.siteName}` }))
</script>

<template>
  <main id="main-content" class="site-shell" tabindex="-1">
    <section class="timeline-panel">
      <header class="timeline-head"><h1>{{ post?.title ?? t('posts') }}</h1></header>
      <div class="feed-status">
        <p>{{ t('readerComingSoon') }}</p>
        <button type="button" @click="back">{{ t('backToFeed') }}</button>
      </div>
      <p id="comments" class="feed-status">{{ t('discussionComingSoon') }}</p>
    </section>
  </main>
</template>
