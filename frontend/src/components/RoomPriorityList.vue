<template>
  <div class="space-y-2">
    <label class="block text-sm font-medium text-gray-700">Room Priority</label>
    <div class="space-y-1">
      <div v-for="(roomId, index) in modelValue" :key="roomId" class="flex items-center gap-2 p-2 bg-white border rounded">
        <span class="text-sm text-gray-400 w-6">{{ index + 1 }}.</span>
        <span class="flex-1 text-sm">{{ getRoomName(roomId) }}</span>
        <button @click="moveUp(index)" :disabled="index === 0" class="text-gray-400 hover:text-gray-600 disabled:opacity-30">&#x2191;</button>
        <button @click="moveDown(index)" :disabled="index === modelValue.length - 1" class="text-gray-400 hover:text-gray-600 disabled:opacity-30">&#x2193;</button>
        <button @click="remove(index)" class="text-red-400 hover:text-red-600">&#x2715;</button>
      </div>
    </div>
    <select @change="addRoom($event)" class="w-full px-3 py-2 border rounded text-sm">
      <option value="">Add room...</option>
      <option v-for="room in availableRooms" :key="room.id" :value="room.id">
        {{ room.name }} - {{ room.secondary_name }}
      </option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoomsStore } from '../stores/rooms'

const props = defineProps<{ modelValue: number[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()
const roomsStore = useRoomsStore()

const availableRooms = computed(() =>
  roomsStore.rooms.filter(r => r.type === 'location' && r.bookable && !props.modelValue.includes(r.id))
)

function getRoomName(id: number): string {
  const room = roomsStore.rooms.find(r => r.id === id)
  return room ? `${room.name} - ${room.secondary_name}` : `Room ${id}`
}

function addRoom(event: Event) {
  const select = event.target as HTMLSelectElement
  const id = parseInt(select.value)
  if (id) {
    emit('update:modelValue', [...props.modelValue, id])
    select.value = ''
  }
}

function remove(index: number) {
  const copy = [...props.modelValue]
  copy.splice(index, 1)
  emit('update:modelValue', copy)
}

function moveUp(index: number) {
  if (index === 0) return
  const copy = [...props.modelValue]
  ;[copy[index - 1], copy[index]] = [copy[index], copy[index - 1]]
  emit('update:modelValue', copy)
}

function moveDown(index: number) {
  if (index === props.modelValue.length - 1) return
  const copy = [...props.modelValue]
  ;[copy[index], copy[index + 1]] = [copy[index + 1], copy[index]]
  emit('update:modelValue', copy)
}
</script>
