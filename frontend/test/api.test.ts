import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { once } from 'node:events'
import test from 'node:test'

import { ApiError, createPublicApi } from '../app/utils/api.ts'

test('client speaks the Go contract over HTTP, including pending and typed errors', async (t) => {
  const requests: string[] = []
  const server = createServer(async (req, res) => {
    requests.push(`${req.method} ${req.url}`)
    res.setHeader('Content-Type', 'application/json')
    if (req.url?.endsWith('/comments') && req.method === 'POST') {
      let body = ''
      for await (const chunk of req) body += chunk
      assert.equal(JSON.parse(body).challenge_token, 'challenge')
      res.writeHead(202).end(JSON.stringify({ ok: true, status: 'pending' }))
    }
    else if (req.url?.endsWith('/likes')) {
      res.writeHead(429, { 'Retry-After': '12' }).end(JSON.stringify({ error: { code: 'like_cooldown', message: 'Wait.' } }))
    }
    else if (req.url?.includes('/posts?')) {
      const url = new URL(req.url, 'http://test')
      assert.equal(url.searchParams.get('cursor'), 'opaque+/=?&')
      res.end(JSON.stringify({ items: [], next_cursor: null }))
    }
    else res.writeHead(404).end(JSON.stringify({ error: { code: 'post_not_found', message: 'Missing.' } }))
  })
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  t.after(() => { server.closeAllConnections(); server.close() })
  const address = server.address()
  assert(address && typeof address !== 'string')
  const api = createPublicApi(`http://127.0.0.1:${address.port}/api/v1/`)
  assert.deepEqual(await api.feed({ limit: 10, cursor: 'opaque+/=?&' }), { items: [], next_cursor: null })
  assert.deepEqual(await api.createComment(1, { content: 'Hello', challenge_token: 'challenge' }), { ok: true, status: 'pending' })
  await assert.rejects(api.post('Привет /?'), error => error instanceof ApiError && error.status === 404 && error.code === 'post_not_found')
  await assert.rejects(api.likePost(1), error => error instanceof ApiError && error.status === 429 && error.retryAfter === 12)
  assert(requests.some(path => path.includes(encodeURIComponent('Привет /?'))))
  assert.equal(requests.length, 4, 'writes/errors must not be automatically retried')
})

test('browser base stays same-origin, abort signal is forwarded and errors are safe', async () => {
  const controller = new AbortController()
  const api = createPublicApi('/api/v1/', async (url, options) => {
    assert.equal(url, '/api/v1/posts')
    assert.equal(options?.signal, controller.signal)
    assert.equal(options?.credentials, 'same-origin')
    return Response.json({ items: [], next_cursor: null })
  })
  await api.feed({}, { signal: controller.signal })
  const html = createPublicApi('/api/v1', async () => new Response('<html>proxy details</html>', { status: 502 }))
  await assert.rejects(html.feed(), error => error instanceof ApiError && error.status === 502 && !error.message.includes('proxy details'))
  const malformed = createPublicApi('/api/v1', async () => new Response('bad JSON'))
  await assert.rejects(malformed.feed(), error => error instanceof ApiError && error.code === 'invalid_response')
  const network = createPublicApi('/api/v1', async () => { throw new TypeError('connection refused') })
  await assert.rejects(network.feed(), error => error instanceof ApiError && error.status === 0)
  controller.abort()
  const cancelled = createPublicApi('/api/v1', async (_url, options) => { options?.signal?.throwIfAborted(); return Response.json({}) })
  await assert.rejects(cancelled.feed({}, { signal: controller.signal }), { name: 'AbortError' })
})
