<script>
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { catInfo } from '../lib/categories.js'
  import { t, errorText, fullDate } from '../lib/i18n.js'
  import { fmt, symbol, MINUS } from '../lib/money.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { app, toast, refresh } from '../lib/store.svelte.js'

  // editable: kendi kurallarım (duraklat/sil); grup görünümünde sadece okunur.
  let { rules, editable = true, onchange, onselect } = $props()

  async function toggle(r) {
    try {
      await api.patch(`/recurring/${r.id}`, { active: !r.active })
      onchange?.()
      refresh()
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }
  async function remove(r) {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/recurring/${r.id}`)
      onchange?.()
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }
</script>

<div class="list">
  {#each rules as r (r.id)}
    {@const c = catInfo(r)}
    <div class="row" class:paused={!r.active}>
      {#snippet info()}
        <span class="ico" style={prefs.theme === 'c' ? `background: ${c.tint}` : ''}><Icon d={c.icon} size={18} /></span>
        <span class="grow">
          <span class="title">{r.description || t('cat.' + c.id)}</span>
          <span class="sub">
            {t('repeat.' + r.frequency)} · {r.active ? t('rec.next', { d: fullDate(r.next_run_on + 'T12:00:00') }) : t('rec.paused')}
            {#if !editable || r.user_id !== app.user?.id} · {r.username}{/if}
          </span>
        </span>
      {/snippet}
      {#if onselect}
        <button type="button" class="pick" onclick={() => onselect(r)}>{@render info()}</button>
      {:else}
        <span class="pick">{@render info()}</span>
      {/if}
      <span class="end">
        <span class="amount">{r.type === 'expense' ? MINUS : '+'}{fmt(r.amount)} {symbol(r.currency)}</span>
        {#if editable && r.user_id === app.user?.id}
          <span class="acts">
            <button type="button" class="mini" aria-label={r.active ? t('rec.pause') : t('rec.resume')} onclick={() => toggle(r)}>
              <Icon d={r.active ? I.pause : I.play} size={15} stroke={2.2} />
            </button>
            <button type="button" class="mini" aria-label={t('common.delete')} onclick={() => remove(r)}><Icon d={I.trash} size={15} /></button>
          </span>
        {/if}
      </span>
    </div>
  {/each}
</div>

<style>
  .pick {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 14px;
    background: none;
    border: none;
    padding: 0;
    color: inherit;
    text-align: left;
  }
  .paused {
    opacity: 0.55;
  }
  .acts {
    display: flex;
    gap: 4px;
  }
  .mini {
    width: 32px;
    height: 32px;
    border-radius: 16px;
    border: 1px solid var(--line2);
    background: transparent;
    color: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
