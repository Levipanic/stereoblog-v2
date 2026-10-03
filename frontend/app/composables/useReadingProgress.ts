import { advanceProgress, parseProgress, progressKey, type ReadingEntry } from '~/utils/progress'

export function useReadingProgress() {
  const entries = useState<Record<string, ReadingEntry>>('reading-progress', () => ({}))
  const loaded = useState('reading-progress-loaded', () => false)
  function load() {
    if (!import.meta.client || loaded.value) return
    try { entries.value = parseProgress(localStorage.getItem(progressKey)) } catch { /* Storage is optional. */ }
    loaded.value = true
  }
  function save(id: number, progress: number, y: number) {
    load()
    entries.value[id] = advanceProgress(entries.value[id], progress, y)
    try { localStorage.setItem(progressKey, JSON.stringify(entries.value)) } catch { /* Keep in-memory progress. */ }
  }
  return { entries, loaded, load, save }
}
