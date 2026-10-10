<script>
  // Aylık rapor: seçili ayın gelir/gider/net'i, geçen aya göre değişim, son 6
  // ayın gelir-gider grafiği ve kategori dağılımı. Transferler ve kart açılış
  // borcu sayılmaz (/summary ile aynı). Her para birimi ayrı gösterilir.
  import Icon from '../components/Icon.svelte'
  import MonthPicker from '../components/MonthPicker.svelte'
  import { I } from '../lib/icons.js'
  import { api } from '../lib/api.js'
  import { t, monthLabel, monthName } from '../lib/i18n.js'
  import { fmt, fmtc, symbol } from '../lib/money.js'
  import { catInfo } from '../lib/categories.js'
  import { route, back } from '../lib/router.svelte.js'
  import { app, monthRange, setMonth } from '../lib/store.svelte.js'

  const N = 6
  let months = $state([]) // [{ month: Date, summary }], eskiden yeniye
  let currency = $state(null)
  let picking = $state(false)
  let selected = $state(N - 1) // grafikte seçili ay (dokunma)

  // Bildirimden "/report?month=2026-10" ile gelindiyse o ay.
  if (route.query?.month) {
    const [y, m] = route.query.month.split('-').map(Number)
    if (y && m) setMonth(new Date(y, m - 1, 1))
  }

  $effect(() => {
    app.version
    const base = app.month
    const list = Array.from({ length: N }, (_, i) => new Date(base.getFullYear(), base.getMonth() - (N - 1 - i), 1))
    Promise.all(list.map((m) => api.get('/summary', monthRange(m)).catch(() => null))).then((r) => {
      months = list.map((m, i) => ({ month: m, summary: r[i] }))
      selected = N - 1
    })
  })

  // Para birimleri: en çok hareket olan önce.
  const currencies = $derived.by(() => {
    const vol = {}
    for (const { summary } of months)
      for (const x of summary?.totals ?? []) vol[x.currency] = (vol[x.currency] || 0) + x.income + x.expense
    return Object.keys(vol).sort((a, b) => vol[b] - vol[a])
  })
  const cur = $derived(currency && currencies.includes(currency) ? currency : currencies[0] || 'TRY')

  const tot = (s) => s?.totals?.find((x) => x.currency === cur) || { income: 0, expense: 0, net: 0 }
  const series = $derived(months.map(({ month, summary }) => ({ month, ...tot(summary) })))
  const now = $derived(series[N - 1] || { income: 0, expense: 0, net: 0 })
  const prev = $derived(series[N - 2] || { income: 0, expense: 0, net: 0 })
  const pct = (a, b) => (b ? Math.round(((a - b) * 100) / b) : null)
  const expDelta = $derived(pct(now.expense, prev.expense))
  const incDelta = $derived(pct(now.income, prev.income))

  // Kategoriler: bu ay ve geçen ay (gider).
  const catsOf = (s) => Object.fromEntries((s?.categories ?? []).filter((c) => c.currency === cur && c.type === 'expense').map((c) => [c.category, c.amount]))
  const cats = $derived.by(() => {
    const a = catsOf(months[N - 1]?.summary)
    const b = catsOf(months[N - 2]?.summary)
    return Object.entries(a)
      .map(([category, amount]) => ({ category, amount, prev: b[category] || 0 }))
      .sort((x, y) => y.amount - x.amount)
  })
  const catMax = $derived(Math.max(1, ...cats.map((c) => c.amount)))

  // Grafik geometrisi (SVG, viewBox ölçeğinde).
  const W = 320, H = 150, PAD_B = 22, BAR = 12, GAP = 2
  const peak = $derived(Math.max(1, ...series.flatMap((s) => [s.income, s.expense])))
  const y = (v) => (H - PAD_B) * (1 - v / peak)
  const colW = W / N
  const sel = $derived(series[selected])

  // Üst kenarı yuvarlak, tabana oturan çubuk.
  function barPath(x, top, w) {
    const base = H - PAD_B
    const h = base - top
    if (h <= 0) return ''
    const r = Math.min(4, h, w / 2)
    return `M${x},${base}V${top + r}Q${x},${top} ${x + r},${top}H${x + w - r}Q${x + w},${top} ${x + w},${top + r}V${base}Z`
  }
</script>

