<script>
  import Logo from '../components/Logo.svelte'
  import Icon from '../components/Icon.svelte'
  import TxRow from '../components/TxRow.svelte'
  import LedgerHead from '../components/LedgerHead.svelte'
  import CountUp from '../components/CountUp.svelte'
  import MonthPicker from '../components/MonthPicker.svelte'
  import AccountSheet from '../components/AccountSheet.svelte'
  import GoalCard from '../components/GoalCard.svelte'
  import GoalView from '../components/GoalView.svelte'
  import InboxSheet from '../components/InboxSheet.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, tv, monthLabel, monthName, dayLabel } from '../lib/i18n.js'
  import { fmt, fmtc, parts, symbol } from '../lib/money.js'
  import { pocketColor } from '../lib/categories.js'
  import { fmtRate } from '../lib/fx.js'
  import { prefs } from '../lib/prefs.svelte.js'
  import { navigate } from '../lib/router.svelte.js'
  import { app, totals, monthTotals, accountMonth } from '../lib/store.svelte.js'

  let recent = $state([])
  let monthOpen = $state(false)
  let newAccount = $state(false)
  let front = $state(null)
  let goals = $state([])
  let rules = $state([])
  let viewing = $state(null)
  let inboxOpen = $state(false)

  const loadGoals = () => api.get('/goals').then((g) => (goals = g)).catch(() => {})
  $effect(() => {
    app.version
    api.get('/transactions', { limit: prefs.theme === 'c' ? 4 : 5 }).then((r) => (recent = r)).catch(() => {})
    loadGoals()
    api.get('/recurring').then((r) => (rules = r.filter((x) => x.active && x.user_id === app.user?.id))).catch(() => {})
  })
  const nextRule = $derived(rules[0])

  const tot = $derived(totals())
  const mt = $derived(monthTotals())
  const ratio = $derived(mt.income > 0 ? Math.round((mt.expense / mt.income) * 100) : null)
  const ticks = $derived.by(() => {
    const filled = ratio === null ? 0 : Math.min(20, (ratio / 100) * 20)
    return Array.from({ length: 20 }, (_, i) => ({
      c: i < Math.floor(filled) ? 'var(--accent)' : i < filled ? '#7E9C22' : '#222227',
      d: (0.35 + i * 0.025).toFixed(3) + 's',
    }))
  })
  const initial = $derived((app.user?.username || '?').charAt(0))
  const others = $derived(tot.others.map(([c, v]) => fmtc(v, c)).join(' · '))

  // C: kart yığını; öndeki kart en altta görünür.
  const cards = $derived.by(() => {
    const list = app.accounts.map((a, i) => ({ ...a, color: pocketColor(i) }))
    const f = front ?? list[0]?.id
    const order = [...list.filter((c) => c.id !== f), ...list.filter((c) => c.id === f)].map((c) => c.id)
    return list.map((c, i) => ({ ...c, z: order.indexOf(c.id) + 1, delay: (0.12 + i * 0.09).toFixed(2), isFront: c.id === f }))
  })
  const stackStep = $derived(app.accounts.length > 3 ? 40 : 53)
  const stackHeight = $derived(176 + Math.max(0, app.accounts.length - 1) * stackStep)

  function pickCard(c) {
    if (c.isFront) navigate(`/accounts/${c.id}`)
    else front = c.id
  }
  const kindLabel = (k) => t('acc.kind.' + k)
</script>

