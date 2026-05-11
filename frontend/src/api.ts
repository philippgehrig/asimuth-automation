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
    localStorage.removeItem('app_token')
    window.location.reload()
    throw new Error('Unauthorized')
  }
  if (!resp.ok) {
    throw new Error(`API error: ${resp.status}`)
  }
  if (resp.status === 204) return undefined as T
  return resp.json()
}

export const api = {
  getBookings: () => request<any[]>('/bookings'),
  createBooking: (data: any) => request<{ id: string }>('/bookings', { method: 'POST', body: JSON.stringify(data) }),
  deleteBooking: (id: string) => request<void>(`/bookings/${id}`, { method: 'DELETE' }),
  getRecurrences: () => request<any[]>('/recurrences'),
  createRecurrence: (data: any) => request<{ id: string }>('/recurrences', { method: 'POST', body: JSON.stringify(data) }),
  updateRecurrence: (id: string, data: any) => request<void>(`/recurrences/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteRecurrence: (id: string) => request<void>(`/recurrences/${id}`, { method: 'DELETE' }),
  getRooms: () => request<any[]>('/rooms'),
  getStatus: () => request<{ asimut_connected: boolean }>('/settings/status'),
}
