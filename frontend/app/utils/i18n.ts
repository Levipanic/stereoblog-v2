const ru = {
  home: 'Главная',
  skipToContent: 'К содержимому',
  settings: 'Настройки',
  language: 'Язык',
  theme: 'Тема',
  light: 'Светлая',
  dark: 'Тёмная',
  posts: 'Посты',
  chronological: 'Обратная хронология',
  preparingFeed: 'Готовим новую ленту.',
}

export type MessageKey = keyof typeof ru

const en: Record<MessageKey, string> = {
  home: 'Home',
  skipToContent: 'Skip to content',
  settings: 'Settings',
  language: 'Language',
  theme: 'Theme',
  light: 'Light',
  dark: 'Dark',
  posts: 'Posts',
  chronological: 'Newest first',
  preparingFeed: 'The new feed is on its way.',
}

export const messages = { ru, en }
