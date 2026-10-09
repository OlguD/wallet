<script>
  import { onMount } from 'svelte'
  import Splash from './components/Splash.svelte'
  import Nav from './components/Nav.svelte'
  import Toast from './components/Toast.svelte'
  import AddSheet from './components/AddSheet.svelte'
  import Tour from './components/Tour.svelte'
  import OfflineBar from './components/OfflineBar.svelte'
  import { net, flush } from './lib/offline.svelte.js'
  import TransferSheet from './components/TransferSheet.svelte'
  import Inbox from './pages/Inbox.svelte'
  import Budgets from './pages/Budgets.svelte'
  import { pendingTour } from './lib/tours.js'
  import Login from './pages/Login.svelte'
  import Home from './pages/Home.svelte'
  import Transactions from './pages/Transactions.svelte'
  import Accounts from './pages/Accounts.svelte'
  import AccountDetail from './pages/AccountDetail.svelte'
  import Groups from './pages/Groups.svelte'
  import GroupDetail from './pages/GroupDetail.svelte'
  import Settings from './pages/Settings.svelte'
  import Goals from './pages/Goals.svelte'
  import Recurring from './pages/Recurring.svelte'
  import Rates from './pages/Rates.svelte'
  import Payees from './pages/Payees.svelte'
  import { setUnauthorizedHandler } from './lib/api.js'
  import { route } from './lib/router.svelte.js'
  import { app, loadSession, signedOut, refresh, openAdd, toast } from './lib/store.svelte.js'
  import { takeSharedFile, uploadReceipt } from './lib/receipt.js'
  import { t, errorText } from './lib/i18n.js'

  let splash = $state(true)

  onMount(() => {
    setUnauthorizedHandler(signedOut)
    const minSplash = new Promise((r) => setTimeout(r, 1500))
    Promise.all([loadSession(), minSplash]).then(() => (splash = false))

    // Uygulamaya geri dönülünce (ana ekrandan) kuyruğu gönder ve verileri tazele.
    const onVisible = () => {
      if (document.visibilityState !== 'visible') return
      flush()
      refresh().catch(() => {})
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  })

  // Android: paylaş → Cüzdan ile gelen dekontu oku ve ekleme ekranını aç.
  async function handleShared() {
    const file = await takeSharedFile().catch(() => null)
    if (!file) return
    toast(t('rc.reading'))
    try {
      const r = await uploadReceipt(file, { source: 'share' })
      openAdd({ receipt: r })
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }

  $effect(() => {
    if (app.user && !splash) {
      handleShared()
      flush()
    }
  })

  // Kuyruk bildirimleri: kuyruğa alındı / gönderildi / reddedildi.
  let lastTick = net.queuedTick
  $effect(() => {
    if (net.queuedTick !== lastTick) {
      lastTick = net.queuedTick
      toast(t('off.queued'))
    }
  })
  $effect(() => {
    if (!net.failed.length) return
    const f = net.failed[0]
    toast(t('off.failed', { e: f.error || f.label }), 'err')
    net.failed = net.failed.slice(1)
  })

  // Görülmemiş tanıtım turu varsa ana sayfa açıkken (panel yokken) bir kez göster.
  $effect(() => {
    if (!app.user || splash || app.tour || app.sheet || route.name !== 'home') return
    const id = pendingTour(app.user.seen_tours || [])
    if (!id) return
    const timer = setTimeout(() => {
      if (!app.sheet && route.name === 'home') app.tour = id
    }, 900)
    return () => clearTimeout(timer)
  })

  const pages = { home: Home, transactions: Transactions, accounts: Accounts, account: AccountDetail, groups: Groups, group: GroupDetail, settings: Settings, goals: Goals, recurring: Recurring, rates: Rates, payees: Payees, inbox: Inbox, budgets: Budgets }
  const Page = $derived(pages[route.name] || Home)
</script>

{#if app.ready && !splash}
  {#if app.user}
    {#key route.path}
      <main class="scroller"><Page /></main>
    {/key}
    {#if route.name !== 'settings'}<Nav />{/if}
    {#if app.sheet?.kind === 'transfer'}
      {#key app.sheet}<TransferSheet sheet={app.sheet} />{/key}
    {:else if app.sheet}
      {#key app.sheet}<AddSheet sheet={app.sheet} />{/key}
    {/if}
  {:else}
    <main class="scroller"><Login /></main>
  {/if}
{/if}
{#if app.user}<OfflineBar />{/if}
{#if app.tour && app.user}
  {#key app.tour}<Tour id={app.tour} />{/key}
{/if}
{#if splash}<Splash />{/if}
<Toast />
