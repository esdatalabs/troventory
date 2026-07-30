<script setup>
import AppToasts from './components/AppToasts.vue'
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        Troventory
        <small>home inventory</small>
      </div>
      <nav class="nav">
        <router-link to="/">Dashboard</router-link>
        <router-link to="/locations">Locations</router-link>
        <router-link to="/items">Items</router-link>
        <router-link to="/search">Search</router-link>
        <router-link to="/export">Export</router-link>
      </nav>
    </aside>
    <main class="main">
      <router-view />
    </main>
    <AppToasts />
  </div>
</template>