<div class="page">
  <header class="topbar">
    <button type="button" class="icon-btn" aria-label={t('nav.back')} onclick={() => back('/')}><Icon d={I.back} size={20} stroke={2} /></button>
    <h1 class="title" style="flex: 1">{t('rep.title')}</h1>
  </header>
  <div class="rule"></div>

  <div class="chips" style="margin: 0">
    <button type="button" class="chip small" onclick={() => (picking = true)}><Icon d={I.calendar} size={15} />{monthLabel(app.month)}<Icon d={I.down} size={13} /></button>
    {#if currencies.length > 1}
      {#each currencies as c}
        <button type="button" class="chip small" aria-pressed={cur === c} onclick={() => (currency = c)}>{c}</button>
      {/each}
    {/if}
  </div>

  <section class="kpis">
    <div class="kpi">
      <span class="k">{t('common.income')}</span>
      <span class="v num pos">+{fmtc(now.income, cur)}</span>
      {#if incDelta !== null}<span class="d">{t('rep.vs_prev', { v: (incDelta > 0 ? '+' : '') + incDelta + '%' })}</span>{/if}
    </div>
    <div class="kpi">
      <span class="k">{t('common.expense')}</span>
      <span class="v num">−{fmtc(now.expense, cur)}</span>
      {#if expDelta !== null}<span class="d" class:up={expDelta > 0}>{t('rep.vs_prev', { v: (expDelta > 0 ? '+' : '') + expDelta + '%' })}</span>{/if}
    </div>
    <div class="kpi wide">
      <span class="k">{t('rep.net')}</span>
      <span class="v num" class:pos={now.net > 0} class:neg={now.net < 0}>{fmtc(now.net, cur, 'always')}</span>
      <span class="d">{now.income ? t('rep.saved_pct', { v: Math.round((now.net * 100) / now.income) }) : ''}</span>
    </div>
  </section>

  <section class="section">
    <h2 class="h-section">{t('rep.trend', { n: N })}</h2>
    <div class="legend">
      <span><i class="sw inc"></i>{t('common.income')}</span>
      <span><i class="sw exp"></i>{t('common.expense')}</span>
    </div>
    {#if sel}
      <p class="readout num">
        <b>{monthName(sel.month)}</b> · +{fmt(sel.income)} / −{fmt(sel.expense)} {symbol(cur)}
      </p>
    {/if}
    <svg viewBox="0 0 {W} {H}" class="chart" role="img" aria-label={t('rep.trend', { n: N })}>
      <line x1="0" x2={W} y1={H - PAD_B} y2={H - PAD_B} class="base" />
      {#each series as s, i}
        {@const cx = colW * i + colW / 2}
        <g class="col" class:on={i === selected}>
          <path class="inc" d={barPath(cx - BAR - GAP / 2, y(s.income), BAR)} />
          <path class="exp" d={barPath(cx + GAP / 2, y(s.expense), BAR)} />
          <text x={cx} y={H - 6} text-anchor="middle" class="lbl">{monthName(s.month).slice(0, 3)}</text>
          <!-- Dokunma alanı çubuktan geniş -->
          <rect x={colW * i} y="0" width={colW} height={H} fill="transparent" role="button" tabindex="0"
            aria-label="{monthName(s.month)}: +{fmt(s.income)} / −{fmt(s.expense)} {symbol(cur)}"
            onclick={() => (selected = i)} onkeydown={(e) => e.key === 'Enter' && (selected = i)} onpointerenter={() => (selected = i)} />
        </g>
      {/each}
    </svg>
    <table class="tbl num">
      <thead><tr><th></th><th>{t('common.income')}</th><th>{t('common.expense')}</th><th>{t('rep.net')}</th></tr></thead>
      <tbody>
        {#each [...series].reverse() as s}
          <tr><td>{monthName(s.month).slice(0, 3)}</td><td>{fmt(s.income)}</td><td>{fmt(s.expense)}</td><td class:pos={s.net > 0} class:neg={s.net < 0}>{fmt(s.net, 'always')}</td></tr>
        {/each}
      </tbody>
    </table>
  </section>

  <section class="section">
    <h2 class="h-section">{t('rep.categories')}</h2>
    {#each cats as c (c.category)}
      {@const info = catInfo({ category: c.category, type: 'expense' })}
      {@const d = pct(c.amount, c.prev)}
      <div class="cat">
        <span class="ico" style="background: {info.tint}; color: #1c1a17"><Icon d={info.icon} size={16} /></span>
        <span class="grow">
          <span class="top"><span>{t('cat.' + c.category)}</span><span class="num">{fmtc(c.amount, cur)}</span></span>
          <span class="bar"><span style="width: {(c.amount * 100) / catMax}%"></span></span>
          <span class="d" class:up={d > 0}>{c.prev ? t('rep.vs_prev', { v: (d > 0 ? '+' : '') + d + '%' }) : t('rep.new_cat')}</span>
        </span>
      </div>
    {:else}
      <p class="empty">{t('common.empty')}</p>
    {/each}
  </section>
</div>

{#if picking}<MonthPicker onclose={() => (picking = false)} />{/if}

<style>
  .kpis {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .kpi {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 14px;
    border-radius: var(--r-card, 16px);
    background: var(--surface);
    border: 1px solid var(--line);
  }
  .kpi.wide {
    grid-column: 1 / -1;
  }
  .kpi .k,
  .d {
    font-size: 12px;
    color: var(--muted);
  }
  .kpi .v {
    font-size: 20px;
    font-weight: 700;
  }
  .d.up {
    color: var(--expense);
    font-weight: 600;
  }
  .legend {
    display: flex;
    gap: 14px;
    font-size: 13px;
    color: var(--muted);
  }
  .legend span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .sw {
    width: 10px;
    height: 10px;
    border-radius: 3px;
  }
  .sw.inc,
  .chart .inc {
    background: var(--income);
    fill: var(--income);
  }
  .sw.exp,
  .chart .exp {
    background: var(--expense);
    fill: var(--expense);
  }
  .readout {
    font-size: 14px;
  }
  .chart {
    width: 100%;
    height: auto;
    display: block;
  }
  .chart .base {
    stroke: var(--line2);
    stroke-width: 1;
  }
  .chart .lbl {
    font-size: 10px;
    fill: var(--muted);
  }
  .chart .col:not(.on) path {
    opacity: 0.45;
  }
  .chart .col.on .lbl {
    fill: var(--fg);
    font-weight: 700;
  }
  .chart rect {
    cursor: pointer;
    outline: none;
  }
  .tbl {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  .tbl th {
    font-weight: 500;
    color: var(--muted);
    text-align: right;
    padding: 4px 0;
  }
  .tbl td {
    text-align: right;
    padding: 5px 0;
    border-top: 1px solid var(--line);
  }
  .tbl td:first-child,
  .tbl th:first-child {
    text-align: left;
  }
  .cat {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .cat .ico {
    flex: 0 0 34px;
    width: 34px;
    height: 34px;
  }
  .cat .grow {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .cat .top {
    display: flex;
    justify-content: space-between;
    font-size: 14px;
    font-weight: 600;
  }
</style>
