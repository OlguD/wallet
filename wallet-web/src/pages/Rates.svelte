<script>
  import Icon from '../components/Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t, timeLabel, dayLabel } from '../lib/i18n.js'
  import { CURRENCIES, symbol, parseAmount, fmt } from '../lib/money.js'
  import { rate, convert, fmtRate } from '../lib/fx.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { back } from '../lib/router.svelte.js'
  import { app, loadRates } from '../lib/store.svelte.js'

  const FLAGS = { USD: '🇺🇸', EUR: '🇪🇺', GBP: '🇬🇧', TRY: '🇹🇷' }

  let ch = $state('online')
  let from = $state('USD')
  let to = $state('TRY')
  let amountText = $state('100')
  let loading = $state(!app.rates)

  $effect(() => {
    loadRates().finally(() => (loading = false))
    // Sayfa açıkken kurlar dakikada bir tazelenir (sunucu 2 dk önbellekler).
    const id = setInterval(loadRates, 60000)
    return () => clearInterval(id)
  })

  const r = $derived(app.rates)
  const c = $derived(r ? (ch === 'desk' ? r.desk : r.online) : null)
  const amount = $derived(parseAmount(amountText) || 0)
  const result = $derived(r ? convert(amount, from, to, ch) : null)
  const k = $derived(r ? rate(from, to, ch) : null)
  const updated = $derived(r ? `${dayLabel(r.updated_at)} ${timeLabel(r.updated_at)}` : '')

  function swap() {
    ;[from, to] = [to, from]
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('fx.title')}</h1>
  </header>
  <div class="rule"></div>

  <div class="seg" role="group">
    <button type="button" aria-pressed={ch === 'online'} onclick={() => (ch = 'online')}>{t('fx.online')}</button>
    <button type="button" aria-pressed={ch === 'desk'} onclick={() => (ch = 'desk')}>{t('fx.desk')}</button>
  </div>

  {#if c}
    <section class="card table">
      <div class="tr th"><span></span><span>{t('fx.buy')}</span><span>{t('fx.sell')}</span></div>
      {#each ['USD', 'EUR', 'GBP'] as cur, i}
        <div class="tr rise" style="animation-delay: {i * 0.05}s">
          <span class="cur"><span class="flag" aria-hidden="true">{FLAGS[cur]}</span>{cur}</span>
          <span class="num">{fmtRate(c.rates[cur]?.buy)}</span>
          <span class="num">{fmtRate(c.rates[cur]?.sell)}</span>
        </div>
      {/each}
    </section>
    {#if Object.keys(c.cross).length}
      <section class="section">
        <h2 class="h-section">{t('fx.cross')}</h2>
        <div class="card table">
          {#each Object.entries(c.cross).sort() as [pair, q]}
            <div class="tr"><span class="cur">{pair}</span><span class="num">{fmtRate(q.buy)}</span><span class="num">{fmtRate(q.sell)}</span></div>
          {/each}
        </div>
      </section>
    {/if}
    <p class="hint">
      {t('fx.source')} · {t('fx.updated', { d: updated })}
      {#if r.stale}<br /><span class="warn">{t('fx.stale')}</span>{/if}
    </p>

    <section class="section">
      <h2 class="h-section">{t('fx.calc')}</h2>
      <div class="card calc">
        <div class="line">
          <input class="input num big" bind:value={amountText} inputmode="decimal" aria-label={t('fx.amount')} />
          <select class="input pick" bind:value={from} aria-label={t('fx.amount')}>
            {#each CURRENCIES as cur}<option value={cur}>{FLAGS[cur]} {cur}</option>{/each}
          </select>
        </div>
        <button type="button" class="swap" aria-label={t('fx.swap')} onclick={swap}><Icon d={I.swap} size={18} stroke={2} /></button>
        <div class="line">
          <output class="input num big out">{result === null ? '—' : fmt(result)}</output>
          <select class="input pick" bind:value={to} aria-label={t('fx.result')}>
            {#each CURRENCIES as cur}<option value={cur}>{FLAGS[cur]} {cur}</option>{/each}
          </select>
        </div>
        <p class="hint">
          {#if k !== null && from !== to}{t('fx.rate_line', { from, to, v: fmtRate(k) })} · {/if}{t('fx.hint')}
        </p>
      </div>
    </section>
  {:else if loading}
    <p class="empty">{t('common.loading')}</p>
  {:else}
    <div class="empty">
      <p>{t('fx.unavailable')}</p>
      <button type="button" class="btn small ghost" onclick={() => { loading = true; loadRates().finally(() => (loading = false)) }}>{t('common.retry')}</button>
    </div>
  {/if}
</div>

<style>
  .seg button {
    flex: 1;
  }
  .table {
    display: flex;
    flex-direction: column;
  }
  .tr {
    display: grid;
    grid-template-columns: 1.2fr 1fr 1fr;
    align-items: center;
    min-height: 48px;
    border-top: 1px solid var(--line);
  }
  .tr:first-child {
    border-top: none;
  }
  .tr > span:not(.cur) {
    text-align: right;
  }
  .th {
    min-height: 32px;
    font-size: 12px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .tr .num {
    font-size: 17px;
  }
  .cur {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 600;
  }
  .flag {
    font-size: 20px;
  }
  .warn {
    color: var(--danger);
  }
  .calc {
    display: flex;
    flex-direction: column;
    gap: 8px;
    position: relative;
  }
  .line {
    display: flex;
    gap: 8px;
  }
  .big {
    flex: 1;
    min-width: 0;
    font-size: 22px;
  }
  .out {
    display: flex;
    align-items: center;
    color: var(--fg);
    background: var(--surface2);
  }
  .pick {
    flex: 0 0 112px;
    width: 112px;
    appearance: none;
    -webkit-appearance: none;
    font-weight: 600;
  }
  .swap {
    align-self: center;
    width: 40px;
    height: 40px;
    border-radius: 20px;
    border: 1px solid var(--line2);
    background: var(--surface);
    color: var(--fg);
    display: flex;
    align-items: center;
    justify-content: center;
    transform: rotate(90deg);
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
