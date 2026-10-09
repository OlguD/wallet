<script>
  import Icon from '../components/Icon.svelte'
  import GroupSheet from '../components/GroupSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t } from '../lib/i18n.js'
  import { fmtc } from '../lib/money.js'
  import { navigate } from '../lib/router.svelte.js'
  import { app } from '../lib/store.svelte.js'

  let creating = $state(false)
  let info = $state({}) // id -> { members, mine: [{currency, net}] }

  $effect(() => {
    app.version
    for (const g of app.groups) {
      Promise.all([api.get(`/groups/${g.id}`), api.get(`/groups/${g.id}/balances`)]).then(([d, b]) => {
        info[g.id] = { members: d.members, mine: b.balances.filter((x) => x.user_id === app.user?.id && x.net !== 0) }
      })
    }
  })
</script>

<div class="page">
  <header class="topbar">
    <h1 class="title">{t('grp.title')}</h1>
    <button type="button" class="icon-btn" aria-label={t('grp.new')} onclick={() => (creating = true)}><Icon d={I.plus} size={20} stroke={2.2} /></button>
  </header>
  <div class="rule"></div>

  <div class="list">
    {#each app.groups as g, i (g.id)}
      {@const inf = info[g.id]}
      <button type="button" class="row rise" style="animation-delay: {i * 0.05}s" onclick={() => navigate(`/groups/${g.id}`)}>
        <span class="ico"><Icon d={I.users} size={18} /></span>
        <span class="grow">
          <span class="title">{g.name}</span>
          <span class="sub">{inf ? inf.members.map((m) => m.username).join(', ') : ''}</span>
        </span>
        <span class="end">
          {#if inf}
            {#each inf.mine as n}
              <span class="amount" class:pos={n.net > 0}>{n.net > 0 ? t('grp.you_get', { v: fmtc(n.net, n.currency) }) : t('grp.you_owe', { v: fmtc(-n.net, n.currency) })}</span>
            {:else}
              <span class="after">{t('grp.settled')}</span>
            {/each}
          {/if}
        </span>
      </button>
    {/each}
  </div>
  {#if !app.groups.length}
    <p class="empty">{t('grp.empty')}</p>
    <button type="button" class="btn" onclick={() => (creating = true)}>{t('grp.new')}</button>
  {/if}
</div>

{#if creating}<GroupSheet mode="create" onclose={() => (creating = false)} />{/if}
