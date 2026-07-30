<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../lib/api.js'
import { notify } from '../lib/toast.js'
import { dollarsInputToCents, formatMoney } from '../lib/money.js'

const items = ref([])
const locations = ref([])
const loading = ref(true)
const showArchived = ref(false)

const emptyForm = () => ({
  description: '',
  category: '',
  purchase_date: '',
  purchase_price: '',
  currency: 'USD',
  vendor: '',
  location_name: '',
})
const form = ref(emptyForm())
const submitting = ref(false)

const scanBarcode = ref('')
const scanning = ref(false)

const visibleItems = computed(() => items.value.filter((i) => showArchived.value || !i.archived))
const activeLocationNames = computed(() => locations.value.filter((l) => !l.archived).map((l) => l.name))

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

async function createItem() {
  if (!form.value.description.trim() || !form.value.category.trim()) {
    notify.error('Description and category are required')
    return
  }
  submitting.value = true
  try {
    await api.createItem({
      description: form.value.description.trim(),
      category: form.value.category.trim(),
      purchase_date: form.value.purchase_date || undefined,
      purchase_price_cents: dollarsInputToCents(form.value.purchase_price),
      currency: form.value.currency || 'USD',
      vendor: form.value.vendor || undefined,
      location_name: form.value.location_name || null,
    })
    notify.success(`Item "${form.value.description}" created`)
    form.value = emptyForm()
    await load()
  } catch (err) {
    notify.error(err)
  } finally {
    submitting.value = false
  }
}

async function scan() {
  if (!scanBarcode.value.trim()) return
  scanning.value = true
  try {
    await api.scanItem(scanBarcode.value.trim())
    notify.success(`Scanned draft for barcode "${scanBarcode.value}" — enrich it from the item detail once cataloged, or via the API's /items/enrich.`)
    scanBarcode.value = ''
  } catch (err) {
    notify.error(err)
  } finally {
    scanning.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1>Items</h1>
      <p>Your cataloged belongings — description, category, purchase details, and location.</p>
    </div>
  </div>

  <div class="card">
    <div class="card-header"><h2>Add an item</h2></div>
    <form class="stack" @submit.prevent="createItem">
      <div class="field-grid">
        <label>
          Description
          <input v-model="form.description" placeholder="e.g. Old Microwave" />
        </label>
        <label>
          Category
          <input v-model="form.category" placeholder="e.g. Appliances" />
        </label>
        <label>
          Purchase date
          <input v-model="form.purchase_date" type="date" />
        </label>
        <label>
          Purchase price
          <input v-model="form.purchase_price" type="number" step="0.01" min="0" placeholder="0.00" />
        </label>
        <label>
          Currency
          <input v-model="form.currency" maxlength="3" />
        </label>
        <label>
          Vendor
          <input v-model="form.vendor" placeholder="e.g. Best Buy" />
        </label>
        <label>
          Location
          <select v-model="form.location_name">
            <option value="">(none)</option>
            <option v-for="n in activeLocationNames" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
      </div>
      <div class="row">
        <button type="submit" :disabled="submitting">Add item</button>
      </div>
    </form>
  </div>

  <div class="card">
    <div class="card-header"><h2>Scan a barcode</h2></div>
    <p style="margin-bottom: 10px">
      Seeds a draft item awaiting enrichment — a stand-in for a physical barcode scanner.
    </p>
    <form class="row" @submit.prevent="scan">
      <input v-model="scanBarcode" placeholder="Barcode / UPC" style="max-width: 220px" />
      <button type="submit" class="secondary" :disabled="scanning">Scan</button>
    </form>
  </div>

  <div class="card">
    <div class="card-header">
      <h2>Catalog</h2>
      <label style="flex-direction: row; align-items: center; gap: 6px">
        <input type="checkbox" v-model="showArchived" style="width: auto" />
        Show archived
      </label>
    </div>
    <p v-if="loading" class="empty-state">Loading…</p>
    <p v-else-if="!visibleItems.length" class="empty-state">No items yet — add one above.</p>
    <table v-else>
      <thead>
        <tr>
          <th>Description</th>
          <th>Category</th>
          <th>Location</th>
          <th>Purchase price</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in visibleItems" :key="item.description">
          <td>
            <router-link class="table-link" :to="`/items/${encodeURIComponent(item.description)}`">
              {{ item.description }}
            </router-link>
          </td>
          <td>{{ item.category }}</td>
          <td>{{ item.location_name || '—' }}</td>
          <td>{{ formatMoney(item.purchase_price_cents, item.currency) }}</td>
          <td>
            <span v-if="item.archived" class="badge muted">Archived</span>
            <span v-else class="badge good">Active</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
