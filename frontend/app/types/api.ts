export type MediaKind = 'image' | 'gif' | 'video' | 'audio' | 'file'

export interface PreviewMedia {
  mediaKind: MediaKind
  src: string
  alt: string
  caption: string
  name: string
}

export type InlineMark = { type: 'bold' | 'italic' | 'code' } | { type: 'link', href: string }
export interface InlineNode {
  type: 'text'
  text: string
  marks?: InlineMark[]
}

export type Block =
  | { type: 'paragraph' | 'quote', text: string, content?: InlineNode[] }
  | { type: 'heading', level: 1 | 2 | 3, text: string, content?: InlineNode[] }
  | { type: 'divider' | 'unknown' }
  | { type: 'media', mediaKind: MediaKind, src: string, spoiler?: boolean, name?: string, alt?: string, caption?: string }

export interface CommentPreview {
  id: number
  parent_id: number | null
  name: string | null
  content: string
  created_at: string
}

export interface Comment extends CommentPreview {
  post_id: number
  likes: number
}

export interface PostSummary {
  id: number
  slug: string
  title: string
  created_at: string
  likes: number
  reading_minutes: number
  preview_text: string
  preview_media: PreviewMedia | null
}

export interface FeedItem extends PostSummary {
  comment_count: number
  comment_previews: CommentPreview[]
}

export interface FeedPage {
  items: FeedItem[]
  next_cursor: string | null
}

export interface Post extends PostSummary {
  blocks: Block[]
}

export type AdminPost = Pick<Post, 'id' | 'title' | 'slug' | 'blocks' | 'preview_media' | 'created_at'>

export interface CommentChallenge {
  token: string
  honeypot_field: string
  expires_in_seconds: number
}

export interface CommentInput {
  content: string
  name?: string
  parent_id?: number | null
  challenge_token: string
  website?: string
  [honeypot: `hp_${string}`]: string
}

export interface CommentSubmission {
  ok: boolean
  status: 'visible' | 'pending'
}

export interface PostLike { success: boolean, post_id: number, likes: number }
export interface CommentLike { success: boolean, comment_id: number, likes: number }
export interface IDResolution { id: number, slug: string }
export interface Health { status: string }
export interface ApiErrorBody { error: { code: string, message: string } }
