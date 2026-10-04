import { expect, test } from '@playwright/test'
const url = '/posts/' + encodeURIComponent('привет-старый-веб') + '#comments'

test('feed replies and direct hashes land at the composer and survive late layout changes', async ({ page }) => {
  await page.goto('/')
  await page.locator('.comment-count').first().click()
  await expect(page).toHaveURL(/#comments$/)
  await expect(page.locator('#comments')).toBeFocused()
  await expect(page.getByRole('textbox', { name: 'Комментарий', exact: true })).toBeInViewport()
  const top = await page.locator('#comments').evaluate(node => Math.round(node.getBoundingClientRect().top))
  await page.evaluate(() => { document.querySelector<HTMLElement>('.post-article')!.style.paddingTop = '600px' })
  await expect.poll(() => page.locator('#comments').evaluate(node => Math.round(node.getBoundingClientRect().top))).toBeCloseTo(top, 0)
  await page.mouse.wheel(0, -300)
  await expect.poll(() => page.locator('#comments').evaluate(node => node.getBoundingClientRect().top)).toBeGreaterThan(top + 100)
  await page.evaluate(() => { document.querySelector<HTMLElement>('.post-article')!.style.paddingTop = '800px' })
  // Browser scroll anchoring may compensate for the insertion; the explicit comments anchor must stay released.
  await expect.poll(() => page.locator('#comments').evaluate(node => node.getBoundingClientRect().top)).toBeGreaterThan(top + 100)
  await page.goto(url)
  await expect(page.getByRole('textbox', { name: 'Комментарий', exact: true })).toBeInViewport()
  await expect(page.locator('#comments')).toBeFocused()
})
