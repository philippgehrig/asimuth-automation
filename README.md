# Asimut Room Booking Bot

Automatically books practice rooms on Asimut (hfm-freiburg.asimut.net) at the exact moment they become available.

## Setup

1. Copy `.env.example` to `.env` and fill in your credentials
2. Run `docker compose up -d`
3. Open `http://localhost:3000` and enter your app password

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
2. The scheduler calculates when the booking window opens (27.5h advance window)
3. At the exact trigger time, the bot logs into Asimut and books the first available room from your priority list
4. If the initial 30-min slot is booked, it immediately extends in 15-min increments until your desired duration is reached
