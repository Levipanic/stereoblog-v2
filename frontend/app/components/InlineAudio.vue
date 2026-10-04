<script setup lang="ts">
import { localMediaSource } from '~/utils/feed'
const props = defineProps<{ src: string, name: string }>()
const { $audio } = useNuxtApp()
const { t } = useReaderSettings()
const playing = computed(() => $audio.state.track?.src === localMediaSource(props.src) && !$audio.state.paused)
const ready = ref(false)
onMounted(() => { ready.value = true })
function toggle() { if (playing.value) $audio.pause(); else void $audio.play(props.src, props.name) }
</script>

<template>
  <div class="inline-audio">
    <span>{{ name }}</span>
    <button type="button" :disabled="!ready" :aria-label="t(playing ? 'pauseAudio' : 'playAudio', { name })" @click="toggle">{{ t(playing ? 'pause' : 'play') }}</button>
  </div>
</template>
