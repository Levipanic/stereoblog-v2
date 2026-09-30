// Disposable real Go + production Nuxt instance behind a same-origin test proxy.
import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { mkdtempSync, mkdirSync, rmSync } from 'node:fs'
import { createServer, request } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { setTimeout } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'

const directory = mkdtempSync(join(tmpdir(), 'stereodamage-browser-'))
const children: ChildProcess[] = []
const proxy = createServer((req, res) => {
  const port = req.url?.startsWith('/api/') || req.url?.startsWith('/uploads/') ? 4011 : 4012
  const upstream = request({ hostname: '127.0.0.1', port, path: req.url, method: req.method, headers: req.headers }, (response) => {
    res.writeHead(response.statusCode ?? 502, response.headers)
    response.pipe(res)
  })
  upstream.on('error', () => { res.writeHead(502).end() })
  req.pipe(upstream)
})

async function close() {
  proxy.closeAllConnections()
  proxy.close()
  await Promise.all(children.map(child => new Promise<void>((resolve) => {
    if (child.exitCode !== null) return resolve()
    child.once('exit', () => resolve())
    child.kill()
  })))
  rmSync(directory, { recursive: true, force: true })
}

process.once('SIGTERM', () => { void close().then(() => process.exit()) })
process.once('SIGINT', () => { void close().then(() => process.exit()) })

async function ready(url: string) {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch(url, { signal: AbortSignal.timeout(1000) })).ok) return } catch { /* Wait for startup. */ }
    await setTimeout(100)
  }
  throw new Error(`Test server did not start: ${url}`)
}

try {
  const binary = join(directory, 'api')
  execFileSync('go', ['build', '-o', binary, './cmd/server'], { cwd: fileURLToPath(new URL('../../../backend/', import.meta.url)), stdio: 'inherit' })
  mkdirSync(join(directory, 'uploads'))
  children.push(spawn(binary, [], {
    env: { ...process.env, GIN_MODE: 'release', APP_ENV: 'test', API_HOST: '127.0.0.1', API_PORT: '4011', DB_PATH: join(directory, 'blog.db'), UPLOADS_PATH: join(directory, 'uploads'), ADMIN_SECRET: 'browser-test-secret' },
    stdio: 'inherit',
  }))
  await ready('http://127.0.0.1:4011/api/v1/health')
  children.push(spawn(process.execPath, ['.output/server/index.mjs'], {
    env: { ...process.env, HOST: '127.0.0.1', PORT: '4012', NUXT_INTERNAL_API_BASE: 'http://127.0.0.1:4011/api/v1', NUXT_PUBLIC_API_BASE: '/api/v1' },
    stdio: 'inherit',
  }))
  await ready('http://127.0.0.1:4012/')
  proxy.listen(4010, '127.0.0.1', () => console.log('BROWSER_TEST_READY'))
}
catch (error) {
  console.error(error)
  await close()
  process.exitCode = 1
}
