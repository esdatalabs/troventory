<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../lib/api.js'
import { notify } from '../lib/toast.js'
import { centsToDollarsInput, dollarsInputToCents, formatMoney } from '../lib/money.js'

const props = defineProps({ description: { type: String, required: true } })
const router = useRouter()

const item = ref(null)
const locations = ref([])
const loading = ref(true)

const editForm = ref(null)
const savingEdit = ref(false)

const priceForm = ref({ amount: '', currency: 'USD', date: '' })
const appraisalForm = ref({ amount: '', currency: 'USD', date: '' })
const depreciationForm = ref({ rate_percent: 0 })
const currentValueDate = ref('')
const currentValue = ref(null)
const enrichBarcode = ref('')

async function load() {
  loading.value = true
  try {
    const [i, locs] = await Promise.all([api.getItem(props.description), api.listLocations()])
    item.value = i
    locations.value = locs
    editForm.value = {
      description: i.description,
      category: i.category,
      purchase_date: i.purchase_date,
      purchase_price: centsToDollarsInput(i.purchase_price_cents),
      currency: i.currency || 'USD',
      vendor: i.vendor,
      location_name: i.location_name || '',
      photos: (i.photos || []).join(', '),
    }
    depreciationForm.value.rate_percent = i.valuation?.depreciation_rate_percent || 0
  } catch (err) {
    notify.error(err)
  } finally {
    loading.value = false
  }
}

watch(() => props.description, load)
onMounted(load)

async function saveEdit() {
  savingEdit.value = true
  try {
    const patch = {
      description: editForm.value.description,
      category: editForm.value.category,
      purchase_date: editForm.value.purchase_date,
      purchase_price_cents: dollarsInputToCents(editForm.value.purchase_price),
      currency: editForm.value.currency,
      vendor: editForm.value.vendor,
      location_name: editForm.value.location_name,
      photos: editForm.value.photos.split(',').map((p) => p.trim()).filter(Boolean),
    }
    const wasRenamed = patch.description !== item.value.description
    await api.updateItem(item.value.description, patch)
    notify.success('Item updated')
    if (wasRenamed) {
      router.replace(`/items/${encodeURIComponent(patch.description)}`)
    } else {
      await load()
    }
  } catch (err) {
    notify.error(err)
  } finally {
    savingEdit.value = false
  }
}

async function archive() {
  if (!confirm(`Archive "${item.value.description}"?`)) return
  try {
    await api.archiveItem(item.value.description)
    notify.success('Item archived')
    await load()
  } catch (err) {
    notify.error(err)
  }
}

async function submitPrice() {
  try {
    await api.recordPrice(item.value.description, {
      amount_cents: dollarsInputToCents(priceForm.value.amount),
      currency: priceForm.value.currency || 'USD',
      date: priceForm.value.date || undefined,
    })
    notify.success('Purchase price recorded')
    priceForm.value.amount = ''
    await load()
  } catch (err) {
    notify.error(err)
  }
}

async function submitAppraisal() {
  try {
    await api.recordAppraisal(item.value.description, {
      amount_cents: dollarsInputToCents(appraisalForm.value.amount),
      currency: appraisalForm.value.currency || 'USD',
      date: appraisalForm.value.date || undefined,
    })
    notify.success('Appraisal recorded')
    appraisalForm.value.amount = ''
    await load()
  } catch (err) {
    notify.error(err)
  }
}

async function submitDepreciation() {
  try {
    await api.setDepreciationRate(item.value.description, Number(depreciationForm.value.rate_percent))
    notify.success('Depreciation rate updated')
    await load()
  } catch (err) {
    notify.error(err)
  }
}

async function computeCurrentValue() {
  try {
    currentValue.value = await api.getCurrentValue(item.value.description, currentValueDate.value || undefined)
  } catch (err) {
    notify.error(err)
  }
}

async function enrich() {
  if (!enrichBarcode.value.trim()) return
  try {
    await api.enrichItem(enrichBarcode.value.trim(), item.value.description)
    notify.success('Item enriched from barcode')
    enrichBarcode.value = ''
    await load()
  } catch (err) {
    notify.error(err)
  }
}
</script>

