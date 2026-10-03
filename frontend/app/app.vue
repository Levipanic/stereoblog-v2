<script setup lang="ts">
const { language, restore } = useReaderSettings()
const { load } = useReadingProgress()
useHead(() => ({ htmlAttrs: { lang: language.value } }))
let dispose: (() => void) | undefined
onMounted(() => { dispose = restore(); load() })
onBeforeUnmount(() => dispose?.())
</script>

<template>
  <SiteHeader />
  <NuxtPage />
</template>
