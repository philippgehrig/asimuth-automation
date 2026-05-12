import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/DashboardView.vue') },
    { path: '/create', component: () => import('./views/CreateBookingView.vue') },
    { path: '/rooms', component: () => import('./views/RoomsView.vue') },
    { path: '/settings', component: () => import('./views/SettingsView.vue') },
  ],
})

export default router
