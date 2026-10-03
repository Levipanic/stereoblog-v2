<script setup lang="ts">
const props = defineProps<{ src: string, alt: string, caption?: string }>()
const { t } = useReaderSettings()
const image = useTemplateRef<HTMLImageElement>('image')
const dialog = useTemplateRef<HTMLDialogElement>('dialog')
const failed = ref(false)
const opened = ref(false)
let overflow = ''

function open() {
  if (!dialog.value || failed.value || opened.value) return
  overflow = document.documentElement.style.overflow
  opened.value = true
  dialog.value.showModal()
  document.documentElement.style.overflow = 'hidden'
}
function closed() {
  if (!opened.value) return
  document.documentElement.style.overflow = overflow
  opened.value = false
}
onMounted(() => { if (image.value?.complete && image.value.naturalWidth === 0) failed.value = true })
watch(() => props.src, () => { failed.value = false; dialog.value?.close() })
onBeforeUnmount(() => { dialog.value?.close(); closed() })
</script>

<template>
  <div class="post-image-frame">
    <button v-if="!failed" type="button" class="image-open" :aria-label="t('openImage', { alt })" @click="open">
      <img ref="image" :src="src" :alt="alt" width="960" height="540" loading="lazy" decoding="async" @error="failed = true">
    </button>
    <p v-else class="image-failed">{{ t('mediaUnavailable') }} — {{ alt }}</p>
  </div>
  <dialog ref="dialog" class="image-viewer" :aria-label="alt" @close="closed" @click="($event.target === dialog) && dialog?.close()">
    <button type="button" autofocus class="viewer-close" @click="dialog?.close()">{{ t('closeImage') }}</button>
    <template v-if="opened">
      <img :src="src" :alt="alt">
      <p v-if="caption">{{ caption }}</p>
    </template>
  </dialog>
</template>
