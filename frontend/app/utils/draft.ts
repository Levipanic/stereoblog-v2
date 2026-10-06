import type { Block, PreviewMedia } from '../types/api.ts'
import { blocksToDocument } from './editor.ts'

export interface Draft { title: string, slug: string, blocks: Block[], preview_media: PreviewMedia | null }

export function parseDraft(raw: string): Draft {
  const value = JSON.parse(raw)
  if (!value || typeof value.title !== 'string' || value.title.length > 160
    || typeof value.slug !== 'string' || !Array.isArray(value.blocks)) throw new Error('Invalid draft')
  blocksToDocument(value.blocks)
  if (value.preview_media !== null) {
    const media = value.preview_media
    if (!media || ['alt', 'caption', 'name'].some(key => typeof media[key] !== 'string')) throw new Error('Invalid preview')
    blocksToDocument([{ ...media, type: 'media' }])
  }
  return { title: value.title, slug: value.slug, blocks: value.blocks, preview_media: value.preview_media }
}
