import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
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
  const allowedRoomIds = ref<number[]>([])
  const loading = ref(false)

  const allowedRooms = computed(() => {
    if (allowedRoomIds.value.length === 0) {
      return rooms.value.filter(r => r.type === 'location' && r.bookable)
    }
    return rooms.value.filter(r => allowedRoomIds.value.includes(r.id))
  })

  async function fetch() {
    loading.value = true
    try {
      const [r, a] = await Promise.all([api.getRooms(), api.getAllowedRooms()])
      rooms.value = r
      allowedRoomIds.value = a
    } finally {
      loading.value = false
    }
  }

  async function setAllowed(ids: number[]) {
    await api.setAllowedRooms(ids)
    allowedRoomIds.value = ids
  }

  return { rooms, allowedRoomIds, allowedRooms, loading, fetch, setAllowed }
})