<template>
  <p v-if="loading" class="empty-state">Loading…</p>
  <template v-else-if="item">
    <div class="page-header">
      <div>
        <router-link to="/items">← All items</router-link>
        <h1>{{ item.description }}</h1>
        <p>
          <span v-if="item.archived" class="badge muted">Archived</span>
          <span v-else class="badge good">Active</span>
        </p>
      </div>
      <button v-if="!item.archived" class="danger" @click="archive">Archive item</button>
    </div>

    <div class="card">
      <div class="card-header"><h2>Details</h2></div>
      <form class="stack" @submit.prevent="saveEdit">
        <div class="field-grid">
          <label>
            Description
            <input v-model="editForm.description" />
          </label>
          <label>
            Category
            <input v-model="editForm.category" />
          </label>
          <label>
            Purchase date
            <input v-model="editForm.purchase_date" type="date" />
          </label>
          <label>
            Purchase price
            <input v-model="editForm.purchase_price" type="number" step="0.01" min="0" />
          </label>
          <label>
            Currency
            <input v-model="editForm.currency" maxlength="3" />
          </label>
          <label>
            Vendor
            <input v-model="editForm.vendor" />
          </label>
          <label>
            Location
            <select v-model="editForm.location_name">
              <option value="">(none)</option>
              <option v-for="l in locations.filter((l) => !l.archived)" :key="l.name" :value="l.name">
                {{ l.name }}
              </option>
            </select>
          </label>
          <label>
            Photos (comma-separated)
            <input v-model="editForm.photos" />
          </label>
        </div>
        <div class="row">
          <button type="submit" :disabled="savingEdit">Save changes</button>
        </div>
      </form>
    </div>

    <div class="card">
      <div class="card-header"><h2>Enrich from barcode</h2></div>
      <p style="margin-bottom: 10px">Fills in whichever of this item's fields are still empty from a barcode/UPC lookup.</p>
      <form class="row" @submit.prevent="enrich">
        <input v-model="enrichBarcode" placeholder="Barcode / UPC" style="max-width: 220px" />
        <button type="submit" class="secondary">Enrich</button>
      </form>
    </div>

    <div class="card">
      <div class="card-header"><h2>Valuation</h2></div>
      <div class="stack">
        <div>
          <h3>Purchase price</h3>
          <form class="row" @submit.prevent="submitPrice">
            <input v-model="priceForm.amount" type="number" step="0.01" min="0" placeholder="Amount" style="max-width: 120px" />
            <input v-model="priceForm.currency" maxlength="3" style="max-width: 70px" />
            <input v-model="priceForm.date" type="date" />
            <button type="submit" class="secondary small">Record</button>
          </form>
        </div>

        <div>
          <h3>New appraisal</h3>
          <form class="row" @submit.prevent="submitAppraisal">
            <input v-model="appraisalForm.amount" type="number" step="0.01" min="0" placeholder="Amount" style="max-width: 120px" />
            <input v-model="appraisalForm.currency" maxlength="3" style="max-width: 70px" />
            <input v-model="appraisalForm.date" type="date" />
            <button type="submit" class="secondary small">Record</button>
          </form>
          <table v-if="item.valuation?.appraisals?.length" style="margin-top: 8px">
            <thead>
              <tr><th>As of</th><th>Amount</th></tr>
            </thead>
            <tbody>
              <tr v-for="(a, idx) in item.valuation.appraisals" :key="idx">
                <td>{{ a.as_of }}</td>
                <td>{{ formatMoney(a.amount_cents, a.currency) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div>
          <h3>Depreciation rate</h3>
          <form class="row" @submit.prevent="submitDepreciation">
            <input v-model="depreciationForm.rate_percent" type="number" min="0" max="100" style="max-width: 100px" />
            <span>% / year</span>
            <button type="submit" class="secondary small">Set</button>
          </form>
        </div>

        <div>
          <h3>Current value</h3>
          <form class="row" @submit.prevent="computeCurrentValue">
            <input v-model="currentValueDate" type="date" />
            <button type="submit" class="secondary small">Compute</button>
          </form>
          <p v-if="currentValue" style="margin-top: 8px; color: var(--text-primary); font-weight: 600">
            {{ formatMoney(currentValue.value.amount_cents, currentValue.value.currency) }}
            <span style="color: var(--text-muted); font-weight: 400"> as of {{ currentValue.as_of }}</span>
          </p>
        </div>
      </div>
    </div>
  </template>
</template>
