<script>
  import Logo from '../components/Logo.svelte'
  import ThemePicker from '../components/ThemePicker.svelte'
  import { api } from '../lib/api.js'
  import { t, errorText } from '../lib/i18n.js'
  import { prefs, setLang } from '../lib/prefs.svelte.js'
  import { signedIn } from '../lib/store.svelte.js'
  import Icon from '../components/Icon.svelte'
  import { I } from '../lib/icons.js'
  import { passkeySupported, loginWithPasskey, cancelled } from '../lib/passkey.js'

  let mode = $state('login')
  let username = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(e) {
    e.preventDefault()
    error = ''
    if (mode === 'register') {
      if (!/^[a-zA-Z0-9_.-]{3,32}$/.test(username.trim())) return (error = t('err.username'))
      if (password.length < 8 || password.length > 72) return (error = t('err.password'))
    }
    busy = true
    try {
      const res = await api.post(mode === 'login' ? '/auth/login' : '/auth/register', { username: username.trim(), password })
      await signedIn(res.user)
    } catch (err) {
      error = errorText(err)
    } finally {
      busy = false
    }
  }

  // Face ID: Ayarlar'dan passkey eklemiş kullanıcı adı yazmadan girer.
  async function faceId() {
    error = ''
    busy = true
    try {
      const res = await loginWithPasskey()
      await signedIn(res.user)
    } catch (err) {
      if (!cancelled(err)) error = err?.status ? errorText(err) : t('pk.failed')
    } finally {
      busy = false
    }
  }
</script>

<div class="login">
  <div class="brand rise">
    <Logo size={84} />
    <h1>{t('app.name')}</h1>
    <p>{t('auth.tagline')}</p>
  </div>

  <form class="form rise" style="animation-delay: .15s" onsubmit={submit}>
    <div class="seg" role="group">
      <button type="button" aria-pressed={mode === 'login'} onclick={() => { mode = 'login'; error = '' }}>{t('auth.login')}</button>
      <button type="button" aria-pressed={mode === 'register'} onclick={() => { mode = 'register'; error = '' }}>{t('auth.register')}</button>
    </div>
    <label class="field">
      <span>{t('auth.username')}</span>
      <input class="input" bind:value={username} autocomplete="username" autocapitalize="none" autocorrect="off" spellcheck="false" required />
      {#if mode === 'register'}<small class="hint">{t('auth.username_hint')}</small>{/if}
    </label>
    <label class="field">
      <span>{t('auth.password')}</span>
      <input class="input" type="password" bind:value={password} autocomplete={mode === 'login' ? 'current-password' : 'new-password'} required />
      {#if mode === 'register'}<small class="hint">{t('auth.password_hint')}</small>{/if}
    </label>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <button class="btn" disabled={busy}>{mode === 'login' ? t('auth.login') : t('auth.register')}</button>
    {#if mode === 'login' && passkeySupported()}
      <button type="button" class="btn ghost" disabled={busy} onclick={faceId}><Icon d={I.faceid} size={18} />{t('pk.login')}</button>
    {/if}
  </form>

  <div class="prefs rise" style="animation-delay: .3s">
    <ThemePicker compact />
    <div class="seg" role="group">
      <button type="button" aria-pressed={prefs.lang === 'tr'} onclick={() => setLang('tr')}>TR</button>
      <button type="button" aria-pressed={prefs.lang === 'en'} onclick={() => setLang('en')}>EN</button>
    </div>
  </div>
</div>

<style>
  .login {
    min-height: 100%;
    padding: calc(env(safe-area-inset-top, 0px) + 48px) 24px calc(env(safe-area-inset-bottom, 0px) + 24px);
    display: flex;
    flex-direction: column;
    gap: 32px;
    max-width: 480px;
    margin: 0 auto;
  }
  .brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    text-align: center;
  }
  .brand h1 {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 48px;
    font-weight: 800;
    letter-spacing: -0.04em;
    line-height: 1;
  }
  .brand p {
    color: var(--muted);
    font-size: 15px;
  }
  .form .seg {
    align-self: center;
  }
  .prefs {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
  }
  .prefs :global(.themes) {
    width: 100%;
  }
</style>
