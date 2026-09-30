<script setup lang="ts">
const { language, restore } = useReaderSettings()
useHead(() => ({ htmlAttrs: { lang: language.value } }))
let dispose: (() => void) | undefined
onMounted(() => { dispose = restore() })
onBeforeUnmount(() => dispose?.())
</script>

<template>
  <SiteHeader />
  <NuxtPage />
</template>
