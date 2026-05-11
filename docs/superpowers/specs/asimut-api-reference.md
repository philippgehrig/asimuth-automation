# Asimut API Reference (Reverse-Engineered)

Base URL: `https://hfm-freiburg.asimut.net`

## Authentication

**Login:** `POST /public/login.php`

Content-Type: `application/x-www-form-urlencoded`

Form params:
- `authenticate-url`: `%2Fpublic%2Fhfm-freiburg.asimut.net`
- `authenticate-useraccount`: email (e.g., `k.gehrig@mh-freiburg.de`)
- `authenticate-password`: password
- `authenticate-verification`: `ok`

Response: HTTP 302 redirect to `/public/hfm-freiburg.asimut.net`

Session is maintained via PHP session cookie (HTTP-only, not visible in HAR exports). All subsequent API calls use the same session.

## Session Validation

**Heartbeat:** `GET /services/v2/heartbeat/me`

Returns user info and booking constraints:
```json
{
  "response": {
    "heartbeat": {
      "loggedin": true,
      "booking_enabled": true,
      "time_rounding_m": 15,
      "booking_open": "00:00",
      "booking_closes": "24:00",
      "me": {
        "id": 965,
        "name": "Klara",
        "surname": "Gehrig",
        "username": "[BM]Ob956",
        "useraccount": "K.Gehrig@mh-freiburg.de",
        "minimum_booking_length": 30,
        "maximum_booking_length": 180,
        "minimum_booking_gap": 60,
        "booking_horizon": "2026-05-13T22:04:00+02:00"
      }
    }
  }
}
```

Key constraints:
- `minimum_booking_length`: 30 minutes
- `maximum_booking_length`: 180 minutes (3 hours max)
- `minimum_booking_gap`: 60 minutes between bookings
- `time_rounding_m`: 15 minutes (slots snap to 15-min increments)
- `booking_horizon`: the furthest point in time that can currently be booked

## Room/Location Endpoints

**List all locations:** `GET /services/v2/locations`

```json
{
  "response": {
    "locations": [
      {
        "id": 114,
        "name": "MBP-326",
        "secondary_name": "Klarinette, Holzbläser / Bläser-Korrepetition, Hauptgebäude, 32 m²",
        "bookable": true,
        "type": "location"
      },
      {
        "id": 50,
        "name": "_MTh- Kmp-Mw-Mpä-MPh-EMP (FG1)",
        "secondary_name": "",
        "bookable": true,
        "type": "group"
      }
    ]
  }
}
```

Types: `"location"` (actual rooms) and `"group"` (categories).

**Location info:** `GET /services/v2/locations/location_ids=114;current_date=2026-05-13T20:15:00.000+02:00/info`

Returns availability and details for a specific room at a specific time.

**Location groups:** `GET /services/v2/locationgroups`

**Locations by IDs:** `GET /services/v2/locations/location_ids=1,2,3,4,...`

Returns schedule/occupancy for multiple locations (used for the calendar view).

## Booking Flow

### Step 1: Check FAB menu (pre-flight)

`POST /services/v2/fabmenu/`

```json
{
  "categoryIds": "1,2",
  "startDate": "2026-05-13T20:15:00.000+02:00",
  "locationId": 114
}
```

Response indicates if booking is possible (icons with warning state).

### Step 2: Get event defaults

`POST /services/v2/eventdefault`

```json
{
  "st": "2026-05-13T20:15:00.000+02:00",
  "ca": 1,
  "rs": [{"id": 114}]
}
```

Response provides a pre-filled event object with user info, room details, and default end time (start + 60 min for "Einzelüben" category).

Key response fields:
- `ar`: "Einzelüben" (activity/reason)
- `ca`: 1 (category ID for practice)
- `st`: start time
- `en`: end time (auto-calculated)
- `rs`: rooms array
- `pe`: persons array (auto-filled with logged-in user)
- `ps`: person slots with role info

### Step 3: Check booking validity

`POST /services/v2/event/type=check`

Sends the full event object. Response:
```json
{
  "response": {
    "bookingrules": {
      "issues": [
        {
          "class": "message-info",
          "text": "Ihre Buchung ist vorläufig. Die Buchung kann ab 60 Minuten vor bis 5 Minuten nach Beginn bestätigt werden.",
          "type": "general"
        }
      ],
      "clashing_person_ids": []
    },
    "event_ids": [0],
    "success": true
  }
}
```

