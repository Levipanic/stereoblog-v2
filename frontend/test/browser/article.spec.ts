import { expect, test } from '@playwright/test'

const articleURL = '/posts/' + encodeURIComponent('привет-старый-веб')

test('article SSR renders canonical blocks, rich text, spoilers and all media kinds', async ({ page }, testInfo) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  const response = await page.goto(articleURL)
  const html = await response!.text()
  expect(html).toContain('Legacy paragraph: &lt;script&gt;')
  await page.screenshot({ path: testInfo.outputPath('article-light.png') })
  await expect(page.locator('.post-body strong em')).toHaveText('Bold words')
  await expect(page.locator('.post-body a[href="https://example.com/"]')).toHaveText(' safe link')
  await expect(page.locator('.post-body code')).toHaveText(' inline code')
  await expect(page.locator('.post-body blockquote')).toContainText('Слова остаются')
  await expect(page.locator('.post-body hr')).toHaveCount(1)
  await expect(page.locator('.post-body video')).toHaveAttribute('preload', 'none')
  await expect(page.locator('#persistent-audio')).toHaveAttribute('preload', 'none')
  await expect(page.locator('.post-body .inline-audio')).toContainText('Тестовый трек')
  await expect(page.getByRole('link', { name: 'Заметки.txt' })).toHaveAttribute('download', 'Заметки.txt')
  await expect(page.getByAltText('Скрытые горы')).not.toBeVisible()
  await page.getByText('Спойлер — раскрыть', { exact: true }).click()
  await expect(page.getByAltText('Скрытые горы')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  expect(errors).toEqual([])
})

test('missing article has a localized 404 and direct post navigation shows the correct content', async ({ page }) => {
  const response = await page.goto('/posts/missing-post')
  expect(response!.status()).toBe(404)
  await expect(page.getByRole('heading', { name: 'Пост не найден' })).toBeVisible()
  await page.goto('/posts/fixture-2')
  await expect(page.locator('.post-body')).toContainText('A quiet place for writing')
})
