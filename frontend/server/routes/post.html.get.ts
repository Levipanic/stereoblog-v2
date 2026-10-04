import { ApiError, createPublicApi } from '../../app/utils/api'
import { postPath } from '../../app/utils/feed'

export default defineEventHandler(async (event) => {
  const ids = getRequestURL(event).searchParams.getAll('id')
  const raw = ids[0] ?? ''
  const id = Number(raw)
  if (ids.length !== 1 || !/^\d+$/.test(raw) || !Number.isSafeInteger(id) || id <= 0) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid post ID' })
  }
  const config = useRuntimeConfig(event)
  let slug: string
  try {
    const post = await createPublicApi(config.internalApiBase).resolvePost(id)
    if (typeof post.slug !== 'string' || !post.slug) throw new Error('Invalid resolution')
    slug = post.slug
  }
  catch (error) {
    throw createError({ statusCode: error instanceof ApiError && error.status === 404 ? 404 : 502, statusMessage: error instanceof ApiError && error.status === 404 ? 'Post not found' : 'Post lookup unavailable' })
  }
  // No fragment in Location: browsers inherit the original #comments themselves.
  return sendRedirect(event, postPath(slug), import.meta.dev ? 302 : 301)
})
