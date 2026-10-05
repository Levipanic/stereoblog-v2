// Disposable real Go + production Nuxt instance behind a same-origin test proxy.
import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { createServer, request } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { setTimeout } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import { DatabaseSync } from 'node:sqlite'
import { feedItems, fixtureImage } from '../fixtures/feed.ts'
import { articleBlocks } from '../fixtures/article.ts'

const directory = mkdtempSync(join(tmpdir(), 'stereodamage-browser-'))
const children: ChildProcess[] = []
const development = process.env.NUXT_E2E_DEV === '1'
const proxy = createServer((req, res) => {
  const port = !development && (req.url?.startsWith('/api/') || req.url?.startsWith('/uploads/')) ? 4011 : 4012
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
    env: { ...process.env, GIN_MODE: 'release', APP_ENV: 'test', API_HOST: '127.0.0.1', API_PORT: '4011', DB_PATH: join(directory, 'blog.db'), UPLOADS_PATH: join(directory, 'uploads'), ADMIN_SECRET: 'browser-test-secret', ADMIN_LOGIN_RATE_LIMIT_MAX: '100', COMMENT_COOLDOWN_SECONDS: '1', COMMENT_BURST_MAX: '100' },
    stdio: 'inherit',
  }))
  await ready('http://127.0.0.1:4011/api/v1/health')
  const db = new DatabaseSync(join(directory, 'blog.db'))
  const insertPost = db.prepare('INSERT INTO posts (id, slug, title, blocks_json, preview_media, likes_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)')
  const insertComment = db.prepare('INSERT INTO comments (id, post_id, parent_id, name, content, created_at) VALUES (?, ?, ?, ?, ?, ?)')
  for (const post of feedItems) {
    const blocks = post.id === 1 ? [{ type: 'paragraph', text: post.preview_text }, ...articleBlocks] : [{ type: 'paragraph', text: post.preview_text }]
    insertPost.run(post.id, post.slug, post.title, JSON.stringify(blocks), post.preview_media ? JSON.stringify(post.preview_media) : null, post.likes, post.created_at)
    for (const comment of post.comment_previews) insertComment.run(comment.id, post.id, comment.parent_id, comment.name, comment.content, comment.created_at)
  }
  db.close()
  writeFileSync(join(directory, 'uploads', 'fixture.svg'), fixtureImage)
  writeFileSync(join(directory, 'uploads', 'fixture.txt'), 'Fixture attachment')
  const wav = Buffer.alloc(44 + 16000 * 2 * 20)
  wav.write('RIFF'); wav.writeUInt32LE(wav.length - 8, 4); wav.write('WAVEfmt ', 8)
  wav.writeUInt32LE(16, 16); wav.writeUInt16LE(1, 20); wav.writeUInt16LE(1, 22)
  wav.writeUInt32LE(16000, 24); wav.writeUInt32LE(32000, 28); wav.writeUInt16LE(2, 32); wav.writeUInt16LE(16, 34)
  wav.write('data', 36); wav.writeUInt32LE(wav.length - 44, 40)
  writeFileSync(join(directory, 'uploads', 'fixture.wav'), wav)
  const webCommand = development ? ['node_modules/nuxt/bin/nuxt.mjs', 'dev', '--host', '127.0.0.1', '--port', '4012'] : ['.output/server/index.mjs']
  children.push(spawn(process.execPath, webCommand, {
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
