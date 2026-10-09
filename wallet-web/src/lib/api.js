// Go API istemcisi. Oturum HttpOnly cookie ile taşınır (aynı origin, /api proxy).
// Bağlantı yokken GET'ler önbellekten döner, değişiklikler kuyruğa alınır (offline.svelte.js).
import { net, cacheGet, cacheSet, withPending, enqueue, queueable, uuid, flush } from './offline.svelte.js'

export class ApiError extends Error {
  constructor(status, message, data = null) {
    super(message)
    this.status = status
    this.data = data // sunucunun ek alanları (attempts_left, retry_after...)
  }
}

let onUnauthorized = () => {}
export function setUnauthorizedHandler(fn) {
  onUnauthorized = fn
}

async function send(method, path, body, key) {
  // FormData (dosya yükleme) olduğu gibi gönderilir; tarayıcı sınırı kendisi ekler.
  const form = body instanceof FormData
  const headers = {}
  if (body !== undefined && !form) headers['Content-Type'] = 'application/json'
  if (key) headers['Idempotency-Key'] = key
  return fetch('/api' + path, {
    method,
    credentials: 'same-origin',
    headers,
    body: body === undefined ? undefined : form ? body : JSON.stringify(body),
  })
}

async function request(method, path, body) {
  const read = method === 'GET'
  const canQueue = !read && !(body instanceof FormData) && queueable(method, path)
  // Bağlantı yokken kuyruğa alınabilen değişiklik doğrudan kuyruğa yazılır.
  if (canQueue && !navigator.onLine) return enqueue(method, path, body)

  const key = read ? null : uuid()
  let res
  try {
    res = await send(method, path, body, key)
  } catch {
    net.online = false
    if (read) {
      const cached = cacheGet(path)
      if (cached !== null) return withPending(path, cached)
    } else if (canQueue) {
      // Sunucuya ulaşmış olabilir: aynı anahtarla kuyruğa al, tekrar işlenmez.
      return enqueue(method, path, body, key)
    }
    throw new ApiError(0, 'network')
  }
  net.online = true
  if (res.status === 204) return null
  let data = null
  try {
    data = await res.json()
  } catch {
    /* boş gövde */
  }
  if (!res.ok) {
    if (res.status === 401 && !path.startsWith('/auth/')) onUnauthorized()
    throw new ApiError(res.status, data?.error || 'unknown', data)
  }
  if (read) {
    cacheSet(path, data)
    return withPending(path, data)
  }
  return data
}

const qs = (params) => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params || {})) if (v !== undefined && v !== null && v !== '') p.set(k, v)
  const s = p.toString()
  return s ? '?' + s : ''
}

export const api = {
  get: (path, params) => request('GET', path + qs(params)),
  // Bağlantı geldiğinde kuyruktakileri gönder.
  flush,
  post: (path, body = {}) => request('POST', path, body),
  patch: (path, body) => request('PATCH', path, body),
  del: (path, body) => request('DELETE', path, body),
  upload: (path, form) => request('POST', path, form),
}
