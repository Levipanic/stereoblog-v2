import { expect, test } from '@playwright/test'

test('editor loads only after admin login and supports ordinary paragraph entry', async ({ page }) => {
  const editorChunks: string[] = []
  page.on('response', async response => {
    if (response.url().endsWith('.js') && (await response.text().catch(() => '')).includes('ProseMirror')) editorChunks.push(response.url())
  })
  await page.goto('/', { waitUntil: 'networkidle' })
  expect(editorChunks).toEqual([])
  await page.goto('/admin', { waitUntil: 'networkidle' })
  expect(editorChunks).toEqual([])
  await page.locator('#admin-secret').fill('browser-test-secret')
  await page.getByRole('button', { name: 'Войти', exact: true }).click()
  const body = page.getByRole('textbox', { name: 'Текст поста' })
  await expect(body).toBeVisible()
  await body.fill('Первый абзац')
  await body.press('End')
  await body.press('Enter')
  await page.keyboard.type('Second paragraph')
  await expect(body.locator('p')).toHaveCount(2)
  expect(editorChunks.length).toBeGreaterThan(0)
})
