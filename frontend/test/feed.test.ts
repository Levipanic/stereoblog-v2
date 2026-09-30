import assert from 'node:assert/strict'
import test from 'node:test'
import { appendFeedPage, localMediaSource, postDate, postPath, previewText } from '../app/utils/feed.ts'
import { feedItems } from './fixtures/feed.ts'

test('appending cursor pages deduplicates across and within batches without replacing existing cards', () => {
  const first = { items: feedItems.slice(0, 10), next_cursor: 'next' }
  const next = { items: [feedItems[0]!, feedItems[10]!, feedItems[10]!, feedItems[11]!], next_cursor: null }
  const result = appendFeedPage(first, next)
  assert.deepEqual(result.items.map(item => item.id), Array.from({ length: 12 }, (_, i) => i + 1))
  assert.equal(result.items[0], first.items[0])
  assert.equal(first.items.length, 10)
  assert.equal(result.next_cursor, null)
})

test('feed formatting is deterministic across SSR/client and safely encodes links', () => {
  assert.equal(postPath('Привет /?#'), `/posts/${encodeURIComponent('Привет /?#')}`)
  assert.equal(postDate('2026-01-01T23:30:00-02:00', 'en'), '2 Jan 2026')
  assert.equal(postDate('invalid', 'ru'), '')
  assert.equal(previewText('  One\n two  three '), 'One two three')
  assert.equal(previewText('😀😀😀', 2), '😀😀…')
})

test('feed media stays inside the uploads namespace', () => {
  assert.equal(localMediaSource('/uploads/Фото.jpg'), '/uploads/%D0%A4%D0%BE%D1%82%D0%BE.jpg')
  for (const src of ['javascript:alert(1)', '//evil.test/x', '/uploads/../.env', '/uploads/%2e%2e/secret', '/uploads/a/%2e%2e%2fsecret', '/uploads/a\\b', '/uploads/a?x=1', '/uploads/%', '/uploads/']) {
    assert.equal(localMediaSource(src), null, src)
  }
})
