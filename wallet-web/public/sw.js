// Uygulama kabuğunu önbelleğe alır; API istekleri her zaman ağdan gider.
const CACHE = 'cuzdan-v4'
// Android paylaş menüsünden gelen dekont, uygulama açılana kadar burada bekler.
const SHARE_CACHE = 'cuzdan-share'
const SHELL = ['/', '/manifest.webmanifest', '/icons/icon-192.png']

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()))
})

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE && k !== SHARE_CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  )
})

// Web Share Target (Android): dosyayı sakla, uygulamayı ?shared=1 ile aç.
async function receiveShare(req) {
  try {
    const form = await req.formData()
    const file = form.getAll('file').find((f) => f && typeof f !== 'string')
    if (file) {
      const cache = await caches.open(SHARE_CACHE)
      await cache.put('/__shared-receipt', new Response(file, {
        headers: { 'Content-Type': file.type || 'application/octet-stream', 'X-File-Name': encodeURIComponent(file.name || 'dekont') },
      }))
    }
  } catch (err) {
    // okunamazsa uygulama yine açılır
  }
  return Response.redirect('/?shared=1', 303)
}

self.addEventListener('fetch', (e) => {
  const req = e.request
  const url = new URL(req.url)
  if (req.method === 'POST' && url.origin === location.origin && url.pathname === '/share-target') {
    e.respondWith(receiveShare(req))
    return
  }
  if (req.method !== 'GET' || url.origin !== location.origin || url.pathname.startsWith('/api/')) return

  // Sayfa gezinmesi: önce ağ, çevrimdışıysa önbellekteki kabuk.
  if (req.mode === 'navigate') {
    e.respondWith(
      fetch(req)
        .then((res) => {
          const copy = res.clone()
          caches.open(CACHE).then((c) => c.put('/', copy))
          return res
        })
        .catch(() => caches.match('/')),
    )
    return
  }

  // Statik dosyalar (hash'li JS/CSS, font, ikon): önbellek, arkada güncelle.
  e.respondWith(
    caches.match(req).then((hit) => {
      const net = fetch(req).then((res) => {
        if (res.ok) {
          const copy = res.clone()
          caches.open(CACHE).then((c) => c.put(req, copy))
        }
        return res
      })
      return hit || net
    }),
  )
})

// Web Push: bildirimi göster; dokununca ilgili sayfayı aç (uygulama açıksa ona geç).
self.addEventListener('push', (e) => {
  let d = {}
  try {
    d = e.data ? e.data.json() : {}
  } catch (err) {
    d = { title: e.data ? e.data.text() : 'Cüzdan' }
  }
  e.waitUntil(
    self.registration.showNotification(d.title || 'Cüzdan', {
      body: d.body || '',
      icon: '/icons/icon-192.png',
      badge: '/icons/icon-192.png',
      tag: d.tag,
      data: { url: d.url || '/inbox' },
    }),
  )
})

self.addEventListener('notificationclick', (e) => {
  e.notification.close()
  const url = new URL(e.notification.data?.url || '/inbox', self.location.origin).href
  e.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((list) => {
      for (const c of list) {
        if ('focus' in c) {
          c.navigate?.(url)
          return c.focus()
        }
      }
      return self.clients.openWindow(url)
    }),
  )
})

