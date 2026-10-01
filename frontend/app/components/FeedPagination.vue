<script setup lang="ts">
const props = defineProps<{ cursor: string | null, loading: boolean, failed: boolean }>()
const emit = defineEmits<{ load: [] }>()
const { t } = useReaderSettings()
const sentinel = useTemplateRef('sentinel')
let observer: IntersectionObserver | undefined
let active = false

function observe() {
  observer?.disconnect()
  if (active && sentinel.value && props.cursor && !props.loading && !props.failed) observer?.observe(sentinel.value)
}

onMounted(() => {
  active = true
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver((entries) => {
    if (active && entries.some(entry => entry.isIntersecting) && props.cursor && !props.loading && !props.failed) emit('load')
  }, { rootMargin: '200px' })
  observe()
})
watch(() => [props.cursor, props.loading, props.failed], observe, { flush: 'post' })
onActivated(() => { active = true; observe() })
onDeactivated(() => { active = false; observer?.disconnect() })
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <div ref="sentinel" class="feed-pagination">
    <p v-if="failed" role="alert">{{ t('moreError') }}</p>
    <p v-if="loading" role="status">{{ t('loadingMore') }}</p>
    <button v-if="cursor" type="button" :disabled="loading" @click="emit('load')">{{ t(failed ? 'retry' : 'loadMore') }}</button>
    <p v-else role="status">{{ t('feedEnd') }}</p>
  </div>
</template>
