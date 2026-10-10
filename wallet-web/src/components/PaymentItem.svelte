<script>
  // Gelen kutusunda bekleyen grup ödemesi: başka üyenin girdiği hesaplaşma.
  // Kullanıcı hangi hesabına geldiğini/hangi hesabından çıktığını seçer ve
  // işler ya da hesaba işlemeden kapatır.
  import Icon from './Icon.svelte'
  import FxAmount from './FxAmount.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, errorText, dayLabel } from '../lib/i18n.js'
  import { fmtc, parseAmount } from '../lib/money.js'
  import { app, refresh, toast } from '../lib/store.svelte.js'

  let { p } = $props()

  const incoming = $derived(p.to_user_id === app.user?.id)
  // Önce ödemenin para birimindeki hesaplar.
  const accounts = $derived([...app.accounts].sort((a, b) => (b.currency === p.currency) - (a.currency === p.currency)))
  let accountId = $state(app.accounts.find((a) => a.currency === p.currency)?.id ?? app.accounts[0]?.id)
  let accAmountText = $state('')
  let busy = $state(false)

  const account = $derived(app.accounts.find((a) => a.id === accountId))
  const cross = $derived(account && account.currency !== p.currency)

  async function act(book) {
    busy = true
    try {
      if (book)
        await api.post(`/settlements/${p.id}/book`, {
          account_id: accountId,
          account_amount: cross ? parseAmount(accAmountText) || undefined : undefined,
        })
      else await api.post(`/settlements/${p.id}/dismiss`)
      toast(book ? t('pay.booked', { acc: account.name }) : t('pay.dismissed'))
      await refresh()
    } catch (e) {
      toast(errorText(e), 'err')
    } finally {
      busy = false
    }
  }
</script>

<div class="card pay rise">
  <div class="head">
    <span class="ico" class:pos={incoming}><Icon d={incoming ? I.arrowDown : I.arrowUp} size={18} /></span>
    <span class="grow">
      <span class="title">{incoming ? t('pay.in_title', { name: p.from_username }) : t('pay.out_title', { name: p.to_username })}</span>
      <span class="sub">{p.group_name} · {dayLabel(p.occurred_at)}</span>
    </span>
    <span class="amount num" class:pos={incoming}>{fmtc(p.amount, p.currency)}</span>
  </div>
  {#if app.accounts.length}
    <span class="q">{incoming ? t('pay.which_in') : t('pay.which_out')}</span>
    <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
      {#each accounts as a (a.id)}
        <button type="button" class="chip small" aria-pressed={accountId === a.id} onclick={() => (accountId = a.id)}>{a.name} · {a.currency}</button>
      {/each}
    </div>
    {#if cross}
      {#key account.id}
        <FxAmount amount={p.amount} currency={p.currency} accountCurrency={account.currency} direction={incoming ? 'in' : 'out'} bind:value={accAmountText} />
      {/key}
    {/if}
  {/if}
  <div class="acts">
    <button type="button" class="btn small ghost" disabled={busy} onclick={() => act(false)}>{t('pay.dismiss')}</button>
    {#if app.accounts.length}
      <button type="button" class="btn small" disabled={busy || (cross && !parseAmount(accAmountText))} onclick={() => act(true)}><Icon d={I.check} size={16} stroke={2.4} />{t('pay.book')}</button>
    {/if}
  </div>
</div>

<style>
  .pay {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .head {
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
  .title {
    font-weight: 600;
  }
  .sub,
  .q {
    font-size: 13px;
    color: var(--muted);
  }
  .amount {
    font-weight: 700;
  }
  .acts {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
