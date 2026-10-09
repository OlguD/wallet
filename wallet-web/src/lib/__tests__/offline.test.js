// Çevrimdışı kuyruk: önbellek, kuyruğa alma, bekleyenlerin listelere/bakiyelere
// yansıması ve bağlantı gelince Idempotency-Key ile sırayla gönderim.
import { describe, it, expect, beforeEach, vi } from 'vitest'

function memoryStorage() {
  const m = new Map()
  return {
    getItem: (k) => (m.has(k) ? m.get(k) : null),
    setItem: (k, v) => m.set(k, String(v)),
    removeItem: (k) => m.delete(k),
    clear: () => m.clear(),
  }
}

let off
beforeEach(async () => {
  vi.resetModules()
  globalThis.localStorage = memoryStorage()
  globalThis.window = globalThis
  globalThis.addEventListener = () => {}
  Object.defineProperty(globalThis, 'navigator', { value: { onLine: false }, configurable: true })
  off = await import('../offline.svelte.js')
  off.cacheSet('/me', { id: 7, username: 'olgu' })
  off.cacheSet('/accounts', [
    { id: 1, name: 'Banka', currency: 'TRY', balance: 100000 },
    { id: 2, name: 'Nakit', currency: 'TRY', balance: 0 },
  ])
})

describe('offline queue', () => {
  it('kuyruğa alınan işlem listede ve bakiyede görünür', () => {
    const tx = off.enqueue('POST', '/transactions', { account_id: 1, type: 'expense', amount: 2500, description: 'Simit' })
    expect(tx.pending).toBe(true)
    expect(tx.id).toBeLessThan(0)
    expect(tx.account_name).toBe('Banka')
    expect(tx.occurred_at).toBeTruthy() // girildiği an tarihlenir
    expect(off.net.pending).toBe(1)

    const accounts = off.withPending('/accounts', off.cacheGet('/accounts'))
    expect(accounts.find((a) => a.id === 1).balance).toBe(97500)

    const list = off.withPending('/transactions?limit=5', [{ id: 10, account_id: 1 }])
    expect(list.map((x) => x.id)).toEqual([tx.id, 10])
    // Başka hesabın listesinde görünmez.
    expect(off.withPending('/accounts/2/transactions?limit=50', [])).toEqual([])
  })

  it('transfer iki hesabın bakiyesini değiştirir', () => {
    off.enqueue('POST', '/transfers', { from_account_id: 1, to_account_id: 2, amount: 30000 })
    const accounts = off.withPending('/accounts', off.cacheGet('/accounts'))
    expect(accounts.map((a) => a.balance)).toEqual([70000, 30000])
  })

  it('gönderilmemiş işlem düzenlenirse kuyruktaki istek güncellenir, silinirse kaldırılır', () => {
    const tx = off.enqueue('POST', '/transactions', { account_id: 1, type: 'expense', amount: 100 })
    off.enqueue('PATCH', `/transactions/${tx.id}`, { amount: 900 })
    expect(off.outbox()).toHaveLength(1)
    expect(off.outbox()[0].body.amount).toBe(900)
    off.enqueue('DELETE', `/transactions/${tx.id}`)
    expect(off.outbox()).toHaveLength(0)
  })

  it('kuyruktaki silme, sunucudan gelen listeden gizlenir', () => {
    off.enqueue('DELETE', '/transactions/10')
    expect(off.withPending('/transactions', [{ id: 10 }, { id: 11 }]).map((x) => x.id)).toEqual([11])
  })

  it('hedef katkısı ilerlemeye yansır', () => {
    off.enqueue('POST', '/goals/3/contributions', { amount: 5000 })
    const [g] = off.withPending('/goals', [{ id: 3, current: 5000, target_amount: 20000, remaining: 15000, progress_pct: 25 }])
    expect(g.current).toBe(10000)
    expect(g.progress_pct).toBe(50)
  })

  it('sadece güvenli istekler kuyruğa alınabilir', () => {
    expect(off.queueable('POST', '/transactions')).toBe(true)
    expect(off.queueable('POST', '/auth/login')).toBe(false)
    expect(off.queueable('POST', '/receipts')).toBe(false)
    expect(off.queueable('POST', '/groups/1/invites')).toBe(false)
  })

  it('flush sırayla ve Idempotency-Key ile gönderir; ağ hatasında durur, reddedileni bildirir', async () => {
    off.enqueue('POST', '/transactions', { account_id: 1, type: 'expense', amount: 1 })
    off.enqueue('POST', '/budgets', { category: 'food', amount: 5 })
    off.enqueue('POST', '/transactions', { account_id: 1, type: 'income', amount: 2 })
    const keys = off.outbox().map((x) => x.key)

    const calls = []
    // 1. istek başarılı, 2. sunucu reddediyor (400), 3. ağ hatası.
    globalThis.fetch = vi.fn(async (url, init) => {
      calls.push({ url, key: init.headers['Idempotency-Key'] })
      if (calls.length === 1) return new Response('{}', { status: 201 })
      if (calls.length === 2) return new Response('{"error":"bu kategori icin zaten butce var"}', { status: 400 })
      throw new TypeError('network')
    })
    const synced = vi.fn()
    off.setSyncedHandler(synced)
    await off.flush()

    expect(calls.map((c) => c.key)).toEqual(keys)
    expect(off.outbox()).toHaveLength(1) // ağ hatasındaki bekliyor
    expect(off.net.failed[0].error).toBe('bu kategori icin zaten butce var')
    expect(off.net.online).toBe(false)
    expect(synced).toHaveBeenCalledWith(2)

    // Bağlantı geldi: kalan aynı anahtarla gönderilir.
    globalThis.fetch = vi.fn(async (url, init) => {
      expect(init.headers['Idempotency-Key']).toBe(keys[2])
      return new Response('{}', { status: 201 })
    })
    await off.flush()
    expect(off.outbox()).toHaveLength(0)
    expect(off.net.pending).toBe(0)
  })

  it('409 (istek işleniyor) ve 5xx sonrası durur, isteği atmaz', async () => {
    off.enqueue('POST', '/transactions', { account_id: 1, type: 'expense', amount: 1 })
    globalThis.fetch = vi.fn(async () => new Response('{}', { status: 503 }))
    await off.flush()
    expect(off.outbox()).toHaveLength(1)
  })
})
