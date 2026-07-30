<script setup>
import { toastState, notify } from '../lib/toast.js'
</script>

<template>
  <div class="toast-stack">
    <div
      v-for="t in toastState.toasts"
      :key="t.id"
      class="toast"
      :class="t.kind"
      @click="notify.dismiss(t.id)"
    >
      {{ t.message }}
    </div>
  </div>
</template>
