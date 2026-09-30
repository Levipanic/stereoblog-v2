import assert from 'node:assert/strict'
import test from 'node:test'
import { runInNewContext } from 'node:vm'

import { readReaderSettings, readerKeys, themeBootstrap } from '../app/utils/reader-settings.ts'
import { messages } from '../app/utils/i18n.ts'

test('reads v1 preferences without modifying progress or obsolete settings', () => {
  const stored = { stereoDamageLanguage: 'en', stereoDamageTheme: 'dark', stereoDamageFeedView: 'grid', stereoDamageSmoothScroll: 'on' }
  const read: string[] = []
  const storage = { getItem(key: string) { read.push(key); return stored[key as keyof typeof stored] ?? null } }
  assert.deepEqual(readReaderSettings(storage), { language: 'en', theme: 'dark', feedView: 'grid' })
  assert.deepEqual(read, [readerKeys.language, readerKeys.theme, readerKeys.feedView])
  const unavailable = { getItem() { throw new Error('Storage blocked') } }
  assert.deepEqual(readReaderSettings(unavailable), { language: undefined, theme: undefined, feedView: 'list' })
  assert.deepEqual(readReaderSettings({ getItem: () => 'invalid' }), { language: undefined, theme: undefined, feedView: 'list' })
})

test('pre-paint bootstrap chooses saved/system theme even with blocked storage', () => {
  for (const stored of ['light', 'dark', 'invalid', null, 'blocked']) {
    const document = { documentElement: { dataset: {} as Record<string, string> } }
    runInNewContext(themeBootstrap, {
      document,
      localStorage: { getItem() { if (stored === 'blocked') throw new Error(); return stored } },
      window: { matchMedia: () => ({ matches: true }) },
    })
    assert.equal(document.documentElement.dataset.theme, stored === 'light' ? 'light' : 'dark')
  }
})

test('both locales cover the same interface strings', () => {
  assert.deepEqual(Object.keys(messages.en).sort(), Object.keys(messages.ru).sort())
  assert(Object.values(messages.en).every(Boolean))
})
