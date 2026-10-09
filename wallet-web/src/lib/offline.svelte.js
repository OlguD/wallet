// Çevrimdışı çalışma: okunan veriler localStorage'da önbelleklenir, değişiklik
// istekleri bağlantı yokken bir kuyruğa (outbox) yazılır ve bağlantı gelince
// sırayla sunucuya gönderilir. Her istek bir Idempotency-Key taşır; böylece
// sunucuya ulaşmış ama yanıtı kaybolmuş bir istek tekrar gönderilince iki kez
// işlenmez.

export const net = $state({
  online: typeof navigator === 'undefined' ? true : navigator.onLine,
  pending: 0, // kuyruktaki istek sayısı
  syncing: false,
  queuedTick: 0, // yeni bir istek kuyruğa girince artar (bildirim için)
  failed: [], // sunucunun reddettiği kuyruk istekleri: { label, error }
})

const OUTBOX = 'wallet.outbox'
const CACHE = 'wallet.cache:'
const INDEX = 'wallet.cache.index'
const MAX_CACHE = 60

function read(key, fallback) {
  try {
    const v = localStorage.getItem(key)
    return v ? JSON.parse(v) : fallback
  } catch {
    return fallback
  }
}

function write(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
    return true
  } catch {
    return false
  }
}

export function uuid() {
  if (crypto.randomUUID) return crypto.randomUUID().replace(/-/g, '')
  // Güvenli olmayan bağlamda (http LAN) randomUUID yok.
  const b = crypto.getRandomValues(new Uint8Array(16))
  return Array.from(b, (x) => x.toString(16).padStart(2, '0')).join('')
}

// ── GET önbelleği ──────────────────────────────────────────────

export function cacheGet(path) {
  return read(CACHE + path, null)?.data ?? null
}

export function cacheSet(path, data) {
  const index = read(INDEX, []).filter((k) => k !== path)
  index.push(path)
  // En eskileri at; kota dolarsa birkaç tane daha at ve yeniden dene.
  while (index.length > MAX_CACHE) localStorage.removeItem(CACHE + index.shift())
  for (let i = 0; i < 5; i++) {
    if (write(CACHE + path, { data, at: Date.now() })) break
    const old = index.shift()
    if (!old || old === path) break
    localStorage.removeItem(CACHE + old)
  }
  write(INDEX, index)
}

// ── Kuyruk ─────────────────────────────────────────────────────

export const outbox = () => read(OUTBOX, [])

function saveOutbox(list) {
  write(OUTBOX, list)
  net.pending = list.length
}
net.pending = outbox().length

// Kuyruğa alınabilen istekler (diğerleri — giriş, dekont yükleme, davet — internet ister).
const QUEUEABLE = [
  ['POST', /^\/transactions$/],
  ['PATCH', /^\/transactions\/-?\d+$/],
  ['DELETE', /^\/transactions\/-?\d+$/],
  ['POST', /^\/transfers$/],
  ['POST', /^\/goals\/\d+\/contributions$/],
  ['POST', /^\/recurring$/],
  ['PATCH', /^\/recurring\/\d+$/],
  ['POST', /^\/budgets$/],
  ['PATCH', /^\/budgets\/\d+$/],
  ['DELETE', /^\/budgets\/\d+$/],
  ['PATCH', /^\/receipts\/\d+$/],
  ['POST', /^\/notifications\/read$/],
  ['POST', /^\/me\/tours$/],
]

export const queueable = (method, path) => QUEUEABLE.some(([m, re]) => m === method && re.test(path))

let tempSeq = 0
const tempId = () => -(Date.now() * 10 + (tempSeq++ % 10))

/**
 * İsteği kuyruğa ekler ve iyimser bir sonuç döner. Henüz gönderilmemiş
 * (geçici id'li) bir işlem düzenlenir/silinirse kuyruktaki oluşturma isteği
 * güncellenir ya da kaldırılır.
 */
export function enqueue(method, path, body, key = uuid()) {
  const list = outbox()
  const tmp = path.match(/^\/transactions\/(-\d+)$/)
  if (tmp) {
    const id = Number(tmp[1])
    const i = list.findIndex((x) => x.tempId === id)
    if (i >= 0) {
      if (method === 'DELETE') list.splice(i, 1)
      else list[i].body = { ...list[i].body, ...body }
      saveOutbox(list)
      net.queuedTick++
      return method === 'DELETE' ? null : pendingTx(list[i])
    }
  }
  const item = { key, method, path, body: body ?? null, at: Date.now() }
  if (method === 'POST' && (path === '/transactions' || path === '/transfers')) {
    // Kayıt, sunucuya ulaştığı an değil girildiği an tarihlenir.
    item.body = { ...body, occurred_at: body?.occurred_at || new Date().toISOString() }
    item.tempId = tempId()
  }
  list.push(item)
  saveOutbox(list)
  net.queuedTick++
  return item.tempId ? pendingTx(item) : { queued: true }
}

