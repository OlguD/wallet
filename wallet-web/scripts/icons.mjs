// Uygulama ikonlarını (PNG) SVG'den üretir. Headless Chromium gerekir:
//   CHROME=/path/to/chrome node scripts/icons.mjs
import { writeFileSync, mkdirSync } from 'node:fs'
import { execFileSync } from 'node:child_process'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const variants = {
  main: { bg: '#2B4BDB', plate: '#6A84F5', coin: '#FFC83D', front: '#FFFFFF' },
  a: { bg: '#0B0B0D', plate: '#2E2E35', coin: '#C6F432', front: '#F4F4F5' },
  b: { bg: '#F4EFE6', plate: '#D8CDBB', coin: '#C2410C', front: '#1C1A17' },
  c: { bg: '#0F5C55', plate: '#2F7D74', coin: '#F2B441', front: '#FFFFFF' },
}

// pad: maskable ikonlarda işaret güvenli alanda kalsın diye küçültülür.
const svg = (v, pad = 1) => `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">
<rect width="64" height="64" fill="${v.bg}"/>
<g transform="translate(32 33.5) scale(${0.78 * pad}) translate(-32 -33.5)">
<rect x="10" y="13" width="44" height="41" rx="11" fill="${v.plate}"/>
<circle cx="32" cy="31" r="8.5" fill="${v.coin}"/>
<path d="M10 31 Q32 43 54 31 V43 A11 11 0 0 1 43 54 H21 A11 11 0 0 1 10 43 Z" fill="${v.front}"/>
</g></svg>`

const chrome = process.env.CHROME
if (!chrome) throw new Error('CHROME ortam degiskeni gerekli')
mkdirSync('public/icons', { recursive: true })
const tmp = join(tmpdir(), 'wallet-icons')
mkdirSync(tmp, { recursive: true })

function render(name, markup, size) {
  const html = join(tmp, name + '.html')
  writeFileSync(html, `<html><body style="margin:0">${markup.replace('<svg ', `<svg width="${size}" height="${size}" `)}</body></html>`)
  execFileSync(chrome, ['--headless', '--disable-gpu', '--hide-scrollbars', `--window-size=${size},${size}`,
    '--force-device-scale-factor=1', `--screenshot=${join(process.cwd(), 'public/icons', name + '.png')}`, 'file://' + html],
    { stdio: 'ignore' })
}

for (const [k, v] of Object.entries(variants)) {
  render(`apple-touch-icon-${k}`, svg(v), 180)
  writeFileSync(`public/icons/icon-${k}.svg`, svg(v))
}
render('icon-192', svg(variants.main), 192)
render('icon-512', svg(variants.main), 512)
render('icon-maskable-512', svg(variants.main, 0.8), 512)
console.log('ikonlar uretildi')
