// Offline asset generation only; the committed PNG is served statically.
import { readFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import { chromium } from '@playwright/test'

const browser = await chromium.launch()
try {
  const page = await browser.newPage({ viewport: { width: 1200, height: 630 }, deviceScaleFactor: 1 })
  const svg = await readFile(new URL('../public/og-default.svg', import.meta.url), 'utf8')
  await page.setContent(`<style>body{margin:0}</style>${svg}`)
  await page.screenshot({ path: fileURLToPath(new URL('../public/og-default.png', import.meta.url)) })
}
finally { await browser.close() }
