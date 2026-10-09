<script>
  import Sheet from './Sheet.svelte'
  import { api } from '../lib/api.js'
  import { t, errorText } from '../lib/i18n.js'
  import { parseAmount, kurusToKeypad, decimalSep, symbol } from '../lib/money.js'
  import { app, refresh, toast } from '../lib/store.svelte.js'

  // s: { from_user_id, from_username, to_user_id, to_username, amount, currency }
  let { groupId, s, onclose, onsaved } = $props()

  let amount = $state(kurusToKeypad(s.amount).replace('.', decimalSep()))
  const myAccounts = app.accounts.filter((a) => a.currency === s.currency)
  let toAccount = $state(myAccounts.length > 0)
  let accountId = $state(myAccounts[0]?.id ?? null)
  let busy = $state(false)

  async function submit(e) {
    e.preventDefault()
    const k = parseAmount(amount)
    if (!k || busy) return
    busy = true
    try {
      await api.post(`/groups/${groupId}/settlements`, {
        from_user_id: s.from_user_id,
        to_user_id: s.to_user_id,
        amount: k,
        currency: s.currency,
        account_id: toAccount ? accountId : undefined,
      })
      toast(t('settle.saved'))
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

<Sheet {onclose} top={180} label={t('settle.title')}>
  <form class="form" onsubmit={submit}>
    <h2 class="h-section" style="padding-top: 4px">{t('settle.title')}</h2>
    <p class="who"><b>{s.from_username}</b> → <b>{s.to_username}</b></p>
    <label class="field">
      <span>{t('ledger.amount')} ({symbol(s.currency)})</span>
      <input class="input num" bind:value={amount} inputmode="decimal" required />
    </label>
    {#if myAccounts.length}
      <label class="toggle">
        <input type="checkbox" bind:checked={toAccount} />
        <span>{t('settle.to_account')}</span>
      </label>
      {#if toAccount}
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each myAccounts as a}
            <button type="button" class="chip small" aria-pressed={accountId === a.id} onclick={() => (accountId = a.id)}>{a.name}</button>
          {/each}
        </div>
      {/if}
    {/if}
    <button class="btn" disabled={busy || !parseAmount(amount)}>{t('common.save')}</button>
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
