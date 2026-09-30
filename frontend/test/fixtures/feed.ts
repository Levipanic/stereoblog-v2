import type { FeedPage, FeedItem, MediaKind } from '../../app/types/api.ts'

const kinds: MediaKind[] = ['image', 'audio', 'video', 'file', 'gif']
export const feedItems: FeedItem[] = Array.from({ length: 12 }, (_, index) => ({
  id: index + 1,
  slug: index === 0 ? 'привет-старый-веб' : `fixture-${index + 1}`,
  title: index === 0 ? 'Привет, старый веб' : `A long read about the personal web — ${index + 1}`,
  created_at: new Date(Date.UTC(2026, 8, 20 - index, 12)).toISOString(),
  likes: index === 0 ? 7 : 0,
  reading_minutes: 1,
  preview_text: index === 0
    ? 'Личный сайт — это место для длинных историй, музыки и разговоров. Всё важное остаётся рядом с текстом.'
    : 'A quiet place for writing, reading and sharing. This is a synthetic post for the browser checks.',
  preview_media: index < kinds.length ? {
    mediaKind: kinds[index]!, src: '/uploads/fixture.svg', alt: 'Синтетическая иллюстрация',
    caption: index === 0 ? 'Тестовая иллюстрация — данные не из production.' : '',
    name: `attachment-${index + 1}`,
  } : null,
  comment_count: index === 0 ? 2 : 0,
  comment_previews: index === 0 ? [
    { id: 10, parent_id: null, name: null, content: 'Хорошо, когда у текста есть свой дом.', created_at: '2026-09-20T13:00:00Z' },
    { id: 11, parent_id: 10, name: 'Reader', content: '<script>window.commentExecuted=true</script>', created_at: '2026-09-20T12:30:00Z' },
  ] : [],
}))

export const firstFeedPage: FeedPage = { items: feedItems.slice(0, 10), next_cursor: 'fixture-next-page' }

export const fixtureImage = `<svg xmlns="http://www.w3.org/2000/svg" width="960" height="540" viewBox="0 0 960 540">
  <rect width="960" height="540" fill="#574964"/>
  <circle cx="740" cy="140" r="70" fill="#ffdab3"/>
  <path d="M0 480L250 200L430 410L650 270L960 520V540H0Z" fill="#9f8383"/>
  <path d="M0 500L340 400L520 470L780 360L960 500V540H0Z" fill="#2a2440"/>
</svg>`
