import { messages, type MessageKey } from '~/utils/i18n'
import { readReaderSettings, readerKeys, type FeedView, type Language, type Theme } from '~/utils/reader-settings'

export function useReaderSettings() {
  // Mirror only language in a cookie so repeat requests can SSR the chosen UI locale.
  const languageCookie = useCookie<string>(readerKeys.language, { path: '/', sameSite: 'lax', maxAge: 31536000 })
  const language = useState<Language>('reader-language', () => languageCookie.value === 'en' ? 'en' : 'ru')
  const theme = useState<Theme>('reader-theme', () => 'light')
  const feedView = useState<FeedView>('reader-feed-view', () => 'list')
  const explicitTheme = useState('reader-explicit-theme', () => false)

  function save(key: string, value: string) {
    if (import.meta.client) { try { localStorage.setItem(key, value) } catch { /* Settings still work in memory. */ } }
  }

  function setLanguage(value: Language) {
    language.value = value
    languageCookie.value = value
    save(readerKeys.language, value)
  }

  function applyTheme(value: Theme) {
    theme.value = value
    if (import.meta.client) document.documentElement.dataset.theme = value
  }

  function setTheme(value: Theme) {
    explicitTheme.value = true
    applyTheme(value)
    save(readerKeys.theme, value)
  }

  function restore() {
    if (!import.meta.client) return
    const systemTheme = () => window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    try {
      const saved = readReaderSettings(localStorage)
      explicitTheme.value = !!saved.theme
      applyTheme(saved.theme ?? systemTheme())
      if (saved.language) setLanguage(saved.language)
      feedView.value = saved.feedView
    }
    catch { applyTheme(systemTheme()) }

    const query = window.matchMedia('(prefers-color-scheme: dark)')
    const update = () => { if (!explicitTheme.value) applyTheme(systemTheme()) }
    query.addEventListener('change', update)
    return () => query.removeEventListener('change', update)
  }

  function t(key: MessageKey, values: Record<string, string | number> = {}) {
    return messages[language.value][key].replace(/\{(\w+)\}/g, (match, name: string) => String(values[name] ?? match))
  }

  return { language, theme, feedView, setLanguage, setTheme, restore, t }
}
