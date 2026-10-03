<script setup lang="ts">
import type { Block } from '~/types/api'
import { articleHeadings } from '~/utils/content'
const props = defineProps<{ blocks: Block[] }>()
const { t } = useReaderSettings()
const headings = computed(() => articleHeadings(props.blocks))
const disclosure = useTemplateRef<HTMLDetailsElement>('disclosure')
const active = ref('')
let dispose: (() => void) | undefined

function select(id: string) {
  active.value = id
  if (window.matchMedia('(max-width: 1099px)').matches && disclosure.value) disclosure.value.open = false
}
onMounted(() => {
  const wide = window.matchMedia('(min-width: 1100px)')
  const sync = () => { if (disclosure.value) disclosure.value.open = wide.matches }
  sync()
  wide.addEventListener('change', sync)
  const nodes = headings.value.map(heading => document.getElementById(heading.id)).filter((node): node is HTMLElement => !!node)
  const observer = typeof IntersectionObserver === 'undefined' ? undefined : new IntersectionObserver(() => {
    active.value = nodes.filter(node => node.getBoundingClientRect().top <= innerHeight * 0.35).at(-1)?.id ?? nodes[0]?.id ?? ''
  }, { rootMargin: '-72px 0px -65% 0px' })
  nodes.forEach(node => observer?.observe(node))
  dispose = () => { observer?.disconnect(); wide.removeEventListener('change', sync) }
})
onBeforeUnmount(() => dispose?.())
</script>

<template>
  <nav class="post-toc" :aria-label="t('contents')">
    <details ref="disclosure">
      <summary>{{ t('contents') }}</summary>
      <ol>
        <li v-for="heading in headings" :key="heading.id" :class="{ 'toc-nested': heading.level > 1 }">
          <a :href="`#${heading.id}`" :aria-current="active === heading.id ? 'location' : undefined" @click="select(heading.id)">{{ heading.text }}</a>
        </li>
      </ol>
    </details>
  </nav>
</template>
