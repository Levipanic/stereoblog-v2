import { audioVolume, audioVolumeKey } from '~/utils/audio'
import { localMediaSource } from '~/utils/feed'

export default defineNuxtPlugin(() => {
  // Per-app state, not a module singleton: no playback state shared between SSR requests.
  const state = reactive({ track: null as { src: string, title: string } | null, paused: true, time: 0, duration: 0, volume: 1, failed: false })
  let media: HTMLAudioElement | null = null
  let generation = 0
  function attach(element: HTMLAudioElement) {
    media = element
    try { state.volume = audioVolume(localStorage.getItem(audioVolumeKey)) } catch { /* Optional storage. */ }
    media.volume = state.volume
  }
  function sync() {
    if (!media || !state.track) return
    state.paused = media.paused
    state.time = Number.isFinite(media.currentTime) ? media.currentTime : 0
    state.duration = Number.isFinite(media.duration) ? media.duration : 0
  }
  function volumeChanged() {
    if (!media) return
    state.volume = media.volume
    try { localStorage.setItem(audioVolumeKey, String(state.volume)) } catch { /* Keep current volume. */ }
  }
  function setVolume(value: number) { if (media) { media.volume = audioVolume(value); volumeChanged() } }
  function pause() { generation++; media?.pause(); sync() }
  async function play(src: string, title: string) {
    const safeSrc = localMediaSource(src)
    if (!media || !safeSrc) return
    const request = ++generation
    if (state.track?.src !== safeSrc || media.error) {
      media.pause()
      state.track = { src: safeSrc, title }
      state.time = 0
      state.duration = 0
      media.src = safeSrc
    }
    state.failed = false
    try { await media.play() }
    catch { if (request === generation) state.failed = true }
    if (request === generation) sync()
  }
  function close() {
    generation++
    state.track = null
    state.paused = true
    state.time = state.duration = 0
    state.failed = false
    media?.pause()
    media?.removeAttribute('src')
    media?.load()
  }
  function failed() { if (state.track && media?.error) { state.failed = true; state.paused = true } }
  return { provide: { audio: { state, attach, sync, volumeChanged, setVolume, pause, play, close, failed } } }
})