Note: "Ihre Buchung ist vorläufig" = "Your booking is provisional. The booking can be confirmed from 60 minutes before to 5 minutes after the start."

### Step 4: Save booking

`POST /services/v2/event/type=save`

Same payload as check. Response:
```json
{
  "response": {
    "event_ids": [470262],
    "success": true
  }
}
```

The returned `event_id` (470262) is needed for extensions.

## Extension Flow

Extensions use PATCH on the existing event, changing only the `en` (end) field.

### Step 1: Check extension

`PATCH /services/v2/event/event_id=470262;type=check`

Payload is the full event with updated `en` time. The extension observed was:
- Original booking: 20:15 – 21:15 (60 min initial, since eventdefault returned 60min)
- First extension check: `en` changed to `21:15:00` (this was the original — confirming the check)
- Second extension check: `en` changed to `21:30:00` (+15 min)

### Step 2: Save extension

`PATCH /services/v2/event/event_id=470262;type=save`

Same payload as check with the new end time. Success response same as booking.

**Extension pattern:** Each extension adds 15 minutes to the end time. Extensions can be done immediately after the initial booking (no waiting required based on the HAR evidence — both initial booking and extension happened in the same session).

## Other Endpoints

- `GET /services/v2/eventgroups/category_id=1` — list event groups for a category
- `GET /services/v2/quota/date=2026-05-13T00:00:00.000+02:00` — daily booking quota
- `GET /services/v2/event/event_id=470262` — get event details
- `GET /services/v2/event/event_id=470262;type=links` — get event links
- `GET /services/v2/arrangement/event_id=470262;direction=forward;load_from=` — arrangement view
- `GET /services/v2/categories` — list all categories
- `GET /services/v2/roles` — list roles
- `GET /services/v2/menu/type=main` — main menu structure

## Request Headers

All API requests use:
```
Accept: application/json, text/plain, */*
Content-Type: application/json
Origin: https://hfm-freiburg.asimut.net
Referer: https://hfm-freiburg.asimut.net/...
```

No explicit Authorization header — session is cookie-based (PHPSESSID).

## Event Object Schema

```json
{
  "id": 0,
  "ac": "",
  "ai": 0,
  "ar": "Einzelüben",
  "ca": 1,
  "de": "<p>description</p>",
  "en": "2026-05-13T21:15:00+02:00",
  "ev": "",
  "li": "",
  "pr": 0,
  "cl": [],
  "ri": {
    "e": true,
    "c": false,
    "r": true,
    "p": true,
    "a": false
  },
  "rs": [
    {
      "id": 114,
      "dn": "MBP-326 (Klarinette, Holzbläser / Bläser-Korrepetition, Hauptgebäude, 32 m²)"
    }
  ],
  "st": "2026-05-13T20:15:00+02:00",
  "vi": "visible",
  "pe": [
    {
      "id": 965,
      "ro": 1,
      "dn": "Teiln: Klara Gehrig ([BM]Ob956)",
      "ac": []
    }
  ],
  "ps": [
    {
      "me": false,
      "ri": 1,
      "rs": "Teilnehmer*in",
      "rh": "Teilnehmer*in",
      "ii": false,
      "is": null,
      "rc": 1,
      "bo": [
        {
          "id": 965,
          "fn": "Klara",
          "ln": "Gehrig",
          "un": "([BM]Ob956)"
        }
      ]
    }
  ]
}
```

Field key (inferred):
- `id`: event ID (0 for new)
- `ai`: arrangement ID
- `ar`: activity reason
- `ca`: category ID
- `de`: description (HTML)
- `en`: end time
- `st`: start time
- `rs`: rooms (resources)
- `pe`: persons
- `ps`: person slots
- `ri`: rights/permissions
- `vi`: visibility
- `cl`: classes/labels
- `pr`: priority

## Booking Constraints Summary

- Initial booking: minimum 30 min, maximum 180 min
- Time rounding: 15-minute increments
- Booking gap: minimum 60 min between user's bookings
- Extensions: +15 min increments via PATCH
- Booking is provisional until confirmed (60 min before to 5 min after start)
- Advance window: ~27.5 hours (booking_horizon confirms this)
