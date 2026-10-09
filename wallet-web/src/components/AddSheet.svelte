<script>
  // Hızlı ekleme / düzenleme. Üç temanın kendi yerleşimi var; durum ve
  // kaydetme mantığı ortak.
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import Keypad from './Keypad.svelte'
  import OptionSheet from './OptionSheet.svelte'
  import SplitSheet from './SplitSheet.svelte'
  import { I } from '../lib/icons.js'
  import { ACCEPT, uploadReceipt, receiptUrl, fmtIban } from '../lib/receipt.js'
  import { api } from '../lib/api.js'
  import { catsFor, catInfo, pocketColor } from '../lib/categories.js'
  import { t, errorText, longDate, dayLabel } from '../lib/i18n.js'
  import { fmt, symbol, MINUS, keypadPress, keypadDisplay, keypadToKurus, kurusToKeypad, parseAmount } from '../lib/money.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { app, closeSheet, refresh, toast, accountIndex } from '../lib/store.svelte.js'

  let { sheet } = $props()

  const editing = sheet.kind === 'edit'
  const orig = sheet.tx || null
  const preset = sheet.preset || {}
  const me = app.user?.id
  const ownedByMe = !editing || orig.user_id === me

  const pad = (n) => String(n).padStart(2, '0')
  const toYmd = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const today = toYmd(new Date())

  let type = $state(orig?.type || preset.type || 'expense')
  let amt = $state(orig ? kurusToKeypad(orig.amount) : '')
  let accountId = $state(orig?.account_id ?? preset.accountId ?? app.accounts[0]?.id ?? null)
  let category = $state(orig ? orig.category : 'groceries')
  let groupId = $state(orig ? orig.group_id : (preset.groupId ?? null))
  let note = $state(orig?.description || '')
  let date = $state(orig ? toYmd(new Date(orig.occurred_at)) : today)
  let repeat = $state(preset.repeat || 'none')
  // Düzenli ödemeden gelmiş işlem düzenlenirken değişiklik kurala da yansısın.
  const fromRule = editing && ownedByMe && !!orig.recurring_id
  let applyFuture = $state(true)
  // Yeni işlem veya (kurala bağlı olmayan) kendi işlemim düzenli yapılabilir.
  const canRepeat = !editing || (ownedByMe && !orig.recurring_id)
  let split = $state({ mode: 'all', userIds: [], shares: {} })
  let splitTouched = $state(false)
  let members = $state([])
  let saving = $state(false)
  let picker = $state(null) // 'account' | 'group' | 'category' | 'repeat' | 'split'

  // Dekont ve karşı taraf (alıcı/gönderen).
  let receiptId = $state(orig?.receipt_id ?? null)
  let cpName = $state(orig?.counterparty_name || '')
  let cpIban = $state(orig?.counterparty_iban || '')
  let cpBank = $state(orig?.counterparty_bank || '')
  let reading = $state(null) // null | { stage: 'read'|'upload', p?: 0..1 }
  let fileInput = $state()

  function applyReceipt(r) {
    receiptId = r.id
    const p = r.parsed
    if (!p) return toast(t('rc.unreadable'), 'err')
    if (p.type && p.type !== type && !editing) setType(p.type)
    if (p.amount && !editing) amt = kurusToKeypad(p.amount)
    if (p.currency && account?.currency !== p.currency && !editing) {
      const other = app.accounts.find((a) => a.currency === p.currency)
      if (other) accountId = other.id
    }
    if (p.date && p.date <= today && !editing) date = p.date
    if (p.description && !note.trim()) note = p.description
    if (p.counterparty_name) cpName = p.counterparty_name
    if (p.counterparty_iban) cpIban = p.counterparty_iban
    if (p.counterparty_bank) cpBank = p.counterparty_bank
    if (!note.trim() && p.counterparty_name) note = (p.type === 'income' ? '← ' : '→ ') + p.counterparty_name
    const found = [p.amount, p.counterparty_name || p.counterparty_iban, p.date].filter(Boolean).length
    toast(found ? t('rc.read_ok') : t('rc.unreadable'), found ? 'ok' : 'err')
  }

  async function pickFile(e) {
    const file = e.currentTarget.files?.[0]
    e.currentTarget.value = ''
    if (!file) return
    reading = { stage: 'read' }
    try {
      const r = await uploadReceipt(file, { onstage: (stage, p) => (reading = { stage, p }) })
      if (r.duplicate && r.transaction_id && r.transaction_id !== orig?.id) return toast(t('rc.already_used'), 'err')
      applyReceipt(r)
    } catch (err) {
      toast(errorText(err), 'err')
    } finally {
      reading = null
    }
  }
  if (preset.receipt) applyReceipt(preset.receipt)

  const cpPayload = () => ({
    counterparty_name: cpName.trim() || null,
    counterparty_iban: cpIban.replace(/\s/g, '').toUpperCase() || null,
    counterparty_bank: cpBank.trim() || null,
  })

  const amount = $derived(keypadToKurus(amt))
  const account = $derived(app.accounts.find((a) => a.id === accountId))
  const currency = $derived(account?.currency || orig?.currency || 'TRY')
  const group = $derived(app.groups.find((g) => g.id === groupId))
  const sign = $derived(type === 'expense' ? MINUS : '+')
  const cats = $derived(catsFor(type))

  // Bakiye önizlemesi: düzenlemede eski etkisi geri alınır.
  const before = $derived(account?.balance ?? 0)
  const after = $derived.by(() => {
    const signed = (k, ty) => (ty === 'expense' ? -k : k)
    let b = before + signed(amount, type)
    if (editing && orig.account_id === accountId) b -= signed(orig.amount, orig.type)
    return b
  })

  // Grup seçilince üyeleri yükle (paylaşım için).
  $effect(() => {
    const gid = groupId
    members = []
    if (!gid) return
    api.get(`/groups/${gid}`).then((g) => {
      if (groupId !== gid) return
      members = g.members
      if (editing && orig.group_id === gid && orig.splits?.length && !splitTouched) {
        const ids = orig.splits.map((s) => s.user_id)
        const amounts = orig.splits.map((s) => s.amount)
        const equalish = Math.max(...amounts) - Math.min(...amounts) <= 1
        split = equalish
          ? { mode: ids.length === g.members.length ? 'all' : 'equal', userIds: ids, shares: {} }
          : { mode: 'exact', userIds: [], shares: Object.fromEntries(orig.splits.map((s) => [s.user_id, kurusToKeypad(s.amount)])) }
      }
    })
  })

  function setType(next) {
    if (type === next) return
    type = next
    category = next === 'income' ? 'salary' : 'groceries'
    if (next === 'income' && !editing) groupId = null
  }

  function setGroup(id) {
    groupId = id
    split = { mode: 'all', userIds: [], shares: {} }
    splitTouched = editing
  }

  const splitLabel = $derived.by(() => {
    if (split.mode === 'all') return t('split.equal_all')
    if (split.mode === 'equal') return t('split.equal_n', { n: split.userIds.length })
    return t('split.exact')
  })

  const repeatOpts = ['none', 'weekly', 'monthly', 'yearly']

  // Mevcut işlemi düzenli yaparken ilk tekrar: işlem gününden itibaren
  // bugünden sonraki ilk vade (geçmiş dönemler geriye dönük oluşturulmasın).
  function nextAfterToday(ymd, freq) {
    const [y, m, d] = ymd.split('-').map(Number)
    for (let i = 1; i < 2000; i++) {
      let n
      if (freq === 'weekly') n = new Date(y, m - 1, d + 7 * i)
      else {
        const months = freq === 'yearly' ? 12 * i : i
        const last = new Date(y, m - 1 + months + 1, 0).getDate()
        n = new Date(y, m - 1 + months, Math.min(d, last))
      }
      if (toYmd(n) > today) return toYmd(n)
    }
    return today
  }

  function splitPayload() {
    if (!groupId) return undefined
    if (split.mode === 'all') return undefined
    if (split.mode === 'equal') return { type: 'equal', user_ids: split.userIds }
    return {
      type: 'exact',
      shares: Object.entries(split.shares)
        .map(([id, v]) => ({ user_id: Number(id), amount: parseAmount(v) || 0 }))
        .filter((s) => s.amount > 0),
    }
  }

  function occurredAt() {
    // Seçili gün bugünse sunucu "şimdi"yi kullanır; değilse günün ortası.
    if (date === today) return undefined
    const [y, m, d] = date.split('-').map(Number)
    return new Date(y, m - 1, d, 12, 0, 0).toISOString()
  }

  async function save() {
    if (saving) return
    if (!amount) return toast(t('add.need_amount'), 'err')
    if (!accountId) return toast(t('add.need_account'), 'err')
    if (split.mode === 'exact' && groupId) {
      const s = Object.values(split.shares).reduce((a, v) => a + (parseAmount(v) || 0), 0)
      if (s !== amount) {
        picker = 'split'
        return toast(t('split.sum_error'), 'err')
      }
    }
    saving = true
    try {
      const description = note.trim() || null
      if (editing) {
        const body = {}
        if (type !== orig.type) body.type = type
        if (amount !== orig.amount) body.amount = amount
        if (category !== orig.category) body.category = category
        if ((description || null) !== (orig.description || null)) body.description = description
        if (date !== toYmd(new Date(orig.occurred_at))) body.occurred_at = occurredAt() || new Date().toISOString()
        if (ownedByMe && groupId !== orig.group_id) body.group_id = groupId
        if (groupId && splitTouched && groupId === orig.group_id) {
          body.split = split.mode === 'all' ? { type: 'equal', user_ids: members.map((m) => m.user_id) } : splitPayload()
        } else if (groupId && splitTouched && split.mode !== 'all') {
          body.split = splitPayload()
        }
        if (ownedByMe) {
          const cp = cpPayload()
          for (const k of Object.keys(cp)) if ((cp[k] || null) !== (orig[k] || null)) body[k] = cp[k]
          if (receiptId && receiptId !== orig.receipt_id) body.receipt_id = receiptId
        }
        if (Object.keys(body).length) await api.patch(`/transactions/${orig.id}`, body)
        if (fromRule && applyFuture) {
          const rb = {}
          if (body.amount !== undefined) rb.amount = amount
          if (body.category !== undefined && type === orig.type) rb.category = category
          if (body.description !== undefined) rb.description = description
          if (Object.keys(rb).length) await api.patch(`/recurring/${orig.recurring_id}`, rb)
        }
        if (canRepeat && repeat !== 'none') {
          await api.post('/recurring', {
            account_id: accountId, group_id: groupId, type, amount, category, description,
            split: groupId ? splitPayload() : undefined, frequency: repeat, start_on: nextAfterToday(date, repeat),
          })
          toast(t('add.repeat_saved'))
        } else toast(t('add.saved'))
      } else if (repeat !== 'none' && !receiptId) {
        await api.post('/recurring', {
          account_id: accountId, group_id: groupId, type, amount, category, description,
          split: splitPayload(), frequency: repeat, start_on: date,
        })
        toast(t('add.repeat_saved'))
      } else {
        await api.post('/transactions', {
          account_id: accountId, group_id: groupId, type, amount, category, description,
          occurred_at: occurredAt(), split: splitPayload(), ...cpPayload(), receipt_id: receiptId || undefined,
        })
        // Dekontlu işlem düzenli yapılırsa: bu işlem kaydedilir, kural sonraki dönemden başlar.
        if (repeat !== 'none') {
          await api.post('/recurring', {
            account_id: accountId, group_id: groupId, type, amount, category, description,
            split: splitPayload(), frequency: repeat, start_on: nextAfterToday(date, repeat),
          })
        }
        navigator.vibrate?.([10, 40, 10])
        toast(repeat !== 'none' ? t('add.repeat_saved') : t('add.saved'))
      }
      closeSheet()
      refresh()
    } catch (e) {
      toast(errorText(e), 'err')
    } finally {
      saving = false
    }
  }

  async function remove() {
    if (!confirm(t('common.confirm_delete'))) return
    try {
      await api.del(`/transactions/${orig.id}`)
      toast(t('add.deleted'))
      closeSheet()
      refresh()
    } catch (e) {
      toast(errorText(e), 'err')
    }
  }

  const accountOpts = $derived(
    app.accounts.map((a) => ({ value: a.id, label: a.name, sub: `${fmt(a.balance)} ${symbol(a.currency)}` })),
  )
  const groupOpts = $derived([{ value: null, label: t('common.none') }, ...app.groups.map((g) => ({ value: g.id, label: g.name }))])
  const catOpts = $derived(cats.map((c) => ({ value: c.id, label: t('cat.' + c.id), icon: c.icon, tint: c.tint })))
  const title = $derived(editing ? t('add.edit_title') : prefs.theme === 'b' ? t('add.title.b') : t('add.title'))
  const lockedAccount = editing
  const lockedGroup = editing && !ownedByMe
  const saveLabel = $derived(
    prefs.theme === 'b' ? t('add.save.b') : prefs.theme === 'c' ? `${t('common.save')} · ${sign}${fmt(amount)} ${symbol(currency)}` : t('common.save'),
  )
