// Canlı güncelleme: sunucu bu kullanıcıya bildirim yazınca (ödeme, davet,
// ortak harcama…) "inbox" olayı gelir; gelen kutusu ve veriler tazelenir.
// Bağlantı koparsa EventSource kendisi yeniden bağlanır (retry: 5 sn).
let es = null
let timer = null

export function startLive(onChange) {
  stopLive()
  if (typeof EventSource === 'undefined') return
  es = new EventSource('/api/events')
  es.addEventListener('inbox', () => {
    // Art arda gelen olaylar tek tazelemede birleşir.
    clearTimeout(timer)
    timer = setTimeout(onChange, 300)
  })
}

export function stopLive() {
  clearTimeout(timer)
  es?.close()
  es = null
}
