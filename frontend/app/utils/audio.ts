export const audioVolumeKey = 'stereodamage_audio_volume_v1'
export function audioVolume(value: string | number | null) {
  if (value === null || value === '') return 1
  const number = Number(value)
  return Number.isFinite(number) ? Math.max(0, Math.min(1, number)) : 1
}
