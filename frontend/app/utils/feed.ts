import type { Language } from './reader-settings.ts'

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
