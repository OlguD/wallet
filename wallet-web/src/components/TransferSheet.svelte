<script>
  // Kendi hesapların arasında para aktarma. Mevcut bir transferde (tx) sadece
  // tarih/açıklama düzenlenir veya silinir; tutar için silip yeniden oluşturulur.
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { convert, rate, fmtRate } from '../lib/fx.js'
  import { t, errorText, longDate } from '../lib/i18n.js'
  import { fmt, fmtc, symbol, parseAmount, kurusToKeypad, decimalSep } from '../lib/money.js'
  import { app, closeSheet, refresh, toast } from '../lib/store.svelte.js'

  let { sheet } = $props()
  const tx = sheet.tx || null // mevcut transferin herhangi bir bacağı
  const preset = sheet.preset || {}

  const pad = (n) => String(n).padStart(2, '0')
  const toYmd = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const today = toYmd(new Date())

  // Mevcut transferde yön: gider bacağı kaynak.
  const outLeg = tx ? (tx.type === 'expense' ? tx : null) : null
  let fromId = $state(tx ? (tx.type === 'expense' ? tx.account_id : tx.transfer_account_id) : (preset.fromId ?? app.accounts[0]?.id))
  let toId = $state(tx ? (tx.type === 'expense' ? tx.transfer_account_id : tx.account_id) : (preset.toId ?? app.accounts.find((a) => a.id !== fromId)?.id))
  let amountText = $state(tx ? kurusToKeypad(tx.amount).replace('.', decimalSep()) : '')
  let toAmountText = $state('')
  let toTouched = $state(false)
  let note = $state(tx?.description || '')
  let date = $state(tx ? toYmd(new Date(tx.occurred_at)) : today)
  let busy = $state(false)

  const from = $derived(app.accounts.find((a) => a.id === fromId))
  const to = $derived(app.accounts.find((a) => a.id === toId))
  const amount = $derived(parseAmount(amountText) || 0)
  const crossCurrency = $derived(from && to && from.currency !== to.currency)
  const fxRate = $derived(crossCurrency ? rate(from.currency, to.currency) : null)

  // Döviz farklıysa hedef tutarı kurdan öner (elle değiştirilmediyse).
  $effect(() => {
    if (!crossCurrency || toTouched || tx) return
    const v = convert(amount, from.currency, to.currency)
    toAmountText = v ? kurusToKeypad(v).replace('.', decimalSep()) : ''
  })

  function pickFrom(id) {
    fromId = id
    if (toId === id) toId = app.accounts.find((a) => a.id !== id)?.id
    toTouched = false
  }

  function occurredAt() {
    if (date === today) return undefined
    const [y, m, d] = date.split('-').map(Number)
    return new Date(y, m - 1, d, 12, 0, 0).toISOString()
  }

  async function save(e) {
    e.preventDefault()
    if (busy) return
    busy = true
    try {
      if (tx) {
        const body = {}
        if ((note.trim() || null) !== (tx.description || null)) body.description = note.trim() || null
        if (date !== toYmd(new Date(tx.occurred_at))) body.occurred_at = occurredAt() || new Date().toISOString()
        if (Object.keys(body).length) await api.patch(`/transactions/${tx.id}`, body)
        toast(t('add.saved'))
      } else {
        if (!amount) return toast(t('add.need_amount'), 'err')
        await api.post('/transfers', {
          from_account_id: fromId, to_account_id: toId, amount,
          to_amount: crossCurrency ? parseAmount(toAmountText) || undefined : undefined,
          description: note.trim() || undefined, occurred_at: occurredAt(),
        })
        navigator.vibrate?.([10, 40, 10])
        toast(t('tr.saved'))
      }
      closeSheet()
      refresh()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }

  async function remove() {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/transactions/${tx.id}`)
      toast(t('tr.deleted'))
      closeSheet()
      refresh()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }
</script>

<Sheet onclose={closeSheet} top={56} label={t('tr.title')}>
  <form class="form" onsubmit={save}>
    <h2 class="h-section" style="padding-top: 4px">{t('tr.title')}</h2>
    {#if app.accounts.length < 2}
      <p class="hint">{t('tr.need_two')}</p>
    {:else}
      <div class="field">
        <span>{t('tr.from')}</span>
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each app.accounts as a}
            <button type="button" class="chip small" aria-pressed={fromId === a.id} disabled={!!tx} onclick={() => pickFrom(a.id)}>
              {a.name} · {fmt(a.balance)} {symbol(a.currency)}
            </button>
          {/each}
        </div>
      </div>
      <div class="arrow" aria-hidden="true"><Icon d={I.arrowDown} size={20} stroke={2} /></div>
      <div class="field">
        <span>{t('tr.to')}</span>
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each app.accounts.filter((a) => a.id !== fromId) as a}
            <button type="button" class="chip small" aria-pressed={toId === a.id} disabled={!!tx} onclick={() => { toId = a.id; toTouched = false }}>
              {a.name} · {fmt(a.balance)} {symbol(a.currency)}
            </button>
          {/each}
        </div>
      </div>
      <label class="field">
        <span>{t('tr.amount')} ({symbol(from?.currency)})</span>
        <input class="input num big" bind:value={amountText} inputmode="decimal" placeholder="0" disabled={!!tx} />
      </label>
      {#if crossCurrency && !tx}
        <label class="field">
          <span>{t('tr.to_amount', { c: symbol(to.currency) })}</span>
          <input class="input num" bind:value={toAmountText} inputmode="decimal" placeholder="0" oninput={() => (toTouched = true)} />
          {#if fxRate}<small class="hint">{t('tr.rate', { from: from.currency, to: to.currency, v: fmtRate(fxRate) })}</small>{/if}
        </label>
      {/if}
      <label class="field">
        <span>{t('add.date')}</span>
        <input class="input" type="date" bind:value={date} max={today} />
      </label>
      <label class="field">
        <span>{t('add.note')}</span>
        <input class="input" bind:value={note} maxlength="255" placeholder={from && to ? `${from.name} → ${to.name}` : ''} />
      </label>
      {#if from && to && amount && !tx}
        <p class="preview">
          {from.name}: {fmt(from.balance)} → <b>{fmt(from.balance - amount)}</b> {symbol(from.currency)}<br />
          {to.name}: {fmt(to.balance)} → <b>{fmt(to.balance + (crossCurrency ? parseAmount(toAmountText) || 0 : amount))}</b> {symbol(to.currency)}
        </p>
      {/if}
      <p class="hint">{tx ? t('tr.locked') : t('tr.hint')}</p>
      <button class="btn" disabled={busy || (!tx && (!amount || !toId))}>{t('common.save')}</button>
      {#if tx}<button type="button" class="btn danger" onclick={remove}>{t('common.delete')}</button>{/if}
    {/if}
  </form>
</Sheet>

<style>
  .arrow {
    align-self: center;
    color: var(--muted);
    margin: -8px 0;
  }
  .big {
    font-size: 22px;
  }
  .preview {
    font-size: 14px;
    color: var(--muted);
    line-height: 1.6;
  }
  .preview b {
    color: var(--fg);
  }
</style>
