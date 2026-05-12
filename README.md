# Asimut Room Booking Bot

Automatically books practice rooms on Asimut (hfm-freiburg.asimut.net) at the exact moment they become available.

## Docker Images

| Container | Image | Port | Purpose |
|-----------|-------|------|---------|
| Backend | `philippgehrig/asimut-bot-backend` | 8080 | Go API, scheduler, SQLite database |
| Frontend | `philippgehrig/asimut-bot-frontend` | 3000 | nginx serving Vue SPA, proxies `/api` to backend |

## Setup (Docker Compose)

1. Copy `.env.example` to `.env` and fill in your credentials
2. Run `docker compose up -d`
3. Open `http://localhost:3000` and enter your app password

## Setup (Unraid / Individual Containers)

Both containers need to be on the same Docker network so the frontend can reach the backend.

### 1. Create a custom network

```bash
docker network create asimut
```

### 2. Start the backend

```bash
docker run -d \
  --name asimut-backend \
  --network asimut \
  -p 8080:8080 \
  -v /mnt/user/appdata/asimut:/data \
  -e ASIMUT_EMAIL=your.email@mh-freiburg.de \
  -e ASIMUT_PASSWORD=your_password \
  -e APP_PASSWORD=choose_a_web_ui_password \
  -e DATABASE_PATH=/data/asimut.db \
  --restart unless-stopped \
  philippgehrig/asimut-bot-backend:latest
```

### 3. Start the frontend

The frontend nginx proxies `/api` requests to `http://backend:8080`. On Unraid, the container name must be `backend` OR you need to set a network alias:

```bash
docker run -d \
  --name asimut-frontend \
  --network asimut \
  --network-alias frontend \
  -p 3000:3000 \
  --restart unless-stopped \
  philippgehrig/asimut-bot-frontend:latest
```

The backend container needs the network alias `backend`:

```bash
docker network connect --alias backend asimut asimut-backend
```

Alternatively, if your Unraid setup uses a reverse proxy (like Nginx Proxy Manager), you can expose only the backend port and have your proxy handle routing.

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ASIMUT_EMAIL` | Yes | Login email for hfm-freiburg.asimut.net |
| `ASIMUT_PASSWORD` | Yes | Login password |
| `APP_PASSWORD` | Yes | Password to access the web UI |
| `DATABASE_PATH` | No | SQLite file path (default: `/data/asimut.db`) |
| `PORT` | No | Backend port (default: `8080`) |

### Volumes

| Container | Mount | Purpose |
|-----------|-------|---------|
| Backend | `/data` | SQLite database persistence |

## Development

### Backend
```bash
cd backend
go run .
```

### Frontend
```bash
cd frontend
yarn install
yarn dev
```

## How it works

1. Create a booking wish via the web UI (date, time, duration, room priority list)
2. The scheduler calculates when the booking window opens (47h30m advance: at time T you can book the slot starting at T-30min, two days from now)
3. At the exact trigger time, the bot logs into Asimut and books the first available room from your priority list
4. If the initial 30-min slot is booked, it immediately extends in 15-min increments until your desired duration is reached
