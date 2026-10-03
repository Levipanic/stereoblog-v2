import assert from 'node:assert/strict'
import test from 'node:test'
import { articleHeadings, hasTableOfContents, safeLink } from '../app/utils/content.ts'
import { articleBlocks } from './fixtures/article.ts'

test('heading anchors are deterministic, unique and unaffected by paragraphs before headings', () => {
  const headings = articleHeadings(articleBlocks)
  assert.equal(headings.length, 6)
  assert.equal(new Set(headings.map(h => h.id)).size, 6)
  assert.equal(articleHeadings([{ type: 'paragraph', text: 'Added' }, ...articleBlocks])[0]?.id, headings[0]?.id)
  assert(hasTableOfContents(articleBlocks))
  assert(!hasTableOfContents([{ type: 'heading', level: 1, text: 'Short' }]))
})

test('inline links reject active schemes, credentials and disguised external URLs', () => {
  for (const value of ['javascript:alert(1)', 'data:text/html,x', '//evil.test', '/\\evil', 'https://u:p@example.com', 'https://example.com/\n']) assert.equal(safeLink(value), null)
  for (const value of ['https://example.com/a', 'mailto:owner@example.com', '/posts/example', '#heading']) assert.equal(safeLink(value), value)
})
