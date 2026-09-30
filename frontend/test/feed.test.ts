import assert from 'node:assert/strict'
import test from 'node:test'
import { localMediaSource, postDate, postPath, previewText } from '../app/utils/feed.ts'

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
