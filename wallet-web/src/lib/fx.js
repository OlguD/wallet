// Döviz çevirme (Sun Döviz kurları). Mantık sundoviz.com hesaplayıcısıyla aynı:
//  döviz → TL: alış, TL → döviz: satış,
//  döviz → döviz: cross_mode ise çapraz kur, değilse TL üzerinden (alış / satış).
import { app } from './store.svelte.js'
import { prefs } from './prefs.svelte.js'

const channelOf = (rates, ch) => (ch === 'desk' ? rates.desk : rates.online)

/** 1 birim `from` kaç birim `to` eder; kur yoksa null. */
export function rate(from, to, ch = 'online', rates = app.rates) {
  if (from === to) return 1
  if (!rates) return null
  const c = channelOf(rates, ch)
  const r = c.rates
  if (to === 'TRY') return r[from]?.buy ?? null
  if (from === 'TRY') return r[to] ? 1 / r[to].sell : null
  if (rates.cross_mode) {
    const direct = c.cross[`${from}/${to}`]
    if (direct) return direct.buy
    const rev = c.cross[`${to}/${from}`]
    if (rev) return 1 / rev.sell
    return null
  }
  return r[from] && r[to] ? r[from].buy / r[to].sell : null
}

/** Kuruş cinsinden tutarı çevirir; kur yoksa null. */
export function convert(kurus, from, to, ch = 'online', rates = app.rates) {
  const k = rate(from, to, ch, rates)
  return k === null ? null : Math.round(kurus * k)
}

/** Kurun gösterimi: 4 hane. */
export const fmtRate = (v) =>
  v == null ? '—' : new Intl.NumberFormat(prefs.lang === 'en' ? 'en-US' : 'tr-TR', { minimumFractionDigits: 2, maximumFractionDigits: 4 }).format(v)
