<script setup lang="ts">
import type { Block } from '~/types/api'
defineProps<{ blocks: Block[] }>()
const { t } = useReaderSettings()
</script>

<template>
  <div class="post-body">
    <template v-for="(block, index) in blocks" :key="index">
      <p v-if="block.type === 'paragraph'"><PostInline :text="block.text" :content="block.content" /></p>
      <component :is="`h${block.level + 1}`" v-else-if="block.type === 'heading'"><PostInline :text="block.text" :content="block.content" /></component>
      <blockquote v-else-if="block.type === 'quote'"><PostInline :text="block.text" :content="block.content" /></blockquote>
      <hr v-else-if="block.type === 'divider'">
      <component :is="block.spoiler ? 'details' : 'div'" v-else-if="block.type === 'media'" class="media-block">
        <summary v-if="block.spoiler">{{ t('revealSpoiler') }}</summary>
        <PostMedia :block="block" />
      </component>
      <p v-else class="unsupported-block">{{ t('unsupportedBlock') }}</p>
    </template>
  </div>
</template>
