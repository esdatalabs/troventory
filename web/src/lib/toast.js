import { reactive } from 'vue'

// A tiny singleton toast queue — good enough for a demo dashboard without
// pulling in a UI library. Views call notify.error()/notify.success() and
// AppToasts.vue (mounted once in App.vue) renders whatever is queued.

let nextId = 1
const state = reactive({ toasts: [] })

function push(kind, message) {
  const id = nextId++
  state.toasts.push({ id, kind, message })
  setTimeout(() => dismiss(id), 5000)
}

function dismiss(id) {
  const idx = state.toasts.findIndex((t) => t.id === id)
  if (idx !== -1) state.toasts.splice(idx, 1)
}

export const toastState = state
export const notify = {
  success: (message) => push('success', message),
  error: (message) => push('error', message instanceof Error ? message.message : message),
  dismiss,
}
