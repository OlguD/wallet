import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// /api istekleri Go sunucusuna aktarılır; böylece tarayıcı açısından
// her şey aynı origin'dir ve oturum cookie'si sorunsuz çalışır.
const api = {
  '/api': {
    target: process.env.API_URL || 'http://localhost:8080',
    changeOrigin: true,
    rewrite: (p) => p.replace(/^\/api/, ''),
  },
}

export default defineConfig({
  plugins: [
    svelte({
      // Formlar başlangıç değerini prop'tan bilerek bir kez alır.
      onwarn(w, handler) {
        if (w.code === 'state_referenced_locally') return
        handler(w)
      },
    }),
  ],
  server: { port: 5173, proxy: api },
  preview: { port: 4173, proxy: api },
})
