<script>
  import Icon from '../components/Icon.svelte'
  import ThemePicker from '../components/ThemePicker.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, errorText, monthLabel } from '../lib/i18n.js'
  import { prefs, setLang } from '../lib/prefs.svelte.js'
  import { back, navigate } from '../lib/router.svelte.js'
  import { app, signedOut, toast, monthRange } from '../lib/store.svelte.js'
  import { net, clearOffline } from '../lib/offline.svelte.js'
  import { pushState, enablePush, disablePush, needsInstall } from '../lib/push.js'

  const standalone = window.matchMedia?.('(display-mode: standalone)').matches || navigator.standalone === true

  // iOS Kestirme: dekontları paylaş menüsünden gelen kutusuna gönderir.
  const inboxUrl = `${location.origin}/api/inbox`
  let token = $state(null)
  let hasToken = $state(!!app.user?.has_inbox_token)

  async function createToken() {
    if (hasToken && !confirm(t('sc.has'))) return
    try {
      token = (await api.post('/me/inbox-token')).token
      hasToken = true
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }
  async function revokeToken() {
    if (!confirm(t('common.confirm_delete'))) return
    await api.del('/me/inbox-token').catch(() => {})
    token = null
    hasToken = false
  }
  async function copy(v) {
    try {
      await navigator.clipboard.writeText(v)
      toast(t('sc.copied'))
    } catch {
      prompt(t('sc.copy'), v)
    }
  }

  // Şifre değiştirme
  let pw = $state({ current: '', next: '', again: '' })
  let pwBusy = $state(false)
  async function changePassword(e) {
    e.preventDefault()
    if (pw.next !== pw.again) return toast(t('pw.mismatch'), 'err')
    pwBusy = true
    try {
      await api.post('/me/password', { current_password: pw.current, new_password: pw.next })
      pw = { current: '', next: '', again: '' }
      toast(t('pw.done'))
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      pwBusy = false
    }
  }

  // Bildirimler
  let push = $state('off')
  let pushBusy = $state(false)
  pushState().then((s) => (push = s)).catch(() => {})
  async function togglePush() {
    pushBusy = true
    try {
      push = push === 'on' ? await disablePush() : await enablePush()
      if (push === 'denied') toast(t('push.denied'), 'err')
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      pushBusy = false
    }
  }

  // Hesap silme (veriler kalır)
  let delOpen = $state(false)
  let delPw = $state('')
  async function deleteAccount(e) {
    e.preventDefault()
    if (!confirm(t('del.sure'))) return
    try {
      await api.del('/me', { password: delPw })
      clearOffline()
      signedOut()
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }

  const exportUrl = $derived(`/api/export/transactions.csv?from=${monthRange().from}&to=${monthRange().to}`)

  function replayTour() {
    navigate('/')
    setTimeout(() => (app.tour = 'welcome'), 400)
  }

  async function logout() {
    // Gönderilmemiş değişiklikler çıkışta silinir; önce uyar.
    if (net.pending && !confirm(t('off.logout_warn', { n: net.pending }))) return
    try {
      await api.post('/auth/logout')
    } catch {
      /* çevrimdışı: yine de cihazdan çık */
    } finally {
      clearOffline()
      signedOut()
    }
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('set.title')}</h1>
  </header>
  <div class="rule"></div>

  <section class="card me">
    <span class="avatar">{app.user?.username?.charAt(0)}</span>
    <span class="grow">
      <span class="name">{app.user?.username}</span>
      <span class="muted small">{t('set.account')}</span>
    </span>
  </section>

  <section class="section">
    <h2 class="h-section">{t('set.theme')}</h2>
    <ThemePicker />
  </section>

  <section class="section">
    <h2 class="h-section">{t('set.lang')}</h2>
    <div class="seg" role="group" style="align-self: flex-start">
      <button type="button" aria-pressed={prefs.lang === 'tr'} onclick={() => setLang('tr')}>Türkçe</button>
      <button type="button" aria-pressed={prefs.lang === 'en'} onclick={() => setLang('en')}>English</button>
    </div>
  </section>

  <section class="section">
    <h2 class="h-section">{t('set.install')}</h2>
    <div class="card install">
      <Icon d={standalone ? I.check : I.share} size={22} />
      <p>{standalone ? t('set.installed') : t('set.install_hint')}</p>
    </div>
  </section>

  <section class="section" data-tour="push">
    <h2 class="h-section">{t('push.title')}</h2>
    <div class="card sc">
      <p>{t('push.what')}</p>
      {#if push === 'unsupported'}
        {#if needsInstall()}
          <p class="muted">{t('push.ios_hint')}</p>
          <ol class="steps">
            <li>{t('pp.install1')}</li>
            <li>{t('pp.install2')}</li>
            <li>{t('pp.install3')}</li>
            <li>{t('pp.install4')}</li>
          </ol>
        {:else}
          <p class="muted">{t('push.unsupported')}</p>
        {/if}
      {:else if push === 'denied'}
        <p class="warn">{t('push.denied')}</p>
        <ol class="steps">
          <li>{t('pp.denied1')}</li>
          <li>{t('pp.denied2')}</li>
          <li>{t('pp.denied3')}</li>
        </ol>
      {:else}
        {#if push === 'on'}<p class="muted"><Icon d={I.check} size={14} /> {t('push.enabled')}</p>{/if}
        <div class="row-btns">
          <button type="button" class="btn small" class:ghost={push === 'on'} disabled={pushBusy} onclick={togglePush}>
            <Icon d={I.bell} size={16} />{push === 'on' ? t('push.off') : t('push.on')}
          </button>
          {#if push === 'on'}<button type="button" class="btn small ghost" onclick={() => api.post('/push/test')}>{t('push.test')}</button>{/if}
        </div>
      {/if}
    </div>
  </section>

  <section class="section">
    <h2 class="h-section">{t('set.data')}</h2>
    <a class="btn ghost" href={exportUrl} download><Icon d={I.download} size={18} />{t('tx.export')} · {monthLabel(app.month)}</a>
  </section>

  <section class="section" data-tour="password">
    <h2 class="h-section">{t('set.security')}</h2>
    <form class="card sc" onsubmit={changePassword}>
      <strong>{t('pw.title')}</strong>
      <input class="input" type="password" autocomplete="current-password" placeholder={t('pw.current')} bind:value={pw.current} required />
      <input class="input" type="password" autocomplete="new-password" placeholder={t('pw.new')} bind:value={pw.next} minlength="8" maxlength="72" required />
      <input class="input" type="password" autocomplete="new-password" placeholder={t('pw.new2')} bind:value={pw.again} minlength="8" maxlength="72" required />
      <small class="muted">{t('pw.hint')}</small>
      <button class="btn small" disabled={pwBusy || pw.next.length < 8}><Icon d={I.lock} size={16} />{t('pw.title')}</button>
    </form>
  </section>

  <section class="section" id="shortcut">
    <h2 class="h-section">{t('sc.title')}</h2>
    <div class="card sc">
      <p>{t('sc.intro')}</p>
      <div class="kv">
        <span class="k">{t('sc.url')}</span>
        <code>{inboxUrl}</code>
        <button type="button" class="mini" aria-label={t('sc.copy')} onclick={() => copy(inboxUrl)}><Icon d={I.copy} size={15} /></button>
      </div>
      {#if token}
        <div class="kv">
          <span class="k">{t('sc.token')}</span>
          <code>{token}</code>
          <button type="button" class="mini" aria-label={t('sc.copy')} onclick={() => copy(token)}><Icon d={I.copy} size={15} /></button>
        </div>
        <p class="warn">{t('sc.token_once')}</p>
      {:else if hasToken}
        <p class="muted">{t('sc.has')}</p>
      {/if}
      <div class="row-btns">
        <button type="button" class="btn small" onclick={createToken}>{hasToken ? t('sc.rotate') : t('sc.create')}</button>
        {#if hasToken}<button type="button" class="btn small ghost" onclick={revokeToken}>{t('sc.revoke')}</button>{/if}
      </div>
      <ol class="steps">
        {#each [1, 2, 3, 4, 5] as n}<li>{t('sc.step' + n)}</li>{/each}
      </ol>
      <p class="muted">{t('sc.android')}</p>
      <p class="muted">{t('sc.https')}</p>
    </div>
  </section>

  <button type="button" class="btn ghost" onclick={replayTour}><Icon d={I.play} size={16} />{t('tour.replay')}</button>
  <button type="button" class="btn danger" onclick={logout}><Icon d={I.logout} size={18} />{t('auth.logout')}</button>

  <section class="section">
    {#if !delOpen}
      <button type="button" class="link-btn danger-text" onclick={() => (delOpen = true)}>{t('del.title')}</button>
    {:else}
      <form class="card sc" onsubmit={deleteAccount}>
        <strong>{t('del.title')}</strong>
        <p class="muted">{t('del.body')}</p>
        <input class="input" type="password" autocomplete="current-password" placeholder={t('del.confirm')} bind:value={delPw} required />
        <div class="row-btns">
          <button type="button" class="btn small ghost" onclick={() => (delOpen = false)}>{t('common.cancel')}</button>
          <button class="btn small danger" disabled={!delPw}>{t('del.button')}</button>
        </div>
      </form>
    {/if}
  </section>
  <p class="hint" style="text-align: center">{t('set.version', { v: '1.0.0' })}</p>
</div>

<style>
  .me {
    display: flex;
    align-items: center;
    gap: 14px;
  }
  .me .grow {
    display: flex;
    flex-direction: column;
  }
  .me .name {
    font-size: 18px;
    font-weight: 600;
  }
  .small {
    font-size: 13px;
  }
  .sc {
    display: flex;
    flex-direction: column;
    gap: 12px;
    font-size: 14px;
    line-height: 1.45;
  }
  .kv {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }
  .kv .k {
    flex: 0 0 auto;
    font-size: 12px;
    color: var(--muted);
    width: 56px;
  }
  .kv code {
    flex: 1;
    min-width: 0;
    overflow-wrap: anywhere;
    font-family: var(--font-num), ui-monospace, monospace;
    font-size: 13px;
    background: var(--surface2);
    padding: 6px 8px;
    border-radius: 8px;
  }
  .mini {
    width: 36px;
    height: 36px;
    flex: 0 0 36px;
    border-radius: 18px;
    border: 1px solid var(--line2);
    background: transparent;
    color: var(--fg);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .danger-text {
    color: var(--danger);
    align-self: center;
  }
  .warn {
    color: var(--danger);
    font-weight: 600;
  }
  .row-btns {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .steps {
    margin: 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .install {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    font-size: 14px;
    line-height: 1.45;
    color: var(--muted);
  }
  .install :global(svg) {
    flex: 0 0 auto;
    color: var(--fg);
  }
  .steps {
    margin: 0;
    padding-left: 22px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 14px;
  }
</style>
