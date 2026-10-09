<script>
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t, tv } from '../lib/i18n.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { route, navigate } from '../lib/router.svelte.js'
  import { openAdd } from '../lib/store.svelte.js'

  const items = [
    { name: 'home', path: '/', icon: I.grid, label: 'nav.home' },
    { name: 'transactions', path: '/transactions', icon: I.list, label: 'nav.tx' },
    { name: 'add' },
    { name: 'accounts', path: '/accounts', icon: I.wallet, label: 'nav.accounts' },
    { name: 'groups', path: '/groups', icon: I.users, label: 'nav.groups' },
  ]
  const active = (it) => route.name === it.name || (it.name === 'accounts' && ['account', 'goals', 'recurring'].includes(route.name)) || (it.name === 'groups' && route.name === 'group')
</script>

<nav class="nav nav-{prefs.theme}" aria-label="Ana menü">
  {#each items as it}
    {#if it.name === 'add'}
      <button type="button" class="plus" data-tour="nav-add" aria-label={t('nav.add')} onclick={() => openAdd()}>
        <Icon d={I.plus} size={26} stroke={prefs.theme === 'c' ? 2.6 : 2.3} />
      </button>
    {:else}
      <a href={it.path} class="item" data-tour={'nav-' + it.name} aria-current={active(it) ? 'page' : undefined} aria-label={tv(it.label, prefs.theme)}
        onclick={(e) => { e.preventDefault(); navigate(it.path) }}>
        {#if prefs.theme !== 'b'}<Icon d={it.icon} size={22} stroke={prefs.theme === 'c' ? 2 : 1.8} />{/if}
        {#if prefs.theme !== 'a'}<span>{tv(it.label, prefs.theme)}</span>{/if}
      </a>
    {/if}
  {/each}
</nav>

<style>
  .nav {
    position: fixed;
    z-index: 20;
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    align-items: center;
  }
  .item {
    justify-self: center;
    display: flex;
    align-items: center;
    justify-content: center;
    text-decoration: none;
    transition: color 0.2s, background-color 0.2s;
  }
  .plus {
    justify-self: center;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.15s cubic-bezier(0.3, 1.5, 0.5, 1);
  }
  .plus:active {
    transform: scale(0.9);
  }

  /* A · Gece: yüzen cam hap, sadece ikon */
  .nav-a {
    left: 16px;
    right: 16px;
    bottom: calc(env(safe-area-inset-bottom, 0px) + 10px);
    height: 66px;
    border-radius: 33px;
    background: rgba(26, 26, 30, 0.86);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid #2a2a30;
    padding: 0 6px;
  }
  .nav-a .item {
    width: 48px;
    height: 48px;
    border-radius: 24px;
    color: #8a8a93;
  }
  .nav-a .item[aria-current='page'] {
    color: #f4f4f5;
    background: #2a2a30;
  }
  .nav-a .plus {
    width: 54px;
    height: 54px;
    border-radius: 27px;
    background: #c6f432;
    color: #0b0b0d;
    animation: pulse 2.6s ease-out 1.2s 2;
  }
  @keyframes pulse {
    0% { box-shadow: 0 0 0 0 rgba(198, 244, 50, 0.55); }
    70% { box-shadow: 0 0 0 14px rgba(198, 244, 50, 0); }
    100% { box-shadow: 0 0 0 0 rgba(198, 244, 50, 0); }
  }

  /* B · Defter: çift çizgili alt bant, sadece yazı */
  .nav-b {
    left: 0;
    right: 0;
    bottom: 0;
    padding: 8px 12px calc(env(safe-area-inset-bottom, 0px) + 8px);
    background: #f4efe6;
    border-top: 3px double #1c1a17;
  }
  .nav-b .item {
    min-height: 44px;
    font-size: 13px;
    font-weight: 500;
    color: #6b6358;
  }
  .nav-b .item[aria-current='page'] {
    font-weight: 600;
    color: #1c1a17;
    text-decoration: underline;
    text-decoration-color: #c2410c;
    text-decoration-thickness: 2px;
    text-underline-offset: 6px;
  }
  .nav-b .plus {
    width: 54px;
    height: 54px;
    border-radius: 16px;
    background: #c2410c;
    color: #fff;
    margin-top: -4px;
  }

  /* C · Cepler: beyaz yüzen bar, ikon + etiket */
  .nav-c {
    left: 16px;
    right: 16px;
    bottom: calc(env(safe-area-inset-bottom, 0px) + 8px);
    height: 68px;
    border-radius: 26px;
    background: #fff;
    box-shadow: 0 12px 32px rgba(30, 26, 18, 0.14);
    padding: 0 6px;
  }
  .nav-c .item {
    flex-direction: column;
    gap: 2px;
    min-height: 48px;
    font-size: 11px;
    font-weight: 600;
    color: #6e685d;
  }
  .nav-c .item[aria-current='page'] {
    color: #0f5c55;
    font-weight: 700;
  }
  .nav-c .plus {
    width: 56px;
    height: 56px;
    border-radius: 20px;
    background: #1e1a12;
    color: #f2b441;
  }
</style>
