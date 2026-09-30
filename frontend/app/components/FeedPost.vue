<script setup lang="ts">
import type { FeedItem } from '~/types/api'
import { ApiError } from '~/utils/api'
import { localMediaSource, postDate, postPath, previewText } from '~/utils/feed'
import type { MessageKey } from '~/utils/i18n'

const props = defineProps<{ post: FeedItem, first?: boolean }>()
const { t, language } = useReaderSettings()
const api = usePublicApi()
const path = computed(() => postPath(props.post.slug))
const mediaSrc = computed(() => props.post.preview_media && localMediaSource(props.post.preview_media.src))
const imageFailed = ref(false)
const previewImage = useTemplateRef<HTMLImageElement>('previewImage')
const likes = ref(props.post.likes)
const pending = ref(false)
const hydrated = ref(false)
const feedback = ref<MessageKey | null>(null)
const controller = new AbortController()

watch(() => props.post.likes, value => { likes.value = value })
watch(mediaSrc, () => { imageFailed.value = false })
onMounted(() => {
  hydrated.value = true
  // An eager SSR image can fail before Vue attaches its error listener.
  if (previewImage.value?.complete && previewImage.value.naturalWidth === 0) imageFailed.value = true
})
onBeforeUnmount(() => controller.abort())

async function like() {
  if (pending.value) return
  pending.value = true
  feedback.value = null
  try {
    const result = await api.likePost(props.post.id, { signal: controller.signal })
    likes.value = result.likes
    feedback.value = 'likeSaved'
  }
  catch (error) {
    if (controller.signal.aborted) return
    feedback.value = error instanceof ApiError && error.status === 429 ? 'likeLimited'
      : error instanceof ApiError && error.status === 404 ? 'likeMissing' : 'likeError'
  }
  finally { pending.value = false }
}
</script>

<template>
  <article class="feed-post" :aria-labelledby="`post-title-${post.id}`">
    <div class="post-meta">
      <time :datetime="post.created_at">{{ postDate(post.created_at, language) }}</time>
      <span v-if="post.reading_minutes > 0">{{ t('readingTime', { minutes: post.reading_minutes }) }}</span>
      <slot name="reading-status" />
    </div>
    <h2 :id="`post-title-${post.id}`" class="post-title">
      <!-- ponytail: use NuxtLink when V2-401 adds the article route; native links avoid resolving a route that does not exist yet. -->
      <a :href="path">{{ post.title }}</a>
    </h2>
    <p v-if="post.preview_text" class="post-preview">{{ previewText(post.preview_text) }}</p>

    <figure v-if="post.preview_media && mediaSrc" class="feed-media">
      <a v-if="post.preview_media.mediaKind === 'image' || post.preview_media.mediaKind === 'gif'" :href="path" class="feed-image-frame">
        <img
          v-if="!imageFailed" ref="previewImage" :src="mediaSrc" :alt="post.preview_media.alt || post.title"
          width="960" height="540" :loading="first ? 'eager' : 'lazy'" decoding="async"
          @error="imageFailed = true"
        >
        <span v-else class="media-fallback">{{ t('mediaUnavailable') }}</span>
      </a>
      <a v-else :href="path" class="feed-attachment">
        <span>{{ t(post.preview_media.mediaKind) }}</span>
        <span v-if="post.preview_media.name" class="attachment-name">{{ post.preview_media.name }}</span>
      </a>
      <figcaption v-if="post.preview_media.caption">{{ post.preview_media.caption }}</figcaption>
    </figure>

    <div class="post-actions">
      <a class="read-post" :href="path">{{ t('openPost') }} <span aria-hidden="true">→</span></a>
      <button type="button" :disabled="pending || !hydrated" :aria-label="t('likePost', { title: post.title })" :aria-busy="pending" @click="like">
        <span aria-hidden="true">♡</span> {{ t('like') }} · <span class="like-count">{{ likes }}</span>
      </button>
      <a class="comment-count" :href="`${path}#comments`">{{ t('comments', { count: post.comment_count }) }}</a>
    </div>
    <p v-if="feedback" class="like-feedback" role="status">{{ t(feedback) }}</p>
    <a v-if="post.comment_previews.length" class="comment-preview-link" :href="`${path}#comments`" :aria-label="t('discussion', { title: post.title })">
      <ul class="comment-previews">
        <li v-for="comment in post.comment_previews.slice(0, 2)" :key="comment.id">
          <strong>{{ comment.name || t('anonymous') }}:</strong> {{ previewText(comment.content, 120) }}
        </li>
      </ul>
    </a>
  </article>
</template>