{#snippet header()}
  <header class="topbar">
    {#if prefs.theme === 'a'}
      <div class="row-left">
        <Logo size={34} />
        <button type="button" class="month-a" onclick={() => (monthOpen = true)}>
          <span>{monthLabel(app.month)}</span><Icon d={I.down} size={14} stroke={2.2} />
        </button>
      </div>
    {:else if prefs.theme === 'b'}
      <div class="row-left">
        <Logo size={30} />
        <span class="wordmark">{t('app.name')}</span>
      </div>
    {:else}
      <div class="row-left">
        <Logo size={34} />
        <span class="hello">{t('home.hello', { name: app.user?.username })}</span>
      </div>
    {/if}
    <div class="row-left">
      {#if prefs.theme === 'b'}<button type="button" class="month-b" onclick={() => (monthOpen = true)}>{monthLabel(app.month)}</button>{/if}
      <button type="button" class="avatar" data-tour="settings" aria-label={t('nav.profile')} onclick={() => navigate('/settings')}>{initial}</button>
    </div>
  </header>
  <div class="rule"></div>
{/snippet}

<div class="page home-{prefs.theme}">
  {@render header()}

  {#if app.inbox.length}
    <button type="button" class="card inbox rise" data-tour="inbox" onclick={() => (inboxOpen = true)}>
      <span class="ico"><Icon d={I.inbox} size={18} /></span>
      <span class="grow">
        <span class="title">{t('rc.inbox')}</span>
        <span class="sub">{t('rc.inbox_n', { n: app.inbox.length })}</span>
      </span>
      <span class="badge num">{app.inbox.length}</span>
    </button>
  {/if}

  {#if prefs.theme === 'a'}
    <!-- ───── A · Gece ───── -->
    <section class="total-a" data-tour="total" aria-label={t('home.total')}>
      <span class="h-section">{t('home.total')}</span>
      <CountUp value={tot.amount}>
        {#snippet children(v)}
          {@const p = parts(v)}
          <div class="big-a"><span class="i">{p.int}</span><span class="d">{p.sep}{p.dec}</span><span class="c">{symbol(tot.main)}</span></div>
        {/snippet}
      </CountUp>
      {#if others}<span class="others">{tot.converted ? t('fx.incl', { v: others }) : t('home.other_currencies', { v: others })}</span>{/if}
      <div class="tiles-a">
        <button type="button" class="tile-a" onclick={() => (monthOpen = true)}>
          <span class="k">{t('common.income')}</span><span class="v num pos">+{fmt(mt.income)}</span>
        </button>
        <button type="button" class="tile-a" onclick={() => (monthOpen = true)}>
          <span class="k">{t('common.expense')}</span><span class="v num">{mt.expense ? '−' : ''}{fmt(mt.expense)}</span>
        </button>
      </div>
      <div class="ticks-wrap">
        <div class="ticks">
          {#each ticks as tk}<span class="tick" style="background: {tk.c}; animation-delay: {tk.d}"></span>{/each}
        </div>
        <span class="ratio">
          {#if ratio === null}{t('home.no_income')}{:else}{t('home.spent_ratio', { p: ratio })}{/if}
          · {t('common.net').toLowerCase()} <b>{fmt(mt.net, 'always')} {symbol(tot.main)}</b>
        </span>
      </div>
    </section>

    <section class="section">
      <h2 class="h-section">{t('home.accounts')}</h2>
      <div class="chips">
        {#each app.accounts as a}
          <a class="acc-chip" href={`/accounts/${a.id}`} onclick={(e) => { e.preventDefault(); navigate(`/accounts/${a.id}`) }}>
            <span class="n">{a.name}</span><span class="b num">{fmt(a.balance)}</span>
          </a>
        {/each}
        <button type="button" class="acc-chip add" onclick={() => (newAccount = true)}><Icon d={I.plus} size={16} stroke={2.2} />{app.accounts.length ? '' : t('home.add_account')}</button>
      </div>
    </section>
  {:else if prefs.theme === 'b'}
    <!-- ───── B · Defter ───── -->
    <section class="rise total-b" data-tour="total" style="animation-delay: .1s" aria-label={t('home.total')}>
      <span class="h-section">{t('home.total')}</span>
      <CountUp value={tot.amount}>
        {#snippet children(v)}
          <div class="big-b"><span>{fmt(v)}</span><span class="c">{symbol(tot.main)}</span></div>
        {/snippet}
      </CountUp>
      {#if others}<span class="others">{tot.converted ? t('fx.incl', { v: others }) : t('home.other_currencies', { v: others })}</span>{/if}
    </section>
    <button type="button" class="rise sum-b" style="animation-delay: .18s" onclick={() => (monthOpen = true)}>
      <span><small>{t('common.income')}</small><b class="pos">+{fmt(mt.income)}</b></span>
      <span><small>{t('common.expense')}</small><b>{mt.expense ? '−' : ''}{fmt(mt.expense)}</b></span>
      <span><small>{t('common.net')}</small><b>{fmt(mt.net, 'always')}</b></span>
    </button>
    <section class="rise section" style="animation-delay: .26s; gap: 4px">
      <h2 class="h-section" style="margin-bottom: 4px">{t('home.accounts')}</h2>
      {#each app.accounts as a}
        <a class="leader" href={`/accounts/${a.id}`} onclick={(e) => { e.preventDefault(); navigate(`/accounts/${a.id}`) }}>
          <span class="n">{a.name}</span><span class="dots" aria-hidden="true"></span><span class="b">{fmt(a.balance)}</span>
        </a>
      {/each}
      <button type="button" class="link-btn" onclick={() => (newAccount = true)}><Icon d={I.plus} size={15} stroke={2.2} />{t('home.add_account')}</button>
    </section>
  {:else}
    <!-- ───── C · Cepler ───── -->
    <section class="rise total-c" data-tour="total" aria-label={t('home.total')}>
      <span class="lbl">{t('home.pockets_total', { n: app.accounts.length })}</span>
      <CountUp value={tot.amount}>
        {#snippet children(v)}<span class="big-c">{fmt(v)} {symbol(tot.main)}</span>{/snippet}
      </CountUp>
      {#if others}<span class="others">{tot.converted ? t('fx.incl', { v: others }) : t('home.other_currencies', { v: others })}</span>{/if}
    </section>
    {#if app.accounts.length}
      <section class="stack" style="height: {stackHeight}px" aria-label={t('home.accounts')}>
        {#each cards as c}
          {@const m = accountMonth(c.id)}
          <div class="slot" style="animation-delay: {c.delay}s; z-index: {c.z}">
            <button type="button" class="pocket" style="top: {(c.z - 1) * stackStep}px; background: {c.color.bg}; color: {c.color.fg}" onclick={() => pickCard(c)}>
              <svg aria-hidden="true" width="420" height="140" viewBox="0 0 420 140" class="wave"><path d="M0 20 Q210 110 420 20 V140 H0 Z" fill={c.color.fg} /></svg>
              <span class="top"><span class="n">{c.name}</span><span class="k">{kindLabel(c.kind)}</span></span>
              <span class="bot">
                <span class="b">{fmt(c.balance)} {symbol(c.currency)}</span>
                <span class="k">{t('home.this_month', { v: fmtc(m.income - m.expense, c.currency, 'always') })}</span>
              </span>
            </button>
          </div>
        {/each}
      </section>
    {:else}
      <button type="button" class="btn ghost" onclick={() => (newAccount = true)}><Icon d={I.plus} size={18} />{t('home.add_account')}</button>
    {/if}
    <section class="rise tiles-c" style="animation-delay: .35s">
      <button type="button" class="tile-c in" onclick={() => (monthOpen = true)}>
        <span class="k"><Icon d={I.arrowDown} size={16} stroke={2.4} />{t('home.month_income', { m: monthName(app.month) })}</span>
        <span class="v">{fmt(mt.income)}</span>
      </button>
      <button type="button" class="tile-c out" onclick={() => (monthOpen = true)}>
        <span class="k"><Icon d={I.arrowUp} size={16} stroke={2.4} />{t('home.month_expense', { m: monthName(app.month) })}</span>
        <span class="v">{fmt(mt.expense)}</span>
      </button>
    </section>
  {/if}

  <!-- Birikim hedefleri ve düzenli ödemeler (ortak) -->
  {#if app.accounts.length}
    <section class="section" data-tour="goals">
      <div class="section-head">
        <h2 class="h-section">{t('goal.title')}</h2>
        <a href="/goals" onclick={(e) => { e.preventDefault(); navigate('/goals') }}>{goals.length ? t('common.all') : t('goal.new')}</a>
      </div>
      {#each goals.slice(0, 2) as g, i (g.id)}
        <GoalCard goal={g} delay={0.3 + i * 0.06} onclick={() => (viewing = g)} />
      {/each}
      {#if !goals.length}
        <button type="button" class="card shortcut" onclick={() => navigate('/goals')}>
          <span class="ico"><Icon d={I.target} size={18} /></span>
          <span class="grow"><span class="sub">{t('goal.empty')}</span></span>
          <Icon d={I.right} size={16} stroke={2} />
        </button>
      {/if}
    </section>
    <section class="section">
      <button type="button" class="card shortcut" data-tour="payees" onclick={() => navigate('/payees')}>
        <span class="ico"><Icon d={I.person} size={18} /></span>
        <span class="grow">
          <span class="title">{t('cp.title')}</span>
          <span class="sub">{t('tour.payees.body')}</span>
        </span>
        <Icon d={I.right} size={16} stroke={2} />
      </button>
      <button type="button" class="card shortcut" data-tour="rates" onclick={() => navigate('/rates')}>
        <span class="ico"><Icon d={I.swap} size={18} /></span>
        <span class="grow">
          <span class="title">{t('fx.title')}</span>
          <span class="sub num">
            {#if app.rates}USD {fmtRate(app.rates.online.rates.USD?.sell)} · EUR {fmtRate(app.rates.online.rates.EUR?.sell)} · GBP {fmtRate(app.rates.online.rates.GBP?.sell)}
            {:else}{t('fx.source')}{/if}
          </span>
        </span>
        <Icon d={I.right} size={16} stroke={2} />
      </button>
      <button type="button" class="card shortcut" data-tour="recurring" onclick={() => navigate('/recurring')}>
        <span class="ico"><Icon d={I.repeat} size={18} /></span>
        <span class="grow">
          <span class="title">{t('rec.title')}</span>
          <span class="sub">
            {#if nextRule}{t('rec.count', { n: rules.length })} · {nextRule.description || t('cat.' + (nextRule.category || 'other'))} {t('rec.next', { d: dayLabel(nextRule.next_run_on + 'T12:00:00') }).toLowerCase()}
            {:else}{t('rec.empty')}{/if}
          </span>
        </span>
        <Icon d={I.right} size={16} stroke={2} />
      </button>
    </section>
  {/if}

  <!-- Son işlemler (ortak) -->
  <section class="section recent">
    <div class="section-head">
      <h2 class="h-section">{prefs.theme === 'b' ? t('home.recent.b') : t('home.recent')}</h2>
      <a href="/transactions" onclick={(e) => { e.preventDefault(); navigate('/transactions') }}>
        {prefs.theme === 'b' ? t('home.open_ledger') : t('common.all')}
      </a>
    </div>
    {#if prefs.theme === 'b'}<LedgerHead />{/if}
    <div class={prefs.theme === 'b' ? '' : 'list'}>
      {#each recent as tx, i (tx.id)}
        <TxRow {tx} delay={0.4 + i * 0.06} />
      {/each}
    </div>
    {#if !recent.length}
      <p class="empty">{app.accounts.length ? t('common.empty') : t('home.no_accounts')}</p>
    {/if}
  </section>
</div>

{#if monthOpen}<MonthPicker onclose={() => (monthOpen = false)} />{/if}
{#if newAccount}<AccountSheet onclose={() => (newAccount = false)} />{/if}
{#if inboxOpen}<InboxSheet onclose={() => (inboxOpen = false)} />{/if}
{#if viewing}<GoalView goal={viewing} onclose={() => (viewing = null)} onchange={loadGoals} />{/if}

<style>
  .inbox {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    text-align: left;
    color: var(--fg);
    border-color: var(--accent);
  }
  .inbox .grow {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .inbox .title {
    font-weight: 600;
  }
  .inbox .sub {
    font-size: 13px;
    color: var(--muted);
  }
  .badge {
    min-width: 26px;
    height: 26px;
    padding: 0 8px;
    border-radius: 13px;
    background: var(--accent);
    color: var(--accent-fg);
    font-size: 13px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .shortcut {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    text-align: left;
    color: var(--fg);
  }
  .shortcut .grow {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .shortcut .title {
    font-weight: 600;
    font-size: 15px;
  }
  .shortcut .sub {
    font-size: 13px;
    color: var(--muted);
  }
  .row-left {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }
  .others {
    font-size: 13px;
    color: var(--muted);
  }

  /* A */
  .month-a {
    height: 40px;
    padding: 0 12px 0 14px;
    border-radius: 20px;
    border: 1px solid #26262c;
    background: #141417;
    font-size: 14px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .total-a {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .big-a {
    display: flex;
    align-items: baseline;
  }
  .big-a .i {
    font-size: clamp(48px, 16vw, 64px);
    font-weight: 600;
    letter-spacing: -0.045em;
    line-height: 0.9;
  }
  .big-a .d,
  .big-a .c {
    font-size: 30px;
    font-weight: 500;
    letter-spacing: -0.02em;
    color: var(--muted2);
  }
  .big-a .c {
    margin-left: 6px;
  }
  .tiles-a {
    display: flex;
    gap: 8px;
  }
  .tile-a {
    flex: 1 1 0;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    text-align: left;
  }
  .tile-a .k {
    font-size: 12px;
    color: var(--muted);
  }
  .tile-a .v {
    font-size: 16px;
    font-weight: 500;
  }
  .ticks-wrap {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .ticks {
    display: grid;
    grid-template-columns: repeat(20, minmax(0, 1fr));
    gap: 3px;
  }
  .tick {
    height: 18px;
    border-radius: 3px;
    transform-origin: 50% 100%;
    animation: tick 0.5s cubic-bezier(0.3, 1.5, 0.5, 1) both;
  }
  @keyframes tick {
    from { transform: scaleY(0); opacity: 0; }
    to { transform: scaleY(1); opacity: 1; }
  }
  .ratio {
    font-size: 13px;
    color: var(--muted);
  }
  .ratio b {
    color: var(--fg);
    font-weight: 400;
  }
  .acc-chip {
    flex: 0 0 auto;
    height: 48px;
    padding: 0 16px;
    border-radius: 24px;
    background: var(--surface);
    border: 1px solid var(--line);
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--fg);
    white-space: nowrap;
    font-size: 14px;
  }
  .acc-chip .n {
    color: var(--muted);
  }
  .acc-chip .b {
    font-weight: 500;
  }
  .acc-chip.add {
    color: var(--muted);
    gap: 6px;
  }

  /* B */
  .wordmark {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 22px;
    font-weight: 800;
    letter-spacing: -0.04em;
  }
  .month-b {
    border: none;
    background: none;
    font-size: 14px;
    color: var(--muted);
    min-height: 44px;
    padding: 0;
  }
  .total-b {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .big-b {
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-family: 'Newsreader', serif;
    font-size: clamp(44px, 15vw, 60px);
    font-weight: 500;
    letter-spacing: -0.03em;
    line-height: 1;
  }
  .big-b .c {
    font-size: 30px;
    font-weight: 400;
    color: var(--muted);
  }
  .sum-b {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    border: none;
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    background: none;
    padding: 0;
    text-align: left;
    color: var(--fg);
  }
  .sum-b > span {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 12px 10px;
    border-left: 1px solid var(--line);
  }
  .sum-b > span:first-child {
    border-left: none;
    padding-left: 0;
  }
  .sum-b small {
    font-size: 12px;
    color: var(--muted);
  }
  .sum-b b {
    font-family: 'Newsreader', serif;
    font-size: 17px;
    font-weight: 500;
  }
  .leader {
    display: flex;
    align-items: baseline;
    gap: 8px;
    min-height: 36px;
    padding-top: 8px;
    color: var(--fg);
  }
  .leader .n {
    font-size: 16px;
    font-weight: 500;
  }
  .leader .dots {
    flex: 1 1 auto;
    border-bottom: 1.5px dotted var(--line2);
    transform: translateY(-4px);
  }
  .leader .b {
    font-family: 'Newsreader', serif;
    font-size: 18px;
    font-weight: 500;
  }

  /* C */
  .hello {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 24px;
    font-weight: 800;
    letter-spacing: -0.035em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .total-c {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .total-c .lbl {
    font-size: 14px;
    font-weight: 500;
    color: var(--muted);
  }
  .big-c {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: clamp(36px, 12vw, 46px);
    font-weight: 800;
    letter-spacing: -0.04em;
    line-height: 1.02;
  }
  .stack {
    position: relative;
    margin-top: -6px;
  }
  .slot {
    position: absolute;
    inset: 0;
    pointer-events: none;
    animation: cin 0.7s cubic-bezier(0.2, 0.9, 0.25, 1.15) both;
  }
  @keyframes cin {
    from { opacity: 0; transform: translateY(80px) rotate(4deg); }
    to { opacity: 1; transform: none; }
  }
  .pocket {
    pointer-events: auto;
    position: absolute;
    left: 0;
    right: 0;
    height: 176px;
    border: none;
    border-radius: 26px;
    text-align: left;
    padding: 18px 20px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    overflow: hidden;
    box-shadow: 0 -2px 0 rgba(255, 255, 255, 0.18) inset, 0 10px 24px rgba(30, 26, 18, 0.16);
    transition: top 0.5s cubic-bezier(0.34, 1.4, 0.5, 1), transform 0.5s cubic-bezier(0.34, 1.4, 0.5, 1);
  }
  .pocket:active {
    transform: scale(0.98);
  }
  .wave {
    position: absolute;
    left: -20px;
    bottom: -40px;
    opacity: 0.13;
  }
  .pocket .top,
  .pocket .bot {
    position: relative;
    display: flex;
  }
  .pocket .top {
    justify-content: space-between;
    align-items: center;
  }
  .pocket .bot {
    flex-direction: column;
    gap: 2px;
  }
  .pocket .n {
    font-size: 16px;
    font-weight: 700;
  }
  .pocket .k {
    font-size: 13px;
    font-weight: 600;
    opacity: 0.85;
  }
  .pocket .b {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 32px;
    font-weight: 800;
    letter-spacing: -0.03em;
  }
  .tiles-c {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .tile-c {
    border: none;
    border-radius: 20px;
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    text-align: left;
  }
  .tile-c.in {
    background: #d7eee2;
    color: #0e4d33;
  }
  .tile-c.out {
    background: #fadfcf;
    color: #74290b;
  }
  .tile-c .k {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 600;
  }
  .tile-c .v {
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 22px;
    font-weight: 800;
    letter-spacing: -0.02em;
  }
  .recent {
    gap: 4px;
  }
  .home-c .recent {
    gap: 6px;
  }
</style>
