<script>
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { GOAL_ICONS } from '../lib/categories.js'
  import { t, errorText } from '../lib/i18n.js'
  import { parseAmount, kurusToKeypad, decimalSep, fmt, symbol } from '../lib/money.js'
  import { app, toast } from '../lib/store.svelte.js'

  let { goal = null, onclose, onsaved, ondeleted } = $props()

  let name = $state(goal?.name || '')
  let icon = $state(goal?.icon || 'target')
  let accountId = $state(goal?.account_id ?? app.accounts[0]?.id)
  let target = $state(goal ? kurusToKeypad(goal.target_amount).replace('.', decimalSep()) : '')
  let initial = $state('')
  let date = $state(goal?.target_date || '')
  let busy = $state(false)

  const account = $derived(app.accounts.find((a) => a.id === accountId))

  async function save(e) {
    e.preventDefault()
    const amount = parseAmount(target)
    if (!name.trim() || !amount || busy) return
    busy = true
    try {
      let saved
      if (goal) saved = await api.patch(`/goals/${goal.id}`, { name, icon, target_amount: amount, target_date: date || null })
      else
        saved = await api.post('/goals', {
          account_id: accountId, name, icon, target_amount: amount, target_date: date || null,
          initial_amount: parseAmount(initial) || 0,
        })
      onsaved?.(saved)
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
      await api.del(`/goals/${goal.id}`)
      ondeleted?.()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }
</script>

<Sheet {onclose} top={90} label={goal ? goal.name : t('goal.new')}>
  <form class="form" onsubmit={save}>
    <h2 class="h-section" style="padding-top: 4px">{goal ? t('common.edit') : t('goal.new')}</h2>
    <label class="field">
      <span>{t('goal.name')}</span>
      <input class="input" bind:value={name} placeholder={t('goal.name_ph')} maxlength="64" required />
    </label>
    <div class="field">
      <span>{t('goal.icon')}</span>
      <div class="icons">
        {#each GOAL_ICONS as k}
          <button type="button" class="ic" aria-pressed={icon === k} aria-label={k} onclick={() => (icon = k)}><Icon d={I[k]} size={20} /></button>
        {/each}
      </div>
    </div>
    <label class="field">
      <span>{t('goal.target')}</span>
      <input class="input num" bind:value={target} inputmode="decimal" placeholder="0" required />
    </label>
    {#if !goal}
      <div class="field">
        <span>{t('goal.account')}</span>
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each app.accounts as a}
            <button type="button" class="chip small" aria-pressed={accountId === a.id} onclick={() => (accountId = a.id)}>{a.name}</button>
          {/each}
        </div>
        {#if account}<small class="muted">{t('goal.account_note', { acc: account.name, v: `${fmt(account.balance)} ${symbol(account.currency)}` })}</small>{/if}
      </div>
      <label class="field">
        <span>{t('goal.initial')}</span>
        <input class="input num" bind:value={initial} inputmode="decimal" placeholder="0" />
      </label>
    {/if}
    <label class="field">
      <span>{t('goal.date')}</span>
      <input class="input" type="date" bind:value={date} />
    </label>
    <button class="btn" disabled={busy || !name.trim() || !parseAmount(target)}>{goal ? t('common.save') : t('common.create')}</button>
    {#if goal}<button type="button" class="btn danger" onclick={remove}>{t('common.delete')}</button>{/if}
  </form>
</Sheet>

<style>
  .icons {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .ic {
    width: 46px;
    height: 46px;
    border-radius: var(--r-input);
    border: 1px solid var(--line2);
    background: var(--surface);
    color: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .ic[aria-pressed='true'] {
    background: var(--chip-on-bg);
    border-color: var(--chip-on-bg);
    color: var(--chip-on-fg);
  }
</style>