// Kuyruktaki oluşturma isteğinden listede gösterilecek "bekliyor" işlemi.
function pendingTx(item) {
  const b = item.body || {}
  const accounts = cacheGet('/accounts') || []
  const me = cacheGet('/me')
  const acc = (id) => accounts.find((a) => a.id === id)
  if (item.path === '/transfers') {
    return {
      id: item.tempId, pending: true, type: 'expense', amount: b.amount, account_id: b.from_account_id,
      account_name: acc(b.from_account_id)?.name ?? '', currency: acc(b.from_account_id)?.currency,
      transfer_peer_id: item.tempId - 1, transfer_account_id: b.to_account_id, transfer_account_name: acc(b.to_account_id)?.name ?? '',
      description: b.description ?? null, occurred_at: b.occurred_at, user_id: me?.id, username: me?.username,
    }
  }
  return {
    id: item.tempId, pending: true, account_id: b.account_id, group_id: b.group_id ?? null, type: b.type, amount: b.amount,
    category: b.category ?? null, description: b.description ?? null, occurred_at: b.occurred_at,
    counterparty_name: b.counterparty_name ?? null, account_name: acc(b.account_id)?.name ?? '',
    currency: acc(b.account_id)?.currency, user_id: me?.id, username: me?.username, receipt_id: b.receipt_id ?? null,
  }
}

const deletedIds = (list) =>
  new Set(list.filter((x) => x.method === 'DELETE' && /^\/transactions\/\d+$/.test(x.path)).map((x) => Number(x.path.split('/').pop())))

/**
 * Önbellekten ya da ağdan gelen GET yanıtına kuyruktaki değişiklikleri uygular:
 * bekleyen işlemler listelere eklenir, silinenler gizlenir, bakiyeler güncellenir.
 */
export function withPending(path, data) {
  const list = outbox()
  if (!list.length || data == null) return data
  const [base, query = ''] = path.split('?')
  const params = new URLSearchParams(query)

  if (base === '/accounts' && Array.isArray(data)) {
    const delta = {}
    const add = (id, v) => (delta[id] = (delta[id] || 0) + v)
    for (const x of list) {
      if (x.method !== 'POST' || !x.body) continue
      if (x.path === '/transactions') add(x.body.account_id, x.body.type === 'expense' ? -x.body.amount : x.body.amount)
      if (x.path === '/transfers') {
        add(x.body.from_account_id, -x.body.amount)
        add(x.body.to_account_id, x.body.to_amount ?? x.body.amount)
      }
    }
    return data.map((a) => (delta[a.id] ? { ...a, balance: a.balance + delta[a.id] } : a))
  }

  const accMatch = base.match(/^\/accounts\/(\d+)\/transactions$/)
  if ((base === '/transactions' || accMatch) && Array.isArray(data) && !params.get('offset') && !params.get('q') && !params.get('counterparty')) {
    const accountId = accMatch ? Number(accMatch[1]) : params.get('account_id') ? Number(params.get('account_id')) : null
    const groupId = params.get('group_id') ? Number(params.get('group_id')) : null
    const gone = deletedIds(list)
    const pending = list
      .filter((x) => x.tempId)
      .map(pendingTx)
      .filter((tx) => (accountId === null || tx.account_id === accountId) && (groupId === null || tx.group_id === groupId))
      .reverse()
    return [...pending, ...data.filter((tx) => !gone.has(tx.id))]
  }

  if (base === '/goals' && Array.isArray(data)) {
    const add = {}
    for (const x of list) {
      const m = x.method === 'POST' && x.path.match(/^\/goals\/(\d+)\/contributions$/)
      if (m) add[m[1]] = (add[m[1]] || 0) + (x.body?.amount || 0)
    }
    return data.map((g) => {
      if (!add[g.id]) return g
      const current = g.current + add[g.id]
      return { ...g, current, remaining: Math.max(0, g.target_amount - current), progress_pct: Math.min(100, Math.floor((Math.max(0, current) * 100) / g.target_amount)) }
    })
  }
  return data
}

// ── Gönderim ───────────────────────────────────────────────────

let onSynced = () => {}
export const setSyncedHandler = (fn) => (onSynced = fn)

const labelOf = (x) => `${x.method} ${x.path}`

/** Kuyruğu sırayla gönderir. Ağ hatasında durur, sonra tekrar denenir. */
export async function flush() {
  if (net.syncing || !outbox().length) return
  net.syncing = true
  let sent = 0
  try {
    for (;;) {
      const list = outbox()
      const x = list[0]
      if (!x) break
      let res
      try {
        res = await fetch('/api' + x.path, {
          method: x.method,
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json', 'Idempotency-Key': x.key },
          body: x.body == null ? undefined : JSON.stringify(x.body),
        })
      } catch {
        net.online = false
        break
      }
      net.online = true
      // 401: oturum yok (tekrar giriş sonrası denenecek); 409/5xx: daha sonra tekrar.
      if (res.status === 401 || res.status === 409 || res.status >= 500) break
      if (!res.ok) {
        let msg = ''
        try {
          msg = (await res.json()).error || ''
        } catch {}
        net.failed = [...net.failed, { label: labelOf(x), error: msg, body: x.body }]
      }
      saveOutbox(outbox().filter((y) => y.key !== x.key))
      sent++
    }
  } finally {
    net.syncing = false
  }
  if (sent) onSynced(sent)
}

export function clearOffline() {
  try {
    for (const k of read(INDEX, [])) localStorage.removeItem(CACHE + k)
    localStorage.removeItem(INDEX)
    localStorage.removeItem(OUTBOX)
  } catch {}
  net.pending = 0
  net.failed = []
}

if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    net.online = true
    flush()
  })
  window.addEventListener('offline', () => (net.online = false))
  // Kuyrukta bekleyen varsa periyodik dene (online olayı her zaman gelmez).
  setInterval(() => net.pending && flush(), 30000)
}
