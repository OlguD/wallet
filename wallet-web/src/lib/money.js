// Para her zaman kuruş (tam sayı) olarak taşınır; sadece gösterimde bölünür.
import { prefs } from './prefs.svelte.js'

const SYMBOLS = { TRY: '₺', USD: '$', EUR: '€', GBP: '£' }
export const CURRENCIES = ['TRY', 'USD', 'EUR', 'GBP']
export const MINUS = '−'

export const symbol = (c) => SYMBOLS[c] || c || '₺'
const locale = () => (prefs.lang === 'en' ? 'en-US' : 'tr-TR')

const formatters = {}
function nf() {
  const l = locale()
  return (formatters[l] ||= new Intl.NumberFormat(l, { minimumFractionDigits: 2, maximumFractionDigits: 2 }))
}

/** 123456 → "1.234,56"; sign: 'auto' (sadece eksi), 'always' (+/−), 'never' */
export function fmt(kurus, sign = 'auto') {
  const n = Number(kurus) || 0
  const s = nf().format(Math.abs(n) / 100)
  if (sign === 'never') return s
  if (n < 0) return MINUS + s
  if (sign === 'always' && n > 0) return '+' + s
  return s
}

/** Tutar + para birimi simgesi: "1.234,56 ₺" */
export const fmtc = (kurus, currency, sign) => `${fmt(kurus, sign)} ${symbol(currency)}`

/** Büyük gösterim için tam ve ondalık kısmı ayırır. */
export function parts(kurus) {
  const s = fmt(Math.abs(kurus), 'never')
  const sep = decimalSep()
  const i = s.lastIndexOf(sep)
  return { int: (kurus < 0 ? MINUS : '') + s.slice(0, i), dec: s.slice(i + 1), sep }
}

export const decimalSep = () => (prefs.lang === 'en' ? '.' : ',')
const groupSep = () => (prefs.lang === 'en' ? ',' : '.')

// Tuş takımı girdisi: içeride "642.3" gibi tutulur (nokta = ondalık).
export function keypadPress(cur, key) {
  if (key === 'del') return cur.slice(0, -1)
  if (key === 'dec') return cur.includes('.') ? cur : (cur || '0') + '.'
  const [i, d] = cur.split('.')
  if (d !== undefined) return d.length < 2 ? cur + key : cur
  if (i === '0' || i === '') return key
  return i.length < 9 ? cur + key : cur
}

export function keypadDisplay(cur) {
  if (!cur) return '0'
  const [i, d] = cur.split('.')
  const grouped = (i || '0').replace(/\B(?=(\d{3})+(?!\d))/g, groupSep())
  return d === undefined ? grouped : grouped + decimalSep() + d
}

export function keypadToKurus(cur) {
  if (!cur) return 0
  const [i, d = ''] = cur.split('.')
  return Number(i || 0) * 100 + Number((d + '00').slice(0, 2))
}

export function kurusToKeypad(k) {
  const i = Math.floor(k / 100)
  const d = k % 100
  return d === 0 ? String(i) : `${i}.${String(d).padStart(2, '0').replace(/0$/, '')}`
}

/** Serbest metin ("12,5" / "12.50") → kuruş; geçersizse null. */
export function parseAmount(text) {
  const s = String(text || '').trim().replace(/\s/g, '')
  if (!s) return null
  const norm = s.replace(/[.,](?=\d{3}(\D|$))/g, '').replace(',', '.')
  if (!/^\d+(\.\d{0,2})?$/.test(norm)) return null
  return keypadToKurus(norm)
}
