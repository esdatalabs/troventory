<script setup>
import { ref, computed, onMounted, provide } from 'vue'
import { api } from '../lib/api.js'
import { notify } from '../lib/toast.js'
import LocationTreeNode from '../components/LocationTreeNode.vue'

const locations = ref([])
const loading = ref(true)

const form = ref({ name: '', parent: '' })
const submitting = ref(false)

const childrenMap = computed(() => {
  const map = new Map()
  for (const loc of locations.value) {
    const key = loc.parent || ''
    if (!map.has(key)) map.set(key, [])
    map.get(key).push(loc)
  }
  for (const list of map.values()) list.sort((a, b) => a.name.localeCompare(b.name))
  return map
})

const topLevel = computed(() => childrenMap.value.get('') || [])
const allNames = computed(() => locations.value.filter((l) => !l.archived).map((l) => l.name))

async function load() {
  loading.value = true
  try {
    locations.value = await api.listLocations()
  } catch (err) {
    notify.error(err)
  } finally {
    loading.value = false
  }
}

async function createLocation() {
  if (!form.value.name.trim()) {
    notify.error('Name is required')
    return
  }
  submitting.value = true
  try {
    await api.createLocation({ name: form.value.name.trim(), parent: form.value.parent || null })
    notify.success(`Location "${form.value.name}" created`)
    form.value = { name: '', parent: '' }
    await load()
  } catch (err) {
    notify.error(err)
  } finally {
    submitting.value = false
  }
}

provide('locationActions', {
  async rename(name, newName) {
    try {
      await api.renameLocation(name, newName)
      notify.success(`Renamed "${name}" to "${newName}"`)
      await load()
    } catch (err) {
      notify.error(err)
    }
  },
  async move(name, parent) {
    try {
      await api.moveLocation(name, parent)
      notify.success(`Moved "${name}"`)
      await load()
    } catch (err) {
      notify.error(err)
    }
  },
  async archive(name) {
    try {
      await api.archiveLocation(name)
      notify.success(`Archived "${name}"`)
      await load()
    } catch (err) {
      notify.error(err)
    }
  },
})

onMounted(load)
</script>

<template>
  <div class="page-header">
    <div>
      <h1>Locations</h1>
      <p>Rooms, containers, and shelves — nested as deep as you like.</p>
    </div>
  </div>

  <div class="card">
    <div class="card-header"><h2>Add a location</h2></div>
    <form class="row" @submit.prevent="createLocation">
      <label style="flex: 1; min-width: 160px">
        Name
        <input v-model="form.name" placeholder="e.g. Garage Shelf A" />
      </label>
      <label style="flex: 1; min-width: 160px">
        Parent (optional)
        <select v-model="form.parent">
          <option value="">(top level)</option>
          <option v-for="n in allNames" :key="n" :value="n">{{ n }}</option>
        </select>
      </label>
      <button type="submit" :disabled="submitting">Add location</button>
    </form>
  </div>

  <div class="card">
    <div class="card-header"><h2>Hierarchy</h2></div>
    <p v-if="loading" class="empty-state">Loading…</p>
    <p v-else-if="!topLevel.length" class="empty-state">No locations yet — add one above.</p>
    <ul v-else class="tree">
      <LocationTreeNode
        v-for="loc in topLevel"
        :key="loc.name"
        :location="loc"
        :children-map="childrenMap"
        :all-names="allNames"
      />
    </ul>
  </div>
</template>
