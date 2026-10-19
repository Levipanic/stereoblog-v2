import assert from 'node:assert/strict'
import test from 'node:test'
import { audioClock, audioVolume } from '../app/utils/audio.ts'
test('legacy volume preserves mute and clamps malformed/out-of-range preferences', () => {
  assert.equal(audioVolume('0'), 0)
  assert.equal(audioVolume('0.35'), 0.35)
  assert.equal(audioVolume('nonsense'), 1)
  assert.equal(audioVolume(null), 1)
  assert.equal(audioVolume(-1), 0)
  assert.equal(audioVolume(2), 1)
})

test('audio clock formats player time labels', () => {
  assert.equal(audioClock(0), '0:00')
  assert.equal(audioClock(9.9), '0:09')
  assert.equal(audioClock(200), '3:20')
  assert.equal(audioClock(Number.NaN), '0:00')
})
