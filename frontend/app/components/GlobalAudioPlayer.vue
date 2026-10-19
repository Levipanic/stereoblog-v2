<script setup lang="ts">
import { audioClock } from '~/utils/audio'
const { $audio } = useNuxtApp()
const { t } = useReaderSettings()
const route = useRoute()
const media = useTemplateRef<HTMLAudioElement>('media')
onMounted(() => { if (media.value) $audio.attach(media.value) })
onBeforeUnmount(() => $audio.close())
useHead(() => ({ bodyAttrs: { class: [route.path.startsWith('/admin') ? 'admin-page' : '', $audio.state.track ? 'has-audio-player' : ''].filter(Boolean).join(' ') } }))
const progress = computed(() => $audio.state.duration ? $audio.state.time / $audio.state.duration : 0)
function seek(event: Event) { $audio.seek(Number((event.target as HTMLInputElement).value)) }
</script>

<template>
  <section v-show="$audio.state.track" class="mini-player" :class="{ 'is-error': $audio.state.failed }" :aria-label="t('audioPlayer')">
    <div class="mini-audio-main">
      <button type="button" class="mini-audio-toggle" :aria-label="t($audio.state.paused ? 'play' : 'pause')" @click="$audio.toggle()">{{ $audio.state.paused ? '▶' : '❚❚' }}</button>
      <div class="mini-audio-meta">
        <p class="mini-audio-title">{{ $audio.state.track?.title }}</p>
        <p class="mini-audio-time">{{ audioClock($audio.state.time) }} / {{ audioClock($audio.state.duration) }}</p>
      </div>
      <button type="button" class="mini-audio-close" :aria-label="t('closeAudio')" @click="$audio.close()">×</button>
    </div>
    <input
      class="mini-audio-progress"
      type="range" min="0" :max="Math.max($audio.state.duration, 0.01)" step="0.1"
      :value="$audio.state.time" :disabled="!$audio.state.duration"
      :style="{ '--progress': `${progress * 100}%` }"
      :aria-label="t('audioPosition')"
      @input="seek"
    >
    <div class="mini-audio-volume">
      <span class="mini-audio-volume-icon" aria-hidden="true">vol</span>
      <input class="mini-audio-volume-slider" type="range" min="0" max="1" step="0.01" :value="$audio.state.volume" :aria-label="t('volume')" @input="$audio.setVolume(Number(($event.target as HTMLInputElement).value))">
    </div>
    <!-- The engine element stays hidden; the visible controls above drive it. -->
    <audio id="persistent-audio" ref="media" class="audio-engine" preload="none" @play="$audio.sync" @pause="$audio.sync" @timeupdate="$audio.sync" @loadedmetadata="$audio.sync" @durationchange="$audio.sync" @ended="$audio.sync" @volumechange="$audio.volumeChanged" @error="$audio.failed" />
    <p v-if="$audio.state.failed" class="audio-status-label" role="status">{{ t('audioUnavailable') }}</p>
  </section>
</template>
