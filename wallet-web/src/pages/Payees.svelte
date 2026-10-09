<script>
  // "Kime gitti / kimden geldi" raporu: karşı tarafa göre toplamlar.
  import Icon from '../components/Icon.svelte'
  import TxRow from '../components/TxRow.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { fmtIban } from '../lib/receipt.js'
  import { t, monthLabel, dayLabel } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { back } from '../lib/router.svelte.js'
  import { app, monthRange } from '../lib/store.svelte.js'

  let type = $state('expense')
  let period = $state('month') // 'month' | 'all'
  let rows = $state([])
  let loading = $state(true)
  let open = $state(null) // key
  let txs = $state([])

  $effect(() => {
    app.version
    const params = { type, ...(period === 'month' ? monthRange() : {}) }
    loading = true
    api.get('/reports/counterparties', params).then((r) => (rows = r)).catch(() => (rows = [])).finally(() => (loading = false))
  })

  // Para birimine göre toplam ve en büyük tutara göre çubuk oranı.
  const maxTotal = $derived(Math.max(1, ...rows.map((r) => r.total)))

  async function toggle(r) {
    if (open === r.key) return (open = null)
    open = r.key
    txs = []
    const params = { counterparty: r.key, limit: 20, ...(period === 'month' ? monthRange() : {}) }
    txs = await api.get('/transactions', params).catch(() => [])
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{type === 'expense' ? t('cp.title') : t('cp.title_in')}</h1>
  </header>
  <div class="rule"></div>

  <div class="seg" role="group">
    <button type="button" aria-pressed={type === 'expense'} onclick={() => (type = 'expense')}>{t('cp.out')}</button>
    <button type="button" aria-pressed={type === 'income'} onclick={() => (type = 'income')}>{t('cp.in')}</button>
  </div>
  <div class="chips" style="margin: -8px 0 0">
    <button type="button" class="chip small" aria-pressed={period === 'month'} onclick={() => (period = 'month')}>{monthLabel(app.month)}</button>
    <button type="button" class="chip small" aria-pressed={period === 'all'} onclick={() => (period = 'all')}>{t('cp.all_time')}</button>
  </div>

  <div class="list">
    {#each rows as r, i (r.key + r.currency)}
      <button type="button" class="row cp rise" style="animation-delay: {i * 0.04}s" aria-expanded={open === r.key} onclick={() => toggle(r)}>
        <span class="ico"><Icon d={I.person} size={18} /></span>
        <span class="grow">
          <span class="title">{r.name || t('cp.unnamed')}</span>
          <span class="sub">{[r.bank, r.iban ? fmtIban(r.iban) : null].filter(Boolean).join(' · ')}</span>
          <span class="sub">{t('cp.count', { n: r.count })} · {t('cp.last', { d: dayLabel(r.last_at + 'T12:00:00') })}</span>
          <span class="bar"><span style="width: {(r.total / maxTotal) * 100}%"></span></span>
        </span>
        <span class="end"><span class="amount" class:pos={type === 'income'}>{fmtc(r.total, r.currency)}</span></span>
      </button>
      {#if open === r.key}
        <div class="sub-list">
          {#each txs as tx (tx.id)}<TxRow {tx} />{/each}
        </div>
      {/if}
    {/each}
  </div>
  {#if !loading && !rows.length}
    <div class="empty">
      <Icon d={I.receipt} size={40} stroke={1.4} />
      <p>{t('cp.empty')}</p>
    </div>
  {/if}
</div>

<style>
  .seg button {
    flex: 1;
  }
  .cp .grow {
    gap: 3px;
  }
  .cp .bar {
    margin-top: 6px;
    height: 5px;
  }
  .sub-list {
    padding-left: 16px;
    border-left: 2px solid var(--line2);
    margin: 0 0 8px 20px;
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
