<script setup lang="ts">
import { ApiError } from '~/utils/api'
import type { MessageKey } from '~/utils/i18n'
const { t } = useReaderSettings()
const { write } = useAdminSession()
const busy = ref(false)
const error = ref<MessageKey | null>(null)
const archiveURL = ref('')
const controller = new AbortController()
let releaseTimer: ReturnType<typeof setTimeout> | undefined
function release() {
  clearTimeout(releaseTimer)
  if (archiveURL.value) URL.revokeObjectURL(archiveURL.value)
  archiveURL.value = ''
}
onBeforeUnmount(() => { controller.abort(); release() })
async function download() {
  if (busy.value) return
  busy.value = true; error.value = null; release()
  try {
    // ponytail: Blob retains the full archive client-side; use a streamed download if backups outgrow device memory.
    const archive = await write<Blob>('/backup', {}, 'POST', { responseType: 'blob', signal: AbortSignal.any([controller.signal, AbortSignal.timeout(15 * 60_000)]) })
    archiveURL.value = URL.createObjectURL(archive)
    const link = document.createElement('a')
    link.href = archiveURL.value
    link.download = 'stereodamage-backup.zip'
    document.body.append(link)
    link.click()
    link.remove()
    releaseTimer = setTimeout(release, 5 * 60_000)
  }
  catch (cause) {
    if (!controller.signal.aborted) error.value = cause instanceof ApiError && cause.status === 429 ? 'backupLimited' : 'adminError'
  }
  finally { busy.value = false }
}
</script>

<template>
  <div :aria-busy="busy">
    <p>{{ t('backupContents') }}</p>
    <button type="button" :disabled="busy" @click="download">{{ t(busy ? 'backupWorking' : 'backupDownload') }}</button>
    <p v-if="busy" role="status">{{ t('backupWait') }}</p>
    <p v-if="error" role="alert">{{ t(error) }}</p>
    <p v-if="archiveURL" role="status">{{ t('backupReady') }} <a :href="archiveURL" download="stereodamage-backup.zip">{{ t('backupSave') }}</a></p>
  </div>
</template>
