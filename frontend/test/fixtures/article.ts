import type { Block } from '../../app/types/api.ts'

export const articleBlocks: Block[] = [
  { type: 'paragraph', text: 'Legacy paragraph: <script>window.articleExecuted=true</script>' },
  { type: 'paragraph', text: 'Rich content', content: [
    { type: 'text', text: 'Bold words', marks: [{ type: 'bold' }, { type: 'italic' }] },
    { type: 'text', text: ' safe link', marks: [{ type: 'link', href: 'https://example.com/' }] },
    { type: 'text', text: ' inline code', marks: [{ type: 'code' }] },
  ] },
  ...Array.from({ length: 6 }, (_, section): Block[] => [
    { type: 'heading', level: section === 0 ? 1 : 2, text: `Раздел ${section + 1}` },
    ...Array.from({ length: 3 }, (): Block => ({ type: 'paragraph', text: 'Длинная история о личном интернете. Чтение должно быть удобным на телефоне и компьютере. '.repeat(8) })),
  ]).flat(),
  { type: 'quote', text: 'Слова остаются важнее интерфейса.' },
  { type: 'divider' },
  { type: 'media', mediaKind: 'image', src: '/uploads/fixture.svg', alt: 'Горы', caption: 'Подпись изображения' },
  { type: 'media', mediaKind: 'gif', src: '/uploads/fixture.svg', spoiler: true, alt: 'Скрытые горы' },
  { type: 'media', mediaKind: 'audio', src: '/uploads/fixture.wav', name: 'Тестовый трек' },
  { type: 'media', mediaKind: 'video', src: '/uploads/fixture.mp4', name: 'Видео' },
  { type: 'media', mediaKind: 'file', src: '/uploads/fixture.txt', name: 'Заметки.txt' },
  { type: 'unknown' },
]
