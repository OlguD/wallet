import { mount } from 'svelte'
import './fonts.css'
import './app.css'
import App from './App.svelte'

const app = mount(App, { target: document.getElementById('app') })

// Service worker yalnızca HTTPS (veya localhost) üzerinde çalışır.
if (import.meta.env.PROD && 'serviceWorker' in navigator && window.isSecureContext) {
  navigator.serviceWorker.register('/sw.js').catch(() => {})
}

export default app
