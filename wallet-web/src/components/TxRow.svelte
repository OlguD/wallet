<script>
  import Icon from './Icon.svelte'
  import { catInfo } from '../lib/categories.js'
  import { fmt, MINUS } from '../lib/money.js'
  import { t, dayLabel, timeLabel, shortDate } from '../lib/i18n.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { app, openEdit } from '../lib/store.svelte.js'

  // context: 'personal' (kendi hesap geçmişim) | 'group' (grup harcamaları)
  let { tx, context = 'personal', delay = 0, showTime = true } = $props()

  const cat = $derived(catInfo(tx))
  const title = $derived(tx.description || tx.counterparty_name || t('cat.' + cat.id))
  const groupName = $derived(tx.group_id ? app.groups.find((g) => g.id === tx.group_id)?.name : null)
  const mine = $derived(tx.user_id === app.user?.id)
  const myShare = $derived(tx.splits?.find((s) => s.user_id === app.user?.id)?.amount)
  const amount = $derived((tx.type === 'expense' ? MINUS : '+') + fmt(tx.amount, 'never'))

  const sub = $derived.by(() => {
    if (context === 'group') {
      const who = mine ? t('common.you') : tx.username
      const parts = [dayLabel(tx.occurred_at), t('tx.paid_by', { name: who })]
      return parts.join(' · ')
    }
    const when = showTime ? `${dayLabel(tx.occurred_at)} ${timeLabel(tx.occurred_at)}` : dayLabel(tx.occurred_at)
    return [when, tx.account_name, groupName].filter(Boolean).join(' · ')
  })
  const after = $derived(
    context === 'group'
      ? myShare !== undefined ? t('tx.your_share', { v: fmt(myShare) }) : ''
      : tx.balance_after !== undefined && tx.balance_after !== null ? fmt(tx.balance_after) : '',
  )
</script>

{#if prefs.theme === 'b'}
  <button type="button" class="ledger rise" style="animation-delay: {delay}s" onclick={() => openEdit(tx)}>
    <span class="d">{shortDate(tx.occurred_at)}</span>
    <span class="desc">
      <span class="t">{title}</span>
      <span class="s">{context === 'group' ? t('tx.paid_by', { name: mine ? t('common.you') : tx.username }) : [tx.account_name, groupName].filter(Boolean).join(' · ')}</span>
    </span>
    <span class="amt" class:inc={tx.type === 'income'}>{amount}</span>
    <span class="aft">{context === 'group' ? (myShare !== undefined ? fmt(myShare) : '') : after}</span>
  </button>
{:else}
  <button type="button" class="row rise" style="animation-delay: {delay}s" onclick={() => openEdit(tx)}>
    <span class="ico" style={prefs.theme === 'c' ? `background: ${cat.tint}` : ''}>
      <Icon d={cat.icon} size={prefs.theme === 'c' ? 20 : 18} stroke={prefs.theme === 'c' ? 2 : 1.8} />
    </span>
    <span class="grow">
      <span class="title">{title}</span>
      <span class="sub">{sub}</span>
    </span>
    <span class="end">
      <span class="amount" class:pos={tx.type === 'income'}>{amount}</span>
      {#if after}<span class="after">{after}</span>{/if}
    </span>
  </button>
{/if}

<style>
  .ledger {
    display: grid;
    grid-template-columns: 46px minmax(0, 1fr) 84px 80px;
    gap: 6px;
    align-items: baseline;
    padding: 12px 0;
    border: none;
    border-bottom: 1px solid var(--line);
    background: none;
    text-align: left;
    color: var(--fg);
    width: 100%;
  }
  .d {
    font-size: 13px;
    color: var(--muted);
  }
  .desc {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .t {
    font-size: 15px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .s {
    font-size: 12px;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .amt,
  .aft {
    text-align: right;
    font-family: 'Newsreader', serif;
    font-size: 16px;
  }
  .amt {
    font-weight: 500;
  }
  .amt.inc {
    color: var(--income);
  }
  .aft {
    color: var(--muted);
  }
</style>
