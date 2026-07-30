<script setup>
import { ref } from 'vue'
import { api } from '../lib/api.js'

const format = ref('CSV')
</script>

<template>
  <div class="page-header">
    <div>
      <h1>Export</h1>
      <p>Compile every active item's description, category, valuation, and location into one report.</p>
    </div>
  </div>

  <div class="card">
    <form class="row" @submit.prevent>
      <label>
        Format
        <select v-model="format">
          <option value="CSV">CSV</option>
          <option value="PDF">PDF</option>
        </select>
      </label>
      <a class="table-link" :href="api.exportURL(format)" style="align-self: end">
        <button type="button">Download report</button>
      </a>
    </form>
    <p v-if="format === 'PDF'" style="margin-top: 12px">
      Note: PDF rendering isn't wired up yet — this downloads a plain-text stand-in with the same content.
    </p>
  </div>
</template>
