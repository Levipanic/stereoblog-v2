<script setup lang="ts">
import { appendFeedPage } from '~/utils/feed'

// Keep only this page alive: its loaded cards/cursor survive same-session navigation.
definePageMeta({ keepalive: true })

const api = usePublicApi()
const config = useRuntimeConfig()
const { t, feedView } = useReaderSettings()
const { data: feed, error, status, refresh } = await useAsyncData('public-feed', (_app, { signal }) => api.feed({ limit: 10 }, { signal }))
const loadingMore = ref(false)
const moreFailed = ref(false)
const loadedCursors = new Set<string>()
let controller = new AbortController()
onBeforeUnmount(() => controller.abort())
onBeforeRouteLeave(() => { controller.abort() })
onDeactivated(() => controller.abort())
onActivated(() => { if (controller.signal.aborted) controller = new AbortController() })

async function loadMore() {
  const cursor = feed.value?.next_cursor
  if (!cursor || loadingMore.value || controller.signal.aborted) return
  loadingMore.value = true
  moreFailed.value = false
  const requestController = controller
  try {
    const next = await api.feed({ limit: 10, cursor }, { signal: requestController.signal })
    if (requestController.signal.aborted) return
    if (next.next_cursor === cursor || (next.next_cursor && loadedCursors.has(next.next_cursor))) throw new Error('Cursor did not advance')
    if (feed.value) feed.value = appendFeedPage(feed.value, next)
    loadedCursors.add(cursor)
  }
  catch {
    if (!requestController.signal.aborted) moreFailed.value = true
  }
  finally { loadingMore.value = false }
}

if (import.meta.server && error.value) {
  const event = useRequestEvent()
  if (event) setResponseStatus(event, 502)
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
      <p v-if="status === 'pending'" class="feed-status" role="status">{{ t('loadingFeed') }}</p>
      <div v-else-if="error" class="feed-status feed-error" role="alert">
        <p>{{ t('feedError') }}</p>
        <button type="button" @click="refresh()">{{ t('retry') }}</button>
      </div>
      <ol v-else-if="feed?.items.length" class="feed-list" :class="{ 'feed-grid': feedView === 'grid' }">
        <li v-for="(post, index) in feed.items" :key="post.id">
          <FeedPost :post="post" :first="index === 0" />
        </li>
      </ol>
      <p v-else class="feed-status" role="status">{{ t('emptyFeed') }}</p>
      <FeedPagination v-if="feed?.items.length && !error" :cursor="feed.next_cursor" :loading="loadingMore" :failed="moreFailed" @load="loadMore" />
    </section>
  </main>
</template>
