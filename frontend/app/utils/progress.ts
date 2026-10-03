export const progressKey = 'stereoDamageReadingProgress'
export interface ReadingEntry { opened_at: string, updated_at: string, progress: number, scroll_y: number, completed: boolean }
export function parseProgress(raw: string | null): Record<string, ReadingEntry> {
  try {
    const value: unknown = JSON.parse(raw ?? '{}')
    if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
    const result: Record<string, ReadingEntry> = {}
    for (const [id, entry] of Object.entries(value)) {
      if (!/^[1-9]\d*$/.test(id) || !entry || typeof entry !== 'object') continue
      const e = entry as Partial<ReadingEntry>
      if (typeof e.progress !== 'number' || !Number.isFinite(e.progress)) continue
      result[id] = { opened_at: typeof e.opened_at === 'string' ? e.opened_at : '', updated_at: typeof e.updated_at === 'string' ? e.updated_at : '',
        progress: Math.max(0, Math.min(1, e.progress)), scroll_y: typeof e.scroll_y === 'number' && Number.isFinite(e.scroll_y) ? Math.max(0, e.scroll_y) : 0,
        completed: e.completed === true || e.progress >= 0.9 }
    }
    return result
  }
  catch { return {} }
}
export function advanceProgress(previous: ReadingEntry | undefined, progress: number, y: number, now = new Date().toISOString()): ReadingEntry {
  const next = Math.max(previous?.progress ?? 0, Math.max(0, Math.min(1, progress)))
  return { opened_at: previous?.opened_at || now, updated_at: now, progress: next, scroll_y: Math.max(0, Math.floor(y)), completed: previous?.completed === true || next >= 0.9 }
}
