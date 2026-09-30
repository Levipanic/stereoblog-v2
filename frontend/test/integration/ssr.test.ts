import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { createServer } from 'node:http'
import { setTimeout } from 'node:timers/promises'
import test from 'node:test'
import { firstFeedPage } from '../fixtures/feed.ts'

test('production Nuxt SSR uses its internal API base and embeds initial data', async (t) => {
  let calls = 0
  let mode: 'populated' | 'empty' | 'error' = 'populated'
  const api = createServer((req, res) => {
    assert.equal(req.url, '/api/v1/posts?limit=10')
    calls++
    res.setHeader('Content-Type', 'application/json')
    if (mode === 'error') res.writeHead(503).end(JSON.stringify({ error: { code: 'internal_error', message: 'Private backend detail' } }))
    else res.end(JSON.stringify(mode === 'empty' ? { items: [], next_cursor: null } : firstFeedPage))
  })
  api.listen(0, '127.0.0.1')
  await once(api, 'listening')
  t.after(() => { api.closeAllConnections(); api.close() })
  const address = api.address()
  assert(address && typeof address !== 'string')

  const reserve = createServer()
  reserve.listen(0, '127.0.0.1')
  await once(reserve, 'listening')
  const webAddress = reserve.address()
  assert(webAddress && typeof webAddress !== 'string')
  await new Promise<void>(resolve => reserve.close(() => resolve()))
  const child = spawn(process.execPath, ['.output/server/index.mjs'], {
    env: {
      ...process.env, HOST: '127.0.0.1', PORT: String(webAddress.port),
      NUXT_INTERNAL_API_BASE: `http://127.0.0.1:${address.port}/api/v1`,
      NUXT_PUBLIC_API_BASE: '/api/v1',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  let logs = ''
  child.stdout.on('data', chunk => { logs += chunk })
  child.stderr.on('data', chunk => { logs += chunk })
  t.after(async () => {
    if (child.exitCode === null) { const exited = once(child, 'exit'); child.kill(); await exited }
  })
  let response: Response | undefined
  for (let attempt = 0; attempt < 100; attempt++) {
    try { response = await fetch(`http://127.0.0.1:${webAddress.port}/`); break }
    catch { await setTimeout(100) }
  }
  assert(response, logs)
  assert.equal(response.status, 200, logs)
  const html = await response.text()
  assert.match(html, /<h2[^>]*>.*Привет, старый веб.*<\/h2>/)
  assert.match(html, /Личный сайт — это место/)
  assert.match(html, /Хорошо, когда у текста/)
  assert.match(html, /&lt;script&gt;window.commentExecuted=true&lt;\/script&gt;/)
  assert.match(html, /#comments/)
  assert.match(html, /__NUXT_DATA__/)
  assert.equal(calls, 1, 'SSR must fetch its initial data exactly once')
  mode = 'empty'
  const empty = await fetch(`http://127.0.0.1:${webAddress.port}/`)
  assert.equal(empty.status, 200)
  assert.match(await empty.text(), /Постов пока нет/)
  mode = 'error'
  const failed = await fetch(`http://127.0.0.1:${webAddress.port}/`)
  assert.equal(failed.status, 502)
  const errorHTML = await failed.text()
  assert.match(errorHTML, /Не удалось загрузить ленту/)
  assert.match(errorHTML, /Попробовать ещё раз/)
  // The UI must not print backend internals (Nuxt may serialize the error in its payload).
  const main = errorHTML.match(/<main\b[\s\S]*?<\/main>/)?.[0]
  assert(main)
  assert.doesNotMatch(main, /Private backend detail/)
})
