import { expect, test } from '@playwright/test'
import { firstFeedPage } from '../fixtures/feed'

test('SSR feed hydrates without API fanout, safely renders previews and likes in place', async ({ page }, testInfo) => {
  const errors: string[] = []
  const requests: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  page.on('console', message => { if (message.type() === 'warning' || message.type() === 'error') errors.push(message.text()) })
  page.on('request', request => { if (request.url().includes('/api/v1/')) requests.push(request.url()) })
  const response = await page.goto('/')
  const html = await response!.text()
  expect(html).toContain('Привет, старый веб')
  expect(html).toContain('Личный сайт — это место')
  await expect(page.locator('.feed-post')).toHaveCount(10)
  const first = page.locator('.feed-post').first()
  const button = first.getByRole('button', { name: 'Поставить лайк: Привет, старый веб', exact: true })
  await expect(button).toBeEnabled()
  expect(requests).toEqual([])
  await expect(first.locator('.comment-count')).toHaveText('Комментарии: 2')
  await expect(first.locator('.comment-count')).toHaveAttribute('href', '/posts/%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82-%D1%81%D1%82%D0%B0%D1%80%D1%8B%D0%B9-%D0%B2%D0%B5%D0%B1#comments')
  expect(await first.locator('.comment-previews script').count()).toBe(0)
  await expect(first.locator('img')).toHaveAttribute('loading', 'eager')
  await expect(page.locator('.feed-post img').nth(1)).toHaveAttribute('loading', 'lazy')
  await expect(first.locator('img')).toHaveJSProperty('complete', true)
  expect(await first.locator('img').evaluate(image => (image as HTMLImageElement).naturalWidth)).toBeGreaterThan(0)
  await expect(page.locator('audio')).toHaveCount(1)
  await expect(page.locator('audio')).not.toHaveAttribute('src')
  const count = Number(await first.locator('.like-count').textContent())
  // Different projects share the backend, so like a different fixture in each project.
  const likedPost = testInfo.project.name === 'mobile' ? page.locator('.feed-post').nth(1) : first
  const before = Number(await likedPost.locator('.like-count').textContent())
  await likedPost.locator('.post-actions button').click()
  await expect(likedPost.locator('.like-count')).toHaveText(String(before + 1))
  await expect(likedPost.getByRole('status')).toHaveText('Спасибо! Лайк сохранён.')
  expect(page.url()).toMatch(/\/$/)
  expect(requests).toHaveLength(1)
  expect(requests[0]).toMatch(/\/likes$/)
  expect(count).toBeGreaterThanOrEqual(7)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.evaluate(() => scrollTo(0, 0))
  await page.screenshot({ path: testInfo.outputPath('feed-light.png'), fullPage: true })
  await page.locator('summary').click()
  await page.getByRole('button', { name: 'English', exact: true }).click()
  await page.getByRole('button', { name: 'Dark', exact: true }).click()
  await page.keyboard.press('Escape')
  await expect(first.locator('.comment-count')).toHaveText('Comments: 2')
  await expect(first.locator('.post-meta')).toContainText('min read')
  await page.evaluate(() => scrollTo(0, 0))
  await page.screenshot({ path: testInfo.outputPath('feed-dark.png'), fullPage: true })
  expect(errors).toEqual([])
})

test('like errors remain separate from navigation and image fallback retains its frame', async ({ page }) => {
  await page.route('**/uploads/fixture.svg', route => route.fulfill({ status: 404, body: '' }))
  await page.route('**/api/v1/posts/*/likes', route => route.fulfill({
    status: 429, contentType: 'application/json', headers: { 'Retry-After': '10' },
    body: JSON.stringify({ error: { code: 'like_cooldown', message: 'Wait.' } }),
  }))
  await page.goto('/')
  const first = page.locator('.feed-post').first()
  await expect(first.getByText('Превью недоступно')).toBeVisible()
  const frame = await first.locator('.feed-image-frame').boundingBox()
  expect(frame!.height).toBeGreaterThan(100)
  await expect(first.getByRole('button')).toBeEnabled()
  const before = await first.locator('.like-count').textContent()
  await first.getByRole('button').click()
  await expect(first.getByRole('status')).toHaveText('Подождите перед следующим лайком.')
  await expect(first.locator('.like-count')).toHaveText(before!)
  expect(page.url()).toMatch(/\/$/)
})

test('client-side loading, empty and retry states are usable', async ({ page }) => {
  // /admin is client-rendered and already exists; returning via the brand exercises a client feed request.
  let mode: 'error' | 'empty' | 'populated' = 'error'
  await page.route('**/api/v1/posts?limit=10', async (route) => {
    await new Promise(resolve => setTimeout(resolve, 200))
    await route.fulfill({
      status: mode === 'error' ? 503 : 200, contentType: 'application/json',
      body: JSON.stringify(mode === 'error' ? { error: { code: 'internal_error', message: 'Unavailable' } }
        : mode === 'empty' ? { items: [], next_cursor: null } : firstFeedPage),
    })
  })
  await page.goto('/admin')
  await page.getByRole('link', { name: 'StereoDamage — Главная', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('Не удалось загрузить ленту.')
  mode = 'empty'
  await page.getByRole('button', { name: 'Попробовать ещё раз' }).click()
  await expect(page.getByRole('status')).toHaveText('Загрузка ленты…')
  await expect(page.getByRole('status')).toHaveText('Постов пока нет.')
})
