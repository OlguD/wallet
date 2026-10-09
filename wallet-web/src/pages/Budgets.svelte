<script>
  import Icon from '../components/Icon.svelte'
  import BudgetSheet from '../components/BudgetSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { catInfo } from '../lib/categories.js'
  import { t, monthLabel } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { back } from '../lib/router.svelte.js'
  import { app, shiftMonth, isCurrentMonth, refreshSummary } from '../lib/store.svelte.js'

  let list = $state([])
  let loading = $state(true)
  let sheet = $state(null) // 'new' | budget

  const pad = (n) => String(n).padStart(2, '0')
  const load = () => {
    const m = `${app.month.getFullYear()}-${pad(app.month.getMonth() + 1)}`
    return api.get('/budgets', { month: m }).then((r) => (list = r)).catch(() => {}).finally(() => (loading = false))
  }
  $effect(() => {
    app.version
    app.month
    load()
  })

  function move(d) {
    shiftMonth(d)
    refreshSummary()
  }
  const tone = (b) => (b.pct > 100 ? 'over' : b.pct >= 80 ? 'warn' : '')
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('bud.title')}</h1>
    <button type="button" class="icon-btn" aria-label={t('bud.new')} onclick={() => (sheet = 'new')}><Icon d={I.plus} size={20} stroke={2.2} /></button>
  </header>
  <div class="rule"></div>

  <div class="month-nav">
    <button type="button" class="icon-btn" aria-label="<" onclick={() => move(-1)}><Icon d={I.left} size={18} stroke={2.2} /></button>
    <span class="m">{monthLabel(app.month)}</span>
    <button type="button" class="icon-btn" aria-label=">" disabled={isCurrentMonth()} onclick={() => move(1)}><Icon d={I.right} size={18} stroke={2.2} /></button>
  </div>

  {#each list as b, i (b.id)}
    {@const c = catInfo({ category: b.category, type: 'expense' })}
    <button type="button" class="card bud rise {tone(b)}" style="animation-delay: {i * 0.05}s" onclick={() => (sheet = b)}>
      <span class="top">
        <span class="ico"><Icon d={c.icon} size={18} /></span>
        <span class="grow">
          <span class="name">{t('cat.' + b.category)}</span>
          <span class="sub">{t('bud.spent', { v: fmtc(b.spent, b.currency) })} / {fmtc(b.amount, b.currency)}</span>
        </span>
        <span class="pct num">{b.pct}%</span>
      </span>
      <span class="bar"><span style="width: {Math.min(100, b.pct)}%"></span></span>
      <span class="sub">{b.spent > b.amount ? t('bud.over', { v: fmtc(b.spent - b.amount, b.currency) }) : t('bud.left', { v: fmtc(b.remaining, b.currency) })}</span>
    </button>
  {/each}

  {#if !loading && !list.length}
    <div class="empty">
      <Icon d={I.pie} size={40} stroke={1.4} />
      <p>{t('bud.empty')}</p>
    </div>
  {/if}
  <button type="button" class="btn ghost" onclick={() => (sheet = 'new')}><Icon d={I.plus} size={18} />{t('bud.new')}</button>
</div>

{#if sheet === 'new'}
  <BudgetSheet taken={list.map((b) => b.category + ':' + b.currency)} onclose={() => (sheet = null)} onsaved={load} />
{:else if sheet}
  <BudgetSheet budget={sheet} onclose={() => (sheet = null)} onsaved={load} />
{/if}

<style>
  .month-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: -8px;
  }
  .m {
    font-weight: 600;
  }
  .bud {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    text-align: left;
    color: var(--fg);
  }
  .top {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .grow {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .name {
    font-weight: 600;
  }
  .sub {
    font-size: 13px;
    color: var(--muted);
  }
  .pct {
    font-size: 18px;
    font-weight: 600;
  }
  .warn .bar > span {
    background: #e8a33d;
  }
  .over .bar > span {
    background: var(--danger);
  }
  .over .pct {
    color: var(--danger);
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
