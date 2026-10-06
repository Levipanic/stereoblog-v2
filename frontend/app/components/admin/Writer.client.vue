<script setup lang="ts">
import { EditorContent, useEditor } from '@tiptap/vue-3'
import { Node } from '@tiptap/core'
import Document from '@tiptap/extension-document'
import Paragraph from '@tiptap/extension-paragraph'
import Text from '@tiptap/extension-text'
import Heading from '@tiptap/extension-heading'
import Bold from '@tiptap/extension-bold'
import Italic from '@tiptap/extension-italic'
import Code from '@tiptap/extension-code'
import Link from '@tiptap/extension-link'
import HorizontalRule from '@tiptap/extension-horizontal-rule'
import { UndoRedo } from '@tiptap/extensions'
import type { Block, MediaKind } from '~/types/api'
import { blocksToDocument, documentToBlocks } from '~/utils/editor'
import { safeLink } from '~/utils/content'

const props = defineProps<{ blocks: Block[] }>()
const emit = defineEmits<{ change: [blocks: Block[]], invalid: [] }>()
const { t } = useReaderSettings()
const invalid = ref(false)
const { write } = useAdminSession()
const uploading = ref(false)
const uploadError = ref(false)
const picker = useTemplateRef<HTMLInputElement>('picker')
let insertAt = 0
const selectedMedia = computed(() => editor.value?.isActive('media') ? editor.value.getAttributes('media').block as Extract<Block, { type: 'media' }> | null : null)
const Quote = Node.create({ name: 'quote', group: 'block', content: 'inline*', defining: true,
  parseHTML: () => [{ tag: 'blockquote' }], renderHTML: () => ['blockquote', 0],
})
const Media = Node.create({ name: 'media', group: 'block', atom: true,
  addAttributes: () => ({ block: { default: null, rendered: false } }),
  renderHTML: ({ node }) => ['figure', { 'data-editor-media': '' }, node.attrs.block?.name || t('editorMedia')],
})
const editor = useEditor({
  extensions: [Document, Paragraph, Text, Heading.configure({ levels: [1, 2, 3] }), Bold, Italic, Code,
    Link.configure({ openOnClick: false, autolink: false, isAllowedUri: url => !!safeLink(url) }), HorizontalRule, UndoRedo, Quote, Media],
  content: blocksToDocument(props.blocks),
  editorProps: { attributes: { role: 'textbox', 'aria-multiline': 'true', 'aria-label': t('editorBody') } },
  onUpdate: ({ editor }) => {
    try { emit('change', documentToBlocks(editor.getJSON())); invalid.value = false }
    catch { invalid.value = true; emit('invalid') }
  },
})
onBeforeUnmount(() => editor.value?.destroy())
function chooseMedia() {
  insertAt = editor.value?.state.selection.from ?? 0
  picker.value?.click()
}
async function upload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !editor.value || uploading.value) return
  uploading.value = true
  uploadError.value = false
  editor.value.setEditable(false)
  try {
    const body = new FormData()
    body.append('file', file)
    const media = await write<{ url: string, original_name: string, media_kind: MediaKind }>('/uploads', body)
    editor.value.chain().focus().insertContentAt(insertAt, { type: 'media', attrs: { block: {
      type: 'media', src: media.url, mediaKind: media.media_kind, name: media.original_name,
    } } }).run()
  }
  catch { uploadError.value = true }
  finally { uploading.value = false; editor.value?.setEditable(true); input.value = '' }
}
function metadata(key: 'alt' | 'caption' | 'name' | 'spoiler', value: string | boolean) {
  editor.value?.chain().focus().updateAttributes('media', { block: { ...selectedMedia.value, [key]: value } }).run()
}
function moveMedia(direction: -1 | 1) {
  if (!editor.value || !selectedMedia.value) return
  const { state, view } = editor.value
  const { from, to, $from } = state.selection
  const node = state.doc.nodeAt(from)
  if (!node) return
  const adjacent = direction < 0 ? $from.nodeBefore : state.doc.resolve(to).nodeAfter
  if (!adjacent) return
  const target = direction < 0 ? from - adjacent.nodeSize : from + adjacent.nodeSize
  view.dispatch(state.tr.delete(from, to).insert(target, node))
}
function link() {
  const value = window.prompt(t('editorLinkPrompt'), editor.value?.getAttributes('link').href ?? '')
  if (value === null) return
  if (!value) { editor.value?.chain().focus().extendMarkRange('link').unsetLink().run(); return }
  if (!safeLink(value)) { invalid.value = true; return }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: value }).run()
}
</script>

