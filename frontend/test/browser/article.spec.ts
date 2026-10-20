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
  await expect(page.locator('.media-spoiler:not(.media-spoiler-revealed)')).toHaveCount(1)
  const spoiler = page.locator('.media-spoiler')
  const spoilerContent = spoiler.locator('.media-spoiler-content')
  const spoilerImage = spoilerContent.locator('.image-open')
  const reveal = spoiler.locator('.media-spoiler-overlay')
  await expect(reveal).toBeVisible()
  await expect(spoilerContent).toHaveAttribute('inert', '')
  await reveal.focus()
  await page.keyboard.press('Shift+Tab')
  expect(await spoilerImage.evaluate(element => element === document.activeElement)).toBe(false)
  await reveal.focus()
  await page.keyboard.press('Enter')
  await expect(page.locator('.media-spoiler-revealed')).toHaveCount(1)
  await expect(page.locator('.media-spoiler-overlay')).toHaveCount(0)
  await expect(spoilerContent).not.toHaveAttribute('inert', '')
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

test('article, discussion and TOC fit V2-411 viewport/theme targets', async ({ page }) => {
  const toc = page.getByRole('navigation', { name: 'Оглавление' })
  for (const theme of ['light', 'dark']) {
    for (const width of [360, 768, 1280]) {
      await page.setViewportSize({ width, height: 900 })
      await page.goto(articleURL)
      await page.locator('html').evaluate((element, value) => element.setAttribute('data-theme', value), theme)
      await expect(toc).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `${theme} at ${width}px`).toBe(true)
    }
  }
})
