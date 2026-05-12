<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">&larr; Back</router-link>
      <h1 class="text-xl font-bold">Allowed Rooms</h1>
    </div>
    <p class="text-sm text-gray-500">Select which rooms can be used for bookings. Only these rooms will appear in the priority list when creating bookings.</p>
    <input v-model="search" type="text" placeholder="Search rooms..." class="w-full px-3 py-2 border rounded" />
    <div v-if="roomsStore.loading" class="text-gray-500">Loading...</div>
    <div v-else class="space-y-1">
      <div v-for="room in filteredRooms" :key="room.id" @click="toggle(room.id)" class="flex items-center gap-3 p-3 bg-white border rounded cursor-pointer hover:bg-gray-50" :class="{ 'border-blue-500 bg-blue-50 hover:bg-blue-50': isAllowed(room.id) }">
        <input type="checkbox" :checked="isAllowed(room.id)" class="rounded" @click.stop="toggle(room.id)" />
        <div class="flex-1">
          <p class="text-sm font-medium">{{ room.name }}</p>
          <p class="text-xs text-gray-500">{{ room.secondary_name }}</p>
        </div>
      </div>
    </div>
    <div class="sticky bottom-4">
      <button @click="save" :disabled="!dirty" class="w-full py-3 text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
        Save ({{ selectedIds.length }} rooms selected)
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoomsStore } from '../stores/rooms'

const roomsStore = useRoomsStore()
const search = ref('')
const selectedIds = ref<number[]>([])
const dirty = ref(false)

const filteredRooms = computed(() => {
  const q = search.value.toLowerCase()
  return roomsStore.rooms
    .filter(r => r.type === 'location' && r.bookable)
    .filter(r => r.name.toLowerCase().includes(q) || r.secondary_name.toLowerCase().includes(q))
})

function isAllowed(id: number) {
  return selectedIds.value.includes(id)
}

function toggle(id: number) {
  dirty.value = true
  if (selectedIds.value.includes(id)) {
    selectedIds.value = selectedIds.value.filter(i => i !== id)
  } else {
    selectedIds.value = [...selectedIds.value, id]
  }
}

async function save() {
  await roomsStore.setAllowed(selectedIds.value)
  dirty.value = false
}

onMounted(async () => {
  await roomsStore.fetch()
  selectedIds.value = [...roomsStore.allowedRoomIds]
})
</script>
