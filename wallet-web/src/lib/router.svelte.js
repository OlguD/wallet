// Küçük bir history tabanlı yönlendirici.
const routes = [
  ['home', /^\/$/],
  ['transactions', /^\/transactions$/],
  ['accounts', /^\/accounts$/],
  ['account', /^\/accounts\/(\d+)$/],
  ['groups', /^\/groups$/],
  ['group', /^\/groups\/(\d+)$/],
  ['settings', /^\/settings$/],
  ['goals', /^\/goals$/],
  ['recurring', /^\/recurring$/],
  ['rates', /^\/rates$/],
  ['payees', /^\/payees$/],
  ['inbox', /^\/inbox$/],
  ['budgets', /^\/budgets$/],
  ['report', /^\/report$/],
]

// path sorgu içerebilir ("/report?month=2026-10"); eşleşme yol kısmıyla yapılır.
function match(path) {
  const [p, q = ''] = path.split('?')
  const query = Object.fromEntries(new URLSearchParams(q))
  for (const [name, re] of routes) {
    const m = p.match(re)
    if (m) return { name, id: m[1] ? Number(m[1]) : null, path, query }
  }
  return { name: 'home', id: null, path: '/', query: {} }
}

export const route = $state(match(location.pathname + location.search))

export function navigate(path, { replace = false } = {}) {
  if (path === route.path) return
  history[replace ? 'replaceState' : 'pushState']({}, '', path)
  Object.assign(route, match(path))
}

export function back(fallback = '/') {
  if (history.length > 1 && history.state) history.back()
  else navigate(fallback, { replace: true })
}

window.addEventListener('popstate', () => Object.assign(route, match(location.pathname + location.search)))
