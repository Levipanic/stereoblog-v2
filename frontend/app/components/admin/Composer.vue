<script setup lang="ts">
import type { Block } from '~/types/api'
const { t } = useReaderSettings()
const title = ref('')
const slug = ref('')
const blocks = ref<Block[]>([])
const valid = ref(true)
</script>

<template>
  <div class="composer">
    <p>{{ t('editorTemporary') }}</p>
    <label for="post-title">{{ t('editorTitle') }}</label>
    <input id="post-title" v-model="title" maxlength="160" aria-required="true">
    <LazyAdminWriter :blocks="[]" @change="blocks = $event; valid = true" @invalid="valid = false" />
    <details>
      <summary>{{ t('editorSettings') }}</summary>
      <label for="post-slug">{{ t('editorSlug') }}</label>
      <input id="post-slug" v-model="slug" autocapitalize="none" :spellcheck="false">
      <p>{{ t('editorPreview') }}</p>
    </details>
  </div>
</template>

<style scoped>
.composer { display: grid; gap: 0.75rem; }
input { display: block; width: 100%; min-width: 0; padding: 0.6rem; font: inherit; background: var(--surface); color: var(--text); border: 1px solid var(--border); }
summary { min-height: 44px; cursor: pointer; }
</style>
