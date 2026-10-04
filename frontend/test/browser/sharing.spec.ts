import { expect, test } from '@playwright/test'
const url = '/posts/' + encodeURIComponent('привет-старый-веб')

test('canonical and share metadata exist without JS and the fallback PNG is served', async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false })
  const page = await context.newPage()
  try {
    await page.goto('http://127.0.0.1:4010' + url + '?tracking=test#comments')
    const canonical = await page.locator('link[rel="canonical"]').getAttribute('href')
    expect(new URL(canonical!).pathname).toBe(url)
    expect(new URL(canonical!).search).toBe('')
    expect(new URL(canonical!).hash).toBe('')
    await expect(page.locator('meta[property="og:url"]')).toHaveAttribute('content', canonical!)
    await expect(page.locator('meta[property="og:type"]')).toHaveAttribute('content', 'article')
    await expect(page.locator('meta[property="og:title"]')).toHaveAttribute('content', 'Привет, старый веб')
    await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', /Личный сайт/)
    await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute('content', 'summary_large_image')
    await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', /\/og-default\.png$/)
    const image = await page.request.get('http://127.0.0.1:4010/og-default.png')
    expect(image.status()).toBe(200)
    expect(image.headers()['content-type']).toContain('image/png')
    await page.goto('http://127.0.0.1:4010/posts/fixture-6')
    await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', /\/og-default\.png$/)
  }
  finally { await context.close() }
})

test('Web Share and clipboard fallback use the canonical link, not tracking or fragments', async ({ page }) => {
  await page.addInitScript(() => {
    const state = window as unknown as { shared?: ShareData, copied?: string }
    Object.defineProperty(navigator, 'share', { configurable: true, value: async (data: ShareData) => { state.shared = data } })
    Object.defineProperty(navigator, 'canShare', { value: () => true })
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: async (value: string) => { state.copied = value } } })
  })
  await page.goto(url + '?tracking=yes')
  const canonical = await page.locator('link[rel="canonical"]').getAttribute('href')
  await page.getByRole('button', { name: 'Поделиться', exact: true }).click()
  expect(await page.evaluate(() => (window as unknown as { shared: ShareData }).shared.url)).toBe(canonical)
  await page.evaluate(() => Object.defineProperty(navigator, 'share', { value: undefined }))
  await page.getByRole('button', { name: 'Поделиться', exact: true }).click()
  await expect(page.locator('.share-post [role="status"]')).toHaveText('Ссылка скопирована.')
  expect(await page.evaluate(() => (window as unknown as { copied: string }).copied)).toBe(canonical)
  await page.evaluate(() => { navigator.clipboard.writeText = async () => { throw new Error('Blocked') } })
  await page.getByRole('button', { name: 'Поделиться', exact: true }).click()
  await expect(page.getByRole('textbox', { name: 'Ссылка на пост' })).toHaveValue(canonical!)
})
