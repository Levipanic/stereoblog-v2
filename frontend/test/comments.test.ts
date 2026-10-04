import assert from 'node:assert/strict'
import test from 'node:test'
import { commentThread } from '../app/utils/comments.ts'
import type { Comment } from '../app/types/api.ts'

test('thread preserves replies, orphan records and cycles without recursive stack growth', () => {
  const comment = (id: number, parent_id: number | null): Comment => ({ id, parent_id, post_id: 1, name: null, content: 'text', likes: 0, created_at: '2026-01-01T00:00:00Z' })
  const result = commentThread([comment(2, 1), comment(1, null), comment(3, 2), comment(4, 999), comment(5, 6), comment(6, 5)])
  assert.deepEqual(result.map(item => item.comment.id), [1, 2, 3, 4, 5, 6])
  assert.equal(result[2]?.depth, 2)
  assert.equal(result[2]?.parent?.id, 2)
  assert.equal(commentThread(Array.from({ length: 10000 }, (_, i) => comment(i + 1, i || null))).length, 10000)
})
