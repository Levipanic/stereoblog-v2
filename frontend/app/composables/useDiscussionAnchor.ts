export function useDiscussionAnchor(target: Readonly<Ref<HTMLElement | null>>) {
  const route = useRoute()
  let stop = () => {}
  let mounted = false

  async function align() {
    stop()
    if (!mounted || route.hash !== '#comments' || !target.value) return
    const element = target.value
    const controller = new AbortController()
    let observer: ResizeObserver | undefined
    stop = () => { controller.abort(); observer?.disconnect() }
    // Keep late media/font layout changes from moving the anchor, but never fight reader input.
    for (const event of ['wheel', 'touchstart', 'pointerdown', 'keydown']) window.addEventListener(event, stop, { passive: true, signal: controller.signal })
    const position = () => {
      if (!controller.signal.aborted && route.hash === '#comments') element.scrollIntoView({ block: 'start', behavior: 'instant' })
    }
    await nextTick()
    if (controller.signal.aborted) return
    position()
    element.focus({ preventScroll: true })
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(position)
      const article = element.parentElement?.querySelector('.post-article')
      if (article) observer.observe(article, { box: 'border-box' })
      observer.observe(element, { box: 'border-box' })
    }
    void document.fonts.ready.then(position)
  }

  onMounted(() => { mounted = true; void align() })
  watch(() => route.hash, () => { void align() }, { flush: 'post' })
  onBeforeUnmount(() => { mounted = false; stop() })
}
