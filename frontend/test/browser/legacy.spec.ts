import { expect, test } from '@playwright/test'

test('legacy links redirect permanently in production and retain discussion fragments', async ({ page, request }) => {
  const response = await request.get('/post.html?id=1', { maxRedirects: 0 })
  expect(response.status()).toBe(process.env.NUXT_E2E_DEV === '1' ? 302 : 301)
  expect(response.headers().location).toBe('/posts/' + encodeURIComponent('привет-старый-веб'))
  await page.goto('/post.html?id=1#comments')
  await expect(page).toHaveURL(/\/posts\/.*#comments$/)
  await expect(page.getByRole('textbox', { name: 'Комментарий', exact: true })).toBeInViewport()
  await expect(page.locator('#comments')).toBeFocused()
})

test('invalid or absent legacy IDs fail rather than redirecting home', async ({ request }) => {
  for (const suffix of ['', '?id=0', '?id=-1', '?id=1.5', '?id=1e2', '?id=1&id=2', '?id=9007199254740992', '?id=%3Cscript%3E', '?id=']) {
    const response = await request.get('/post.html' + suffix, { maxRedirects: 0 })
    expect(response.status()).toBe(400)
    expect(response.headers().location).toBeUndefined()
  }
  const missing = await request.get('/post.html?id=999999', { maxRedirects: 0 })
  expect(missing.status()).toBe(404)
  expect(missing.headers().location).toBeUndefined()
})
