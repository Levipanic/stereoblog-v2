<script setup lang="ts">
import type { Block } from '~/types/api'
import { articleHeadings } from '~/utils/content'
const props = defineProps<{ blocks: Block[], preview?: boolean }>()
const anchors = computed(() => new Map(articleHeadings(props.blocks).map(heading => [heading.index, heading.id])))
const { t } = useReaderSettings()
</script>

<template>
  <div class="post-body">
    <template v-for="(block, index) in blocks" :key="index">
      <p v-if="block.type === 'paragraph'"><PostInline :text="block.text" :content="block.content" /></p>
      <component :is="`h${block.level + 1}`" v-else-if="block.type === 'heading'" :id="anchors.get(index)"><PostInline :text="block.text" :content="block.content" /></component>
      <blockquote v-else-if="block.type === 'quote'"><PostInline :text="block.text" :content="block.content" /></blockquote>
      <hr v-else-if="block.type === 'divider'">
      <component :is="block.spoiler ? 'details' : 'div'" v-else-if="block.type === 'media'" class="media-block">
        <summary v-if="block.spoiler">{{ t('revealSpoiler') }}</summary>
        <PostMedia :block="block" :preview="preview" />
      </component>
      <p v-else class="unsupported-block">{{ t('unsupportedBlock') }}</p>
    </template>
  </div>
</template>
