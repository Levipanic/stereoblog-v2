export function safeLink(value: string): string | null {
  if (!value || /[\s\\\u0000-\u001f\u007f]/.test(value) || value.startsWith('//')) return null
  try {
    const url = new URL(value, 'https://content.invalid')
    return ['http:', 'https:', 'mailto:'].includes(url.protocol) && !url.username && !url.password ? value : null
  }
  catch { return null }
}
import type { Block } from '../types/api.ts'

export function articleHeadings(blocks: Block[]) {
  let ordinal = 0
  return blocks.flatMap((block, index) => {
    if (block.type !== 'heading') return []
    const base = block.text.trim().toLowerCase().replace(/[^\p{L}\p{N}]+/gu, '-').replace(/^-+|-+$/g, '').slice(0, 48)
    return [{ index, id: `section-${base || 'heading'}-${++ordinal}`, text: block.text, level: block.level }]
  })
}

export function hasTableOfContents(blocks: Block[]) {
  return articleHeadings(blocks).length >= 3 && blocks.reduce((length, block) => length + ('text' in block ? block.text.length : 0), 0) >= 1000
}
