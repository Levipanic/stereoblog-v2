import type { Comment } from '../types/api.ts'

export function commentThread(comments: Comment[]) {
  const byID = new Map(comments.map(comment => [comment.id, comment]))
  const children = new Map<number, Comment[]>()
  for (const comment of byID.values()) {
    const parent = comment.parent_id && byID.has(comment.parent_id) && comment.parent_id !== comment.id ? comment.parent_id : 0
    const list = children.get(parent) ?? []
    list.push(comment)
    children.set(parent, list)
  }
  const result: { comment: Comment, parent?: Comment, depth: number }[] = []
  const seen = new Set<number>()
  function walk(roots: Comment[]) {
    const stack = roots.map(comment => ({ comment, depth: 0 })).reverse()
    while (stack.length) {
      const item = stack.pop()!
      if (seen.has(item.comment.id)) continue
      seen.add(item.comment.id)
      result.push({ ...item, parent: byID.get(item.comment.parent_id ?? 0) })
      for (const child of [...(children.get(item.comment.id) ?? [])].reverse()) stack.push({ comment: child, depth: item.depth + 1 })
    }
  }
  walk(children.get(0) ?? [])
  // Retain orphan/cyclic legacy records without recursion or hiding their content.
  walk([...byID.values()].filter(comment => !seen.has(comment.id)))
  return result
}
