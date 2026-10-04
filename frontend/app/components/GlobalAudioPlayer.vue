<script setup lang="ts">
const { $audio } = useNuxtApp()
const { t } = useReaderSettings()
const media = useTemplateRef<HTMLAudioElement>('media')
onMounted(() => { if (media.value) $audio.attach(media.value) })
onBeforeUnmount(() => $audio.close())
useHead(() => ({ bodyAttrs: { class: $audio.state.track ? 'has-audio-player' : '' } }))
</script>

<template>
  <section v-show="$audio.state.track" class="mini-player" :aria-label="t('audioPlayer')">
    <div class="mini-player-title"><span>{{ $audio.state.track?.title }}</span><button type="button" @click="$audio.close()">{{ t('closeAudio') }}</button></div>
    <!-- ponytail: native transport controls cover playback/seek; add waveform only as an optional enhancement. -->
    <audio id="persistent-audio" ref="media" controls preload="none" :aria-label="t('audioPlayer')" @play="$audio.sync" @pause="$audio.sync" @timeupdate="$audio.sync" @loadedmetadata="$audio.sync" @durationchange="$audio.sync" @ended="$audio.sync" @volumechange="$audio.volumeChanged" @error="$audio.failed" />
    <label class="audio-volume">{{ t('volume') }} <input type="range" min="0" max="1" step="0.01" :value="$audio.state.volume" @input="$audio.setVolume(Number(($event.target as HTMLInputElement).value))"></label>
    <p v-if="$audio.state.failed" role="status">{{ t('audioUnavailable') }}</p>
  </section>
</template>
