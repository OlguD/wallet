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
]

function match(path) {
  for (const [name, re] of routes) {
    const m = path.match(re)
    if (m) return { name, id: m[1] ? Number(m[1]) : null, path }
  }
  return { name: 'home', id: null, path: '/' }
}

export const route = $state(match(location.pathname))

export function navigate(path, { replace = false } = {}) {
  if (path === route.path) return
  history[replace ? 'replaceState' : 'pushState']({}, '', path)
  Object.assign(route, match(path))
}

export function back(fallback = '/') {
  if (history.length > 1 && history.state) history.back()
  else navigate(fallback, { replace: true })
}

window.addEventListener('popstate', () => Object.assign(route, match(location.pathname)))
