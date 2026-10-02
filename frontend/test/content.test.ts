import assert from 'node:assert/strict'
import test from 'node:test'
import { safeLink } from '../app/utils/content.ts'

test('inline links reject active schemes, credentials and disguised external URLs', () => {
  for (const value of ['javascript:alert(1)', 'data:text/html,x', '//evil.test', '/\\evil', 'https://u:p@example.com', 'https://example.com/\n']) assert.equal(safeLink(value), null)
  for (const value of ['https://example.com/a', 'mailto:owner@example.com', '/posts/example', '#heading']) assert.equal(safeLink(value), value)
})
