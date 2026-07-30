<script setup>
import { ref } from 'vue'
import { api } from '../lib/api.js'
import { notify } from '../lib/toast.js'
import { dollarsInputToCents, formatMoney } from '../lib/money.js'

const filters = ref({ desc: '', category: '', location: '', min: '', max: '' })
const results = ref(null)
const searching = ref(false)
const searched = ref(false)

async function runSearch() {
  searching.value = true
  try {
    const res = await api.search({
      desc: filters.value.desc,
      category: filters.value.category,
      location: filters.value.location,
      min: filters.value.min === '' ? '' : dollarsInputToCents(filters.value.min),
      max: filters.value.max === '' ? '' : dollarsInputToCents(filters.value.max),
    })
    results.value = res.matches
    searched.value = true
  } catch (err) {
    notify.error(err)
  } finally {
    searching.value = false
  }
}

runSearch()
</script>

<template>
  <div class="page-header">
    <div>
      <h1>Search</h1>
      <p>Filter across your active inventory by name, category, location, or value.</p>
    </div>
  </div>

  <div class="card">
    <form class="field-grid" @submit.prevent="runSearch">
      <label>
        Description contains
        <input v-model="filters.desc" />
      </label>
      <label>
        Category
        <input v-model="filters.category" />
      </label>
      <label>
        Location
        <input v-model="filters.location" />
      </label>
      <label>
        Min value
        <input v-model="filters.min" type="number" step="0.01" min="0" />
      </label>
      <label>
        Max value
        <input v-model="filters.max" type="number" step="0.01" min="0" />
      </label>
      <div style="align-self: end">
        <button type="submit" :disabled="searching">Search</button>
      </div>
    </form>
  </div>

  <div class="card">
    <div class="card-header"><h2>Results</h2></div>
    <p v-if="searching" class="empty-state">Searching…</p>
    <p v-else-if="searched && !results.length" class="empty-state">No matches.</p>
    <table v-else-if="results && results.length">
      <thead>
        <tr>
          <th>Description</th>
          <th>Category</th>
          <th>Location</th>
          <th>Current value</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in results" :key="item.description">
          <td>
            <router-link class="table-link" :to="`/items/${encodeURIComponent(item.description)}`">
              {{ item.description }}
            </router-link>
          </td>
          <td>{{ item.category }}</td>
          <td>{{ item.location_name || '—' }}</td>
          <td>{{ formatMoney(item.current_value?.amount_cents, item.current_value?.currency) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
