<script>
  // Alttan açılan panel. full: tam ekran (A/B hızlı ekleme gibi).
  import { fly, fade } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { onMount } from 'svelte'

  let { onclose, full = false, top = 64, label = '', children } = $props()

  onMount(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKey = (e) => e.key === 'Escape' && onclose?.()
    window.addEventListener('keydown', onKey)
    return () => {
      document.body.style.overflow = prev
      window.removeEventListener('keydown', onKey)
    }
  })
</script>

<div class="overlay" transition:fade={{ duration: 220 }} onclick={() => onclose?.()} aria-hidden="true"></div>
<div class="sheet" class:full role="dialog" aria-modal="true" aria-label={label}
  style={full ? '' : `top: calc(env(safe-area-inset-top, 0px) + ${top}px)`}
  transition:fly={{ y: full ? 60 : 500, duration: full ? 420 : 460, easing: cubicOut, opacity: full ? 0 : 1 }}>
  {#if !full}<div class="handle" aria-hidden="true"></div>{/if}
  {@render children()}
</div>

<style>
  .overlay {
    position: absolute;
    inset: 0;
    z-index: 40;
    background: var(--overlay);
  }
  .sheet {
    position: absolute;
    z-index: 41;
    left: 0;
    right: 0;
    bottom: 0;
    background: var(--sheet-bg);
    border-radius: 30px 30px 0 0;
    padding: 10px 20px calc(env(safe-area-inset-bottom, 0px) + 16px);
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow-y: auto;
    overscroll-behavior: contain;
    max-width: 640px;
    min-height: 0;
    margin: 0 auto;
  }
  :global([data-theme='b']) .sheet:not(.full) {
    border-radius: 6px 6px 0 0;
    border-top: 3px double var(--fg);
  }
  .sheet.full {
    top: 0;
    border-radius: 0;
    padding-top: calc(env(safe-area-inset-top, 0px) + 10px);
  }
  .handle {
    align-self: center;
    flex: 0 0 5px;
    width: 40px;
    height: 5px;
    border-radius: 3px;
    background: var(--line2);
  }
</style>
