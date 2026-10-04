<script setup lang="ts">
import { ApiError } from '~/utils/api'
import { postDate } from '~/utils/feed'
import { hasTableOfContents } from '~/utils/content'
import { postMetadata } from '~/utils/sharing'

definePageMeta({ key: route => route.path })
const route = useRoute()
const router = useRouter()
const api = usePublicApi()
const config = useRuntimeConfig()
const { t, language } = useReaderSettings()
const slug = String(route.params.slug)
const article = useTemplateRef<HTMLElement>('article')
const { data: post, error } = await useAsyncData(`post:${slug}`, async (_app, { signal }) => {
  try { return await api.post(slug, { signal }) }
  catch (error) {
    throw createError({ statusCode: error instanceof ApiError && error.status === 404 ? 404 : 502, statusMessage: 'Post unavailable' })
  }
})
if (import.meta.server && error.value) {
  const event = useRequestEvent()
  if (event) setResponseStatus(event, error.value.statusCode ?? 502)
}

function back() {
  const previous = router.options.history.state.back
  if (typeof previous === 'string' && previous.split(/[?#]/)[0] === '/') router.back()
  else navigateTo('/')
}

const metadata = computed(() => post.value ? postMetadata(post.value, config.public.siteUrl, config.public.siteName) : null)
useHead(() => ({
  title: `${post.value?.title ?? t('postNotFound')} — ${config.public.siteName}`,
  link: metadata.value ? [{ rel: 'canonical', href: metadata.value.url }] : [],
  meta: metadata.value ? [
    { name: 'description', content: metadata.value.description },
    { name: 'author', content: config.public.siteHandle },
    { property: 'og:type', content: 'article' },
    { property: 'og:site_name', content: config.public.siteName },
    { property: 'og:title', content: metadata.value.title },
    { property: 'og:description', content: metadata.value.description },
    { property: 'og:url', content: metadata.value.url },
    { property: 'og:image', content: metadata.value.image },
    { property: 'og:image:alt', content: metadata.value.imageAlt },
    { property: 'article:published_time', content: post.value!.created_at },
    { name: 'twitter:card', content: 'summary_large_image' },
    { name: 'twitter:title', content: metadata.value.title },
    { name: 'twitter:description', content: metadata.value.description },
    { name: 'twitter:image', content: metadata.value.image },
    { name: 'twitter:image:alt', content: metadata.value.imageAlt },
  ] : [{ name: 'robots', content: 'noindex' }],
}))
</script>

<template>
  <main id="main-content" class="site-shell" tabindex="-1">
    <button type="button" class="back-to-feed" @click="back">{{ t('backToFeed') }}</button>
    <article v-if="post" ref="article" class="post-article">
      <header class="article-header">
        <h1>{{ post.title }}</h1>
        <div class="post-meta"><time :datetime="post.created_at">{{ postDate(post.created_at, language) }}</time><span>{{ t('readingTime', { minutes: post.reading_minutes }) }}</span></div>
        <div class="article-actions">
          <NuxtLink :to="`${route.path}#comments`" :prefetch="false">{{ t('discussionTitle') }}</NuxtLink>
          <SharePost v-if="metadata" :title="post.title" :url="metadata.url" />
        </div>
      </header>
      <ReadingProgress :post-id="post.id" :article="article" />
      <div class="article-layout" :class="{ 'with-toc': hasTableOfContents(post.blocks) }">
        <PostToc v-if="hasTableOfContents(post.blocks)" :blocks="post.blocks" />
        <PostBody :blocks="post.blocks" />
      </div>
    </article>
    <PostComments v-if="post" :post-id="post.id" />
    <section v-else class="feed-status" role="alert"><h1>{{ t(error?.statusCode === 404 ? 'postNotFound' : 'postUnavailable') }}</h1></section>
  </main>
</template>
