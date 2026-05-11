<template>
  <div class="flex items-center justify-center min-h-screen p-4">
    <form @submit.prevent="handleLogin" class="w-full max-w-sm space-y-4">
      <h1 class="text-2xl font-bold text-center">Asimut Booking Bot</h1>
      <input v-model="password" type="password" placeholder="Password" class="w-full px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500" autofocus />
      <button type="submit" class="w-full py-3 text-white bg-blue-600 rounded-lg hover:bg-blue-700">Login</button>
      <p v-if="error" class="text-sm text-red-500 text-center">{{ error }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { api } from '../api'

const auth = useAuthStore()
const password = ref('')
const error = ref('')

async function handleLogin() {
  auth.login(password.value)
  try {
    await api.getStatus()
  } catch {
    auth.logout()
    error.value = 'Invalid password'
  }
}
</script>