<template>
  <div class="writer">
    <fieldset v-if="editor" class="writer-tools" :disabled="uploading" :aria-label="t('editorBody')">
      <button type="button" :aria-pressed="editor.isActive('bold')" @click="editor.chain().focus().toggleBold().run()">{{ t('editorBold') }}</button>
      <button type="button" :aria-pressed="editor.isActive('italic')" @click="editor.chain().focus().toggleItalic().run()">{{ t('editorItalic') }}</button>
      <button type="button" :aria-pressed="editor.isActive('code')" @click="editor.chain().focus().toggleCode().run()">{{ t('editorCode') }}</button>
      <button type="button" :aria-pressed="editor.isActive('link')" @click="link">{{ t('editorLink') }}</button>
      <button type="button" @click="editor.chain().focus().setParagraph().run()">{{ t('editorParagraph') }}</button>
      <button v-for="level in ([1, 2, 3] as const)" :key="level" type="button" :aria-label="`${t('editorHeading')} ${level}`" :aria-pressed="editor.isActive('heading', { level })" @click="editor.chain().focus().toggleHeading({ level }).run()">H{{ level }}</button>
      <button type="button" :aria-pressed="editor.isActive('quote')" @click="editor.chain().focus().toggleNode('quote', 'paragraph').run()">{{ t('editorQuote') }}</button>
      <button type="button" @click="editor.chain().focus().setHorizontalRule().run()">+ {{ t('editorDivider') }}</button>
      <button type="button" @click="chooseMedia">+ {{ t('insertMedia') }}</button>
      <button type="button" :disabled="!editor.can().undo()" @click="editor.chain().focus().undo().run()">{{ t('editorUndo') }}</button>
      <button type="button" :disabled="!editor.can().redo()" @click="editor.chain().focus().redo().run()">{{ t('editorRedo') }}</button>
    </fieldset>
    <input ref="picker" type="file" hidden @change="upload">
    <p v-if="uploading" role="status">{{ t('uploadingMedia') }}</p>
    <p v-if="uploadError" role="alert">{{ t('adminError') }}</p>
    <fieldset v-if="selectedMedia" :disabled="uploading" class="media-settings">
      <legend>{{ t('insertMedia') }}</legend>
      <label v-for="field in (['name', 'alt', 'caption'] as const)" :key="field">{{ t(field === 'name' ? 'mediaName' : field === 'alt' ? 'mediaAlt' : 'mediaCaption') }}
        <input :value="selectedMedia[field] ?? ''" @change="metadata(field, ($event.target as HTMLInputElement).value)">
      </label>
      <label><input type="checkbox" :checked="selectedMedia.spoiler" @change="metadata('spoiler', ($event.target as HTMLInputElement).checked)">{{ t('mediaSpoiler') }}</label>
      <button type="button" @click="moveMedia(-1)">{{ t('moveUp') }}</button>
      <button type="button" @click="moveMedia(1)">{{ t('moveDown') }}</button>
      <button type="button" @click="editor?.chain().focus().deleteSelection().run()">{{ t('removeMedia') }}</button>
    </fieldset>
    <p v-if="invalid" role="alert">{{ t('editorInvalid') }}</p>
    <slot v-if="editor" :editor="editor" />
    <EditorContent :editor="editor" />
  </div>
</template>

<style scoped>
.writer :deep(.tiptap) { min-height: 22rem; padding: 1rem; border: 1px solid var(--border); background: var(--surface); white-space: pre-wrap; overflow-wrap: anywhere; }
.writer :deep(.tiptap p) { margin-block: 0.6rem; }
.writer-tools { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-block: 0.75rem; border: 0; padding: 0; min-width: 0; }
.media-settings { min-width: 0; }
.media-settings label { display: block; margin-block: 0.5rem; }
.media-settings input:not([type='checkbox']) { display: block; width: 100%; min-width: 0; }
.writer-tools [aria-pressed='true'] { font-weight: bold; background: var(--surface-alt); }
.writer :deep(figure) { border: 1px dashed var(--border); padding: 1rem; }
</style>
