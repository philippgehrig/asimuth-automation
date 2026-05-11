<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">&larr; Back</router-link>
      <h1 class="text-xl font-bold">Rooms</h1>
    </div>
    <input v-model="search" type="text" placeholder="Search rooms..." class="w-full px-3 py-2 border rounded" />
    <div class="space-y-1">
      <div v-for="room in filteredRooms" :key="room.id" class="p-3 bg-white border rounded text-sm">
        <p class="font-medium">{{ room.name }}</p>
        <p class="text-gray-500">{{ room.secondary_name }}</p>
        <p class="text-xs text-gray-400">ID: {{ room.id }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoomsStore } from '../stores/rooms'

const roomsStore = useRoomsStore()
const search = ref('')

const filteredRooms = computed(() => {
  const q = search.value.toLowerCase()
  return roomsStore.rooms
    .filter(r => r.type === 'location' && r.bookable)
    .filter(r => r.name.toLowerCase().includes(q) || r.secondary_name.toLowerCase().includes(q))
})

onMounted(() => { roomsStore.fetch() })
</script>
