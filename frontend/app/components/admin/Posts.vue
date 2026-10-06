<script setup lang="ts">
import type { AdminPost, FeedItem, FeedPage } from '~/types/api'
import { blocksToDocument } from '~/utils/editor'
const props = defineProps<{ revision?: number }>()
const { t } = useReaderSettings()
const { read, write } = useAdminSession()
const items = ref<FeedItem[]>([])
const cursor = ref<string | null>(null)
const busy = ref(false)
const error = ref(false)
const unsupported = ref(false)
const editing = shallowRef<AdminPost | null>(null)
const savedPost = shallowRef<AdminPost | null>(null)
const composer = useTemplateRef<{ canClose: () => boolean }>('composer')
async function load(more = false) {
  if (busy.value) return
  busy.value = true; error.value = false
  try {
    const page = await read<FeedPage>(`/posts?limit=10${more && cursor.value ? `&cursor=${encodeURIComponent(cursor.value)}` : ''}`)
    items.value = more ? [...items.value, ...page.items.filter(item => !items.value.some(existing => existing.id === item.id))] : page.items
    cursor.value = page.next_cursor
  }
  catch { error.value = true }
  finally { busy.value = false }
}
async function edit(id: number) {
  if (busy.value) return
  busy.value = true; error.value = false; unsupported.value = false
  try {
    const post = await read<AdminPost>(`/posts/${id}`)
    try { blocksToDocument(post.blocks) } catch { unsupported.value = true; return }
    editing.value = post
    await nextTick()
    document.getElementById('admin-edit-post')?.scrollIntoView()
  }
  catch { error.value = true }
  finally { busy.value = false }
}
async function remove(post: FeedItem) {
  if (busy.value || !window.confirm(`${t('deletePostConfirm')}\n${post.title}`)) return
  busy.value = true; error.value = false
  try {
    await write(`/posts/${post.id}`, {}, 'DELETE')
    items.value = items.value.filter(item => item.id !== post.id)
    try { localStorage.removeItem(`stereoDamageDraft:${post.id}`) } catch { /* The deleted post cannot be reopened. */ }
  }
  catch { error.value = true }
  finally { busy.value = false }
}
function close() { if (composer.value?.canClose()) { editing.value = null; load() } }
async function saved(post: AdminPost) { savedPost.value = post; editing.value = null; await load() }
onMounted(() => load())
watch(() => props.revision, () => { if (!editing.value) load() })
</script>

<template>
  <div class="admin-posts">
    <p v-if="error" role="alert">{{ t('adminError') }}</p>
    <p v-if="unsupported" role="alert">{{ t('postUnsupported') }}</p>
    <p v-if="savedPost" role="status">{{ t('postSaved') }} <a :href="`/posts/${encodeURIComponent(savedPost.slug)}`">{{ savedPost.title }}</a></p>
    <div v-if="editing" id="admin-edit-post">
      <button type="button" @click="close">{{ t('closeEditor') }}</button>
      <h3>{{ editing.title }}</h3>
      <LazyAdminComposer :key="editing.id" ref="composer" :post="editing" @saved="saved" />
    </div>
    <template v-else>
      <button type="button" :disabled="busy" @click="load()">{{ t('refreshAdmin') }}</button>
      <p v-if="busy" role="status">{{ t('adminWorking') }}</p>
      <p v-else-if="!items.length && !error">{{ t('noAdminPosts') }}</p>
      <ol>
        <li v-for="post in items" :key="post.id" :data-post-id="post.id">
          <a :href="`/posts/${encodeURIComponent(post.slug)}`">{{ post.title }}</a>
          <time>{{ post.created_at }}</time>
          <div class="post-actions">
            <button type="button" :disabled="busy" @click="edit(post.id)">{{ t('editPost') }}</button>
            <button type="button" :disabled="busy" @click="remove(post)">{{ t('deletePost') }}</button>
          </div>
        </li>
      </ol>
      <button v-if="cursor" type="button" :disabled="busy" @click="load(true)">{{ t('loadMore') }}</button>
    </template>
  </div>
</template>

<style scoped>
ol { list-style: none; padding: 0; }
li { padding-block: 0.75rem; border-bottom: 1px solid var(--border); overflow-wrap: anywhere; }
time { display: block; color: var(--muted); font-size: 0.85rem; }
.post-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: 0.4rem; }
#admin-edit-post { scroll-margin-top: 5rem; }
</style>
