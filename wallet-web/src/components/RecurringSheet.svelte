<script>
  // Düzenli ödemeyi düzenleme: tutar/açıklama/bitiş; duraklat ve sil.
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { catsFor } from '../lib/categories.js'
  import { t, errorText, fullDate } from '../lib/i18n.js'
  import { parseAmount, kurusToKeypad, decimalSep, symbol } from '../lib/money.js'
  import { toast, refresh } from '../lib/store.svelte.js'

  let { rule, onclose, onchange } = $props()

  let amountText = $state(kurusToKeypad(rule.amount).replace('.', decimalSep()))
  let description = $state(rule.description || '')
  let category = $state(rule.category)
  let endOn = $state(rule.end_on || '')
  let busy = $state(false)

  async function save(e) {
    e.preventDefault()
    const amount = parseAmount(amountText)
    if (!amount || busy) return
    busy = true
    try {
      const body = {}
      if (amount !== rule.amount) body.amount = amount
      if ((description.trim() || null) !== (rule.description || null)) body.description = description.trim() || null
      if (category !== rule.category) body.category = category
      if ((endOn || null) !== (rule.end_on || null)) body.end_on = endOn || null
      if (Object.keys(body).length) {
        await api.patch(`/recurring/${rule.id}`, body)
        toast(t('rec.saved'))
        onchange?.()
      }
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }

  async function toggle() {
    try {
      await api.patch(`/recurring/${rule.id}`, { active: !rule.active })
      onchange?.()
      refresh()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }

  async function remove() {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/recurring/${rule.id}`)
      onchange?.()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }
</script>

<Sheet {onclose} top={70} label={t('rec.edit')}>
  <form class="form" onsubmit={save}>
    <h2 class="h-section" style="padding-top: 4px">{t('rec.edit')}</h2>
    <p class="hint">
      {rule.account_name} · {t('repeat.' + rule.frequency)} ·
      {rule.active ? t('rec.next', { d: fullDate(rule.next_run_on + 'T12:00:00') }) : t('rec.paused')}
    </p>
    <label class="field">
      <span>{t('common.amount')} ({symbol(rule.currency)})</span>
      <input class="input num" bind:value={amountText} inputmode="decimal" required />
    </label>
    <label class="field">
      <span>{t('add.note')}</span>
      <input class="input" bind:value={description} maxlength="200" />
    </label>
    <div class="field">
      <span>{t('add.category')}</span>
      <div class="chips" style="margin: 0; padding: 0; flex-wrap: wrap">
        {#each catsFor(rule.type) as c}
          <button type="button" class="chip small" aria-pressed={category === c.id} onclick={() => (category = c.id)}>
            <Icon d={c.icon} size={15} />{t('cat.' + c.id)}
          </button>
        {/each}
      </div>
    </div>
    <label class="field">
      <span>{t('rec.end')}</span>
      <input class="input" type="date" bind:value={endOn} min={rule.next_run_on} />
    </label>
    <p class="hint">{t('rec.future_only')}</p>
    <button class="btn" disabled={busy || !parseAmount(amountText)}>{t('common.save')}</button>
    <button type="button" class="btn ghost" onclick={toggle}>
      <Icon d={rule.active ? I.pause : I.play} size={16} stroke={2.2} />{rule.active ? t('rec.pause') : t('rec.resume')}
    </button>
    <button type="button" class="btn danger" onclick={remove}>{t('common.delete')}</button>
  </form>
</Sheet>
