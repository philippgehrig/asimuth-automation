<template>
  <div class="min-h-screen bg-gray-50">
    <router-view v-if="isAuthenticated" />
    <LoginView v-else />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from './stores/auth'
import LoginView from './views/LoginView.vue'

const auth = useAuthStore()
const isAuthenticated = computed(() => auth.isAuthenticated)
</script>
