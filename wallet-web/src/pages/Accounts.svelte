<script>
  import Icon from '../components/Icon.svelte'
  import AccountSheet from '../components/AccountSheet.svelte'
  import GoalCard from '../components/GoalCard.svelte'
  import GoalView from '../components/GoalView.svelte'
  import RecurringSheet from '../components/RecurringSheet.svelte'
  import RecurringList from '../components/RecurringList.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, tv } from '../lib/i18n.js'
  import { fmt, fmtc, symbol } from '../lib/money.js'
  import { pocketColor } from '../lib/categories.js'
  import { convert } from '../lib/fx.js'
  import { cardDebt, cardAvailable } from '../lib/cards.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { navigate } from '../lib/router.svelte.js'
  import { app, accountMonth } from '../lib/store.svelte.js'

  let goals = $state([])
  let rules = $state([])
  let sheet = $state(null) // 'account' | { goal } | { rule }

  function load() {
    api.get('/goals').then((g) => (goals = g)).catch(() => {})
    api.get('/recurring').then((r) => (rules = r)).catch(() => {})
  }
  $effect(() => {
    app.version
    load()
  })

  const open = (a) => navigate(`/accounts/${a.id}`)
</script>

<div class="page">
  <header class="topbar">
    <h1 class="title">{tv('acc.title', prefs.theme)}</h1>
    <button type="button" class="icon-btn" aria-label={t('acc.new')} onclick={() => (sheet = 'account')}><Icon d={I.plus} size={20} stroke={2.2} /></button>
  </header>
  <div class="rule"></div>

  {#if prefs.theme === 'c'}
    <div class="pockets">
      {#each app.accounts as a, i (a.id)}
        {@const col = pocketColor(i)}
        {@const m = accountMonth(a.id)}
        <button type="button" class="pocket rise" style="background: {col.bg}; color: {col.fg}; animation-delay: {i * 0.06}s" onclick={() => open(a)}>
          <span class="n">{a.name}</span>
          <span class="k">{t('acc.kind.' + a.kind)}</span>
          {#if a.kind === 'card'}
            <span class="b">{fmt(cardDebt(a))} {symbol(a.currency)}</span>
            <span class="k">{t('card.debt')}{a.credit_limit ? ' · ' + t('card.available_short', { v: fmtc(cardAvailable(a), a.currency) }) : ''}</span>
          {:else}
            <span class="b">{fmt(a.balance)} {symbol(a.currency)}</span>
          {/if}
          {#if a.currency !== 'TRY' && app.rates}<span class="k">{t('fx.approx', { v: fmtc(convert(a.balance, a.currency, 'TRY'), 'TRY') })}</span>{/if}
          <span class="k">{t('home.this_month', { v: fmtc(m.income - m.expense, a.currency, 'always') })}</span>
        </button>
      {/each}
    </div>
  {:else}
    <div class="list">
      {#each app.accounts as a, i (a.id)}
        {@const m = accountMonth(a.id)}
        <button type="button" class="row rise" style="animation-delay: {i * 0.05}s" onclick={() => open(a)}>
          <span class="ico"><Icon d={a.kind === 'card' ? I.card : a.kind === 'savings' ? I.target : I.wallet} size={18} /></span>
          <span class="grow">
            <span class="title">{a.name}</span>
            {#if a.kind === 'card'}
              <span class="sub">{t('card.debt')}{a.credit_limit ? ' · ' + t('card.available_short', { v: fmtc(cardAvailable(a), a.currency) }) : ''}</span>
            {:else}
              <span class="sub">{t('acc.kind.' + a.kind)} · {t('home.this_month', { v: fmtc(m.income - m.expense, a.currency, 'always') })}</span>
            {/if}
          </span>
          <span class="end">
            <span class="amount" class:neg={a.kind === 'card' && cardDebt(a) > 0}>{a.kind === 'card' ? fmt(cardDebt(a)) : fmt(a.balance)} {symbol(a.currency)}</span>
            {#if a.currency !== 'TRY' && app.rates}<span class="after">{t('fx.approx', { v: fmtc(convert(a.balance, a.currency, 'TRY'), 'TRY') })}</span>{/if}
          </span>
        </button>
      {/each}
    </div>
  {/if}
  {#if !app.accounts.length}<p class="empty">{t('home.no_accounts')}</p>{/if}

  <section class="section">
    <div class="section-head">
      <h2 class="h-section">{t('goal.title')}</h2>
      <a href="/goals" onclick={(e) => { e.preventDefault(); navigate('/goals') }}>{goals.length ? t('common.all') : t('goal.new')}</a>
    </div>
    {#each goals as g (g.id)}
      <GoalCard goal={g} onclick={() => (sheet = { goal: g })} />
    {/each}
    {#if !goals.length}<p class="hint">{t('goal.empty')}</p>{/if}
  </section>

  <section class="section">
    <div class="section-head">
      <h2 class="h-section">{t('rec.title')}</h2>
      <a href="/recurring" onclick={(e) => { e.preventDefault(); navigate('/recurring') }}>{t('common.all')}</a>
    </div>
    {#if rules.length}
      <RecurringList {rules} onchange={load} onselect={(r) => (sheet = { rule: r })} />
    {:else}
      <p class="hint">{t('rec.empty')}</p>
    {/if}
  </section>
</div>

{#if sheet === 'account'}
  <AccountSheet onclose={() => (sheet = null)} />
{:else if sheet?.goal}
  <GoalView goal={sheet.goal} onclose={() => (sheet = null)} onchange={load} />
{:else if sheet?.rule}
  <RecurringSheet rule={sheet.rule} onclose={() => (sheet = null)} onchange={load} />
{/if}

<style>
  .pockets {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .pocket {
    border: none;
    border-radius: 22px;
    padding: 16px;
    min-height: 136px;
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 2px;
    box-shadow: 0 8px 20px rgba(30, 26, 18, 0.14);
  }
  .pocket .n {
    font-weight: 700;
    font-size: 15px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .pocket .k {
    font-size: 12px;
    font-weight: 600;
    opacity: 0.85;
  }
  .pocket .b {
    margin-top: auto;
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 22px;
    font-weight: 800;
    letter-spacing: -0.03em;
  }
</style>
