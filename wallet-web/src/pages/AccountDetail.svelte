<script>
  import Icon from '../components/Icon.svelte'
  import TxRow from '../components/TxRow.svelte'
  import LedgerHead from '../components/LedgerHead.svelte'
  import AccountSheet from '../components/AccountSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t } from '../lib/i18n.js'
  import { fmt, fmtc, symbol } from '../lib/money.js'
  import { pocketColor } from '../lib/categories.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { route, back } from '../lib/router.svelte.js'
  import { app, accountMonth, accountIndex, openAdd } from '../lib/store.svelte.js'

  const PAGE = 50
  let list = $state([])
  let more = $state(false)
  let editing = $state(false)

  const account = $derived(app.accounts.find((a) => a.id === route.id))
  const m = $derived(accountMonth(route.id))
  const col = $derived(pocketColor(Math.max(0, accountIndex(route.id))))

  $effect(() => {
    app.version
    const id = route.id
    api.get(`/accounts/${id}/transactions`, { limit: PAGE }).then((r) => {
      list = r
      more = r.length === PAGE
    })
  })

  async function loadMore() {
    const r = await api.get(`/accounts/${route.id}/transactions`, { limit: PAGE, offset: list.length })
    list = [...list, ...r]
    more = r.length === PAGE
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/accounts')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{account?.name ?? ''}</h1>
    {#if account}<button type="button" class="icon-btn" aria-label={t('common.edit')} onclick={() => (editing = true)}><Icon d={I.edit} size={18} /></button>{/if}
  </header>
  <div class="rule"></div>

  {#if account}
    <section class="hero" class:hero-c={prefs.theme === 'c'} style={prefs.theme === 'c' ? `background: ${col.bg}; color: ${col.fg}` : ''}>
      <span class="k">{t('acc.kind.' + account.kind)} · {account.currency}</span>
      <span class="b">{fmt(account.balance)} <small>{symbol(account.currency)}</small></span>
      <span class="k">{t('acc.month_in_out')}: <b>+{fmt(m.income)}</b> / <b>−{fmt(m.expense)}</b></span>
    </section>
    <button type="button" class="btn ghost" onclick={() => openAdd({ accountId: account.id })}><Icon d={I.plus} size={18} stroke={2.2} />{t('nav.add')}</button>
  {/if}

  <section class="section">
    {#if prefs.theme === 'b' && list.length}<LedgerHead />{/if}
    <div class={prefs.theme === 'b' ? '' : 'list'}>
      {#each list as tx (tx.id)}<TxRow {tx} />{/each}
    </div>
    {#if !list.length}<p class="empty">{t('common.empty')}</p>{/if}
    {#if more}<button type="button" class="btn ghost small" style="align-self: center" onclick={loadMore}>{t('common.more')}</button>{/if}
  </section>
</div>

{#if editing && account}<AccountSheet {account} onclose={() => (editing = false)} />{/if}

<style>
  .hero {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .hero .k {
    font-size: 13px;
    color: var(--muted);
  }
  .hero .k b {
    font-weight: 600;
    color: var(--fg);
  }
  .hero .b {
    font-family: var(--font-display);
    font-weight: var(--display-weight);
    letter-spacing: var(--display-track);
    font-size: 44px;
    line-height: 1;
  }
  .hero .b small {
    font-size: 24px;
    color: var(--muted2);
    font-weight: 500;
  }
  .hero-c {
    border-radius: 26px;
    padding: 20px;
    box-shadow: 0 10px 24px rgba(30, 26, 18, 0.16);
  }
  .hero-c .k,
  .hero-c .k b,
  .hero-c .b small {
    color: inherit;
    opacity: 0.85;
  }
</style>
