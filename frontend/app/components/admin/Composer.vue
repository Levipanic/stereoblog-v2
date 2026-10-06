<script setup lang="ts">
import type { Block, Post } from '~/types/api'
import { parseDraft, type Draft } from '~/utils/draft'
const props = defineProps<{ post?: Post }>()
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
let timer: ReturnType<typeof setTimeout> | undefined
function persist() {
  clearTimeout(timer)
  if (!ready.value || !valid.value) return
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
</script>

<template>
  <div class="composer">
    <p v-if="recovered" role="status">{{ t('draftRecovered') }}</p>
    <p v-if="storageError" role="alert">{{ t('draftStorageError') }}</p>
    <p v-else-if="saved" role="status">{{ t('draftSaved') }}</p>
    <p v-if="!valid" role="alert">{{ t('editorInvalid') }}</p>
    <template v-if="ready">
      <div v-show="!preview">
        <label :for="`${id}-title`">{{ t('editorTitle') }}</label>
        <input :id="`${id}-title`" v-model="draft.title" maxlength="160" aria-required="true">
        <LazyAdminWriter :key="version" :blocks="initialBlocks" @change="update" @invalid="valid = false" @uploading="uploading = $event" />
        <details>
          <summary>{{ t('editorSettings') }}</summary>
          <label :for="`${id}-slug`">{{ t('editorSlug') }}</label>
          <input :id="`${id}-slug`" v-model="draft.slug" :readonly="!!post" autocapitalize="none" :spellcheck="false">
          <p>{{ t('editorPreview') }}</p>
        </details>
      </div>
      <div class="composer-actions">
        <button type="button" :disabled="!valid || uploading" @click="preview = !preview">{{ t(preview ? 'closePreview' : 'previewDraft') }}</button>
        <button type="button" :disabled="uploading" @click="reset">{{ t('discardDraft') }}</button>
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
input { display: block; width: 100%; min-width: 0; padding: 0.6rem; font: inherit; background: var(--surface); color: var(--text); border: 1px solid var(--border); }
summary { min-height: 44px; cursor: pointer; }
.composer-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; }
</style>
