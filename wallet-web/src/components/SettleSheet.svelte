<script>
  // Grup içi ödeme. s verilirse önerilen borcun kapatılması (taraflar sabit);
  // verilmezse "Para gönder": ben → seçilen üye, tutar serbest. Kayıt karşı
  // tarafın gelen kutusuna düşer; o da kendi hesabına işler.
  import Sheet from './Sheet.svelte'
  import FxAmount from './FxAmount.svelte'
  import { api } from '../lib/api.js'
  import { t, errorText } from '../lib/i18n.js'
  import { parseAmount, kurusToKeypad, decimalSep, symbol, CURRENCIES } from '../lib/money.js'
  import { app, refresh, toast } from '../lib/store.svelte.js'

  // s: { from_user_id, from_username, to_user_id, to_username, amount, currency }
  let { groupId, s = null, members = [], onclose, onsaved } = $props()

  const me = app.user?.id
  const free = !s
  const others = members.filter((m) => m.user_id !== me && !m.deleted)
  // Dövizli ödeme: hesap farklı para birimindeyse hesaba işlenen tutar kurla çevrilir.
  const myAccounts = app.accounts

  let toId = $state(s?.to_user_id ?? others[0]?.user_id)
  let amount = $state(s ? kurusToKeypad(s.amount).replace('.', decimalSep()) : '')
  let toAccount = $state(myAccounts.length > 0)
  let accountId = $state(myAccounts[0]?.id ?? null)
  let currency = $state(s?.currency ?? app.accounts[0]?.currency ?? 'TRY')
  let accAmountText = $state('')
  // Borca say: kapalıysa hediye/harçlık; grup borç durumu değişmez.
  let affects = $state(true)
  let busy = $state(false)

  const account = $derived(app.accounts.find((a) => a.id === accountId))
  const cur = $derived(free ? currency : s.currency)
  const cross = $derived(toAccount && account && account.currency !== cur)
  // Kaydı giren ödeyense hesabından çıkar, alansa hesabına girer.
  const outgoing = $derived(free || s.from_user_id === me)
  const toName = $derived(free ? others.find((m) => m.user_id === toId)?.username : null)

  async function submit(e) {
    e.preventDefault()
    const k = parseAmount(amount)
    if (!k || busy || (free && !toId)) return
    busy = true
    try {
      await api.post(`/groups/${groupId}/settlements`, {
        from_user_id: free ? me : s.from_user_id,
        to_user_id: free ? toId : s.to_user_id,
        amount: k,
        currency: cur,
        account_id: toAccount ? accountId : undefined,
        account_amount: cross ? parseAmount(accAmountText) || undefined : undefined,
        affects_balance: free ? affects : true,
      })
      toast(free ? t('pay.sent', { name: toName }) : t('settle.saved'))
      refresh()
      onsaved?.()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }
</script>

<Sheet {onclose} top={free ? 110 : 180} label={free ? t('pay.title') : t('settle.title')}>
  <form class="form" onsubmit={submit}>
    <h2 class="h-section" style="padding-top: 4px">{free ? t('pay.title') : t('settle.title')}</h2>
    {#if free}
      <div class="field">
        <span>{t('pay.to')}</span>
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each others as m (m.user_id)}
            <button type="button" class="chip small" aria-pressed={toId === m.user_id} onclick={() => (toId = m.user_id)}>{m.username}</button>
          {/each}
        </div>
      </div>
    {:else}
      <p class="who"><b>{s.from_username}</b> → <b>{s.to_username}</b></p>
    {/if}
    {#if free}
      <div class="field">
        <span>{t('pay.currency')}</span>
        <div class="chips" style="margin: 0; padding: 0">
          {#each CURRENCIES as c}
            <button type="button" class="chip small" aria-pressed={currency === c} onclick={() => (currency = c)}>{c}</button>
          {/each}
        </div>
      </div>
    {/if}
    <label class="field">
      <span>{t('ledger.amount')} ({symbol(cur)})</span>
      <input class="input num" bind:value={amount} inputmode="decimal" placeholder="0" required />
    </label>
    {#if myAccounts.length}
      <label class="toggle">
        <input type="checkbox" bind:checked={toAccount} />
        <span>{free ? t('pay.from_account') : t('settle.to_account')}</span>
      </label>
      {#if toAccount}
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each myAccounts as a}
            <button type="button" class="chip small" aria-pressed={accountId === a.id} onclick={() => (accountId = a.id)}>{a.name} · {a.currency}</button>
          {/each}
        </div>
        {#if cross}
          {#key `${cur}-${account.currency}`}
            <FxAmount amount={parseAmount(amount) || 0} currency={cur} accountCurrency={account.currency} direction={outgoing ? 'out' : 'in'} bind:value={accAmountText} />
          {/key}
        {/if}
      {/if}
    {/if}
    {#if free}
      <label class="toggle">
        <input type="checkbox" bind:checked={affects} />
        <span>{t('pay.affects')}</span>
      </label>
    {/if}
    <p class="hint">{free && !affects ? t('pay.gift_hint') : t('pay.hint')}</p>
    <button class="btn" disabled={busy || !parseAmount(amount) || (free && !toId) || (cross && !parseAmount(accAmountText))}>{free ? t('pay.send') : t('common.save')}</button>
  </form>
</Sheet>

<style>
  .who {
    font-size: 18px;
  }
  .toggle {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 44px;
    font-size: 15px;
  }
  .toggle input {
    width: 22px;
    height: 22px;
    accent-color: var(--accent);
  }
</style>
