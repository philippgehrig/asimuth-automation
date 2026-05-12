import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('app_token') || '')
  const isAuthenticated = computed(() => token.value !== '')

  function login(password: string) {
    token.value = password
    localStorage.setItem('app_token', password)
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('app_token')
  }

  return { token, isAuthenticated, login, logout }
})
