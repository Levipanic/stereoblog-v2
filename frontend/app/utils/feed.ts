import type { Language } from './reader-settings.ts'
import type { FeedPage } from '../types/api.ts'

export function appendFeedPage(current: FeedPage, next: FeedPage): FeedPage {
  const ids = new Set(current.items.map(item => item.id))
  const added = next.items.filter(item => {
    if (ids.has(item.id)) return false
    ids.add(item.id)
    return true
  })
  return { items: [...current.items, ...added], next_cursor: next.next_cursor }
}

export function postPath(slug: string) {
  return `/posts/${encodeURIComponent(slug)}`
}

export function postDate(value: string, language: Language) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return ''
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-RU' : 'en-GB', {
    day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC',
  }).format(date)
}

export function previewText(value: string, length = 300) {
  const text = value.replace(/\s+/g, ' ').trim()
  const characters = Array.from(text)
  return characters.length > length ? `${characters.slice(0, length).join('').trimEnd()}…` : text
}

export function localMediaSource(value: string): string | null {
  if (!value.startsWith('/uploads/') || /[\\\u0000-\u001f\u007f]/.test(value)) return null
  try {
    const url = new URL(value, 'https://media.invalid')
    const path = decodeURIComponent(url.pathname)
    if (url.origin !== 'https://media.invalid' || !path.startsWith('/uploads/') || url.search || url.hash) return null
    if (path.includes('\\') || path.split('/').slice(1).some(part => !part || part === '.' || part === '..')) return null
    return url.pathname
  }
  catch { return null }
}
