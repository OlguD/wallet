<script>
  import Icon from '../components/Icon.svelte'
  import RecurringList from '../components/RecurringList.svelte'
  import RecurringSheet from '../components/RecurringSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { back } from '../lib/router.svelte.js'
  import { app, openAdd } from '../lib/store.svelte.js'

  let rules = $state([])
  let editing = $state(null)

  const load = () => api.get('/recurring').then((r) => (rules = r)).catch(() => {})
  $effect(() => {
    app.version
    load()
  })

  // Aktif kuralların aylık karşılığı (haftalık ×52/12, yıllık /12), para birimine göre.
  const perMonth = (r) => {
    const n = r.interval || 1
    if (r.frequency === 'weekly') return (r.amount * 52) / 12 / n
    if (r.frequency === 'yearly') return r.amount / 12 / n
    return r.amount / n
  }
  function monthly(type) {
    const by = {}
    for (const r of rules) if (r.active && r.type === type) by[r.currency] = (by[r.currency] || 0) + perMonth(r)
    return Object.entries(by).map(([c, v]) => fmtc(Math.round(v), c)).join(' · ')
  }
  const expense = $derived(monthly('expense'))
  const income = $derived(monthly('income'))
  const mine = $derived(rules.filter((r) => r.user_id === app.user?.id))
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('rec.title')}</h1>
    {#if app.accounts.length}
      <button type="button" class="icon-btn" aria-label={t('rec.new')} onclick={() => openAdd({ repeat: 'monthly' })}><Icon d={I.plus} size={20} stroke={2.2} /></button>
    {/if}
  </header>
  <div class="rule"></div>

  {#if expense || income}
    <section class="tiles">
      {#if expense}<div class="card tile"><span class="k">{t('rec.monthly_total')}</span><span class="v num">{expense}</span></div>{/if}
      {#if income}<div class="card tile"><span class="k">{t('rec.monthly_income')}</span><span class="v num pos">{income}</span></div>{/if}
    </section>
  {/if}

  {#if mine.length}
    <RecurringList rules={mine} onchange={load} onselect={(r) => (editing = r)} />
  {:else}
    <div class="empty">
      <Icon d={I.repeat} size={40} stroke={1.4} />
      <p>{t('rec.empty')}</p>
    </div>
  {/if}

  {#if app.accounts.length}
    <button type="button" class="btn ghost" onclick={() => openAdd({ repeat: 'monthly' })}><Icon d={I.plus} size={18} />{t('rec.new')}</button>
  {/if}
  <p class="hint">{t('rec.how')}</p>
</div>

{#if editing}
  <RecurringSheet rule={editing} onclose={() => (editing = null)} onchange={load} />
{/if}

<style>
  .tiles {
    display: flex;
    gap: 8px;
  }
  .tile {
    flex: 1 1 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .tile .k {
    font-size: 12px;
    color: var(--muted);
  }
  .tile .v {
    font-size: 17px;
    font-weight: 600;
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
