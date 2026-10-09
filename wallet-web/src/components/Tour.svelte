<script>
  // Spot ışıklı tanıtım: hedef öğenin etrafı açık kalır, gerisi kararır;
  // yanında "buradan şunu yapabilirsin" kartı çıkar.
  import { onMount, tick } from 'svelte'
  import { fade } from 'svelte/transition'
  import { api } from '../lib/api.js'
  import { TOURS, idsCoveredBy } from '../lib/tours.js'
  import { t } from '../lib/i18n.js'
  import { app } from '../lib/store.svelte.js'

  let { id } = $props()

  const tour = TOURS.find((x) => x.id === id)
  const steps = tour?.steps || []
  let i = $state(0)
  let rect = $state(null) // spot ışığı (viewport koordinatları)
  let card = $state()
  let cardTop = $state(0)
  let vh = $state(window.innerHeight)

  const step = $derived(steps[i])
  const PAD = 8

  async function measure() {
    vh = window.innerHeight
    const el = step?.target ? document.querySelector(`[data-tour="${step.target}"]`) : null
    if (!el) {
      rect = null
    } else {
      const r0 = el.getBoundingClientRect()
      // Görünmüyorsa kaydırıp ortala, sonra tekrar ölç.
      if (r0.top < 60 || r0.bottom > vh - 100) {
        el.scrollIntoView({ block: 'center', behavior: 'smooth' })
        await new Promise((res) => setTimeout(res, 380))
      }
      const r = el.getBoundingClientRect()
      rect = { x: r.left - PAD, y: r.top - PAD, w: r.width + PAD * 2, h: r.height + PAD * 2 }
    }
    await tick()
    const ch = card?.offsetHeight || 200
    if (!rect) cardTop = Math.max(16, (vh - ch) / 2)
    else if (rect.y + rect.h + 14 + ch < vh - 16) cardTop = rect.y + rect.h + 14
    else cardTop = Math.max(16, rect.y - ch - 14)
  }

  $effect(() => {
    i
    measure()
  })
  onMount(() => {
    const on = () => measure()
    window.addEventListener('resize', on)
    return () => window.removeEventListener('resize', on)
  })

  async function finish() {
    const ids = idsCoveredBy(id)
    app.tour = null
    if (app.user) app.user.seen_tours = [...new Set([...(app.user.seen_tours || []), ...ids])]
    document.querySelector('main.scroller')?.scrollTo({ top: 0, behavior: 'smooth' })
    try {
      await api.post('/me/tours', { ids })
    } catch {
      /* bir sonraki açılışta tekrar denenir */
    }
  }

  const next = () => (i < steps.length - 1 ? i++ : finish())
  const prev = () => i > 0 && i--

  function onKey(e) {
    if (e.key === 'Escape') finish()
    else if (e.key === 'ArrowRight' || e.key === 'Enter') next()
    else if (e.key === 'ArrowLeft') prev()
  }
</script>

<svelte:window onkeydown={onKey} />

{#if step}
  <div class="tour" transition:fade={{ duration: 200 }} role="dialog" aria-modal="true" aria-labelledby="tour-title">
    {#if rect}
      <div class="spot" style="left: {rect.x}px; top: {rect.y}px; width: {rect.w}px; height: {rect.h}px"></div>
    {:else}
      <div class="dim"></div>
    {/if}
    <div class="card-t" bind:this={card} style="top: {cardTop}px">
      <span class="count">{i + 1} / {steps.length}</span>
      <h2 id="tour-title">{t(step.title)}</h2>
      <p>{t(step.body)}</p>
      <div class="acts">
        {#if i < steps.length - 1}<button type="button" class="skip" onclick={finish}>{t('tour.skip')}</button>{/if}
        <span class="sp"></span>
        {#if i > 0}<button type="button" class="btn small ghost" onclick={prev}>{t('tour.prev')}</button>{/if}
        <button type="button" class="btn small" onclick={next}>{i < steps.length - 1 ? t('tour.next') : t('tour.finish')}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .tour {
    position: fixed;
    inset: 0;
    z-index: 60;
  }
  .dim {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.62);
  }
  .spot {
    position: absolute;
    border-radius: 18px;
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.62);
    outline: 2px solid var(--accent);
    outline-offset: 2px;
    transition: left 0.35s cubic-bezier(0.2, 0.8, 0.2, 1), top 0.35s cubic-bezier(0.2, 0.8, 0.2, 1),
      width 0.35s cubic-bezier(0.2, 0.8, 0.2, 1), height 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
    pointer-events: none;
  }
  :global([data-theme='c']) .spot {
    outline-color: #f2b441;
  }
  .card-t {
    position: absolute;
    left: 16px;
    right: 16px;
    max-width: 420px;
    margin: 0 auto;
    background: var(--sheet-bg);
    color: var(--fg);
    border-radius: var(--r-card);
    border: 1px solid var(--line2);
    padding: 18px 18px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    box-shadow: 0 18px 40px rgba(0, 0, 0, 0.35);
    transition: top 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
  }
  .count {
    font-size: 12px;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  h2 {
    font-family: var(--font-display);
    font-weight: var(--display-weight);
    font-size: 21px;
    letter-spacing: -0.02em;
    line-height: 1.15;
  }
  p {
    font-size: 15px;
    line-height: 1.45;
    color: var(--muted);
  }
  .acts {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 6px;
  }
  .sp {
    flex: 1;
  }
  .skip {
    background: none;
    border: none;
    padding: 0;
    min-height: 40px;
    color: var(--muted);
    font-size: 14px;
    font-weight: 600;
  }
</style>