</script>

{#snippet dateChip(cls)}
  <label class="chip small {cls}">
    <Icon d={I.calendar} size={15} />
    <span>{date === today ? t('common.today') : dayLabel(new Date(date + 'T12:00:00'))}</span>
    <input class="date-input" type="date" bind:value={date} max={today} aria-label={t('add.date')} />
  </label>
{/snippet}

{#snippet receiptChip()}
  {#if ownedByMe}
    <button type="button" class="chip small" data-tour="receipt" aria-pressed={!!receiptId} disabled={!!reading} onclick={() => fileInput.click()}>
      <Icon d={I.receipt} size={15} /><span>{receiptId ? t('rc.attached') : t('rc.add')}</span>
    </button>
  {/if}
{/snippet}

{#snippet receiptCard()}
  {#if reading}
    <div class="rc-card reading" aria-live="polite">
      <span class="spinner" aria-hidden="true"></span>
      <span>{reading.stage === 'upload' ? t('rc.uploading') : t('rc.reading')}{reading.p ? ` %${Math.round(reading.p * 100)}` : ''}</span>
    </div>
  {:else if ownedByMe && (cpName || cpIban || receiptId)}
    <div class="rc-card">
      <span class="ico"><Icon d={I.person} size={17} /></span>
      <div class="rc-body">
        <label class="rc-name">
          <span class="sr">{type === 'income' ? t('rc.sender') : t('rc.recipient')}</span>
          <small>{type === 'income' ? t('rc.sender') : t('rc.recipient')}</small>
          <input type="text" bind:value={cpName} maxlength="128" placeholder={t('rc.name_ph')} enterkeyhint="done" />
        </label>
        {#if cpIban || cpBank}<small class="muted num">{[cpBank, fmtIban(cpIban)].filter(Boolean).join(' · ')}</small>{/if}
        {#if receiptId}<a class="rc-link" href={receiptUrl(receiptId)} target="_blank" rel="noopener"><Icon d={I.external} size={13} />{t('rc.view')}</a>{/if}
      </div>
    </div>
  {/if}
{/snippet}

{#snippet extraChips()}
  <div class="chips extra">
    {@render receiptChip()}
    {@render dateChip('')}
    {#if canRepeat}
      <button type="button" class="chip small" aria-pressed={repeat !== 'none'} onclick={() => (picker = 'repeat')}>
        <Icon d={I.repeat} size={15} /><span>{repeat === 'none' && editing ? t('rec.from_tx') : t('repeat.' + repeat)}</span>
      </button>
    {/if}
    {#if fromRule}
      <button type="button" class="chip small" aria-pressed={applyFuture} onclick={() => (applyFuture = !applyFuture)}>
        <Icon d={applyFuture ? I.check : I.repeat} size={15} /><span>{t('rec.apply_future')}</span>
      </button>
    {/if}
    {#if groupId && members.length}
      <button type="button" class="chip small" aria-pressed={split.mode !== 'all'} onclick={() => (picker = 'split')}>
        <Icon d={I.split} size={15} /><span>{splitLabel}</span>
      </button>
    {/if}
  </div>
{/snippet}

{#snippet amountBlock()}
  <section class="amount" aria-live="polite">
    <div class="figure">
      <span class="sign">{sign}</span>
      {#key amt}<span class="value">{keypadDisplay(amt)}</span>{/key}
      {#if prefs.theme === 'b'}<span class="caret" aria-hidden="true"></span>{/if}
      <span class="cur">{symbol(currency)}</span>
    </div>
    {#if account}
      {#if prefs.theme === 'b'}
        <div class="preview-b">
          <span>{t('add.after', { acc: account.name })}</span>
          <span><s>{fmt(before)}</s> → <strong>{fmt(after)} {symbol(currency)}</strong></span>
        </div>
      {:else}
        <span class="preview">{account.name}{prefs.theme === 'c' ? ':' : ''} {fmt(before)} → <strong>{fmt(after)}{prefs.theme === 'c' ? ' ' + symbol(currency) : ''}</strong></span>
      {/if}
    {/if}
  </section>
{/snippet}

<input bind:this={fileInput} type="file" accept={ACCEPT} hidden onchange={pickFile} />
<Sheet onclose={closeSheet} full={prefs.theme !== 'c'} top={64} label={title}>
  <div class="add add-{prefs.theme}">
    <div class="scroll">
    {#if prefs.theme === 'a'}
      <!-- A · Gece -->
      <header class="hdr-a">
        <button type="button" class="icon-btn" aria-label={t('common.close')} onclick={closeSheet}><Icon d={I.x} size={18} stroke={2.2} /></button>
        <div class="seg" role="group">
          <button type="button" aria-pressed={type === 'expense'} onclick={() => setType('expense')}>{t('common.expense')}</button>
          <button type="button" aria-pressed={type === 'income'} onclick={() => setType('income')}>{t('common.income')}</button>
        </div>
        {#if editing}
          <button type="button" class="icon-btn" aria-label={t('common.delete')} onclick={remove}><Icon d={I.trash} size={18} /></button>
        {:else}<span style="width: 44px"></span>{/if}
      </header>
      {@render amountBlock()}
      <div class="tiles">
        <button type="button" class="tile" disabled={lockedAccount} onclick={() => (picker = 'account')}>
          <span class="k">{t('add.account')}</span><span class="v">{account?.name ?? '—'}</span>
        </button>
        <button type="button" class="tile" disabled={lockedGroup} onclick={() => (picker = 'group')}>
          <span class="k">{t('add.group')}</span><span class="v">{group?.name ?? t('common.none')}</span>
        </button>
      </div>
      <div class="chips">
        {#each cats as c}
          <button type="button" class="chip" aria-pressed={category === c.id} onclick={() => (category = c.id)}>
            <Icon d={c.icon} size={17} /><span>{t('cat.' + c.id)}</span>
          </button>
        {/each}
      </div>
      {@render extraChips()}
      {@render receiptCard()}
      <label class="note-a">
        <span class="sr">{t('add.note')}</span>
        <input type="text" placeholder={t('add.note_ph')} bind:value={note} maxlength="255" enterkeyhint="done" />
      </label>
    {:else if prefs.theme === 'b'}
      <!-- B · Defter -->
      <header class="hdr-b">
        <button type="button" class="link-btn" onclick={closeSheet}>{t('common.cancel')}</button>
        <h1>{title}</h1>
        <label class="date-b">
          <span>{longDate(new Date(date + 'T12:00:00'))}</span>
          <input class="date-input" type="date" bind:value={date} max={today} aria-label={t('add.date')} />
        </label>
      </header>
      <div class="rule" style="margin-top: -4px"></div>
      <div class="type-b">
        <div class="seg" role="group">
          <button type="button" aria-pressed={type === 'expense'} onclick={() => setType('expense')}>{t('common.expense')}</button>
          <button type="button" aria-pressed={type === 'income'} onclick={() => setType('income')}>{t('common.income')}</button>
        </div>
        {#if editing}<button type="button" class="link-btn danger-link" onclick={remove}>{t('common.delete')}</button>{/if}
      </div>
      {@render amountBlock()}
      {@render receiptCard()}
      <div class="fields-b">
        {#if ownedByMe}
          <button type="button" class="field-b" data-tour="receipt" disabled={!!reading} onclick={() => fileInput.click()}>
            <span class="k">{t('rc.title')}</span><span class="v">{receiptId ? t('rc.attached') : t('rc.add')}</span><Icon d={I.receipt} size={16} stroke={2} />
          </button>
        {/if}
        <button type="button" class="field-b" disabled={lockedAccount} onclick={() => (picker = 'account')}>
          <span class="k">{t('add.account')}</span><span class="v">{account?.name ?? '—'}</span><Icon d={I.right} size={16} stroke={2} />
        </button>
        <button type="button" class="field-b" onclick={() => (picker = 'category')}>
          <span class="k">{t('add.category')}</span><span class="v">{t('cat.' + (category || 'none'))}</span><Icon d={I.right} size={16} stroke={2} />
        </button>
        <button type="button" class="field-b" disabled={lockedGroup} onclick={() => (picker = 'group')}>
          <span class="k">{t('add.group')}</span><span class="v">{group?.name ?? t('common.none')}</span><Icon d={I.right} size={16} stroke={2} />
        </button>
        {#if groupId && members.length}
          <button type="button" class="field-b" onclick={() => (picker = 'split')}>
            <span class="k">{t('add.split')}</span><span class="v">{splitLabel}</span><Icon d={I.right} size={16} stroke={2} />
          </button>
        {/if}
        {#if canRepeat}
          <button type="button" class="field-b" onclick={() => (picker = 'repeat')}>
            <span class="k">{t('add.repeat')}</span><span class="v">{t('repeat.' + repeat)}</span><Icon d={I.right} size={16} stroke={2} />
          </button>
        {/if}
        {#if fromRule}
          <button type="button" class="field-b" aria-pressed={applyFuture} onclick={() => (applyFuture = !applyFuture)}>
            <span class="k">{t('add.repeat')}</span><span class="v">{t('rec.apply_future')}</span><Icon d={applyFuture ? I.check : I.x} size={16} stroke={2} />
          </button>
        {/if}
        <label class="field-b">
          <span class="k">{t('add.note')}</span>
          <input class="v" type="text" placeholder={t('add.note_ph')} bind:value={note} maxlength="255" enterkeyhint="done" />
        </label>
      </div>
    {:else}
      <!-- C · Cepler -->
      <header class="hdr-c">
        <h1>{title}</h1>
        <div class="seg seg-c" role="group" data-type={type}>
          <button type="button" aria-pressed={type === 'expense'} onclick={() => setType('expense')}>{t('common.expense')}</button>
          <button type="button" aria-pressed={type === 'income'} onclick={() => setType('income')}>{t('common.income')}</button>
        </div>
      </header>
      {@render amountBlock()}
      <div class="accs-c">
        {#each app.accounts as a}
          {@const col = pocketColor(accountIndex(a.id))}
          <button type="button" class="acc-c" aria-pressed={a.id === accountId} disabled={lockedAccount && a.id !== accountId}
            style="background: {col.bg}; color: {col.fg}" onclick={() => (accountId = a.id)}>
            <span class="n">{a.name}</span><span class="b">{fmt(a.balance)}</span>
          </button>
        {/each}
      </div>
      <div class="cats-c">
        {#each cats as c}
          <button type="button" class="cat-c" aria-pressed={category === c.id} onclick={() => (category = c.id)}>
            <span class="circle" style={category === c.id ? '' : `background: ${c.tint}`}><Icon d={c.icon} size={22} stroke={2} /></span>
            <span class="l">{t('cat.' + c.id)}</span>
          </button>
        {/each}
      </div>
      {#if type === 'expense' || groupId}
        <div class="groups-c">
          <span class="k">{t('add.group')}</span>
          <div class="chips">
            <button type="button" class="chip small" aria-pressed={groupId === null} disabled={lockedGroup} onclick={() => setGroup(null)}>{t('common.none')}</button>
            {#each app.groups as g}
              <button type="button" class="chip small" aria-pressed={groupId === g.id} disabled={lockedGroup} onclick={() => setGroup(g.id)}>{g.name}</button>
            {/each}
          </div>
        </div>
      {/if}
      {@render extraChips()}
      {@render receiptCard()}
      <input class="input note-c" type="text" placeholder={t('add.note_ph')} bind:value={note} maxlength="255" enterkeyhint="done" />
      {#if editing}<button type="button" class="link-btn danger-link" onclick={remove}><Icon d={I.trash} size={16} />{t('common.delete')}</button>{/if}
    {/if}
    </div>

    <div class="bottom">
      <Keypad onkey={(k) => (amt = keypadPress(amt, k))} />
      <button type="button" class="btn save" class:glow={prefs.theme === 'a' && amount > 0} disabled={saving} onclick={save}>
        {#if prefs.theme === 'a'}<Icon d={I.check} size={20} stroke={2.4} />{/if}
        <span>{saveLabel}</span>
      </button>
    </div>
  </div>
</Sheet>

{#if picker === 'account'}
  <OptionSheet title={t('add.account')} options={accountOpts} selected={accountId} onpick={(v) => (accountId = v)} onclose={() => (picker = null)} />
{:else if picker === 'group'}
  <OptionSheet title={t('add.group')} options={groupOpts} selected={groupId} onpick={setGroup} onclose={() => (picker = null)} />
{:else if picker === 'category'}
  <OptionSheet title={t('add.category')} options={catOpts} selected={category} onpick={(v) => (category = v)} onclose={() => (picker = null)} />
{:else if picker === 'repeat'}
  <OptionSheet title={t('add.repeat')} options={repeatOpts.map((r) => ({ value: r, label: t('repeat.' + r) }))} selected={repeat}
    onpick={(v) => (repeat = v)} onclose={() => (picker = null)} />
{:else if picker === 'split'}
  <SplitSheet {members} {amount} value={split} onsave={(v) => { split = v; splitTouched = true }} onclose={() => (picker = null)} />
{/if}

<style>
  /* Üst kısım kaydırılabilir; tuş takımı ve kaydet her zaman görünür. */
  .add {
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex: 1 1 auto;
    min-height: 0;
  }
  .scroll {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    display: flex;
    flex-direction: column;
    gap: 14px;
    margin: 0 -20px;
    padding: 0 20px 4px;
  }
  .scroll > :global(*) {
    flex-shrink: 0;
  }
  .bottom {
    flex: 0 0 auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .amount {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }
  .figure {
    display: flex;
    align-items: baseline;
    gap: 6px;
    max-width: 100%;
  }
  .value {
    display: inline-block;
    white-space: nowrap;
  }
  .preview {
    font-family: var(--font-num);
    font-size: 13px;
    color: var(--muted);
  }
  .preview strong {
    color: var(--fg);
    font-weight: 500;
  }
  .date-input {
    position: absolute;
    inset: 0;
    opacity: 0;
    width: 100%;
    height: 100%;
    -webkit-appearance: none;
    appearance: none;
  }
  .chip.small {
    position: relative;
  }
  .extra {
    margin-top: -4px;
  }
  .danger-link {
    color: var(--danger);
    align-self: center;
  }

  /* ── A ── */
  .hdr-a {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 44px;
  }
  .add-a .amount {
    padding: 14px 0 2px;
  }
  .add-a .sign {
    font-size: 34px;
    font-weight: 500;
    color: var(--muted2);
  }
  .add-a .value {
    font-size: clamp(48px, 18vw, 72px);
    font-weight: 600;
    letter-spacing: -0.05em;
    line-height: 0.95;
  }
  .add-a .cur {
    font-size: 30px;
    font-weight: 500;
    color: var(--muted2);
  }
  .tiles {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .tile {
    height: 60px;
    border-radius: 16px;
    border: 1px solid var(--line);
    background: var(--surface);
    text-align: left;
    padding: 0 14px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 3px;
    min-width: 0;
  }
  .tile:disabled {
    opacity: 0.6;
  }
  .tile .k {
    font-size: 12px;
    color: var(--muted);
  }
  .tile .v {
    font-size: 15px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .note-a {
    display: flex;
    height: 44px;
    border-bottom: 1px solid var(--line2);
  }
  .note-a input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
  }
  .note-a input::placeholder {
    color: var(--muted2);
  }
  .save.glow {
    animation: glow 2.4s ease-in-out infinite;
  }
  @keyframes glow {
    0%, 100% { box-shadow: 0 0 0 0 rgba(198, 244, 50, 0); }
    50% { box-shadow: 0 8px 28px -6px rgba(198, 244, 50, 0.55); }
  }

  /* ── B ── */
  .hdr-b {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    align-items: center;
    height: 44px;
  }
  .hdr-b .link-btn {
    font-size: 16px;
  }
  .hdr-b h1 {
    text-align: center;
    font-family: 'Newsreader', serif;
    font-style: italic;
    font-size: 21px;
    font-weight: 400;
    white-space: nowrap;
  }
  .date-b {
    position: relative;
    justify-self: end;
    font-size: 13px;
    color: var(--muted);
    min-height: 44px;
    display: flex;
    align-items: center;
  }
  .type-b {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .add-b .amount {
    align-items: stretch;
  }
  .add-b .figure {
    font-family: 'Newsreader', serif;
  }
  .add-b .sign {
    font-size: 40px;
    color: var(--muted);
  }
  .add-b .value {
    font-size: clamp(56px, 19vw, 78px);
    font-weight: 500;
    letter-spacing: -0.035em;
    line-height: 0.95;
  }
  .caret {
    display: inline-block;
    width: 2px;
    height: 58px;
    background: var(--accent);
    margin-left: 4px;
    align-self: center;
    animation: blink 1.05s steps(1) infinite;
  }
  @keyframes blink {
    0%, 50% { opacity: 1; }
    51%, 100% { opacity: 0; }
  }
  .add-b .cur {
    font-size: 34px;
    color: var(--muted);
    margin-left: 4px;
  }
  .preview-b {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    font-size: 13px;
    color: var(--muted);
    border-top: 1px solid var(--line);
    padding-top: 8px;
  }
  .preview-b s {
    text-decoration-color: var(--line2);
  }
  .preview-b strong {
    color: var(--fg);
    font-weight: 600;
  }
  .fields-b {
    display: flex;
    flex-direction: column;
  }
  .field-b {
    min-height: 46px;
    border: none;
    border-bottom: 1px solid var(--line);
    background: transparent;
    padding: 0;
    display: flex;
    align-items: center;
    gap: 10px;
    text-align: left;
    color: var(--fg);
  }
  .field-b:disabled {
    opacity: 0.55;
  }
  .field-b .k {
    flex: 0 0 82px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .field-b .v {
    flex: 1 1 auto;
    min-width: 0;
    font-family: 'Newsreader', serif;
    font-size: 18px;
    border: none;
    background: transparent;
    outline: none;
    padding: 0;
  }
  .field-b :global(svg) {
    color: var(--muted2);
  }

  /* ── C ── */
  .add-c .scroll {
    gap: 10px;
  }
  .hdr-c {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 44px;
  }
  .hdr-c h1 {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 24px;
    font-weight: 800;
    letter-spacing: -0.03em;
  }
  .seg-c[data-type='income'] button[aria-pressed='true'] {
    background: #0e4d33;
  }
  .add-c .amount {
    gap: 2px;
  }
  .add-c .figure {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-weight: 800;
    letter-spacing: -0.045em;
    font-size: clamp(44px, 15vw, 60px);
    line-height: 1.05;
    gap: 0;
  }
  .add-c .value {
    animation: bump 0.28s cubic-bezier(0.3, 1.6, 0.5, 1);
  }
  .add-c .cur {
    margin-left: 0.2em;
  }
  @keyframes bump {
    from { transform: scale(1.06); }
    to { transform: none; }
  }
  .add-c .preview {
    font-family: var(--font);
    font-weight: 500;
  }
  .add-c .preview strong {
    font-weight: 700;
  }
  .accs-c {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding: 8px 6px 4px;
    margin: 0 -6px;
    scrollbar-width: none;
  }
  .accs-c::-webkit-scrollbar {
    display: none;
  }
  .acc-c {
    flex: 1 0 104px;
    height: 58px;
    border: none;
    border-radius: 16px;
    text-align: left;
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    transition: transform 0.35s cubic-bezier(0.34, 1.5, 0.5, 1), box-shadow 0.2s ease;
  }
  .acc-c[aria-pressed='true'] {
    transform: translateY(-4px);
    box-shadow: 0 0 0 3px #f2f0ea, 0 0 0 5px #1e1a12;
  }
  .acc-c:disabled {
    opacity: 0.4;
  }
  .acc-c .n {
    font-size: 12px;
    font-weight: 700;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .acc-c .b {
    font-size: 13px;
    font-weight: 600;
    opacity: 0.9;
  }
  .cats-c {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px 4px;
  }
  .cat-c {
    border: none;
    background: transparent;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 0;
    color: var(--fg);
  }
  .circle {
    width: 48px;
    height: 48px;
    border-radius: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #1e1a12;
    transition: background-color 0.2s ease, color 0.2s ease, transform 0.3s cubic-bezier(0.34, 1.6, 0.5, 1);
  }
  .cat-c[aria-pressed='true'] .circle {
    background: #1e1a12;
    color: #fff;
    transform: scale(1.08);
  }
  .cat-c .l {
    font-size: 12px;
    font-weight: 600;
  }
  .groups-c {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .groups-c .k {
    font-size: 13px;
    font-weight: 600;
    color: var(--muted);
    flex: 0 0 auto;
  }
  .groups-c .chips {
    margin: 0;
    padding: 2px;
    flex: 1;
  }
  .note-c {
    height: 44px;
  }
  .add-c .bottom {
    gap: 10px;
  }
  .add-c .save {
    color: #f2b441;
  }
  .rc-card {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 12px 14px;
    border-radius: var(--r-card);
    background: var(--surface);
    border: 1px solid var(--line);
  }
  .rc-card.reading {
    align-items: center;
    color: var(--muted);
    font-size: 14px;
  }
  .rc-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .rc-name {
    display: flex;
    flex-direction: column;
  }
  .rc-name small,
  .rc-body > small {
    font-size: 12px;
    color: var(--muted);
  }
  .rc-name input {
    border: none;
    background: transparent;
    padding: 2px 0;
    font-size: 16px;
    font-weight: 600;
    outline: none;
    width: 100%;
  }
  .rc-link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
    font-weight: 600;
    min-height: 32px;
  }
  .spinner {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    border: 2px solid var(--line2);
    border-top-color: var(--accent);
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
