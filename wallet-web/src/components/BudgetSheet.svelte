<script>
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { api } from '../lib/api.js'
  import { EXPENSE_CATS } from '../lib/categories.js'
  import { t, errorText } from '../lib/i18n.js'
  import { CURRENCIES, parseAmount, kurusToKeypad, decimalSep, symbol } from '../lib/money.js'
  import { toast } from '../lib/store.svelte.js'

  let { budget = null, taken = [], onclose, onsaved } = $props()

  let category = $state(budget?.category || EXPENSE_CATS.find((c) => !taken.includes(c.id + ':TRY'))?.id || 'groceries')
  let currency = $state(budget?.currency || 'TRY')
  let amountText = $state(budget ? kurusToKeypad(budget.amount).replace('.', decimalSep()) : '')
  let busy = $state(false)

  async function save(e) {
    e.preventDefault()
    const amount = parseAmount(amountText)
    if (!amount || busy) return
    busy = true
    try {
      if (budget) await api.patch(`/budgets/${budget.id}`, { amount })
      else await api.post('/budgets', { category, currency, amount })
      onsaved?.()
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
      await api.del(`/budgets/${budget.id}`)
      onsaved?.()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }
</script>

<Sheet {onclose} top={110} label={budget ? t('bud.edit') : t('bud.new')}>
  <form class="form" onsubmit={save}>
    <h2 class="h-section" style="padding-top: 4px">{budget ? t('cat.' + budget.category) : t('bud.new')}</h2>
    {#if !budget}
      <div class="field">
        <span>{t('bud.category')}</span>
        <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
          {#each EXPENSE_CATS as c}
            <button type="button" class="chip small" aria-pressed={category === c.id} disabled={taken.includes(c.id + ':' + currency)} onclick={() => (category = c.id)}>
              <Icon d={c.icon} size={15} />{t('cat.' + c.id)}
            </button>
          {/each}
        </div>
      </div>
      <div class="field">
        <span>{t('acc.currency')}</span>
        <div class="seg" role="group" style="align-self: flex-start">
          {#each CURRENCIES as c}<button type="button" aria-pressed={currency === c} onclick={() => (currency = c)}>{c}</button>{/each}
        </div>
      </div>
    {/if}
    <label class="field">
      <span>{t('bud.amount')} ({symbol(currency)})</span>
      <input class="input num" bind:value={amountText} inputmode="decimal" placeholder="0" required />
    </label>
    <button class="btn" disabled={busy || !parseAmount(amountText)}>{budget ? t('common.save') : t('common.create')}</button>
    {#if budget}<button type="button" class="btn danger" onclick={remove}>{t('common.delete')}</button>{/if}
  </form>
</Sheet>
