import { expect, test } from '@playwright/test'
import { feedItems } from '../fixtures/feed'

test('scroll loads each cursor once, retries explicitly and never duplicates cards or concurrent requests', async ({ page }) => {
  let failed = false
  let active = 0
  let maximum = 0
  const cursors: string[] = []
  await page.route('**/api/v1/posts?*cursor=*', async (route) => {
    cursors.push(new URL(route.request().url()).searchParams.get('cursor')!)
    maximum = Math.max(maximum, ++active)
    await new Promise(resolve => setTimeout(resolve, 100))
    if (!failed) {
      failed = true
      await route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":{"code":"unavailable","message":"Retry"}}' })
    }
    else {
      const response = await route.fetch()
      const body = await response.json()
      body.items.unshift(feedItems[0]) // Simulate an overlapping page.
      await route.fulfill({ response, json: body })
    }
    active--
  })
  await page.goto('/')
  await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
  await expect(page.locator('.feed-pagination [role="alert"]')).toBeVisible()
  await expect(page.locator('.feed-post')).toHaveCount(10)
  expect(cursors).toHaveLength(1)
  await page.getByRole('button', { name: 'Попробовать ещё раз' }).click()
  await expect(page.locator('.feed-post')).toHaveCount(20)
  for (const count of [30, 32]) {
    await page.locator('.feed-pagination').scrollIntoViewIfNeeded()
    await expect(page.locator('.feed-post')).toHaveCount(count)
  }
  await expect(page.getByText('Вы дошли до начала ленты.', { exact: true })).toBeVisible()
  expect(cursors).toHaveLength(4)
  expect(cursors[0]).toBe(cursors[1])
  expect(new Set(cursors).size).toBe(3)
  expect(maximum).toBe(1)
  expect(await page.locator('.post-title').evaluateAll(nodes => new Set(nodes.map(node => node.id)).size)).toBe(32)
  await expect(page.locator('.feed-pagination button')).toHaveCount(0)
})

test('keyboard load-more works without IntersectionObserver', async ({ page }) => {
  await page.addInitScript(() => { Object.defineProperty(window, 'IntersectionObserver', { value: undefined }) })
  await page.goto('/')
  await expect(page.locator('.feed-post')).toHaveCount(10)
  const button = page.getByRole('button', { name: 'Загрузить ещё', exact: true })
  await button.focus()
  await button.press('Enter')
  await expect(page.locator('.feed-post')).toHaveCount(20)
  await expect(button).toBeEnabled()
})
