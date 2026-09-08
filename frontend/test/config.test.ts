import assert from 'node:assert/strict'
import test from 'node:test'

import { validateRuntimeConfig } from '../config/validate.ts'

const valid = {
  internalApiBase: 'http://127.0.0.1:8080/api/v1',
  apiBase: '/api/v1',
  siteUrl: 'https://example.com',
  siteName: 'StereoDamage',
  siteHandle: '@stereodamage',
}

test('accepts valid runtime config', () => {
  assert.doesNotThrow(() => validateRuntimeConfig(valid))
})

test('rejects unsafe browser API paths', () => {
  for (const apiBase of ['/', '/api?target=x', '/api#target', 'https://example.com/api', '//example.com/api', '/\\example.com/api']) {
    assert.throws(() => validateRuntimeConfig({ ...valid, apiBase }), /NUXT_PUBLIC_API_BASE/)
  }
})

test('rejects invalid URLs, credentials, and empty metadata', () => {
  assert.throws(() => validateRuntimeConfig({ ...valid, internalApiBase: 'not-a-url' }), /NUXT_INTERNAL_API_BASE/)
  assert.throws(() => validateRuntimeConfig({ ...valid, siteUrl: 'https://user:pass@example.com' }), /NUXT_PUBLIC_SITE_URL/)
  assert.throws(() => validateRuntimeConfig({ ...valid, siteName: ' ' }), /NUXT_PUBLIC_SITE_NAME/)
})
