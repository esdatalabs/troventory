<script setup>
import { ref, inject } from 'vue'

const props = defineProps({
  location: { type: Object, required: true },
  childrenMap: { type: Map, required: true },
  allNames: { type: Array, required: true },
})

const actions = inject('locationActions')

const renaming = ref(false)
const moving = ref(false)
const newName = ref(props.location.name)
const newParent = ref(props.location.parent || '')

const children = () => props.childrenMap.get(props.location.name) || []

async function submitRename() {
  if (newName.value && newName.value !== props.location.name) {
    await actions.rename(props.location.name, newName.value)
  }
  renaming.value = false
}

async function submitMove() {
  await actions.move(props.location.name, newParent.value || null)
  moving.value = false
}

async function archive() {
  if (confirm(`Archive location "${props.location.name}"? This only succeeds if it has no active children.`)) {
    await actions.archive(props.location.name)
  }
}
</script>

<template>
  <li>
    <div class="loc-row">
      <template v-if="renaming">
        <input v-model="newName" size="20" @keyup.enter="submitRename" @keyup.esc="renaming = false" />
        <button class="small" @click="submitRename">Save</button>
        <button class="small secondary" @click="renaming = false">Cancel</button>
      </template>
      <template v-else-if="moving">
        <select v-model="newParent">
          <option value="">(top level)</option>
          <option v-for="n in allNames.filter((n) => n !== location.name)" :key="n" :value="n">{{ n }}</option>
        </select>
        <button class="small" @click="submitMove">Save</button>
        <button class="small secondary" @click="moving = false">Cancel</button>
      </template>
      <template v-else>
        <span class="loc-name">{{ location.name }}</span>
        <span v-if="location.archived" class="badge muted">Archived</span>
        <template v-if="!location.archived">
          <button class="small secondary" @click="renaming = true">Rename</button>
          <button class="small secondary" @click="moving = true">Move</button>
          <button class="small secondary" @click="archive">Archive</button>
        </template>
      </template>
    </div>
    <ul v-if="children().length" class="tree">
      <LocationTreeNode
        v-for="child in children()"
        :key="child.name"
        :location="child"
        :children-map="childrenMap"
        :all-names="allNames"
      />
    </ul>
  </li>
</template>
