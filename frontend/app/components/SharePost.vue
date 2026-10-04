<script setup lang="ts">
const props = defineProps<{ title: string, url: string }>()
const { t } = useReaderSettings()
const busy = ref(false)
const feedback = ref<'linkCopied' | 'shareDone' | 'copyManually' | null>(null)
const manual = useTemplateRef<HTMLInputElement>('manual')

async function share() {
  if (busy.value) return
  busy.value = true
  feedback.value = null
  try {
    const data = { title: props.title, url: props.url }
    if (navigator.share && (!navigator.canShare || navigator.canShare(data))) {
      try { await navigator.share(data); feedback.value = 'shareDone'; return }
      catch (error) { if (error instanceof DOMException && error.name === 'AbortError') return }
    }
    try {
      await navigator.clipboard.writeText(props.url)
      feedback.value = 'linkCopied'
    }
    catch {
      feedback.value = 'copyManually'
      await nextTick()
      manual.value?.focus()
      manual.value?.select()
    }
  }
  finally { busy.value = false }
}
</script>

<template>
  <div class="share-post">
    <button type="button" :disabled="busy" @click="share">{{ t('sharePost') }}</button>
    <span v-if="feedback" role="status">{{ t(feedback) }}</span>
    <input v-if="feedback === 'copyManually'" ref="manual" :value="url" readonly :aria-label="t('postLink')" @focus="manual?.select()">
  </div>
</template>
