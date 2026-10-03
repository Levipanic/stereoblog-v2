<script setup lang="ts">
import type { ReadingEntry } from '~/utils/progress'
const props = defineProps<{ postId: number, article: HTMLElement | null }>()
const { t } = useReaderSettings()
const { entries, load, save } = useReadingProgress()
const resume = ref<ReadingEntry | null>(null)
const current = ref(0)
let timer: ReturnType<typeof setTimeout> | undefined
let touched = false

function persist() {
  if (!props.article) return
  const top = props.article.getBoundingClientRect().top + scrollY
  const range = props.article.offsetHeight - innerHeight
  current.value = range <= 0 ? 1 : Math.max(0, Math.min(1, (scrollY - top) / range))
  if (resume.value && scrollY < 80 && current.value < 0.03) return
  if (touched || !resume.value) save(props.postId, current.value, scrollY)
}
function schedule() {
  touched = true
  if (!timer) timer = setTimeout(() => { timer = undefined; persist() }, 500)
}
function flush() { if (timer) clearTimeout(timer); timer = undefined; persist() }
async function continueReading() {
  const y = resume.value?.scroll_y ?? 0
  resume.value = null
  await nextTick()
  await document.fonts.ready
  requestAnimationFrame(() => { scrollTo({ top: y, behavior: 'instant' }); touched = true; persist() })
}
onMounted(() => {
  load()
  const entry = entries.value[props.postId]
  if (entry && !entry.completed && entry.progress >= 0.03 && entry.scroll_y >= 80 && !location.hash) resume.value = { ...entry }
  persist()
  window.addEventListener('scroll', schedule, { passive: true })
  window.addEventListener('resize', schedule)
  window.addEventListener('pagehide', flush)
})
onBeforeRouteLeave(flush)
onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
  window.removeEventListener('scroll', schedule)
  window.removeEventListener('resize', schedule)
  window.removeEventListener('pagehide', flush)
})
</script>

<template>
  <div class="reading-progress" :aria-label="t('readingProgress')" role="progressbar" :aria-valuenow="Math.round(current * 100)" aria-valuemin="0" aria-valuemax="100"><span :style="{ width: `${current * 100}%` }" /></div>
  <div v-if="resume" class="continue-reading"><button type="button" @click="continueReading">{{ t('continueReading', { percent: Math.round(resume.progress * 100) }) }}</button></div>
</template>
