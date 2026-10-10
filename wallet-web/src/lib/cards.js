// Kredi kartı hesapları: borç negatif bakiyedir; kalan limit = limit − borç.

/** Kartın güncel borcu (kuruş, ≥ 0). */
export const cardDebt = (a) => Math.max(0, -(a?.balance || 0))

/** Kalan (kullanılabilir) limit; limit yoksa null. Fazla ödeme limite eklenir. */
export const cardAvailable = (a) => (a?.credit_limit ? a.credit_limit + (a.balance || 0) : null)

/** Limit kullanım yüzdesi (0–100); limit yoksa null. */
export const cardUsedPct = (a) => (a?.credit_limit ? Math.min(100, Math.round((cardDebt(a) * 100) / a.credit_limit)) : null)

/** Sıradaki son ödeme tarihi (bugün dahil); kısa aylarda ay sonuna çekilir. */
export function nextDue(dueDay, from = new Date()) {
  if (!dueDay) return null
  const today = new Date(from.getFullYear(), from.getMonth(), from.getDate())
  for (let k = 0; k < 2; k++) {
    const y = today.getFullYear()
    const m = today.getMonth() + k
    const last = new Date(y, m + 1, 0).getDate()
    const d = new Date(y, m, Math.min(dueDay, last))
    if (d >= today) return d
  }
  return null
}

/** Bugünden son ödeme tarihine kalan gün. */
export const daysUntil = (d) => (d ? Math.round((d - new Date(new Date().toDateString())) / 86400000) : null)
