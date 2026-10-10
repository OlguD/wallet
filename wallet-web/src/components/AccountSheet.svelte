<script>
  import Sheet from './Sheet.svelte'
  import { api } from '../lib/api.js'
  import { t, errorText } from '../lib/i18n.js'
  import { CURRENCIES, parseAmount, kurusToKeypad, decimalSep } from '../lib/money.js'
  import { refresh, toast } from '../lib/store.svelte.js'
  import { navigate } from '../lib/router.svelte.js'

  // account verilirse düzenleme modu.
  let { account = null, onclose } = $props()

  let name = $state(account?.name || '')
  let kind = $state(account?.kind || 'bank')
  let currency = $state(account?.currency || 'TRY')
  // Kart: limit, son ödeme günü ve (yeni kartta) şu anki borç.
  const toText = (k) => (k ? kurusToKeypad(k).replace('.', decimalSep()) : '')
  let limitText = $state(toText(account?.credit_limit))
  let dueDay = $state(account?.due_day ? String(account.due_day) : '')
  let debtText = $state('')
  let busy = $state(false)
  const kinds = ['bank', 'cash', 'card', 'savings']

  async function save(e) {
    e.preventDefault()
    if (!name.trim() || busy) return
    busy = true
    try {
      const body = { name, kind }
      if (kind === 'card') Object.assign(body, { credit_limit: parseAmount(limitText) || 0, due_day: Number(dueDay) || 0 })
      if (account) await api.patch(`/accounts/${account.id}`, body)
      else await api.post('/accounts', { ...body, currency, opening_debt: kind === 'card' ? parseAmount(debtText) || 0 : 0 })
      await refresh()
      toast(t('add.saved'))
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }

  async function remove() {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/accounts/${account.id}`)
      await refresh()
      onclose()
      navigate('/accounts', { replace: true })
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }
</script>

<Sheet {onclose} top={kind === 'card' ? 70 : account ? 220 : 160} label={account ? t('acc.rename') : t('acc.new')}>
  <form class="form" onsubmit={save}>
    <h2 class="h-section" style="padding-top: 4px">{account ? t('acc.rename') : t('acc.new')}</h2>
    <label class="field">
      <span>{t('acc.name')}</span>
      <input class="input" bind:value={name} placeholder={t('acc.name_ph')} maxlength="64" required />
    </label>
    <div class="field">
      <span>{t('acc.kind')}</span>
      <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
        {#each kinds as k}
          <button type="button" class="chip small" aria-pressed={kind === k} onclick={() => (kind = k)}>{t('acc.kind.' + k)}</button>
        {/each}
      </div>
    </div>
    {#if kind === 'card'}
      <label class="field" data-tour="card-limit">
        <span>{t('card.limit')}</span>
        <input class="input num" bind:value={limitText} inputmode="decimal" placeholder="0" />
      </label>
      {#if !account}
        <label class="field">
          <span>{t('card.opening_debt')}</span>
          <input class="input num" bind:value={debtText} inputmode="decimal" placeholder="0" />
          <small class="muted">{t('card.opening_hint')}</small>
        </label>
      {/if}
      <label class="field">
        <span>{t('card.due_day')}</span>
        <input class="input num" bind:value={dueDay} inputmode="numeric" type="number" min="1" max="31" placeholder={t('card.due_ph')} />
      </label>
    {/if}
    {#if !account}
      <div class="field">
        <span>{t('acc.currency')}</span>
        <div class="chips" style="margin: 0; padding: 0">
          {#each CURRENCIES as c}
            <button type="button" class="chip small" aria-pressed={currency === c} onclick={() => (currency = c)}>{c}</button>
          {/each}
        </div>
      </div>
    {/if}
    <button class="btn" disabled={busy || !name.trim()}>{account ? t('common.save') : t('common.create')}</button>
    {#if account}<button type="button" class="btn danger" onclick={remove}>{t('common.delete')}</button>{/if}
  </form>
</Sheet>
