<script>
  import Sheet from './Sheet.svelte'
  import { api } from '../lib/api.js'
  import { t, errorText } from '../lib/i18n.js'
  import { refresh, toast } from '../lib/store.svelte.js'
  import { navigate } from '../lib/router.svelte.js'

  // mode: 'create' | 'edit' | 'member'
  let { mode = 'create', group = null, onclose, onsaved } = $props()

  let name = $state(group?.name || '')
  let username = $state('')
  let busy = $state(false)

  async function submit(e) {
    e.preventDefault()
    if (busy) return
    busy = true
    try {
      if (mode === 'create') {
        const g = await api.post('/groups', { name })
        await refresh()
        onclose()
        navigate(`/groups/${g.id}`)
        return
      }
      if (mode === 'edit') await api.patch(`/groups/${group.id}`, { name })
      if (mode === 'member') {
        await api.post(`/groups/${group.id}/invites`, { username })
        toast(t('inv.sent'))
      }
      await refresh()
      onsaved?.()
      onclose()
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      busy = false
    }
  }

  async function leave() {
    if (!confirm(t('grp.leave_confirm'))) return
    try {
      await api.post(`/groups/${group.id}/leave`)
      await refresh()
      onclose()
      navigate('/groups', { replace: true })
    } catch (err) {
      toast(errorText(err), 'err')
    }
  }

  const title = mode === 'create' ? t('grp.new') : mode === 'edit' ? t('grp.rename') : t('grp.add_member')
</script>

<Sheet {onclose} top={mode === 'edit' ? 240 : 300} label={title}>
  <form class="form" onsubmit={submit}>
    <h2 class="h-section" style="padding-top: 4px">{title}</h2>
    {#if mode === 'member'}
      <label class="field">
        <span>{t('grp.member_username')}</span>
        <input class="input" bind:value={username} autocapitalize="none" autocorrect="off" spellcheck="false" required />
      </label>
    {:else}
      <label class="field">
        <span>{t('grp.name')}</span>
        <input class="input" bind:value={name} placeholder={t('grp.name_ph')} maxlength="64" required />
      </label>
    {/if}
    <button class="btn" disabled={busy}>{mode === 'create' ? t('common.create') : mode === 'member' ? t('inv.invite') : t('common.save')}</button>
    {#if mode === 'edit'}<button type="button" class="btn danger" onclick={leave}>{t('grp.leave')}</button>{/if}
  </form>
</Sheet>
