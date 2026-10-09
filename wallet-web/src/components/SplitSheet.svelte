<script>
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t } from '../lib/i18n.js'
  import { fmt, parseAmount, kurusToKeypad, decimalSep } from '../lib/money.js'
  import { app } from '../lib/store.svelte.js'

  // value: { mode: 'all' | 'equal' | 'exact', userIds: number[], shares: { [id]: string } }
  let { members, amount, value, onsave, onclose } = $props()

  let mode = $state(value.mode === 'exact' ? 'exact' : 'equal')
  let picked = $state(value.mode === 'all' ? members.map((m) => m.user_id) : [...value.userIds])
  let shares = $state(
    Object.fromEntries(
      members.map((m) => [m.user_id, value.shares?.[m.user_id] ?? (value.mode === 'exact' ? '' : '')]),
    ),
  )

  const sum = $derived(members.reduce((s, m) => s + (parseAmount(shares[m.user_id]) || 0), 0))
  const remaining = $derived(amount - sum)

  function toggle(id) {
    picked = picked.includes(id) ? picked.filter((x) => x !== id) : [...picked, id]
  }

  function fillEqual() {
    const n = members.length
    members.forEach((m, i) => {
      const base = Math.floor(amount / n) + (i < amount % n ? 1 : 0)
      shares[m.user_id] = kurusToKeypad(base).replace('.', decimalSep())
    })
  }

  function save() {
    if (mode === 'equal') {
      if (!picked.length) return
      const all = picked.length === members.length
      onsave({ mode: all ? 'all' : 'equal', userIds: picked, shares: {} })
    } else {
      if (remaining !== 0) return
      onsave({ mode: 'exact', userIds: [], shares: { ...shares } })
    }
    onclose()
  }

  const name = (m) => (m.user_id === app.user?.id ? t('grp.member_you', { name: m.username }) : m.username)
</script>

<Sheet {onclose} top={110} label={t('split.title')}>
  <div class="head">
    <h2 class="h-section">{t('split.title')}</h2>
    <div class="seg" role="group">
      <button type="button" aria-pressed={mode === 'equal'} onclick={() => (mode = 'equal')}>{t('split.mode_equal')}</button>
      <button type="button" aria-pressed={mode === 'exact'} onclick={() => { mode = 'exact'; if (!sum) fillEqual() }}>{t('split.mode_exact')}</button>
    </div>
  </div>

  <div class="list">
    {#each members as m}
      {#if mode === 'equal'}
        <button type="button" class="row" onclick={() => toggle(m.user_id)} aria-pressed={picked.includes(m.user_id)}>
          <span class="check" class:on={picked.includes(m.user_id)}>{#if picked.includes(m.user_id)}<Icon d={I.check} size={16} stroke={2.6} />{/if}</span>
          <span class="grow"><span class="title">{name(m)}</span></span>
          {#if picked.includes(m.user_id)}
            <span class="amount">{fmt(Math.floor(amount / picked.length))}</span>
          {/if}
        </button>
      {:else}
        <label class="row">
          <span class="grow"><span class="title">{name(m)}</span></span>
          <input class="input share num" inputmode="decimal" placeholder="0" bind:value={shares[m.user_id]} />
        </label>
      {/if}
    {/each}
  </div>

  {#if mode === 'exact'}
    <p class="hint" class:error={remaining !== 0}>
      {remaining === 0 ? '✓ ' + fmt(amount) : t('split.remaining', { v: fmt(remaining) })}
    </p>
  {/if}

  <button type="button" class="btn" disabled={mode === 'equal' ? !picked.length : remaining !== 0} onclick={save}>{t('common.done')}</button>
</Sheet>

<style>
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .check {
    width: 26px;
    height: 26px;
    flex: 0 0 26px;
    border-radius: 8px;
    border: 1.5px solid var(--line2);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .check.on {
    background: var(--chip-on-bg);
    color: var(--chip-on-fg);
    border-color: var(--chip-on-bg);
  }
  .share {
    width: 130px;
    height: 44px;
    text-align: right;
  }
</style>
