import { expect, test } from '@playwright/test'

for (const language of ['ru', 'en']) {
  test(`admin login, reload, CSRF, expiry and logout (${language})`, async ({ page, context }) => {
    await page.addInitScript(language => localStorage.setItem('stereoDamageLanguage', language), language)
    const login = language === 'ru' ? 'Войти' : 'Sign in'
    const logout = language === 'ru' ? 'Выйти' : 'Sign out'
    await page.goto('/admin')
    await expect(page.locator('#admin-secret')).toBeVisible()
    expect((await page.request.get('/api/v1/admin/posts')).status()).toBe(401)
    await page.locator('#admin-secret').fill('wrong-secret')
    await page.getByRole('button', { name: login, exact: true }).click()
    await expect(page.getByRole('alert')).toContainText(language === 'ru' ? 'Неверный пароль' : 'Incorrect password')
    await expect(page.locator('#admin-secret')).toHaveValue('')
    async function signIn() {
      await page.locator('#admin-secret').fill('browser-test-secret')
      await page.getByRole('button', { name: login, exact: true }).click()
      await expect(page.getByRole('button', { name: logout, exact: true })).toBeVisible()
    }
    await signIn()
    await expect(page.locator('.admin-home nav a')).toHaveCount(4)
    expect((await context.cookies()).find(cookie => cookie.name === 'admin_session')?.httpOnly).toBe(true)
    expect(await page.evaluate(() => JSON.stringify(localStorage))).not.toContain('browser-test-secret')
    expect((await page.request.post('/api/v1/admin/logout', { data: {} })).status()).toBe(403)
    await page.reload()
    await expect(page.getByRole('button', { name: logout, exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    const request = page.waitForRequest(request => request.url().endsWith('/admin/logout') && request.method() === 'POST')
    await page.getByRole('button', { name: logout, exact: true }).click()
    expect((await request).headers()['x-csrf-token']).toBeTruthy()
    await expect(page.locator('#admin-secret')).toBeVisible()
    expect((await page.request.get('/api/v1/admin/posts')).status()).toBe(401)
    await signIn()
    await context.clearCookies()
    await page.getByRole('button', { name: logout, exact: true }).click()
    await expect(page.locator('#admin-secret')).toBeVisible()
    await expect(page.getByRole('alert')).toContainText(language === 'ru' ? 'Сессия истекла' : 'session expired')
  })
}

test('session check failure offers retry without exposing the admin shell', async ({ page }) => {
  await page.route('**/api/v1/admin/session', route => route.abort())
  await page.goto('/admin')
  await expect(page.getByRole('alert')).toBeVisible()
  await expect(page.locator('.admin-home')).toHaveCount(0)
  await page.unroute('**/api/v1/admin/session')
  await page.getByRole('button', { name: 'Попробовать ещё раз' }).click()
  await expect(page.locator('#admin-secret')).toBeVisible()
})
