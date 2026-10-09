<script>
  // Gelen kutusu: grup davetleri, Kestirme/paylaş ile gelen dekontlar ve bildirimler.
  import Icon from '../components/Icon.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { parseStored } from '../lib/receipt.js'
  import { t, errorText, dayLabel, timeLabel } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { back, navigate } from '../lib/router.svelte.js'
  import { app, openAdd, loadInbox, refresh, toast } from '../lib/store.svelte.js'

  let busy = $state(null)

  $effect(() => {
    loadInbox()
  })

  // Sayfadan çıkarken görülen bildirimler okundu sayılır.
  $effect(() => () => {
    if (app.notifications.unread) api.post('/notifications/read', { ids: [] }).then(loadInbox).catch(() => {})
  })

  async function respond(inv, accept) {
    busy = 'i' + inv.id
    try {
      await api.post(`/invites/${inv.id}/${accept ? 'accept' : 'decline'}`)
      if (accept) {
        toast(t('inv.joined', { g: inv.group_name }))
        await refresh()
        navigate(`/groups/${inv.group_id}`)
      } else await loadInbox()
    } catch (e) {
      toast(errorText(e), 'err')
    } finally {
      busy = null
    }
  }

  async function processReceipt(r) {
    busy = 'r' + r.id
    try {
      openAdd({ receipt: await parseStored(r) })
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
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }

  function openNotification(n) {
    if (n.url && n.url !== '/inbox') navigate(n.url)
  }

  const iconFor = (kind) =>
    kind.startsWith('budget') ? I.target : kind === 'recurring' ? I.repeat : kind.startsWith('invite') ? I.userPlus : kind === 'group_expense' ? I.users : I.bell
  const empty = $derived(!app.invites.length && !app.inbox.length && !app.notifications.items.length)
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('inbox.title')}</h1>
  </header>
  <div class="rule"></div>

  {#if app.invites.length}
    <section class="section">
      <h2 class="h-section">{t('inbox.invites')}</h2>
      {#each app.invites as inv (inv.id)}
        <div class="card invite rise">
          <span class="ico"><Icon d={I.users} size={18} /></span>
          <span class="grow">
            <span class="title">{inv.group_name}</span>
            <span class="sub">{t('inv.from', { name: inv.inviter_name })} · {dayLabel(inv.created_at)}</span>
          </span>
          <span class="acts">
            <button type="button" class="btn small ghost" disabled={busy !== null} onclick={() => respond(inv, false)}>{t('inv.decline')}</button>
            <button type="button" class="btn small" disabled={busy !== null} onclick={() => respond(inv, true)}>{t('inv.accept')}</button>
          </span>
        </div>
      {/each}
    </section>
  {/if}

  {#if app.inbox.length}
    <section class="section">
      <h2 class="h-section">{t('inbox.receipts')}</h2>
      <p class="hint">{t('rc.inbox_hint')}</p>
      <div class="list">
        {#each app.inbox as r (r.id)}
          <div class="row">
            <span class="ico"><Icon d={I.receipt} size={18} /></span>
            <button type="button" class="pick" disabled={busy !== null} onclick={() => processReceipt(r)}>
              <span class="title">{r.parsed?.counterparty_name || r.original_name || t('rc.title')}</span>
              <span class="sub">
                {#if busy === 'r' + r.id}{t('rc.reading')}
                {:else}{dayLabel(r.created_at)} {timeLabel(r.created_at)}{r.parsed?.amount ? ' · ' + fmtc(r.parsed.amount, r.parsed.currency) : ''}{/if}
              </span>
            </button>
            <button type="button" class="mini" aria-label={t('rc.dismiss')} onclick={() => dismiss(r)}><Icon d={I.x} size={15} stroke={2.2} /></button>
          </div>
        {/each}
      </div>
    </section>
  {/if}

  {#if app.notifications.items.length}
    <section class="section">
      <h2 class="h-section">{t('inbox.notifications')}</h2>
      <div class="list">
        {#each app.notifications.items as n (n.id)}
          <button type="button" class="row" class:unread={!n.read_at} onclick={() => openNotification(n)}>
            <span class="ico"><Icon d={iconFor(n.kind)} size={18} /></span>
            <span class="grow">
              <span class="title">{n.title}</span>
              {#if n.body}<span class="sub wrap">{n.body}</span>{/if}
              <span class="sub">{dayLabel(n.created_at)} {timeLabel(n.created_at)}</span>
            </span>
            {#if !n.read_at}<span class="dot" aria-hidden="true"></span>{/if}
          </button>
        {/each}
      </div>
    </section>
  {/if}

  {#if empty}
    <div class="empty">
      <Icon d={I.inbox} size={40} stroke={1.4} />
      <p>{t('inbox.empty')}</p>
    </div>
  {/if}
</div>

<style>
  .invite {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }
  .invite .grow {
    flex: 1;
    min-width: 140px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .invite .title {
    font-weight: 600;
  }
  .invite .sub {
    font-size: 13px;
    color: var(--muted);
  }
  .acts {
    display: flex;
    gap: 8px;
    margin-left: auto;
  }
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
  .wrap {
    white-space: normal !important;
  }
  .unread .title {
    font-weight: 700;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: var(--accent);
    flex: 0 0 9px;
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
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
