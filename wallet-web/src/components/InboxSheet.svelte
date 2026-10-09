<script>
  // Kestirme / paylaş menüsüyle gelen, henüz işleme çevrilmemiş dekontlar.
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { parseStored } from '../lib/receipt.js'
  import { t, errorText, dayLabel, timeLabel } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { app, openAdd, loadInbox, toast } from '../lib/store.svelte.js'

  let { onclose } = $props()
  let busy = $state(null) // işlenen dekont id

  async function process(r) {
    busy = r.id
    try {
      const parsed = await parseStored(r)
      onclose()
      openAdd({ receipt: parsed })
    } catch (e) {
      toast(errorText(e), 'err')
    } finally {
      busy = null
    }
  }

  async function dismiss(r) {
    try {
      await api.patch(`/receipts/${r.id}`, { status: 'dismissed' })
      await loadInbox()
      if (!app.inbox.length) onclose()
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }

  const label = (r) => r.parsed?.counterparty_name || r.original_name || t('rc.title')
</script>

<Sheet {onclose} top={110} label={t('rc.inbox')}>
  <h2 class="h-section" style="padding-top: 4px">{t('rc.inbox')}</h2>
  <p class="hint">{t('rc.inbox_hint')}</p>
  <div class="list">
    {#each app.inbox as r (r.id)}
      <div class="row">
        <span class="ico"><Icon d={I.receipt} size={18} /></span>
        <button type="button" class="grow pick" disabled={busy !== null} onclick={() => process(r)}>
          <span class="title">{label(r)}</span>
          <span class="sub">
            {#if busy === r.id}{t('rc.reading')}
            {:else}{dayLabel(r.created_at)} {timeLabel(r.created_at)}{r.parsed?.amount ? ' · ' + fmtc(r.parsed.amount, r.parsed.currency) : ''}{/if}
          </span>
        </button>
        <button type="button" class="mini" aria-label={t('rc.dismiss')} onclick={() => dismiss(r)}><Icon d={I.x} size={15} stroke={2.2} /></button>
      </div>
    {/each}
  </div>
</Sheet>

<style>
  .pick {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    background: none;
    border: none;
    padding: 0;
    color: inherit;
    text-align: left;
  }
  .mini {
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    border-radius: 16px;
    border: 1px solid var(--line2);
    background: transparent;
    color: var(--muted);
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
