<script>
  // Tutarı 0'dan hedefe sayarak gösterir (tasarımdaki açılış animasyonu).
  import { untrack } from 'svelte'
  let { value = 0, children } = $props()
  let shown = $state(0)
  let raf

  $effect(() => {
    const target = value
    const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
    const from = untrack(() => shown)
    if (reduce || from === target) {
      shown = target
      return
    }
    const dur = 1100
    let start = null
    const tick = (ts) => {
      start ??= ts
      const p = Math.min(1, (ts - start) / dur)
      const e = 1 - Math.pow(1 - p, 3)
      shown = Math.round(from + (target - from) * e)
      if (p < 1) raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  })
</script>

{@render children(shown)}
