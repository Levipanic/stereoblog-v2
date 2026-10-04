import assert from 'node:assert/strict'
import test from 'node:test'
import { postMetadata } from '../app/utils/sharing.ts'
import type { Post } from '../app/types/api.ts'
import { feedItems } from './fixtures/feed.ts'

test('metadata uses configured canonical URLs and only suitable local preview images', () => {
  const post: Post = { ...feedItems[0]!, blocks: [] }
  const data = postMetadata(post, 'https://blog.example/path?ignored=1', 'StereoDamage')
  assert.equal(data.url, 'https://blog.example/posts/' + encodeURIComponent(post.slug))
  assert.equal(data.image, 'https://blog.example/og-default.png') // SVG is unsuitable for social crawlers.
  post.preview_media = { mediaKind: 'image', src: '/uploads/cover.jpg', alt: 'Cover', caption: '', name: '' }
  assert.equal(postMetadata(post, 'https://blog.example', 'StereoDamage').image, 'https://blog.example/uploads/cover.jpg')
  post.preview_media.src = '//evil.example/image.jpg'
  assert.equal(postMetadata(post, 'https://blog.example', 'StereoDamage').image, data.image)
  post.preview_media = null
  post.preview_text = ''
  assert.equal(postMetadata(post, 'https://blog.example', 'StereoDamage').description, post.title)
})
