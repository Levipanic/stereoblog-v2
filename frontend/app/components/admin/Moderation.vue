<script setup lang="ts">
import type { ModerationOverview } from '~/types/api'
const emit = defineEmits<{ count: [value: number | null] }>()
const { t } = useReaderSettings()
const { read, write } = useAdminSession()
const data = ref<ModerationOverview | null>(null)
const busy = ref(false)
const error = ref(false)
const done = ref(false)
async function overview() {
  data.value = null
  emit('count', null)
  data.value = await read<ModerationOverview>('/moderation')
  emit('count', data.value.pending_comments.length)
}
async function refresh() {
  if (busy.value) return
  busy.value = true; error.value = false
  try { await overview() }
  catch { error.value = true }
  finally { busy.value = false }
}
async function act(path: string, remove = false, confirmDelete = false) {
  if (busy.value || (confirmDelete && !window.confirm(t('deleteThreadConfirm')))) return
  busy.value = true; error.value = false; done.value = false
  try {
    await write(path, {}, remove ? 'DELETE' : 'POST')
    done.value = true
    await overview()
  }
  catch { error.value = true }
  finally { busy.value = false }
}
onMounted(refresh)
</script>

<template>
  <div class="moderation">
    <button type="button" :disabled="busy" @click="refresh">{{ t('refreshAdmin') }}</button>
    <p v-if="busy" role="status">{{ t('adminWorking') }}</p>
    <p v-if="error" role="alert">{{ t('adminError') }}</p>
    <p v-if="done" role="status">{{ t('moderationDone') }}</p>
    <template v-if="data">
      <h3>{{ t('pendingComments') }} ({{ data.pending_comments.length }})</h3>
      <p>{{ t('moderationBounded') }}</p>
      <p v-if="!data.pending_comments.length">{{ t('noPendingComments') }}</p>
      <article v-for="comment in data.pending_comments" :key="comment.id" :data-comment-id="comment.id">
        <p><strong>{{ comment.name || t('anonymous') }}</strong> · <time>{{ comment.created_at }}</time></p>
        <p class="comment-content">{{ comment.content }}</p>
        <p v-if="comment.moderation_reason">{{ comment.moderation_reason }}</p>
        <a :href="`/post.html?id=${comment.post_id}#comments`" target="_blank" rel="noopener">{{ t('moderationContext') }} #{{ comment.post_id }}</a>
        <p v-if="comment.parent_id">{{ t('parentComment') }} #{{ comment.parent_id }}</p>
        <div class="moderation-actions">
          <button type="button" :disabled="busy" @click="act(`/comments/${comment.id}/approve`)">{{ t('approveComment') }}</button>
          <button type="button" :disabled="busy" @click="act(`/comments/${comment.id}/reject`)">{{ t('rejectComment') }}</button>
          <button type="button" :disabled="busy" @click="act(`/comments/${comment.id}`, true, true)">{{ t('deleteThread') }}</button>
        </div>
      </article>
      <details>
        <summary>{{ t('moderationHistory') }}</summary>
        <h3>{{ t('activeMutes') }}</h3>
        <p v-if="!data.mutes.length">{{ t('noActiveMutes') }}</p>
        <article v-for="mute in data.mutes" :key="mute.id" :data-mute-id="mute.id">
          <p>{{ t('anonymousHash') }}: <code>{{ mute.ip_hash_short }}</code></p>
          <p>{{ t('mutedUntil') }}: <time>{{ mute.muted_until }}</time> · {{ mute.reason }}</p>
          <button type="button" :disabled="busy" @click="act(`/comment-mutes/${mute.id}`, true)">{{ t('unmuteCommenter') }}</button>
        </article>
        <h3>{{ t('recentAttempts') }}</h3>
        <p v-if="!data.attempts.length">{{ t('noAttempts') }}</p>
        <article v-for="attempt in data.attempts" :key="attempt.id" :data-attempt-id="attempt.id">
          <p><time>{{ attempt.created_at }}</time> · {{ attempt.status }} · {{ attempt.reason }}</p>
          <p>{{ t('anonymousHash') }}: <code>{{ attempt.ip_hash_short }}</code></p>
          <p v-if="attempt.content" class="comment-content">{{ attempt.content }}</p>
          <a v-if="attempt.post_id" :href="`/post.html?id=${attempt.post_id}#comments`" target="_blank" rel="noopener">{{ t('moderationContext') }} #{{ attempt.post_id }}</a>
        </article>
      </details>
    </template>
  </div>
</template>

<style scoped>
article { padding-block: 0.75rem; border-bottom: 1px solid var(--border); overflow-wrap: anywhere; }
.comment-content { white-space: pre-wrap; }
.moderation-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: 0.5rem; }
details { margin-top: 1rem; }
summary { cursor: pointer; min-height: 44px; }
</style>
