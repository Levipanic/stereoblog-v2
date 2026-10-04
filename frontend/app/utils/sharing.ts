import type { Post } from '../types/api.ts'
import { localMediaSource, postPath, previewText } from './feed.ts'

export function postMetadata(post: Post, siteUrl: string, siteName: string) {
  const preview = post.preview_media
  const src = preview && ['image', 'gif'].includes(preview.mediaKind) ? localMediaSource(preview.src) : null
  const image = src && /\.(png|jpe?g|gif|webp)$/i.test(src) ? src : '/og-default.png'
  const text = post.preview_text || post.blocks.flatMap(block => 'text' in block ? [block.text] : []).join(' ') || post.title
  return {
    url: new URL(postPath(post.slug), siteUrl).href,
    title: post.title,
    description: previewText(text, 180),
    image: new URL(image, siteUrl).href,
    imageAlt: image === src ? preview?.alt || post.title : siteName,
  }
}
