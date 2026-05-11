<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">&larr; Back</router-link>
      <h1 class="text-xl font-bold">Settings</h1>
    </div>
    <div class="p-4 bg-white rounded-lg border">
      <h2 class="font-medium mb-2">Asimut Connection</h2>
      <p v-if="status === null" class="text-gray-500">Checking...</p>
      <p v-else-if="status" class="text-green-600">Connected</p>
      <p v-else class="text-red-500">Not connected</p>
    </div>
    <button @click="logout" class="w-full py-2 text-red-600 border border-red-200 rounded-lg hover:bg-red-50">Logout</button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const status = ref<boolean | null>(null)

onMounted(async () => {
  try {
    const result = await api.getStatus()
    status.value = result.asimut_connected
  } catch {
    status.value = false
  }
})

function logout() {
  auth.logout()
  window.location.reload()
}
</script>
