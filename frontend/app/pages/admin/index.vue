<script setup lang="ts">
import { ApiError } from '~/utils/api'
import type { MessageKey } from '~/utils/i18n'

const { t } = useReaderSettings()
const { session, check, login, logout } = useAdminSession()
const secret = ref('')
const busy = ref(false)
const checking = ref(true)
const checkFailed = ref(false)
const postsVersion = ref(0)
const pendingCount = ref<number | null>(null)
const error = ref<MessageKey | null>(null)
const heading = useTemplateRef<HTMLElement>('heading')
const password = useTemplateRef<HTMLInputElement>('password')
const sections = [
  { id: 'write', label: 'adminNew' }, { id: 'posts', label: 'posts' },
  { id: 'moderation', label: 'adminModeration' }, { id: 'backup', label: 'adminBackup' },
] as const
useHead(() => ({ title: t('admin'), meta: [{ name: 'robots', content: 'noindex, nofollow' }] }))

async function run(action: () => Promise<void>, initial = false) {
  if (busy.value) return
  busy.value = true
  error.value = null
  try {
    await action()
    checkFailed.value = false
  }
  catch (cause) {
    error.value = cause instanceof ApiError && cause.code === 'invalid_credentials' ? 'adminInvalid'
      : cause instanceof ApiError && cause.status === 429 ? 'adminLimited'
        : cause instanceof ApiError && cause.status === 401 ? 'adminExpired' : 'adminError'
    if (initial) checkFailed.value = true
  }
  finally {
    secret.value = ''
    busy.value = false
    checking.value = false
    await nextTick()
    if (!initial) (session.value.authenticated ? heading.value : password.value)?.focus()
  }
}
onMounted(() => run(check, true))
</script>

<template>
  <main id="main-content" class="site-shell" tabindex="-1">
    <h1 ref="heading" tabindex="-1">{{ t('admin') }}</h1>
    <p v-if="checking" role="status">{{ t('adminChecking') }}</p>
    <p v-if="error" role="alert">{{ t(error) }}</p>
    <button v-if="checkFailed" type="button" :disabled="busy" @click="run(check, true)">{{ t('retry') }}</button>
    <template v-else-if="!checking">
      <form v-if="!session.authenticated" class="admin-login" :aria-busy="busy" @submit.prevent="run(() => login(secret))">
        <label for="admin-secret">{{ t('adminSecret') }}</label>
        <input id="admin-secret" ref="password" v-model="secret" type="password" autocomplete="current-password" required :disabled="busy">
        <button type="submit" :disabled="busy">{{ t(busy ? 'adminWorking' : 'adminLogin') }}</button>
      </form>
      <div v-else class="admin-home">
        <button type="button" :disabled="busy" @click="run(logout)">{{ t(busy ? 'adminWorking' : 'adminLogout') }}</button>
        <nav :aria-label="t('admin')"><a v-for="section in sections" :key="section.id" :href="`#admin-${section.id}`">{{ t(section.label) }}<span v-if="section.id === 'moderation' && pendingCount !== null"> ({{ pendingCount }})</span></a></nav>
        <section v-for="section in sections" :id="`admin-${section.id}`" :key="section.id">
          <h2>{{ t(section.label) }}</h2>
          <LazyAdminComposer v-if="section.id === 'write'" @saved="postsVersion++" />
          <LazyAdminPosts v-else-if="section.id === 'posts'" :revision="postsVersion" />
          <LazyAdminModeration v-else-if="section.id === 'moderation'" @count="pendingCount = $event" />
          <LazyAdminBackup v-else-if="section.id === 'backup'" />
        </section>
      </div>
    </template>
  </main>
</template>

<style scoped>
.admin-login { display: grid; gap: 0.75rem; max-width: 26rem; margin-top: 1rem; }
.admin-login input { min-width: 0; width: 100%; padding: 0.6rem; border: 1px solid var(--border); border-radius: 4px; background: var(--surface); color: var(--text); }
.admin-home { display: grid; gap: 1rem; margin-top: 1rem; }
.admin-home > button { justify-self: start; }
nav { display: flex; flex-wrap: wrap; gap: 0.5rem 1rem; }
nav a { padding-block: 0.6rem; }
section { padding-block: 1rem; border-top: 1px solid var(--border); scroll-margin-top: 5rem; }
h2 { font-size: 1.2rem; }
</style>
