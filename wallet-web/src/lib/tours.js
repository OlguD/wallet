// Tanıtım turları. Her kullanıcı her turu bir kez görür (sunucuda users.seen_tours).
//
// YENİ ÖZELLİK EKLERKEN: bu listeye yeni (benzersiz kimlikli) bir tur ekle ve
// özelliğin arayüzdeki öğesine data-tour="..." ver. Mevcut kullanıcılar sadece
// görmedikleri yeni turu görür. Yeni kullanıcılar "welcome"u görür; welcome
// bitince o ana kadarki tüm turlar görüldü sayılır (welcome hepsini anlatır),
// bu yüzden yeni özelliğin adımını welcome'a da ekle.
//
// Adım: { target?: 'data-tour değeri', title, body } — hedef yoksa ortada kart.
// route: turun gösterileceği sayfa ('home' varsayılan; alt menü orada görünür).

export const TOURS = [
  {
    id: 'welcome',
    steps: [
      { title: 'tour.welcome.title', body: 'tour.welcome.body' },
      { target: 'total', title: 'tour.total.title', body: 'tour.total.body' },
      { target: 'nav-add', title: 'tour.add.title', body: 'tour.add.body' },
      { target: 'nav-add', title: 'tour.receipt.title', body: 'tour.receipt.body' },
      { target: 'settings', title: 'tour.share.title', body: 'tour.share.body' },
      { target: 'goals', title: 'tour.goals.title', body: 'tour.goals.body' },
      { target: 'payees', title: 'tour.payees.title', body: 'tour.payees.body' },
      { target: 'rates', title: 'tour.rates.title', body: 'tour.rates.body' },
      { target: 'recurring', title: 'tour.recurring.title', body: 'tour.recurring.body' },
      { target: 'nav-transactions', title: 'tour.tx.title', body: 'tour.tx.body' },
      { target: 'nav-accounts', title: 'tour.accounts.title', body: 'tour.accounts.body' },
      { target: 'nav-groups', title: 'tour.groups.title', body: 'tour.groups.body' },
      { title: 'tour.done.title', body: 'tour.done.body' },
    ],
  },
  // Örnek (yeni özellik): { id: 'budgets-2026-11', steps: [{ target: 'budgets', title: '...', body: '...' }] },
]

/** Kullanıcının henüz görmediği ilk tur; yoksa null. */
export function pendingTour(seen = []) {
  if (!seen.includes('welcome')) return 'welcome'
  return TOURS.find((t) => !seen.includes(t.id))?.id ?? null
}

/** Bir tur bitince görüldü sayılacak kimlikler. */
export const idsCoveredBy = (id) => (id === 'welcome' ? TOURS.map((t) => t.id) : [id])
