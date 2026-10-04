<script setup lang="ts">
import type { Comment, CommentChallenge, CommentInput, FeedPage } from '~/types/api'
import { ApiError } from '~/utils/api'
import { commentThread } from '~/utils/comments'
import type { MessageKey } from '~/utils/i18n'

const props = defineProps<{ postId: number }>()
const { t } = useReaderSettings()
const api = usePublicApi()
const { data: feed } = useNuxtData<FeedPage>('public-feed')
const { data: comments, error, status, refresh } = await useAsyncData(`comments:${props.postId}`, (_app, { signal }) => api.comments(props.postId, { signal }))
const thread = computed(() => commentThread(comments.value ?? []))
const name = ref('')
const content = ref('')
const website = ref('')
const honeypot = ref('')
const reply = ref<Comment | null>(null)
const challenge = ref<CommentChallenge | null>(null)
const sending = ref(false)
const ready = ref(false)
const feedback = ref<MessageKey | null>(null)
const textarea = useTemplateRef<HTMLTextAreaElement>('textarea')
const form = useTemplateRef<HTMLFormElement>('form')
const controller = new AbortController()
let expires = 0
let preparing: Promise<void> | undefined
onMounted(() => { ready.value = true })
onBeforeUnmount(() => controller.abort())

async function prepare() {
  if (challenge.value && Date.now() < expires - 1000) return
  if (preparing) return preparing
  preparing = (async () => {
    const value = await api.challenge(props.postId, { signal: controller.signal })
    if (!/^hp_[a-f0-9]{12}$/i.test(value.honeypot_field)) throw new Error('Invalid challenge')
    challenge.value = value
    expires = Date.now() + value.expires_in_seconds * 1000
  })().finally(() => { preparing = undefined })
  return preparing
}
function prepareOnFocus() { void prepare().catch(() => { if (!controller.signal.aborted) feedback.value = 'commentPrepareError' }) }
function selectReply(comment: Comment) {
  if (sending.value) return
  reply.value = comment
  textarea.value?.focus({ preventScroll: true })
  form.value?.scrollIntoView({ block: 'start', behavior: 'instant' })
}
async function submit() {
  if (sending.value || !content.value.trim()) return
  sending.value = true
  feedback.value = null
  try {
    await prepare()
    const token = challenge.value!
    const input: CommentInput = {
      name: name.value.trim(), content: content.value.trim(), parent_id: reply.value?.id ?? null,
      website: website.value, challenge_token: token.token,
    }
    input[token.honeypot_field as `hp_${string}`] = honeypot.value
    const result = await api.createComment(props.postId, input, { signal: controller.signal })
    feedback.value = result.status === 'pending' ? 'commentPending' : 'commentPosted'
    content.value = ''
    reply.value = null
    if (result.status === 'visible') {
      await refresh()
      if (feed.value && comments.value && !error.value) {
        const visible = comments.value
        feed.value = { ...feed.value, items: feed.value.items.map(item => item.id === props.postId
          ? { ...item, comment_count: visible.length, comment_previews: visible.slice(-2).reverse() } : item) }
      }
    }
  }
  catch (error) {
    if (!controller.signal.aborted) feedback.value = error instanceof ApiError && error.status === 429 ? 'commentLimited'
      : error instanceof ApiError && error.status === 400 ? 'commentInvalid' : 'commentSendError'
  }
  finally {
    challenge.value = null
    sending.value = false
  }
}
</script>

<template>
  <section id="comments" class="post-comments" aria-labelledby="comments-title" tabindex="-1">
    <h2 id="comments-title">{{ t('discussionTitle') }}</h2>
    <form ref="form" class="comment-form" :aria-busy="sending" @submit.prevent="submit" @focusin="prepareOnFocus">
      <fieldset :disabled="sending">
        <p v-if="reply" class="reply-context">{{ t('replyTo', { name: reply.name || t('anonymous') }) }} <button type="button" @click="reply = null">{{ t('cancelReply') }}</button></p>
        <label for="comment-name">{{ t('commentName') }}</label>
        <input id="comment-name" v-model="name" name="name" maxlength="80" autocomplete="name" :placeholder="t('anonymous')">
        <label for="comment-content">{{ t('commentText') }}</label>
        <textarea id="comment-content" ref="textarea" v-model="content" name="content" rows="4" maxlength="1000" required />
        <div class="comment-honeypot" aria-hidden="true">
          <input v-model="website" name="website" tabindex="-1" autocomplete="off" aria-label="Website">
          <input v-if="challenge" v-model="honeypot" :name="challenge.honeypot_field" tabindex="-1" autocomplete="off" aria-label="Leave empty">
        </div>
        <button type="submit" :disabled="!ready || !content.trim()">{{ t(sending ? 'commentSending' : 'commentSend') }}</button>
      </fieldset>
      <p v-if="feedback" class="comment-feedback" role="status">{{ t(feedback) }}</p>
    </form>
    <div v-if="error" role="alert"><p>{{ t('commentsLoadError') }}</p><button type="button" @click="refresh()">{{ t('retry') }}</button></div>
    <p v-else-if="status === 'pending'" role="status">{{ t('commentsLoading') }}</p>
    <ol v-else-if="thread.length" class="comment-thread">
      <CommentItem v-for="item in thread" :key="item.comment.id" v-bind="item" @reply="selectReply" />
    </ol>
    <p v-else>{{ t('commentsEmpty') }}</p>
  </section>
</template>
