import { expect, test } from '@playwright/test'

test('Back restores loaded pages, cursor, card state and scroll without a feed refetch', async ({ page }) => {
  const requests: string[] = []
  await page.route('**/api/v1/posts/14/likes', route => route.fulfill({ json: { success: true, post_id: 14, likes: 100 } }))
  page.on('request', request => { if (/\/api\/v1\/posts(?:\?|$)/.test(request.url())) requests.push(request.url()) })
  await page.goto('/')
  await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
  await expect(page.locator('.feed-post')).toHaveCount(20)
  const link = page.locator('#post-title-14 a')
  await link.scrollIntoViewIfNeeded()
  const card = page.getByRole('article', { name: 'A long read about the personal web — 14', exact: true })
  await card.getByRole('button').click()
  await expect(card.locator('.like-count')).toHaveText('100')
  const top = await page.evaluate(() => scrollY)
  const fetchCount = requests.length
  await page.evaluate(() => { (window as unknown as Record<string, unknown>).navigationMarker = 'same-app' })
  await link.click()
  await expect(page).toHaveURL(/\/posts\/fixture-14$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('A long read about the personal web — 14')
  await expect(page).toHaveTitle('A long read about the personal web — 14 — StereoDamage')
  await page.goBack()
  await expect(page.locator('.feed-post')).toHaveCount(20)
  await expect(page).toHaveTitle('StereoDamage — Посты')
  await expect(card.locator('.like-count')).toHaveText('100')
  await expect(card.getByRole('status')).toHaveText('Спасибо! Лайк сохранён.')
  await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(top, 0)
  expect(requests).toHaveLength(fetchCount)
  expect(await page.evaluate(() => (window as unknown as Record<string, unknown>).navigationMarker)).toBe('same-app')

  await link.click()
  await expect(page).toHaveURL(/\/posts\/fixture-14$/)
  await page.getByRole('button', { name: 'Назад к ленте', exact: true }).click()
  await expect(page.locator('.feed-post')).toHaveCount(20)
  await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(top, 0)
  expect(requests).toHaveLength(fetchCount)
  await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
  await expect(page.locator('.feed-post')).toHaveCount(30)
  expect(requests).toHaveLength(fetchCount + 1)

  await page.evaluate(() => scrollTo(0, 0))
  await page.reload()
  await expect(page.locator('.feed-post')).toHaveCount(10)
})

test('leaving during pagination cancels the request and Back can resume from the same cursor', async ({ page }) => {
  let started = 0
  let release: () => void = () => {}
  const gate = new Promise<void>(resolve => { release = resolve })
  await page.route('**/api/v1/posts?*cursor=*', async (route) => {
    started++
    if (started === 1) {
      await gate
      await route.abort().catch(() => {})
    }
    else await route.continue()
  })
  try {
    await page.goto('/')
    await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
    await expect.poll(() => started).toBe(1)
    await page.locator('#post-title-6 a').click()
    await expect(page).toHaveURL(/\/posts\/fixture-6$/)
    release()
    await page.goBack()
    await expect(page.locator('.feed-post')).toHaveCount(10)
    await expect(page.locator('.feed-pagination [role="alert"]')).toHaveCount(0)
    await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
    await expect(page.locator('.feed-post')).toHaveCount(20)
    expect(started).toBe(2)
  }
  finally { release() }
})

test('direct article entry has a safe Back action and grid survives navigation', async ({ page }) => {
  await page.goto('/posts/fixture-2')
  await page.getByRole('button', { name: 'Назад к ленте', exact: true }).click()
  await expect(page).toHaveURL(/\/$/)
  await page.locator('summary').click()
  await page.getByRole('button', { name: 'Плитки', exact: true }).click()
  await page.keyboard.press('Escape')
  await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
  await expect(page.locator('.feed-post')).toHaveCount(20)
  const link = page.locator('#post-title-13 a')
  await link.scrollIntoViewIfNeeded()
  const top = await page.evaluate(() => scrollY)
  await link.click()
  await expect(page).toHaveURL(/\/posts\/fixture-13$/)
  await page.goBack()
  await expect(page.locator('.feed-list')).toHaveClass(/feed-grid/)
  await expect(page.locator('.feed-post')).toHaveCount(20)
  await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(top, 0)
})
