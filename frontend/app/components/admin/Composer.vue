<script setup lang="ts">
import type { Block, AdminPost } from '~/types/api'
import { parseDraft, type Draft } from '~/utils/draft'
import { ApiError } from '~/utils/api'
import type { MessageKey } from '~/utils/i18n'
const props = defineProps<{ post?: AdminPost }>()
const emit = defineEmits<{ saved: [post: AdminPost] }>()
const { write } = useAdminSession()
const { t } = useReaderSettings()
const id = useId()
const key = `stereoDamageDraft:${props.post?.id ?? 'new'}`
const initial = (): Draft => ({ title: props.post?.title ?? '', slug: props.post?.slug ?? '', blocks: props.post?.blocks ?? [], preview_media: props.post?.preview_media ?? null })
const draft = ref<Draft>(initial())
const initialBlocks = shallowRef<Block[]>(draft.value.blocks)
const version = ref(0)
const valid = ref(true)
const uploading = ref(false)
const ready = ref(false)
const storageError = ref(false)
const recovered = ref(false)
const saved = ref(false)
const preview = ref(false)
const busy = ref(false)
const result = shallowRef<AdminPost | null>(null)
const error = ref<MessageKey | null>(null)
const media = computed(() => draft.value.blocks.filter((block): block is Extract<Block, { type: 'media' }> => block.type === 'media'))
function chooseCover(event: Event) {
  const index = Number((event.target as HTMLSelectElement).value)
  if (index === -2) return
  const block = media.value[index]
  draft.value.preview_media = block ? { mediaKind: block.mediaKind, src: block.src, alt: block.alt ?? '', caption: block.caption ?? '', name: block.name ?? '' } : null
}
const coverIndex = computed(() => !draft.value.preview_media ? -1 : media.value.findIndex(block => block.src === draft.value.preview_media?.src) < 0 ? -2 : media.value.findIndex(block => block.src === draft.value.preview_media?.src))
let timer: ReturnType<typeof setTimeout> | undefined
function persist() {
  clearTimeout(timer)
  if (!ready.value || !valid.value || result.value) return
  try { localStorage.setItem(key, JSON.stringify(draft.value)); saved.value = true; storageError.value = false }
  catch { storageError.value = true }
}
onMounted(() => {
  try {
    const raw = localStorage.getItem(key)
    if (raw) { draft.value = parseDraft(raw); initialBlocks.value = draft.value.blocks; recovered.value = true }
  }
  catch { storageError.value = true }
  ready.value = true
  window.addEventListener('beforeunload', beforeUnload)
})
watch(draft, () => { saved.value = false; timer && clearTimeout(timer); timer = setTimeout(persist, 400) }, { deep: true })
function beforeUnload(event: BeforeUnloadEvent) {
  persist()
  if (!valid.value || storageError.value || uploading.value) { event.preventDefault(); event.returnValue = '' }
}
onBeforeUnmount(() => { persist(); window.removeEventListener('beforeunload', beforeUnload) })
function reset() {
  if (!window.confirm(t('discardDraftConfirm'))) return
  clearTimeout(timer)
  draft.value = initial()
  initialBlocks.value = draft.value.blocks
  version.value++
  valid.value = true
  recovered.value = false
  preview.value = false
  persist()
}
function update(blocks: Block[]) { draft.value.blocks = blocks; valid.value = true }
function newPost() {
  draft.value = initial(); initialBlocks.value = draft.value.blocks; version.value++
  recovered.value = false; saved.value = false; preview.value = false; result.value = null
}
defineExpose({ canClose: () => !busy.value && !uploading.value && ((!storageError.value && valid.value) || window.confirm(t('leaveDraftConfirm'))) })
async function publish() {
  if (busy.value || uploading.value || result.value) return
  error.value = null
  if (!valid.value || !draft.value.title.trim() || !draft.value.blocks.some(block => block.type === 'media' || ('text' in block && block.text.trim()))) {
    error.value = 'postInvalid'; return
  }
  persist()
  busy.value = true
  try {
    const input = { ...draft.value, ...(props.post ? { slug: props.post.slug } : {}) }
    result.value = await write<AdminPost>(props.post ? `/posts/${props.post.id}` : '/posts', input, props.post ? 'PUT' : 'POST')
    clearTimeout(timer)
    try { localStorage.removeItem(key) } catch { storageError.value = true }
    emit('saved', result.value)
  }
  catch (cause) {
    error.value = cause instanceof ApiError && cause.code === 'slug_conflict' ? 'slugConflict'
      : cause instanceof ApiError && cause.status === 400 ? 'postInvalid'
        : cause instanceof ApiError && cause.status === 429 ? 'adminLimited' : 'postSaveError'
  }
  finally { busy.value = false }
}
</script>

