# Asimut Room Booking Bot — Design Spec

## Problem

The Hochschule für Musik Freiburg uses Asimut (`hfm-freiburg.asimut.net`) for room booking. Rooms are bookable on a rolling 48-hour window, creating a race condition where students camp the app to grab slots the moment they open. The user's sister wants a bot that wins that race for her.

## Solution

A self-hosted web application that lets her define booking wishes in advance. A scheduler fires precise HTTP requests to Asimut at the exact moment the 48-hour window opens, securing rooms faster than any human can.

## Architecture

```
Docker Compose
├── frontend (Vue.js SPA, static file server, port 3000)
└── backend  (Go binary: REST API + scheduler + Asimut client + SQLite, port 8080)
```

The user's existing nginx instance handles external routing/proxying to these containers.

### Backend (Go)

**Components:**
- **REST API** — serves the frontend, CRUD for booking wishes and recurring schedules
- **Scheduler** — calculates trigger times, fires jobs at the exact moment
- **Asimut HTTP Client** — reverse-engineered HTTP requests mimicking browser behavior (login, book, extend)
- **SQLite** — persistent storage for booking wishes and results

### Frontend (Vue.js)

- Vue 3 + TypeScript
- Tailwind CSS
- Yarn package manager
- Mobile-first responsive design

## Authentication

Simple password protection:
- Single shared password configured via environment variable
- Frontend prompts for password on first visit
- Backend validates on every API request (session-based)

## Data Model

### Booking Wish

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| date | date | Desired booking date |
| start_time | time | Desired start time |
| duration_minutes | int | Total desired duration |
| room_priorities | JSON array | Ordered list of room IDs to try |
| recurrence_id | UUID (nullable) | Link to recurring schedule |
| status | enum | `pending` / `scheduled` / `booked` / `partially_booked` / `failed` |
| result_room | string (nullable) | Room that was actually booked |
| result_duration | int (nullable) | Actual duration secured (minutes) |
| failure_reason | string (nullable) | Why it failed |
| created_at | timestamp | |
| updated_at | timestamp | |

### Recurring Schedule

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| day_of_week | int | 0=Mon, 6=Sun |
| start_time | time | Desired start time |
| duration_minutes | int | Total desired duration |
| room_priorities | JSON array | Ordered list of room IDs |
| active | bool | Can be paused without deletion |
| created_at | timestamp | |

The backend auto-generates individual booking wishes from recurring schedules on a rolling basis (keeps the next 4 weeks populated). Individual occurrences can be cancelled without affecting the series.

## Asimut Interaction

### Session Management

- Pre-authenticate a few seconds before trigger time to have a valid session ready
- Handle session expiry and re-login

### Booking Sequence (at trigger time)

1. Iterate through room priority list
2. For the first available room, book the initial 30-minute slot
3. Attempt extensions in 15-minute increments until desired duration is filled
4. Store result

### Extension Logic

The 30+15+15... pattern:
- Initial booking: 30 minutes
- Each extension: 15 minutes
- Total extensions needed: `(duration - 30) / 15`

Note: The exact extension mechanism (whether extensions can be booked immediately or require waiting) is TBD and will be confirmed during reverse-engineering. The system will be designed so this logic is easy to adjust.

## Scheduler Logic

**Trigger calculation:**
- Desired time: Wednesday 14:00
- Trigger time: Monday 14:00 (48 hours before)

**Precision:**
- Job wakes up slightly early (~500ms)
- Busy-waits with time checks until the exact second
- Fires request immediately

**Failure handling:**
- All rooms taken → status `failed`, stores reason
- Partial success (got room but not full duration) → status `partially_booked`, stores actual duration

## Frontend Pages

### 1. Dashboard
- Upcoming booking wishes with status indicators
- Recent results (success/failure/partial)
- Quick overview of active recurring schedules

### 2. Create Booking
- Date + time picker (for one-time bookings)
- Duration selector
- Room priority list with drag-to-reorder
- Recurrence toggle: one-time / weekly
- For recurring: day-of-week selector + time

### 3. Room List
- Browse/search available rooms from Asimut
- Select rooms to build priority lists

### 4. Settings
- Asimut connection status check
- View/test credentials connectivity

## Deployment

**Docker Compose** with two services:
- `frontend`: builds Vue SPA, serves via lightweight static server
- `backend`: builds Go binary, runs with SQLite volume mount

**Environment variables:**
- `ASIMUT_EMAIL` — login email
- `ASIMUT_PASSWORD` — login password
- `APP_PASSWORD` — web UI access password
- `DATABASE_PATH` — SQLite file location (default: `/data/asimut.db`)

## Open Questions

- Exact Asimut API endpoints (to be reverse-engineered)
- Whether extensions can be booked immediately or require waiting
- Room list: whether it can be pulled from Asimut or needs manual configuration
