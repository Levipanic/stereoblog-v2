import type { JSONContent } from '@tiptap/core'
import type { Block, InlineNode, InlineMark } from '../types/api.ts'
import { safeLink } from './content.ts'

function inline(nodes: JSONContent[] = []): InlineNode[] {
  return nodes.map(node => {
    if (node.type !== 'text' || typeof node.text !== 'string' || node.content) throw new Error('Unsupported inline node')
    const marks: InlineMark[] = (node.marks ?? []).map(mark => {
      if (mark.type === 'link') {
        const href = mark.attrs?.href
        if (typeof href !== 'string' || !safeLink(href)) throw new Error('Unsafe link')
        return { type: 'link', href }
      }
      if (mark.type === 'bold' || mark.type === 'italic' || mark.type === 'code') return { type: mark.type }
      throw new Error('Unsupported mark')
    })
    return { type: 'text', text: node.text, ...(marks.length ? { marks } : {}) }
  })
}

export function documentToBlocks(doc: JSONContent): Block[] {
  if (doc.type !== 'doc' || !Array.isArray(doc.content)) throw new Error('Invalid document')
  return doc.content.map((node): Block => {
    if (node.type === 'horizontalRule') return { type: 'divider' }
    if (node.type === 'media') {
      const block = node.attrs?.block as Block | undefined
      if (!block || block.type !== 'media' || !['image', 'gif', 'video', 'audio', 'file'].includes(block.mediaKind)
        || typeof block.src !== 'string' || !block.src.startsWith('/uploads/') || !safeLink(block.src)) throw new Error('Invalid media')
      return structuredClone(block)
    }
    if (!['paragraph', 'heading', 'quote'].includes(node.type ?? '')) throw new Error('Unsupported block')
    const content = inline(node.content)
    const text = content.map(node => node.text).join('')
    if (node.type === 'heading') {
      const level = node.attrs?.level
      if (level !== 1 && level !== 2 && level !== 3) throw new Error('Invalid heading')
      return { type: 'heading', level, text, content }
    }
    return { type: node.type as 'paragraph' | 'quote', text, content }
  })
}

export function blocksToDocument(blocks: Block[]): JSONContent {
  const doc: JSONContent = { type: 'doc', content: blocks.map(block => {
    if (block.type === 'unknown') throw new Error('Unsupported block')
    if (block.type === 'divider') return { type: 'horizontalRule' }
    if (block.type === 'media') return { type: 'media', attrs: { block: structuredClone(block) } }
    if (!('text' in block)) throw new Error('Unsupported block')
    return {
      type: block.type,
      ...(block.type === 'heading' ? { attrs: { level: block.level } } : {}),
      content: (block.content ?? [{ type: 'text', text: block.text }]).filter(node => node.text !== '').map(node => ({
        type: node.type, text: node.text,
        marks: node.marks?.map(mark => mark.type === 'link' ? { type: 'link', attrs: { href: mark.href } } : { type: mark.type }),
      })),
    }
  }) }
  if (!doc.content?.length) doc.content = [{ type: 'paragraph' }]
  documentToBlocks(doc)
  return doc
}
