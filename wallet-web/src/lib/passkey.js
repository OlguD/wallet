// Face ID / Touch ID ile giriş (WebAuthn passkey). Sunucunun seçenekleri
// base64url metin taşır; tarayıcı API'si ArrayBuffer ister (ve döner).
import { api } from './api.js'

export const passkeySupported = () => typeof window !== 'undefined' && !!window.PublicKeyCredential && !!navigator.credentials

const toBuf = (s) => {
  const b = atob(s.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (s.length % 4)) % 4))
  return Uint8Array.from(b, (c) => c.charCodeAt(0)).buffer
}
const toB64 = (buf) =>
  btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')

const descriptors = (list) => (list || []).map((c) => ({ ...c, id: toBuf(c.id) }))

/** Bu cihaza passkey ekler (Face ID sorar). name: listede görünen ad. */
export async function registerPasskey(name) {
  const { challenge_id, options } = await api.post('/passkeys/register/begin')
  const pk = options.publicKey ?? options
  const cred = await navigator.credentials.create({
    publicKey: {
      ...pk,
      challenge: toBuf(pk.challenge),
      user: { ...pk.user, id: toBuf(pk.user.id) },
      excludeCredentials: descriptors(pk.excludeCredentials),
    },
  })
  const r = cred.response
  const body = {
    id: cred.id,
    rawId: toB64(cred.rawId),
    type: cred.type,
    authenticatorAttachment: cred.authenticatorAttachment ?? undefined,
    clientExtensionResults: cred.getClientExtensionResults?.() ?? {},
    response: {
      clientDataJSON: toB64(r.clientDataJSON),
      attestationObject: toB64(r.attestationObject),
      transports: r.getTransports?.() ?? [],
    },
  }
  const q = new URLSearchParams({ challenge: challenge_id, name })
  return api.post(`/passkeys/register/finish?${q}`, body)
}

/** Kullanıcı adı sormadan Face ID ile giriş; { user, token } döner. */
export async function loginWithPasskey() {
  const { challenge_id, options } = await api.post('/auth/passkey/begin')
  const pk = options.publicKey ?? options
  const cred = await navigator.credentials.get({
    publicKey: { ...pk, challenge: toBuf(pk.challenge), allowCredentials: descriptors(pk.allowCredentials) },
  })
  const r = cred.response
  const body = {
    id: cred.id,
    rawId: toB64(cred.rawId),
    type: cred.type,
    authenticatorAttachment: cred.authenticatorAttachment ?? undefined,
    clientExtensionResults: cred.getClientExtensionResults?.() ?? {},
    response: {
      clientDataJSON: toB64(r.clientDataJSON),
      authenticatorData: toB64(r.authenticatorData),
      signature: toB64(r.signature),
      userHandle: r.userHandle ? toB64(r.userHandle) : undefined,
    },
  }
  return api.post(`/auth/passkey/finish?challenge=${encodeURIComponent(challenge_id)}`, body)
}

/** Kullanıcı Face ID penceresini kapattıysa hata göstermeye gerek yok. */
export const cancelled = (e) => e?.name === 'NotAllowedError' || e?.name === 'AbortError'
