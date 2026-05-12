import type { BookingWish } from './stores/bookings'
import type { RecurringSchedule } from './stores/recurrences'
import type { Room } from './stores/rooms'
import { useAuthStore } from './stores/auth'

const API_BASE = '/api'

function getToken(): string {
  return localStorage.getItem('app_token') || ''
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`,
      ...options.headers,
    },
  })
  if (resp.status === 401) {
    const auth = useAuthStore()
    auth.logout()
    throw new Error('Unauthorized (401)')
  }
  if (!resp.ok) {
    throw new Error(`API error: ${resp.status}`)
  }
  if (resp.status === 204) return undefined as T
  return resp.json()
}

export const api = {
  getBookings: () => request<BookingWish[]>('/bookings'),
  createBooking: (data: Partial<BookingWish>) => request<{ id: string }>('/bookings', { method: 'POST', body: JSON.stringify(data) }),
  deleteBooking: (id: string) => request<void>(`/bookings/${id}`, { method: 'DELETE' }),
  getRecurrences: () => request<RecurringSchedule[]>('/recurrences'),
  createRecurrence: (data: Partial<RecurringSchedule>) => request<{ id: string }>('/recurrences', { method: 'POST', body: JSON.stringify(data) }),
  updateRecurrence: (id: string, data: Partial<RecurringSchedule>) => request<void>(`/recurrences/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteRecurrence: (id: string) => request<void>(`/recurrences/${id}`, { method: 'DELETE' }),
  getRooms: () => request<Room[]>('/rooms'),
  getStatus: () => request<{ asimut_connected: boolean }>('/settings/status'),
}
