<script setup lang="ts">
import type { Block } from '~/types/api'
import { localMediaSource } from '~/utils/feed'
const props = defineProps<{ block: Extract<Block, { type: 'media' }> }>()
const { t } = useReaderSettings()
const src = computed(() => localMediaSource(props.block.src))
</script>

<template>
  <figure class="post-media">
    <template v-if="src">
      <img v-if="block.mediaKind === 'image' || block.mediaKind === 'gif'" :src="src" :alt="block.alt || block.name || t('image')" width="960" height="540" loading="lazy" decoding="async">
      <video v-else-if="block.mediaKind === 'video'" :src="src" controls playsinline preload="none" :aria-label="block.name || t('video')" />
      <audio v-else-if="block.mediaKind === 'audio'" :src="src" controls preload="none" :aria-label="block.name || t('audio')" />
      <a v-else :href="src" :download="block.name || true">{{ block.name || t('file') }}</a>
    </template>
    <p v-else>{{ t('mediaUnavailable') }}</p>
    <figcaption v-if="block.caption">{{ block.caption }}</figcaption>
  </figure>
</template>
