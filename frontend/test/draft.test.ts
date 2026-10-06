import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parseDraft } from '../app/utils/draft.ts'

test('draft recovery validates canonical text, media and preview without accepting damaged storage', () => {
  const draft = { title: 'Draft', slug: '', blocks: [{ type: 'paragraph', text: 'Keep me' }, { type: 'media', mediaKind: 'audio', src: '/uploads/music.mp3', spoiler: true }], preview_media: null }
  assert.deepEqual(parseDraft(JSON.stringify(draft)), draft)
  for (const raw of ['{', 'null', JSON.stringify({ ...draft, blocks: [{ type: 'unknown' }] }), JSON.stringify({ ...draft, preview_media: { src: 'javascript:bad' } }), JSON.stringify({ ...draft, title: 42 })]) {
    assert.throws(() => parseDraft(raw))
  }
})
