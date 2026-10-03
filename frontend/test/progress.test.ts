import assert from 'node:assert/strict'
import test from 'node:test'
import { parseProgress, advanceProgress } from '../app/utils/progress.ts'
test('v1 progress is validated, monotonic and completed at 90 percent', () => {
  const old = parseProgress('{"1":{"progress":0.5,"scroll_y":800,"opened_at":"old"}}')['1']
  assert.equal(old?.scroll_y, 800)
  const next = advanceProgress(old, 0.2, 120, 'now')
  assert.equal(next.progress, 0.5)
  assert.equal(next.opened_at, 'old')
  assert.equal(advanceProgress(next, 0.9, 900).completed, true)
  for (const value of ['null', '[]', '{broken', '{"1":{"progress":"invalid"}}']) assert.deepEqual(parseProgress(value), {})
})
