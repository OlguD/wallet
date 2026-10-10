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
      { target: 'inbox', title: 'tour.inbox.title', body: 'tour.inbox.body' },
      { target: 'budgets', title: 'tour.budgets.title', body: 'tour.budgets.body' },
      { target: 'nav-add', title: 'tour.transfer.title', body: 'tour.transfer.body' },
      { target: 'nav-transactions', title: 'tour.search.title', body: 'tour.search.body' },
      { target: 'settings', title: 'tour.security.title', body: 'tour.security.body' },
      { title: 'tour.offline.title', body: 'tour.offline.body' },
      { target: 'nav-accounts', title: 'tour.cards.title', body: 'tour.cards.body' },
      { target: 'goals', title: 'tour.goalfx.title', body: 'tour.goalfx.body' },
      { target: 'nav-groups', title: 'tour.grouppay.title', body: 'tour.grouppay.body' },
      { target: 'push-prompt', title: 'tour.push.title', body: 'tour.push.body' },
      { title: 'tour.done.title', body: 'tour.done.body' },
    ],
  },
  // 2026-10-10: gelen kutusu + davetler, bütçeler, transfer, arama/CSV, bildirim + şifre, çevrimdışı.
  {
    id: 'update-2026-10-10',
    steps: [
      { title: 'tour.update.title', body: 'tour.update.body' },
      { target: 'inbox', title: 'tour.inbox.title', body: 'tour.inbox.body' },
      { target: 'nav-groups', title: 'tour.invite.title', body: 'tour.invite.body' },
      { target: 'budgets', title: 'tour.budgets.title', body: 'tour.budgets.body' },
      { target: 'nav-add', title: 'tour.transfer.title', body: 'tour.transfer.body' },
      { target: 'nav-transactions', title: 'tour.search.title', body: 'tour.search.body' },
      { target: 'settings', title: 'tour.security.title', body: 'tour.security.body' },
      { title: 'tour.offline.title', body: 'tour.offline.body' },
    ],
  },
  // 2026-10-10 (öğleden sonra): kredi kartı limit/borç takibi, dövizli birikim hedefi.
  {
    id: 'cards-goal-currency',
    steps: [
      { title: 'tour.update.title', body: 'tour.update.body' },
      { target: 'nav-accounts', title: 'tour.cards.title', body: 'tour.cards.body' },
      { target: 'goals', title: 'tour.goalfx.title', body: 'tour.goalfx.body' },
    ],
  },
  // 2026-10-10 (akşam): gruptakilere (dövizli) para gönderme, gelen kutusunda hesaba işleme.
  {
    id: 'group-payments',
    steps: [
      { title: 'tour.update.title', body: 'tour.update.body' },
      { target: 'nav-groups', title: 'tour.grouppay.title', body: 'tour.grouppay.body' },
      { target: 'inbox', title: 'tour.inbox.title', body: 'tour.inbox.body' },
    ],
  },
  // 2026-10-10: bildirimleri açmaya yönlendirme (ana sayfa kartı + adımlar).
  {
    id: 'push-prompt',
    steps: [{ target: 'push-prompt', title: 'tour.push.title', body: 'tour.push.body' }],
  },
]

/** Kullanıcının henüz görmediği ilk tur; yoksa null. */
export function pendingTour(seen = []) {
  if (!seen.includes('welcome')) return 'welcome'
  return TOURS.find((t) => !seen.includes(t.id))?.id ?? null
}

/** Bir tur bitince görüldü sayılacak kimlikler. */
export const idsCoveredBy = (id) => (id === 'welcome' ? TOURS.map((t) => t.id) : [id])
