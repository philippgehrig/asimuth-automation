<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">&larr; Back</router-link>
      <h1 class="text-xl font-bold">New Booking</h1>
    </div>
    <form @submit.prevent="submit" class="space-y-4">
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium">Recurring</label>
        <input type="checkbox" v-model="isRecurring" class="rounded" />
      </div>
      <div v-if="!isRecurring">
        <label class="block text-sm font-medium text-gray-700">Date</label>
        <input v-model="form.date" type="date" class="w-full px-3 py-2 border rounded" required />
      </div>
      <div v-else>
        <label class="block text-sm font-medium text-gray-700">Day of Week</label>
        <select v-model.number="form.day_of_week" class="w-full px-3 py-2 border rounded">
          <option v-for="(name, i) in dayNames" :key="i" :value="i">{{ name }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700">Start Time</label>
        <input v-model="form.start_time" type="time" step="900" class="w-full px-3 py-2 border rounded" required />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700">Duration (minutes)</label>
        <select v-model.number="form.duration_minutes" class="w-full px-3 py-2 border rounded">
          <option :value="30">30 min</option>
          <option :value="45">45 min</option>
          <option :value="60">1 hour</option>
          <option :value="90">1.5 hours</option>
          <option :value="120">2 hours</option>
          <option :value="150">2.5 hours</option>
          <option :value="180">3 hours</option>
        </select>
      </div>
      <RoomPriorityList v-model="form.room_priorities" />
      <button type="submit" :disabled="form.room_priorities.length === 0" class="w-full py-3 text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
        {{ isRecurring ? 'Create Recurring Schedule' : 'Create Booking' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBookingsStore } from '../stores/bookings'
import { useRecurrencesStore } from '../stores/recurrences'
import { useRoomsStore } from '../stores/rooms'
import RoomPriorityList from '../components/RoomPriorityList.vue'

const router = useRouter()
const bookingsStore = useBookingsStore()
const recurrencesStore = useRecurrencesStore()
const roomsStore = useRoomsStore()

const isRecurring = ref(false)
const dayNames = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

const form = reactive({
  date: '',
  day_of_week: 0,
  start_time: '14:00',
  duration_minutes: 60,
  room_priorities: [] as number[],
})

onMounted(() => { roomsStore.fetch() })

async function submit() {
  if (isRecurring.value) {
    await recurrencesStore.create({
      day_of_week: form.day_of_week,
      start_time: form.start_time,
      duration_minutes: form.duration_minutes,
      room_priorities: form.room_priorities,
    })
  } else {
    await bookingsStore.create({
      date: form.date,
      start_time: form.start_time,
      duration_minutes: form.duration_minutes,
      room_priorities: form.room_priorities,
    })
  }
  router.push('/')
}
</script>
