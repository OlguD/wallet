<script>
  // Ana sayfada bildirimleri açmaya yönlendiren kart. Duruma göre adım adım
  // anlatır: iPhone Safari'deyse önce ana ekrana ekleme, izin reddedildiyse
  // telefon ayarları, değilse tek dokunuşla açma. "Şimdi değil" bu cihazda
  // 7 gün gizler.
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t, errorText } from '../lib/i18n.js'
  import { pushState, enablePush, needsInstall } from '../lib/push.js'
  import { api } from '../lib/api.js'
  import { toast } from '../lib/store.svelte.js'

  const KEY = 'wallet.pushPromptHiddenUntil'
  let state = $state(null) // install | off | denied | null (gösterme)
  let busy = $state(false)

  function hidden() {
    try {
      return Number(localStorage.getItem(KEY) || 0) > Date.now()
    } catch {
      return false
    }
  }

  $effect(() => {
    if (hidden()) return
    if (needsInstall()) {
      state = 'install'
      return
    }
    pushState()
      .then((s) => (state = s === 'off' || s === 'denied' ? s : null))
      .catch(() => {})
  })

  function later() {
    try {
      localStorage.setItem(KEY, String(Date.now() + 7 * 864e5))
    } catch {}
    state = null
  }

  async function enable() {
    busy = true
    try {
      const s = await enablePush()
      if (s === 'on') {
        toast(t('push.enabled'))
        api.post('/push/test').catch(() => {})
        state = null
      } else state = 'denied'
    } catch (e) {
      toast(errorText(e), 'err')
    } finally {
      busy = false
    }
  }
</script>

{#if state}
  <section class="card prompt rise" data-tour="push-prompt">
    <div class="head">
      <span class="ico"><Icon d={I.bell} size={18} /></span>
      <span class="grow">
        <span class="title">{t('pp.title')}</span>
        <span class="sub">{t('push.what')}</span>
      </span>
    </div>
    {#if state === 'install'}
      <ol class="steps">
        <li>{t('pp.install1')}</li>
        <li>{t('pp.install2')}</li>
        <li>{t('pp.install3')}</li>
        <li>{t('pp.install4')}</li>
      </ol>
    {:else if state === 'denied'}
      <ol class="steps">
        <li>{t('pp.denied1')}</li>
        <li>{t('pp.denied2')}</li>
        <li>{t('pp.denied3')}</li>
      </ol>
    {:else}
      <p class="sub">{t('pp.off_hint')}</p>
    {/if}
    <div class="acts">
      <button type="button" class="btn small ghost" onclick={later}>{t('pp.later')}</button>
      {#if state === 'off'}
        <button type="button" class="btn small" disabled={busy} onclick={enable}><Icon d={I.bell} size={16} />{t('push.on')}</button>
      {/if}
    </div>
  </section>
{/if}

<style>
  .prompt {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .grow {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .title {
    font-weight: 600;
  }
  .sub {
    font-size: 13px;
    color: var(--muted);
  }
  .steps {
    margin: 0;
    padding-left: 22px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 14px;
  }
  .acts {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
