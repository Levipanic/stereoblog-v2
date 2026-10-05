import { test } from 'node:test'
import assert from 'node:assert/strict'
import { blocksToDocument, documentToBlocks } from '../app/utils/editor.ts'
import type { Block } from '../app/types/api.ts'

test('legacy and rich blocks preserve text, marks, media and spoilers through editor round trips', () => {
  const blocks: Block[] = [
    { type: 'paragraph', text: 'Legacy\nsecond line' },
    { type: 'heading', level: 2, text: 'Heading' },
    { type: 'quote', text: 'Quote' }, { type: 'divider' },
    { type: 'paragraph', text: 'obsolete fallback', content: [{ type: 'text', text: 'Rich', marks: [{ type: 'bold' }, { type: 'italic' }, { type: 'code' }, { type: 'link', href: '/posts/example' }] }] },
    { type: 'media', mediaKind: 'audio', src: '/uploads/a.mp3', spoiler: true, caption: 'Caption', alt: 'Alt', name: 'Track' },
  ]
  const result = documentToBlocks(blocksToDocument(blocks))
  assert.equal((result[0] as { text: string }).text, 'Legacy\nsecond line')
  assert.equal((result[4] as { text: string }).text, 'Rich')
  assert.deepEqual(result[5], blocks[5])
  assert.deepEqual(documentToBlocks(blocksToDocument(result)), result)
  assert.throws(() => blocksToDocument([{ type: 'unknown' }]))
  assert.throws(() => documentToBlocks({ type: 'doc', content: [{ type: 'script' }] }))
  assert.throws(() => documentToBlocks({ type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'bad', marks: [{ type: 'link', attrs: { href: 'javascript:alert(1)' } }] }] }] }))
})
