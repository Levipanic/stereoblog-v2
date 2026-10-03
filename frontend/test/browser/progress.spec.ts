import { expect, test } from '@playwright/test'
test('v1 progress resumes, is throttled, completes and updates the cached feed', async ({ page }) => {
  await page.addInitScript(() => {
    if (!localStorage.getItem('stereoDamageReadingProgress')) localStorage.setItem('stereoDamageReadingProgress', JSON.stringify({ 1: { progress: 0.4, scroll_y: 1000, opened_at: '2024-01-01T00:00:00Z', completed: false } }))
    const original = Storage.prototype.setItem
    const state = window as unknown as { progressWrites: number }
    state.progressWrites = 0
    Storage.prototype.setItem = function (key, value) {
      if (key === 'stereoDamageReadingProgress') state.progressWrites++
      return original.call(this, key, value)
    }
  })
  await page.goto('/')
  await expect(page.locator('.feed-post').first().locator('.reading-status')).toHaveText('Начато (40%)')
  await page.locator('.post-title a').first().click()
  await page.getByRole('button', { name: 'Продолжить с 40%' }).click()
  await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(1000, 0)
  const before = await page.evaluate(() => (window as unknown as { progressWrites: number }).progressWrites)
  await page.evaluate(() => { for (let i = 0; i < 100; i++) window.dispatchEvent(new Event('scroll')) })
  await page.waitForTimeout(600)
  const after = await page.evaluate(() => (window as unknown as { progressWrites: number }).progressWrites)
  expect(after - before).toBeLessThanOrEqual(2)
  await page.evaluate(() => scrollTo(0, document.body.scrollHeight))
  await expect.poll(() => page.evaluate(() => JSON.parse(localStorage.getItem('stereoDamageReadingProgress')!)['1'].completed)).toBe(true)
  await page.getByRole('button', { name: 'Назад к ленте' }).click()
  await expect(page.locator('.feed-post').first().locator('.reading-status')).toHaveText('Прочитано')
})
