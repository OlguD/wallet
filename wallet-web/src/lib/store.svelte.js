// Uygulama durumu: oturum, hesaplar, gruplar, seçili ay ve aylık özet.
import { api } from './api.js'
import { convert } from './fx.js'
import { setSyncedHandler } from './offline.svelte.js'
import { t } from './i18n.js'

const firstOfMonth = (d = new Date()) => new Date(d.getFullYear(), d.getMonth(), 1)
const ymd = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

export const app = $state({
  ready: false,
  user: null,
  accounts: [],
  groups: [],
  month: firstOfMonth(),
  summary: null,
  rates: null, // Sun Döviz kurları (GET /rates)
  inbox: [], // işlenmeyi bekleyen dekontlar (Kestirme / paylaş)
  invites: [], // bekleyen grup davetleri
  notifications: { items: [], unread: 0 },
  tour: null, // gösterilen tanıtım turunun kimliği
  // Veri değişince artar; sayfalar $effect ile izleyip yeniden yükler.
  version: 0,
  sheet: null, // { kind: 'add' | 'edit', tx?, preset? }
  toast: null,
})

export function monthRange(m = app.month) {
  const to = new Date(m.getFullYear(), m.getMonth() + 1, 1)
  return { from: ymd(m), to: ymd(to) }
}

export function shiftMonth(delta) {
  app.month = new Date(app.month.getFullYear(), app.month.getMonth() + delta, 1)
}
export const setMonth = (d) => (app.month = firstOfMonth(d))
export const isCurrentMonth = () => app.month.getTime() === firstOfMonth().getTime()

export async function loadSession() {
  try {
    app.user = await api.get('/me')
    await refresh()
  } catch {
    app.user = null
  } finally {
    app.ready = true
  }
}

export async function refresh() {
  if (!app.user) return
  // Çevrimdışıyken bazıları önbellekte olmayabilir; olanlar yine güncellenir.
  const [accounts, groups, summary] = await Promise.allSettled([
    api.get('/accounts'),
    api.get('/groups'),
    api.get('/summary', monthRange()),
  ])
  if (accounts.status === 'fulfilled') app.accounts = accounts.value
  if (groups.status === 'fulfilled') app.groups = groups.value
  if (summary.status === 'fulfilled') app.summary = summary.value
  app.version++
  loadRates()
  loadInbox()
}

// Gelen kutusu: bekleyen dekontlar, grup davetleri ve bildirimler.
export function loadInbox() {
  return Promise.all([
    api.get('/receipts', { status: 'pending' }).then((r) => (app.inbox = r)),
    api.get('/invites').then((r) => (app.invites = r)),
    api.get('/notifications').then((r) => (app.notifications = r)),
  ]).catch(() => {})
}

/** Gelen kutusu rozeti: okunmamış bildirim + bekleyen dekont + davet. */
export const inboxCount = () => app.notifications.unread + app.inbox.length

export const openTransfer = (preset = {}) => (app.sheet = { kind: 'transfer', preset })

// Kurlar ayrı yüklenir; kaynak erişilemezse uygulama yine çalışır.
export function loadRates() {
  return api.get('/rates').then((r) => (app.rates = r)).catch(() => {})
}

export async function refreshSummary() {
  if (!app.user) return
  app.summary = await api.get('/summary', monthRange())
}

export async function signedIn(user) {
  app.user = user
  // Giriş yanıtında tur/anahtar bilgisi yok; tam profili al.
  app.user = await api.get('/me').catch(() => user)
  return refresh()
}

export function signedOut() {
  app.user = null
  app.accounts = []
  app.groups = []
  app.summary = null
  app.sheet = null
  app.inbox = []
  app.invites = []
  app.notifications = { items: [], unread: 0 }
}

let toastTimer
export function toast(text, kind = 'ok') {
  app.toast = { text, kind, id: Date.now() }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (app.toast = null), 2600)
}

export const openAdd = (preset = {}) => (app.sheet = { kind: 'add', preset })
export const openEdit = (tx) => (app.sheet = tx.transfer_peer_id ? { kind: 'transfer', tx } : { kind: 'edit', tx })
export const closeSheet = () => (app.sheet = null)

/** Toplam bakiye: para birimine göre; ana para birimi TRY (yoksa ilk hesabınki). */
export function totals() {
  const by = {}
  for (const a of app.accounts) by[a.currency] = (by[a.currency] || 0) + a.balance
  const main = by.TRY !== undefined ? 'TRY' : app.accounts[0]?.currency || 'TRY'
  const others = Object.entries(by).filter(([c]) => c !== main)
  // Kurlar varsa dövizli hesaplar ana para birimine (alış kuruyla) çevrilip toplama eklenir.
  let amount = by[main] || 0
  let converted = false
  if (others.length && app.rates) {
    const conv = others.map(([c, v]) => convert(v, c, main))
    if (conv.every((v) => v !== null)) {
      amount += conv.reduce((a, v) => a + v, 0)
      converted = true
    }
  }
  return { main, amount, others, converted }
}

export function monthTotals() {
  const tot = app.summary?.totals || []
  const main = totals().main
  return tot.find((x) => x.currency === main) || { currency: main, income: 0, expense: 0, net: 0 }
}

export const accountMonth = (id) => app.summary?.accounts?.find((a) => a.account_id === id) || { income: 0, expense: 0 }
export const accountIndex = (id) => app.accounts.findIndex((a) => a.id === id)

// Kuyruktaki değişiklikler sunucuya ulaşınca verileri tazele.
setSyncedHandler((n) => {
  toast(t('off.synced', { n }))
  refresh().catch(() => {})
})

