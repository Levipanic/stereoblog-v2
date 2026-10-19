<script setup lang="ts">
import { audioClock } from '~/utils/audio'
import { localMediaSource } from '~/utils/feed'
const props = defineProps<{ src: string, name: string }>()
const { $audio } = useNuxtApp()
const { t } = useReaderSettings()
const active = computed(() => $audio.state.track?.src === localMediaSource(props.src))
const playing = computed(() => active.value && !$audio.state.paused)
const progress = computed(() => active.value && $audio.state.duration ? $audio.state.time / $audio.state.duration : 0)
const ready = ref(false)
onMounted(() => { ready.value = true })
function toggle() { if (playing.value) $audio.pause(); else void $audio.play(props.src, props.name) }
</script>

<template>
  <div class="inline-audio" :class="{ 'is-active': active, 'is-error': active && $audio.state.failed }">
    <div class="audio-player-head">
      <button type="button" class="audio-play-toggle" :disabled="!ready" :aria-label="t(playing ? 'pauseAudio' : 'playAudio', { name })" @click="toggle">{{ playing ? '❚❚' : '▶' }}</button>
      <div class="audio-meta">
        <p class="audio-title">{{ name }}</p>
        <p v-if="active && $audio.state.duration" class="audio-duration-badge">{{ audioClock($audio.state.duration) }}</p>
      </div>
    </div>
    <div class="audio-waveform" aria-hidden="true">
      <span class="audio-fallback-progress"><span class="audio-fallback-progress-fill" :style="{ width: `${progress * 100}%` }" /></span>
    </div>
    <div class="audio-player-foot">
      <p class="audio-time-label">{{ active ? `${audioClock($audio.state.time)} / ${audioClock($audio.state.duration)}` : t('play') }}</p>
      <p v-if="active && $audio.state.failed" class="audio-status-label" role="status">{{ t('audioUnavailable') }}</p>
    </div>
  </div>
</template>
