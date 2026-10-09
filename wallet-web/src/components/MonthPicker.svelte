<script>
  import Sheet from './Sheet.svelte'
  import { t, monthLabel } from '../lib/i18n.js'
  import { app, setMonth, refreshSummary } from '../lib/store.svelte.js'

  let { onclose } = $props()
  const now = new Date()
  const months = Array.from({ length: 18 }, (_, i) => new Date(now.getFullYear(), now.getMonth() - i, 1))

  function pick(m) {
    setMonth(m)
    refreshSummary()
    onclose()
  }
</script>

<Sheet {onclose} top={200} label={t('month.pick')}>
  <h2 class="h-section" style="padding: 4px 0">{t('month.pick')}</h2>
  <div class="list">
    {#each months as m}
      <button type="button" class="row" onclick={() => pick(m)} aria-pressed={m.getTime() === app.month.getTime()}>
        <span class="grow"><span class="title">{monthLabel(m)}</span></span>
        {#if m.getTime() === app.month.getTime()}<span class="dot"></span>{/if}
      </button>
    {/each}
  </div>
</Sheet>

<style>
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 5px;
    background: var(--accent);
  }
  :global([data-theme='c']) .dot {
    background: #0f5c55;
  }
</style>
