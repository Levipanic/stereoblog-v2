import { readFile } from 'node:fs/promises'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { expect, test } from '@playwright/test'

test('full backup download, retry, busy guard and expired session', async ({ page, context }, info) => {
  test.setTimeout(120_000)
  const ru = info.project.name === 'desktop'
  await page.addInitScript(language => localStorage.setItem('stereoDamageLanguage', language), ru ? 'ru' : 'en')
  await page.goto('/admin')
  await page.locator('#admin-secret').fill('browser-test-secret')
  await page.getByRole('button', { name: ru ? 'Войти' : 'Sign in', exact: true }).click()
  const area = page.locator('#admin-backup')
  const button = area.getByRole('button')
  let release!: () => void
  const held = new Promise<void>(resolve => { release = resolve })
  let calls = 0
  await page.route('**/api/v1/admin/backup', async route => {
    calls++
    expect(route.request().headers()['x-csrf-token']).toBeTruthy()
    await held
    await route.fulfill({ status: 503, json: { error: { code: 'unavailable', message: 'Unavailable' } } })
  })
  await button.click()
  await expect(button).toBeDisabled()
  await expect(area.getByRole('status')).toContainText(ru ? 'Оставьте страницу открытой' : 'Keep this page open')
  expect(calls).toBe(1)
  release()
  await expect(area.getByRole('alert')).toBeVisible()
  await page.unroute('**/api/v1/admin/backup')
  const download = page.waitForEvent('download', { timeout: 90_000 })
  let responsePromise = page.waitForResponse('**/api/v1/admin/backup')
  await button.click()
  let response = await responsePromise
  // The real server deliberately permits only one backup per minute across all sessions.
  if (response.status() === 429) {
    await expect(area.getByRole('alert')).toContainText(ru ? 'Подождите минуту' : 'Wait a minute')
    await page.waitForTimeout((Number(response.headers()['retry-after']) + 1) * 1000)
    responsePromise = page.waitForResponse('**/api/v1/admin/backup')
    await button.click()
    response = await responsePromise
  }
  expect(response.status()).toBe(200)
  const archive = await download
  expect(archive.suggestedFilename()).toBe('stereodamage-backup.zip')
  const bytes = await readFile((await archive.path())!)
  expect(bytes.length).toBe(Number(response.headers()['content-length']))
  execFileSync('go', ['test', './internal/backup', '-run', '^TestBrowserDownloadedBackup$', '-count=1'], {
    cwd: fileURLToPath(new URL('../../../backend/', import.meta.url)),
    env: { ...process.env, STEREODAMAGE_TEST_BACKUP: (await archive.path())! },
    timeout: 30_000,
  })
  expect(bytes.subarray(0, 4)).toEqual(Buffer.from([0x50, 0x4b, 0x03, 0x04]))
  for (const name of ['manifest.json', 'data/blog.db', 'uploads/fixture.svg', 'uploads/fixture.wav']) expect(bytes.includes(Buffer.from(name))).toBe(true)
  await expect(area.getByRole('link', { name: ru ? 'Сохранить ZIP' : 'Save ZIP' })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await context.clearCookies()
  await button.click()
  await expect(page.locator('#admin-secret')).toBeVisible()
  await expect(area).toHaveCount(0)
})