<template>
  <div class="composer">
    <p v-if="result" role="status">{{ t('postSaved') }} <a :href="`/posts/${encodeURIComponent(result.slug)}`">{{ result.title }}</a></p>
    <button v-if="result && !post" type="button" @click="newPost">{{ t('startAnotherPost') }}</button>
    <p v-if="error" role="alert">{{ t(error) }}</p>
    <p v-if="recovered && !result" role="status">{{ t('draftRecovered') }}</p>
    <p v-if="storageError" role="alert">{{ t('draftStorageError') }}</p>
    <p v-else-if="saved && !result" role="status">{{ t('draftSaved') }}</p>
    <p v-if="!valid" role="alert">{{ t('editorInvalid') }}</p>
    <template v-if="ready && !result">
      <fieldset v-show="!preview" :disabled="busy" class="writing-fields">
        <label :for="`${id}-title`">{{ t('editorTitle') }}</label>
        <input :id="`${id}-title`" v-model="draft.title" maxlength="160" aria-required="true">
        <LazyAdminWriter :key="version" :blocks="initialBlocks" :disabled="busy" @change="update" @invalid="valid = false" @uploading="uploading = $event" />
        <details>
          <summary>{{ t('editorSettings') }}</summary>
          <label :for="`${id}-slug`">{{ t('editorSlug') }}</label>
          <input :id="`${id}-slug`" v-model="draft.slug" :readonly="!!post" autocapitalize="none" :spellcheck="false">
          <label :for="`${id}-cover`">{{ t('coverMedia') }}</label>
          <select :id="`${id}-cover`" :value="coverIndex" @change="chooseCover">
            <option :value="-1">{{ t('automaticCover') }}</option>
            <option v-if="coverIndex === -2" :value="-2">{{ t('existingCover') }}</option>
            <option v-for="(block, index) in media" :key="index" :value="index">{{ block.name || block.src }}</option>
          </select>
        </details>
      </fieldset>
      <div class="composer-actions">
        <button type="button" :disabled="!valid || uploading || busy" @click="preview = !preview">{{ t(preview ? 'closePreview' : 'previewDraft') }}</button>
        <button type="button" :disabled="uploading || busy" @click="reset">{{ t('discardDraft') }}</button>
        <button type="button" :disabled="!valid || uploading || busy" @click="publish">{{ t(busy ? 'adminWorking' : post ? 'updatePost' : 'publishPost') }}</button>
      </div>
      <article v-if="preview" class="post-article" :aria-label="t('previewDraft')">
        <h1>{{ draft.title }}</h1>
        <PostBody :blocks="draft.blocks" preview />
      </article>
    </template>
  </div>
</template>

<style scoped>
.composer { display: grid; gap: 0.75rem; }
.writing-fields { border: 0; padding: 0; margin: 0; min-width: 0; }
select { max-width: 100%; min-height: 44px; }
input { display: block; width: 100%; min-width: 0; padding: 0.6rem; font: inherit; background: var(--surface); color: var(--text); border: 1px solid var(--border); }
summary { min-height: 44px; cursor: pointer; }
.composer-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; }
</style>
