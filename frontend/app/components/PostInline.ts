import { defineComponent, h, type PropType, type VNodeChild } from 'vue'
import type { InlineNode } from '../types/api'
import { safeLink } from '../utils/content'

export default defineComponent({
  props: { text: { type: String, default: '' }, content: { type: Array as PropType<InlineNode[]>, default: undefined } },
  setup: props => () => h('span', props.content?.length ? props.content.map((node) => {
    if (node.type !== 'text') return ''
    return (node.marks ?? []).reduceRight<VNodeChild>((child, mark) => {
      if (mark.type === 'link') {
        const href = safeLink(mark.href)
        return href ? h('a', { href, rel: 'noopener noreferrer' }, [child]) : child
      }
      const tag = { bold: 'strong', italic: 'em', code: 'code' }[mark.type]
      return tag ? h(tag, [child]) : child
    }, node.text)
  }) : props.text),
})
