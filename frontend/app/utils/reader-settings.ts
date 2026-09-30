export type Language = 'ru' | 'en'
export type Theme = 'light' | 'dark'
export type FeedView = 'list' | 'grid'

export const readerKeys = {
  language: 'stereoDamageLanguage',
  theme: 'stereoDamageTheme',
  feedView: 'stereoDamageFeedView',
} as const

export function readReaderSettings(storage: Pick<Storage, 'getItem'>) {
  const read = (key: string) => { try { return storage.getItem(key) } catch { return null } }
  const language = read(readerKeys.language)
  const theme = read(readerKeys.theme)
  return {
    language: language === 'ru' || language === 'en' ? language : undefined,
    theme: theme === 'light' || theme === 'dark' ? theme : undefined,
    feedView: read(readerKeys.feedView) === 'grid' ? 'grid' : 'list',
  } as { language?: Language, theme?: Theme, feedView: FeedView }
}

// Runs in <head> before styles/first paint; Vue never owns this DOM attribute.
export const themeBootstrap = `(() => {
  let theme;
  try { theme = localStorage.getItem('${readerKeys.theme}'); } catch {}
  document.documentElement.dataset.theme = theme === 'light' || theme === 'dark'
    ? theme : (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
})();`
