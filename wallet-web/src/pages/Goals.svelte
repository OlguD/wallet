<script>
  import Icon from '../components/Icon.svelte'
  import GoalCard from '../components/GoalCard.svelte'
  import GoalSheet from '../components/GoalSheet.svelte'
  import GoalView from '../components/GoalView.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t } from '../lib/i18n.js'
  import { back } from '../lib/router.svelte.js'
  import { app } from '../lib/store.svelte.js'

  let goals = $state([])
  let creating = $state(false)
  let viewing = $state(null)

  const load = () => api.get('/goals').then((g) => (goals = g)).catch(() => {})
  $effect(() => {
    app.version
    load()
  })
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('goal.title')}</h1>
    {#if app.accounts.length}
      <button type="button" class="icon-btn" aria-label={t('goal.new')} onclick={() => (creating = true)}><Icon d={I.plus} size={20} stroke={2.2} /></button>
    {/if}
  </header>
  <div class="rule"></div>

  {#each goals as g, i (g.id)}
    <GoalCard goal={g} delay={i * 0.05} onclick={() => (viewing = g)} />
  {/each}

  {#if !goals.length}
    <div class="empty">
      <Icon d={I.target} size={40} stroke={1.4} />
      <p>{app.accounts.length ? t('goal.empty') : t('home.no_accounts')}</p>
    </div>
  {/if}
  {#if app.accounts.length}
    <button type="button" class="btn ghost" onclick={() => (creating = true)}><Icon d={I.plus} size={18} />{t('goal.new')}</button>
  {/if}
</div>

{#if creating}
  <GoalSheet onclose={() => (creating = false)} onsaved={(g) => { load(); viewing = g }} />
{/if}
{#if viewing}
  <GoalView goal={viewing} onclose={() => (viewing = null)} onchange={load} />
{/if}

<style>
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }
</style>
