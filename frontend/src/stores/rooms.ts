import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export interface Room {
  id: number
  name: string
  secondary_name: string
  bookable: boolean
  type: string
}

export const useRoomsStore = defineStore('rooms', () => {
  const rooms = ref<Room[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    try {
      rooms.value = await api.getRooms()
    } finally {
      loading.value = false
    }
  }

  return { rooms, loading, fetch }
})
