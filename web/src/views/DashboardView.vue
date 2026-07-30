<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../lib/api.js'
import { notify } from '../lib/toast.js'
import { formatMoney } from '../lib/money.js'

const items = ref([])
const locations = ref([])
const loading = ref(true)

const activeItems = computed(() => items.value.filter((i) => !i.archived))
const archivedCount = computed(() => items.value.length - activeItems.value.length)
const activeLocations = computed(() => locations.value.filter((l) => !l.archived))
const recentItems = computed(() => activeItems.value.slice(-6).reverse())

const recordedValue = computed(() => {
  let cents = 0
  let currency = ''
  for (const item of activeItems.value) {
    if (item.valuation?.purchase_price?.currency) {
      currency = item.valuation.purchase_price.currency
      cents += item.valuation.purchase_price.amount_cents
    }
  }
  return { cents, currency }
})

async function load() {
  loading.value = true
  try {
    const [i, l] = await Promise.all([api.listItems(), api.listLocations()])
    items.value = i
    locations.value = l
  } catch (err) {
    notify.error(err)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1>Dashboard</h1>
      <p>An overview of what's tracked in your inventory.</p>
    </div>
  </div>

  <div class="stat-grid">
    <div class="stat-tile">
      <div class="stat-label">Active items</div>
      <div class="stat-value">{{ activeItems.length }}</div>
    </div>
    <div class="stat-tile">
      <div class="stat-label">Archived items</div>
      <div class="stat-value">{{ archivedCount }}</div>
    </div>
    <div class="stat-tile">
      <div class="stat-label">Locations</div>
      <div class="stat-value">{{ activeLocations.length }}</div>
    </div>
    <div class="stat-tile">
      <div class="stat-label">Recorded purchase value</div>
      <div class="stat-value">{{ formatMoney(recordedValue.cents, recordedValue.currency) }}</div>
    </div>
  </div>

  <div class="card">
    <div class="card-header">
      <h2>Recently added items</h2>
      <router-link to="/items">View all items →</router-link>
    </div>
    <p v-if="loading" class="empty-state">Loading…</p>
    <p v-else-if="!recentItems.length" class="empty-state">
      No items yet — <router-link to="/items">add your first one</router-link>.
    </p>
    <table v-else>
      <thead>
        <tr>
          <th>Description</th>
          <th>Category</th>
          <th>Location</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in recentItems" :key="item.description">
          <td>
            <router-link class="table-link" :to="`/items/${encodeURIComponent(item.description)}`">
              {{ item.description }}
            </router-link>
          </td>
          <td>{{ item.category }}</td>
          <td>{{ item.location_name || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
