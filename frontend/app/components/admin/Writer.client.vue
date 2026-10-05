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
import type { Block } from '~/types/api'
import { blocksToDocument, documentToBlocks } from '~/utils/editor'
import { safeLink } from '~/utils/content'

const props = defineProps<{ blocks: Block[] }>()
const emit = defineEmits<{ change: [blocks: Block[]], invalid: [] }>()
const { t } = useReaderSettings()
const Quote = Node.create({ name: 'quote', group: 'block', content: 'inline*', defining: true,
  parseHTML: () => [{ tag: 'blockquote' }], renderHTML: () => ['blockquote', 0],
})
const Media = Node.create({ name: 'media', group: 'block', atom: true,
  addAttributes: () => ({ block: { default: null, rendered: false } }),
  renderHTML: ({ node }) => ['figure', { 'data-editor-media': '' }, node.attrs.block?.name || t('file')],
})
const editor = useEditor({
  extensions: [Document, Paragraph, Text, Heading.configure({ levels: [1, 2, 3] }), Bold, Italic, Code,
    Link.configure({ openOnClick: false, autolink: false, isAllowedUri: url => !!safeLink(url) }), HorizontalRule, UndoRedo, Quote, Media],
  content: blocksToDocument(props.blocks),
  editorProps: { attributes: { role: 'textbox', 'aria-multiline': 'true', 'aria-label': t('editorBody') } },
  onUpdate: ({ editor }) => {
    try { emit('change', documentToBlocks(editor.getJSON())) }
    catch { emit('invalid') }
  },
})
onBeforeUnmount(() => editor.value?.destroy())
</script>

<template>
  <div class="writer">
    <slot v-if="editor" :editor="editor" />
    <EditorContent :editor="editor" />
  </div>
</template>

<style scoped>
.writer :deep(.tiptap) { min-height: 22rem; padding: 1rem; border: 1px solid var(--border); background: var(--surface); white-space: pre-wrap; overflow-wrap: anywhere; }
.writer :deep(.tiptap p) { margin-block: 0.6rem; }
.writer :deep(figure) { border: 1px dashed var(--border); padding: 1rem; }
</style>
