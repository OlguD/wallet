<script>
  // Hedef detayı: ilerleme, para ekle/çek, hareket geçmişi.
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import GoalSheet from './GoalSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { goalIcon } from '../lib/categories.js'
  import { t, errorText, fullDate, dayLabel } from '../lib/i18n.js'
  import { fmt, fmtc, symbol, parseAmount, MINUS } from '../lib/money.js'
  import { toast } from '../lib/store.svelte.js'

  let { goal: initialGoal, onclose, onchange } = $props()

  let goal = $state(initialGoal)
  let history = $state([])
  let mode = $state('in') // 'in' | 'out'
  let amountText = $state('')
  let note = $state('')
  let busy = $state(false)
  let editing = $state(false)

  function loadHistory() {
    api.get(`/goals/${goal.id}/contributions`).then((h) => (history = h)).catch(() => {})
  }
  loadHistory()

  function updated(g) {
    goal = g
    onchange?.()
  }

  async function submit(e) {
    e.preventDefault()
    const amount = parseAmount(amountText)
    if (!amount || busy) return
    busy = true
    try {
      const g = await api.post(`/goals/${goal.id}/contributions`, { amount: mode === 'in' ? amount : -amount, note: note.trim() || null })
      toast(t(mode === 'in' ? 'goal.added' : 'goal.taken', { v: fmtc(amount, goal.currency) }))
      if (mode === 'in' && g.remaining === 0 && goal.remaining > 0) navigator.vibrate?.([20, 60, 20, 60, 40])
      amountText = ''
      note = ''
      updated(g)
      loadHistory()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }

  async function removeEntry(c) {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      updated(await api.del(`/goals/${goal.id}/contributions/${c.id}`))
      loadHistory()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }

  // Halka: r=52 → çevre ≈ 326.7
  const C = 2 * Math.PI * 52
</script>

{#if editing}
  <GoalSheet {goal} onclose={() => (editing = false)} onsaved={updated} ondeleted={() => { onchange?.(); onclose() }} />
{:else}
  <Sheet {onclose} top={56} label={goal.name}>
    <header class="head">
      <span class="ico"><Icon d={goalIcon(goal)} size={18} /></span>
      <h2 class="name">{goal.name}</h2>
      <button type="button" class="icon-btn" aria-label={t('common.edit')} onclick={() => (editing = true)}><Icon d={I.edit} size={18} /></button>
    </header>

    <section class="ring-wrap">
      <svg viewBox="0 0 120 120" class="ring" aria-hidden="true">
        <circle cx="60" cy="60" r="52" class="track" />
        <circle cx="60" cy="60" r="52" class="fill" stroke-dasharray={C} stroke-dashoffset={C * (1 - goal.progress_pct / 100)} />
      </svg>
      <div class="ring-text">
        <span class="pct num">{goal.progress_pct}%</span>
        <span class="muted small">{t('goal.saved', { v: fmtc(goal.current, goal.currency) })}</span>
        <span class="muted small">{t('goal.of', { v: fmtc(goal.target_amount, goal.currency) })}</span>
      </div>
    </section>

    <p class="status">
      {#if goal.remaining === 0}{t('goal.done')}
      {:else}
        {t('goal.left', { v: fmtc(goal.remaining, goal.currency) })}
        {#if goal.monthly_needed} · {t('goal.monthly_hint', { v: fmtc(goal.monthly_needed, goal.currency) })}{/if}
      {/if}
    </p>
    {#if goal.target_date}<p class="hint center">{t('goal.date')}: {fullDate(goal.target_date + 'T12:00:00')} · {goal.account_name}</p>{/if}

    <form class="form box" onsubmit={submit}>
      <div class="seg" role="group">
        <button type="button" aria-pressed={mode === 'in'} onclick={() => (mode = 'in')}>{t('goal.deposit')}</button>
        <button type="button" aria-pressed={mode === 'out'} onclick={() => (mode = 'out')} disabled={goal.current === 0}>{t('goal.withdraw')}</button>
      </div>
      <div class="amount-row">
        <input class="input num" bind:value={amountText} inputmode="decimal" placeholder="0" aria-label={t('goal.target')} />
        <span class="cur">{symbol(goal.currency)}</span>
      </div>
      <input class="input" bind:value={note} placeholder={t('goal.note_ph')} maxlength="120" />
      <button class="btn" disabled={busy || !parseAmount(amountText)}>
        <Icon d={mode === 'in' ? I.plus : I.minus} size={18} stroke={2.4} />{mode === 'in' ? t('goal.deposit') : t('goal.withdraw')}
      </button>
    </form>

    <section class="section">
      <h3 class="h-section">{t('goal.history')}</h3>
      <div class="list">
        {#each history as c (c.id)}
          <div class="row">
            <span class="grow">
              <span class="title">{c.note || (c.amount > 0 ? t('goal.deposit') : t('goal.withdraw'))}</span>
              <span class="sub">{dayLabel(c.occurred_at)}</span>
            </span>
            <span class="amount num" class:pos={c.amount > 0}>{c.amount > 0 ? '+' : MINUS}{fmt(Math.abs(c.amount))}</span>
            <button type="button" class="mini" aria-label={t('common.delete')} onclick={() => removeEntry(c)}><Icon d={I.trash} size={15} /></button>
          </div>
        {/each}
      </div>
      {#if !history.length}<p class="hint">{t('goal.no_history')}</p>{/if}
    </section>
  </Sheet>
{/if}

<style>
  .head {
    display: flex;
    align-items: center;
    gap: 12px;
    padding-top: 4px;
  }
  .name {
    flex: 1;
    min-width: 0;
    font-family: var(--font-display);
    font-weight: var(--display-weight);
    font-size: 22px;
    letter-spacing: -0.02em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ring-wrap {
    position: relative;
    width: 200px;
    height: 200px;
    align-self: center;
    flex: 0 0 auto;
  }
  .ring {
    width: 100%;
    height: 100%;
    transform: rotate(-90deg);
  }
  .track,
  .fill {
    fill: none;
    stroke-width: 10;
  }
  .track {
    stroke: var(--surface2);
  }
  .fill {
    stroke: var(--accent);
    stroke-linecap: round;
    transition: stroke-dashoffset 0.9s cubic-bezier(0.2, 0.8, 0.2, 1);
  }
  :global([data-theme='c']) .fill {
    stroke: #0f5c55;
  }
  .ring-text {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    text-align: center;
    padding: 0 28px;
  }
  .pct {
    font-size: 38px;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  .small {
    font-size: 12px;
  }
  .status {
    text-align: center;
    font-size: 15px;
    font-weight: 500;
  }
  .center {
    text-align: center;
    margin-top: -6px;
  }
  .box {
    gap: 10px;
    flex: 0 0 auto;
  }
  .box .seg button {
    flex: 1;
  }
  .amount-row {
    position: relative;
  }
  .amount-row .input {
    font-size: 22px;
    padding-right: 40px;
  }
  .amount-row .cur {
    position: absolute;
    right: 14px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--muted);
    font-size: 18px;
  }
  .amount {
    font-size: 15px;
  }
  .mini {
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    border-radius: 16px;
    border: 1px solid var(--line2);
    background: transparent;
    color: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
