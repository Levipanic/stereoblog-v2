import { expect, test } from '@playwright/test'

test('grid preference persists, shares cards and stays single-column on phones', async ({ page }, testInfo) => {
  const requests: string[] = []
  page.on('request', request => { if (request.url().includes('/api/v1/posts')) requests.push(request.url()) })
  await page.goto('/')
  await page.locator('summary').click()
  await page.getByRole('button', { name: 'Плитки', exact: true }).click()
  await expect(page.locator('.feed-list')).toHaveClass(/feed-grid/)
  await expect(page.locator('.feed-post')).toHaveCount(10)
  expect(requests).toEqual([])
  const first = await page.locator('.feed-list > li').nth(0).boundingBox()
  const second = await page.locator('.feed-list > li').nth(1).boundingBox()
  if (testInfo.project.name === 'desktop') {
    expect(second!.y).toBe(first!.y)
    expect(second!.x).toBeGreaterThan(first!.x)
  }
  else {
    expect(second!.y).toBeGreaterThan(first!.y)
    expect(second!.x).toBe(first!.x)
  }
  await page.reload()
  await expect(page.locator('.feed-list')).toHaveClass(/feed-grid/)
  expect(await page.evaluate(() => localStorage.getItem('stereoDamageFeedView'))).toBe('grid')
  await page.locator('summary').click()
  await page.getByRole('button', { name: 'Лента', exact: true }).click()
  await expect(page.locator('.feed-list')).not.toHaveClass(/feed-grid/)
  expect(await page.evaluate(() => localStorage.getItem('stereoDamageFeedView'))).toBe('list')
  await expect(page.locator('.feed-post').first().getByRole('button')).toBeEnabled()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})
