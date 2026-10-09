// Cihaza özel tercihler (tema, dil). localStorage'a yazılır.
const read = (k, ok, def) => {
  try {
    const v = localStorage.getItem(k)
    return ok.includes(v) ? v : def
  } catch {
    return def
  }
}

const THEME_COLOR = { a: '#0B0B0D', b: '#F4EFE6', c: '#F2F0EA' }

export const prefs = $state({
  theme: read('wallet.theme', ['a', 'b', 'c'], 'a'),
  lang: read('wallet.lang', ['tr', 'en'], navigator.language?.startsWith('en') ? 'en' : 'tr'),
})

function apply() {
  const root = document.documentElement
  root.dataset.theme = prefs.theme
  root.lang = prefs.lang
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', THEME_COLOR[prefs.theme])
  // Ana ekrana eklenirken Safari o anki ikonu kullanır: seçili temanın ikonu.
  document.getElementById('apple-icon')?.setAttribute('href', `/icons/apple-touch-icon-${prefs.theme}.png`)
}
apply()

export function setTheme(t) {
  prefs.theme = t
  try {
    localStorage.setItem('wallet.theme', t)
  } catch {}
  apply()
}

export function setLang(l) {
  prefs.lang = l
  try {
    localStorage.setItem('wallet.lang', l)
  } catch {}
  apply()
}
