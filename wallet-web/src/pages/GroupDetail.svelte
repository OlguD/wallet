<script>
  import Icon from '../components/Icon.svelte'
  import TxRow from '../components/TxRow.svelte'
  import LedgerHead from '../components/LedgerHead.svelte'
  import GroupSheet from '../components/GroupSheet.svelte'
  import SettleSheet from '../components/SettleSheet.svelte'
  import RecurringList from '../components/RecurringList.svelte'
  import MonthPicker from '../components/MonthPicker.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, errorText, monthLabel, dayLabel } from '../lib/i18n.js'
  import { fmt, fmtc, symbol } from '../lib/money.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { route, back } from '../lib/router.svelte.js'
  import { app, monthRange, openAdd, toast } from '../lib/store.svelte.js'

  let tab = $state('tx')
  let group = $state(null)
  let txs = $state([])
  let balances = $state({ balances: [], suggestions: [] })
  let settlements = $state([])
  let summary = $state(null)
  let rules = $state([])
  let sheet = $state(null) // 'edit' | 'member' | 'pay' | { settle } | 'month'
  let notFound = $state(false)

  const me = $derived(app.user?.id)

  function load() {
    const id = route.id
    const range = monthRange(app.month)
    api.get(`/groups/${id}`).then((g) => (group = g)).catch(() => (notFound = true))
    api.get(`/groups/${id}/transactions`, { limit: 100 }).then((r) => (txs = r)).catch(() => {})
    api.get(`/groups/${id}/balances`).then((r) => (balances = r)).catch(() => {})
    api.get(`/groups/${id}/settlements`).then((r) => (settlements = r)).catch(() => {})
    api.get(`/groups/${id}/summary`, range).then((r) => (summary = r)).catch(() => {})
    api.get(`/groups/${id}/recurring`).then((r) => (rules = r)).catch(() => {})
  }
  $effect(() => {
    app.version
    app.month
    route.id
    load()
  })

  const myNets = $derived(balances.balances.filter((b) => b.user_id === me && b.net !== 0))
  const name = (id, username) => (id === me ? t('common.you') : username)

  async function cancelInvite(inv) {
    if (!confirm(t('inv.cancel') + '?')) return
    try {
      group = await api.del(`/groups/${route.id}/invites/${inv.id}`)
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }

  async function removeSettlement(s) {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/groups/${route.id}/settlements/${s.id}`)
      load()
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/groups')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{group?.name ?? ''}</h1>
    {#if group}<button type="button" class="icon-btn" aria-label={t('grp.rename')} onclick={() => (sheet = 'edit')}><Icon d={I.settings} size={19} /></button>{/if}
  </header>
  <div class="rule"></div>

  {#if notFound}
    <p class="empty">{t('err.not_found')}</p>
  {:else if group}
    <section class="members">
      {#each group.members as m}
        <span class="member" class:gone={m.deleted} title={m.deleted ? t('grp.deleted_member', { name: m.username }) : m.username}>
          <span class="avatar sm">{m.username.charAt(0)}</span>
          <span class="mn">{m.user_id === me ? t('common.you') : m.username}</span>
        </span>
      {/each}
      {#each group.invites || [] as inv (inv.id)}
        <button type="button" class="member pending" title={t('inv.cancel')} onclick={() => cancelInvite(inv)}>
          <span class="avatar sm">{inv.invitee_name.charAt(0)}</span>
          <span class="mn">{inv.invitee_name}</span>
          <span class="tag">{t('inv.pending')}</span>
        </button>
      {/each}
      <button type="button" class="member add" data-tour="invite" onclick={() => (sheet = 'member')}>
        <span class="avatar sm"><Icon d={I.userPlus} size={18} /></span>
        <span class="mn">{t('common.add')}</span>
      </button>
    </section>

    <div class="status card">
      {#each myNets as n}
        <span class:pos={n.net > 0}>{n.net > 0 ? t('grp.you_get', { v: fmtc(n.net, n.currency) }) : t('grp.you_owe', { v: fmtc(-n.net, n.currency) })}</span>
      {:else}
        <span>{t('grp.settled')}</span>
      {/each}
      <span class="acts">
        {#if group.members.some((m) => m.user_id !== me && !m.deleted)}
          <button type="button" class="btn small ghost" data-tour="group-pay" onclick={() => (sheet = 'pay')}><Icon d={I.swap} size={16} />{t('pay.button')}</button>
        {/if}
        <button type="button" class="btn small" onclick={() => openAdd({ groupId: group.id })}><Icon d={I.plus} size={16} stroke={2.4} />{t('common.expense')}</button>
      </span>
    </div>

    <div class="tabs" role="tablist">
      {#each ['tx', 'debts', 'summary'] as k}
        <button type="button" role="tab" aria-selected={tab === k} onclick={() => (tab = k)}>{t('grp.tab.' + k)}</button>
      {/each}
    </div>

    {#if tab === 'tx'}
      {#if prefs.theme === 'b' && txs.length}<LedgerHead last="grp.share" />{/if}
      <div class={prefs.theme === 'b' ? '' : 'list'}>
        {#each txs as tx (tx.id)}<TxRow {tx} context="group" />{/each}
      </div>
      {#if !txs.length}<p class="empty">{t('common.empty')}</p>{/if}
    {:else if tab === 'debts'}
      <section class="section">
        <h2 class="h-section">{t('grp.net')}</h2>
        <div class="list">
          {#each balances.balances as b}
            <div class="row">
              <span class="avatar sm">{b.username.charAt(0)}</span>
              <span class="grow"><span class="title">{b.user_id === me ? t('grp.member_you', { name: b.username }) : b.username}</span></span>
              <span class="amount" class:pos={b.net > 0}>{fmt(b.net, 'always')} {symbol(b.currency)}</span>
            </div>
          {/each}
        </div>
      </section>
      <section class="section">
        <h2 class="h-section">{t('grp.tab.debts')}</h2>
        {#each balances.suggestions as s}
          <div class="card debt">
            <span class="grow">
              <span class="title">{t('grp.owes', { from: name(s.from_user_id, s.from_username), to: name(s.to_user_id, s.to_username) })}</span>
              <span class="amount">{fmtc(s.amount, s.currency)}</span>
            </span>
            {#if s.from_user_id === me || s.to_user_id === me}
              <button type="button" class="btn small" onclick={() => (sheet = { settle: s })}><Icon d={I.check} size={16} stroke={2.4} />{t('grp.mark_paid')}</button>
            {/if}
          </div>
        {:else}
          <p class="empty">{t('grp.no_debts')}</p>
        {/each}
      </section>
      {#if settlements.length}
        <section class="section">
          <h2 class="h-section">{t('grp.settlements')}</h2>
          <div class="list">
            {#each settlements as s (s.id)}
              <div class="row">
                <span class="ico"><Icon d={I.swap} size={18} /></span>
                <span class="grow">
                  <span class="title">{name(s.from_user_id, s.from_username)} → {name(s.to_user_id, s.to_username)}</span>
                  <span class="sub">{dayLabel(s.occurred_at)}</span>
                </span>
                <span class="amount">{fmtc(s.amount, s.currency)}</span>
                {#if [s.from_user_id, s.to_user_id, s.created_by].includes(me)}
                  <button type="button" class="mini" aria-label={t('common.delete')} onclick={() => removeSettlement(s)}><Icon d={I.trash} size={15} /></button>
                {/if}
              </div>
            {/each}
          </div>
        </section>
      {/if}
    {:else}
      <button type="button" class="chip small" style="align-self: flex-start" onclick={() => (sheet = 'month')}>
        <Icon d={I.calendar} size={15} />{monthLabel(app.month)}<Icon d={I.down} size={13} />
      </button>
      {#each summary?.currencies ?? [] as c}
        <section class="section">
          <div class="card total">
            <span class="muted">{t('grp.total_spent')}</span>
            <span class="big">{fmtc(c.total_expense, c.currency)}</span>
          </div>
          <div class="list">
            {#each c.members as m}
              <div class="row">
                <span class="avatar sm">{m.username.charAt(0)}</span>
                <span class="grow">
                  <span class="title">{m.user_id === me ? t('grp.member_you', { name: m.username }) : m.username}</span>
                  <span class="sub">{t('grp.paid')} {fmt(m.paid)} · {t('grp.share')} {fmt(m.share)}</span>
                </span>
                <span class="amount" class:pos={m.paid - m.share > 0}>{fmt(m.paid - m.share, 'always')}</span>
              </div>
            {/each}
          </div>
        </section>
      {:else}
        <p class="empty">{t('common.empty')}</p>
      {/each}
      {#if rules.length}
        <section class="section">
          <h2 class="h-section">{t('grp.recurring')}</h2>
          <RecurringList {rules} editable={false} />
        </section>
      {/if}
    {/if}
  {/if}
</div>

{#if sheet === 'edit'}
  <GroupSheet mode="edit" {group} onclose={() => (sheet = null)} onsaved={load} />
{:else if sheet === 'member'}
  <GroupSheet mode="member" {group} onclose={() => (sheet = null)} onsaved={load} />
{:else if sheet === 'month'}
  <MonthPicker onclose={() => (sheet = null)} />
{:else if sheet?.settle}
  <SettleSheet groupId={route.id} s={sheet.settle} onclose={() => (sheet = null)} onsaved={load} />
{:else if sheet === 'pay' && group}
  <SettleSheet groupId={route.id} members={group.members} onclose={() => (sheet = null)} onsaved={load} />
{/if}

<style>
  .members {
    display: flex;
    gap: 14px;
    overflow-x: auto;
    padding-bottom: 2px;
  }
  .member {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    flex: 0 0 auto;
    background: none;
    border: none;
    padding: 0;
    color: var(--fg);
  }
  .member.add {
    color: var(--muted);
  }
  .avatar.sm {
    width: 40px;
    height: 40px;
    flex-basis: 40px;
    font-size: 15px;
  }
  .mn {
    font-size: 12px;
    max-width: 64px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .status {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    font-size: 16px;
    font-weight: 600;
  }
  .status > span {
    flex: 1;
  }
  .status > .acts {
    flex: 0 0 auto;
    display: flex;
    gap: 8px;
  }
  .debt {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .debt .grow {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }
  .debt .title {
    font-weight: 600;
  }
  .debt .amount {
    font-family: var(--font-num);
    font-size: 18px;
  }
  .total {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .total .big {
    font-family: var(--font-display);
    font-weight: var(--display-weight);
    letter-spacing: var(--display-track);
    font-size: 34px;
  }
  .amount {
    font-family: var(--font-num);
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
  .member.gone {
    opacity: 0.45;
  }
  .member.pending {
    opacity: 0.7;
    border: none;
    background: none;
    padding: 0;
    color: inherit;
  }
  .member .tag {
    font-size: 10px;
    color: var(--muted);
  }
</style>
