<script setup lang="ts">
import type { Comment } from '~/types/api'
import { ApiError } from '~/utils/api'
import { postDate } from '~/utils/feed'
import type { MessageKey } from '~/utils/i18n'

const props = defineProps<{ comment: Comment, parent?: Comment, depth: number }>()
const emit = defineEmits<{ reply: [comment: Comment] }>()
const { t, language } = useReaderSettings()
const api = usePublicApi()
const likes = ref(props.comment.likes)
const busy = ref(false)
const feedback = ref<MessageKey | null>(null)
const controller = new AbortController()
watch(() => props.comment.likes, value => { likes.value = value })
onBeforeUnmount(() => controller.abort())
async function like() {
  if (busy.value) return
  busy.value = true
  try {
    likes.value = (await api.likeComment(props.comment.id, { signal: controller.signal })).likes
    feedback.value = 'likeSaved'
  }
  catch (error) {
    if (!controller.signal.aborted) feedback.value = error instanceof ApiError && error.status === 429 ? 'likeLimited' : 'commentLikeError'
  }
  finally { busy.value = false }
}
</script>

<template>
  <li :id="`comment-${comment.id}`" class="comment-item" :style="{ marginInlineStart: `${Math.min(depth, 2) * 0.75}rem` }">
    <div class="comment-meta"><strong>{{ comment.name || t('anonymous') }}</strong><time :datetime="comment.created_at">{{ postDate(comment.created_at, language) }}</time></div>
    <a v-if="parent" class="comment-parent" :href="`#comment-${parent.id}`">{{ t('replyTo', { name: parent.name || t('anonymous') }) }}</a>
    <p class="comment-text">{{ comment.content }}</p>
    <div class="comment-actions">
      <button type="button" :disabled="busy" :aria-label="t('likeComment', { name: comment.name || t('anonymous') })" @click="like">♡ {{ likes }}</button>
      <button type="button" @click="emit('reply', comment)">{{ t('reply') }}</button>
    </div>
    <p v-if="feedback" role="status">{{ t(feedback) }}</p>
  </li>
</template>
