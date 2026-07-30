import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', name: 'dashboard', component: () => import('./views/DashboardView.vue') },
  { path: '/locations', name: 'locations', component: () => import('./views/LocationsView.vue') },
  { path: '/items', name: 'items', component: () => import('./views/ItemsView.vue') },
  { path: '/items/:description', name: 'item-detail', component: () => import('./views/ItemDetailView.vue'), props: true },
  { path: '/search', name: 'search', component: () => import('./views/SearchView.vue') },
  { path: '/export', name: 'export', component: () => import('./views/ExportView.vue') },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
