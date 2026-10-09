// Dekont metnini cihazda çıkarır; dosya içeriği hiçbir üçüncü servise gitmez.
//  PDF   → pdf.js (metin katmanı)
//  Resim → tesseract.js OCR (Türkçe dil verisi ilk kullanımda indirilir)
import { api } from './api.js'

export const ACCEPT = 'application/pdf,image/jpeg,image/png,image/webp,image/heic,image/heif'

const isPdf = (f) => f.type === 'application/pdf' || /\.pdf$/i.test(f.name || '')
const isOcrImage = (f) => /^image\/(jpeg|png|webp)$/.test(f.type)

// Aynı satırdaki parçaları x sırasıyla birleştirir; büyük boşluk = sütun ayracı " | ".
async function pdfText(file) {
  const pdfjs = await import('pdfjs-dist')
  const workerUrl = (await import('pdfjs-dist/build/pdf.worker.min.mjs?url')).default
  pdfjs.GlobalWorkerOptions.workerSrc = workerUrl
  const doc = await pdfjs.getDocument({ data: await file.arrayBuffer() }).promise
  const out = []
  for (let n = 1; n <= Math.min(doc.numPages, 3); n++) {
    const page = await doc.getPage(n)
    const { items } = await page.getTextContent()
    const rows = []
    for (const it of items) {
      if (!it.str?.trim()) continue
      const [, , , , x, y] = it.transform
      let row = rows.find((r) => Math.abs(r.y - y) < 3)
      if (!row) rows.push((row = { y, parts: [] }))
      row.parts.push({ x, w: it.width, s: it.str })
    }
    rows.sort((a, b) => b.y - a.y)
    for (const r of rows) {
      r.parts.sort((a, b) => a.x - b.x)
      let line = ''
      let end = null
      for (const p of r.parts) {
        if (end !== null) line += p.x - end > 24 ? ' | ' : p.x - end > 1.5 ? ' ' : ''
        line += p.s
        end = p.x + p.w
      }
      out.push(line.trim())
    }
  }
  return out.join('\n')
}

async function ocrText(file, onprogress) {
  const { createWorker } = await import('tesseract.js')
  const worker = await createWorker('tur', 1, {
    logger: (m) => m.status === 'recognizing text' && onprogress?.(m.progress),
  })
  try {
    const { data } = await worker.recognize(file)
    return data.text
  } finally {
    worker.terminate()
  }
}

/** Dosyanın metni; okunamazsa '' (dosya yine saklanır, alanlar elle doldurulur). */
export async function extractText(file, onprogress) {
  try {
    if (isPdf(file)) return await pdfText(file)
    if (isOcrImage(file)) return await ocrText(file, onprogress)
  } catch (e) {
    console.warn('dekont okunamadı', e)
  }
  return ''
}

/** Dosyayı okuyup yükler; sunucu alanları ayrıştırıp Receipt döner. */
export async function uploadReceipt(file, { source = 'app', onstage } = {}) {
  onstage?.('read')
  const text = await extractText(file, (p) => onstage?.('read', p))
  onstage?.('upload')
  const form = new FormData()
  form.append('file', file, file.name || 'dekont')
  if (text) form.append('text', text)
  return api.upload(`/receipts${source === 'share' ? '?source=share' : ''}`, form)
}

/** Gelen kutusundaki (Kestirmeyle gelmiş) dekontu cihazda okuyup ayrıştırır. */
export async function parseStored(receipt, onstage) {
  if (receipt.has_text) return receipt
  onstage?.('read')
  const res = await fetch(`/api/receipts/${receipt.id}/file`, { credentials: 'same-origin' })
  if (!res.ok) return receipt
  const blob = await res.blob()
  const file = new File([blob], receipt.original_name || 'dekont', { type: receipt.mime_type })
  const text = await extractText(file, (p) => onstage?.('read', p))
  if (!text.trim()) return receipt
  return api.post(`/receipts/${receipt.id}/parse`, { text })
}

/** Android paylaş menüsünden gelen dosya (service worker'ın sakladığı); yoksa null. */
export async function takeSharedFile() {
  // Giriş yapılmamışsa dosya önbellekte bekler; girişten sonra yine alınır.
  if (new URLSearchParams(location.search).has('shared')) history.replaceState({}, '', location.pathname)
  if (!('caches' in window)) return null
  const cache = await caches.open('cuzdan-share')
  const res = await cache.match('/__shared-receipt')
  if (!res) return null
  await cache.delete('/__shared-receipt')
  const blob = await res.blob()
  const name = decodeURIComponent(res.headers.get('X-File-Name') || 'dekont')
  return new File([blob], name, { type: blob.type })
}

export const receiptUrl = (id) => `/api/receipts/${id}/file`

/** IBAN'ı 4'lü gruplar: TR12 0006 2000 ... */
export const fmtIban = (iban) => (iban || '').replace(/(.{4})/g, '$1 ').trim()
