import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    // Listen on every interface, not just localhost, so the dev server is
    // reachable from other devices on the network (or from outside a
    // devcontainer) — not just the machine it's running on.
    host: true,
  },
})
