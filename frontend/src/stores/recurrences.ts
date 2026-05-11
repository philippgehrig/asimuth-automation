import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export interface RecurringSchedule {
  id: string
  day_of_week: number
  start_time: string
  duration_minutes: number
  room_priorities: number[]
  active: boolean
  created_at: string
}

export const useRecurrencesStore = defineStore('recurrences', () => {
  const recurrences = ref<RecurringSchedule[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    recurrences.value = await api.getRecurrences()
    loading.value = false
  }

  async function create(data: Partial<RecurringSchedule>) {
    await api.createRecurrence(data)
    await fetch()
  }

  async function toggleActive(id: string, active: boolean) {
    await api.updateRecurrence(id, { active })
    await fetch()
  }

  async function remove(id: string) {
    await api.deleteRecurrence(id)
    await fetch()
  }

  return { recurrences, loading, fetch, create, toggleActive, remove }
})
