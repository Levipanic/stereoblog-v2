import { expect, test } from '@playwright/test'

for (const language of ['ru', 'en']) {
  test(`public recovery and settings in ${language}`, async ({ page }) => {
    await page.addInitScript((language) => localStorage.setItem('stereoDamageLanguage', language), language)
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/')
    await expect(page.locator('html')).toHaveAttribute('lang', language)
    await expect(page.locator('a[href^="/admin"]')).toHaveCount(0)
    await page.route('**/api/v1/posts/fixture-2', route => route.fulfill({ status: 503, json: { error: { code: 'unavailable', message: 'Unavailable' } } }))
    await page.locator('a[href="/posts/fixture-2"]').first().click()
    await expect(page.getByRole('heading', { name: language === 'ru' ? 'Не удалось загрузить пост' : 'Could not load the post' })).toBeVisible()
    await page.unroute('**/api/v1/posts/fixture-2')
    await page.getByRole('button', { name: language === 'ru' ? 'Попробовать ещё раз' : 'Try again' }).click()
    await expect(page.locator('.post-body')).toBeVisible()
    for (const theme of ['dark', 'light']) {
      await page.locator('.reader-settings summary').click()
      await page.getByRole('button', { name: language === 'ru' ? (theme === 'dark' ? 'Тёмная' : 'Светлая') : (theme === 'dark' ? 'Dark' : 'Light'), exact: true }).click()
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme)
      await page.keyboard.press('Escape')
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    }
  })
}
