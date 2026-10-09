<script>
  import Icon from '../components/Icon.svelte'
  import TxRow from '../components/TxRow.svelte'
  import LedgerHead from '../components/LedgerHead.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, tv, monthLabel, dayHeader } from '../lib/i18n.js'
  import { fmt, symbol } from '../lib/money.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { app, monthRange, shiftMonth, refreshSummary, isCurrentMonth, monthTotals } from '../lib/store.svelte.js'

  let list = $state([])
  let loading = $state(true)
  let filter = $state('all') // all | income | expense
  let accountFilter = $state(null)
  let query = $state('')
  let q = $state('') // gecikmeli arama metni

  // Yazarken her tuşta istek atılmasın.
  $effect(() => {
    const v = query.trim()
    const id = setTimeout(() => (q = v), 280)
    return () => clearTimeout(id)
  })

  $effect(() => {
    app.version
    const range = monthRange(app.month)
    const acc = accountFilter
    const search = q
    loading = true
    // Arama yapılırken tüm zamanlarda aranır.
    api
      .get('/transactions', { ...(search ? {} : range), account_id: acc, q: search, limit: 200 })
      .then((r) => (list = r))
      .finally(() => (loading = false))
  })
  const exportUrl = $derived(`/api/export/transactions.csv?from=${monthRange(app.month).from}&to=${monthRange(app.month).to}`)

  function move(d) {
    shiftMonth(d)
    refreshSummary()
  }

  const shown = $derived(filter === 'all' ? list : list.filter((x) => x.type === filter))
  const days = $derived.by(() => {
    const out = []
    for (const tx of shown) {
      const d = new Date(tx.occurred_at)
      const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
      let g = out.at(-1)
      if (!g || g.key !== key) out.push((g = { key, date: d, items: [], net: 0 }))
      g.items.push(tx)
      g.net += tx.type === 'income' ? tx.amount : -tx.amount
    }
    return out
  })
  const mt = $derived(monthTotals())
</script>

<div class="page">
  <header class="topbar">
    <h1 class="title">{tv('tx.title', prefs.theme)}</h1>
    <div class="month-nav">
      <button type="button" class="icon-btn" aria-label="<" onclick={() => move(-1)}><Icon d={I.left} size={18} stroke={2.2} /></button>
      <span class="m">{monthLabel(app.month)}</span>
      <button type="button" class="icon-btn" aria-label=">" disabled={isCurrentMonth()} onclick={() => move(1)}><Icon d={I.right} size={18} stroke={2.2} /></button>
    </div>
  </header>
  <div class="rule"></div>

  <div class="sum">
    <div><small>{t('common.income')}</small><b class="num pos">+{fmt(mt.income)}</b></div>
    <div><small>{t('common.expense')}</small><b class="num">{mt.expense ? '−' : ''}{fmt(mt.expense)}</b></div>
    <div><small>{t('common.net')} · {symbol(mt.currency)}</small><b class="num">{fmt(mt.net, 'always')}</b></div>
  </div>

  <div class="search-row" data-tour="search">
    <label class="search">
      <Icon d={I.search} size={17} />
      <input type="search" bind:value={query} placeholder={t('tx.search')} enterkeyhint="search" />
      {#if query}<button type="button" aria-label={t('common.close')} onclick={() => (query = '')}><Icon d={I.x} size={15} stroke={2.2} /></button>{/if}
    </label>
    <a class="icon-btn dl" href={exportUrl} download aria-label={t('tx.export')} title={t('tx.export')}><Icon d={I.download} size={18} /></a>
  </div>

  <div class="chips">
    {#each ['all', 'expense', 'income'] as f}
      <button type="button" class="chip small" aria-pressed={filter === f} onclick={() => (filter = f)}>
        {f === 'all' ? t('tx.filter_all') : t('common.' + f)}
      </button>
    {/each}
    {#if app.accounts.length > 1}
      <span class="sep"></span>
      {#each app.accounts as a}
        <button type="button" class="chip small" aria-pressed={accountFilter === a.id} onclick={() => (accountFilter = accountFilter === a.id ? null : a.id)}>{a.name}</button>
      {/each}
    {/if}
  </div>

  {#if prefs.theme === 'b' && days.length}<LedgerHead />{/if}
  {#each days as day (day.key)}
    <section class="day">
      {#if prefs.theme !== 'b'}
        <div class="day-head">
          <span>{dayHeader(day.date)}</span>
          <span class="num">{fmt(day.net, 'always')}</span>
        </div>
      {/if}
      <div class={prefs.theme === 'b' ? '' : 'list'}>
        {#each day.items as tx (tx.id)}<TxRow {tx} showTime={false} />{/each}
      </div>
    </section>
  {/each}
  {#if !loading && !days.length}<p class="empty">{q ? t('tx.no_results') : t('tx.empty')}</p>{/if}
</div>

<style>
  .month-nav {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .month-nav .m {
    font-size: 14px;
    font-weight: 600;
    min-width: 96px;
    text-align: center;
  }
  .month-nav .icon-btn {
    width: 36px;
    height: 36px;
    flex-basis: 36px;
  }
  .month-nav .icon-btn:disabled {
    opacity: 0.3;
  }
  .sum {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
  .sum > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 12px 10px;
    border-radius: var(--r-card);
    background: var(--surface);
    border: 1px solid var(--line);
    min-width: 0;
  }
  .sum small {
    font-size: 12px;
    color: var(--muted);
  }
  .sum b {
    font-size: 14px;
    letter-spacing: -0.02em;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  :global([data-theme='b']) .sum {
    gap: 0;
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
  }
  :global([data-theme='b']) .sum > div {
    background: none;
    border: none;
    border-left: 1px solid var(--line);
    border-radius: 0;
    padding: 12px 10px;
  }
  :global([data-theme='b']) .sum > div:first-child {
    border-left: none;
    padding-left: 0;
  }
  :global([data-theme='b']) .sum b {
    font-size: 17px;
  }
  :global([data-theme='c']) .sum > div {
    border: none;
  }
  :global([data-theme='c']) .sum b {
    font-weight: 700;
  }
  .sep {
    flex: 0 0 1px;
    background: var(--line2);
    margin: 6px 2px;
  }
  .day {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .day-head {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
    font-weight: 600;
    color: var(--muted);
    padding: 0 2px;
  }
  .day + .day {
    margin-top: -6px;
  }
  .search-row {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: -8px;
  }
  .search {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 44px;
    padding: 0 12px;
    border-radius: 22px;
    background: var(--surface);
    border: 1px solid var(--line);
    color: var(--muted);
  }
  .search input {
    flex: 1;
    min-width: 0;
    border: none;
    background: transparent;
    outline: none;
    color: var(--fg);
  }
  .search button {
    border: none;
    background: none;
    color: var(--muted);
    padding: 4px;
    display: flex;
  }
  .dl {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--fg);
  }
</style>
