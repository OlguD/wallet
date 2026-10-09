<script>
  import Icon from './Icon.svelte'
  import { goalIcon } from '../lib/categories.js'
  import { t } from '../lib/i18n.js'
  import { fmt, fmtc } from '../lib/money.js'

  let { goal: g, onclick, delay = 0 } = $props()

  const daysLeft = $derived.by(() => {
    if (!g.target_date) return null
    const [y, m, d] = g.target_date.split('-').map(Number)
    const now = new Date()
    return Math.ceil((new Date(y, m - 1, d) - new Date(now.getFullYear(), now.getMonth(), now.getDate())) / 86400000)
  })
</script>

<button type="button" class="card goal rise" style="animation-delay: {delay}s" {onclick}>
  <span class="g-top">
    <span class="ico"><Icon d={goalIcon(g)} size={18} /></span>
    <span class="g-head">
      <span class="g-name">{g.name}</span>
      <span class="g-sub">{fmt(g.current)} / {fmtc(g.target_amount, g.currency)}</span>
    </span>
    <span class="pct num">{g.progress_pct}%</span>
  </span>
  <span class="bar"><span style="width: {g.progress_pct}%"></span></span>
  <span class="g-foot">
    <span>
      {#if g.remaining === 0}{t('goal.done')}
      {:else if g.monthly_needed}{t('goal.monthly', { v: fmtc(g.monthly_needed, g.currency) })}
      {:else}{t('goal.left', { v: fmtc(g.remaining, g.currency) })}{/if}
    </span>
    {#if daysLeft !== null && g.remaining > 0}
      <span>{daysLeft < 0 ? t('goal.overdue') : t('goal.days_left', { n: daysLeft })}</span>
    {/if}
  </span>
</button>

<style>
  .goal {
    display: flex;
    flex-direction: column;
    gap: 10px;
    text-align: left;
    color: var(--fg);
    width: 100%;
  }
  .g-top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .g-head {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .g-name {
    font-weight: 600;
    font-size: 16px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .g-sub,
  .g-foot {
    font-size: 13px;
    color: var(--muted);
  }
  .g-foot {
    display: flex;
    justify-content: space-between;
    gap: 8px;
  }
  .pct {
    font-size: 18px;
    font-weight: 600;
  }
</style>
