# Travel Planner Viewer (Go + AngularJS + Leaflet)

A travel itinerary viewer and editor with support for managing multiple trip plans.

Current features:
- Plan list with search and sorting
- Plan detail map with layer switching
- Plan and stop creation, editing, and deletion in admin mode
- Excel import for generating plans, including automatic geocoding by place name

---

## 1. Project Structure

```text
howPlan/
├─ backend/
│  ├─ cmd/server/main.go
│  ├─ internal/
│  ├─ data/
│  │  ├─ index.json
│  │  └─ plans/
│  └─ go.mod
├─ frontend/
│  ├─ index.html
│  └─ app/
├─ start.sh
└─ README.md
```

---

## 2. Starting the App

Use the startup script only. You do not need to run `go run` manually.

Viewer mode:

```bash
./start.sh
```

Admin mode:

```bash
./start.sh true
```

Notes:
- The startup script automatically chooses an available random port between `18080` and `19080`.
- After startup, the page opens automatically in Chrome.
- The frontend and backend are served from the same origin, so there is no separate frontend service to start.

---

## 3. Admin Mode

- Start with `./start.sh true` to enable admin mode.
- You can also add `?admin=true` to the URL to request admin capabilities temporarily, for example: `http://localhost:18xxx/?admin=true#!/plans`.

---

## 4. Data Files

Plan data is written to:
- `backend/data/index.json`
- `backend/data/plans/*.json`

The repository currently includes one real itinerary by default:
- `cebu-bohol-siquijor-moalboal`

---

## 5. Troubleshooting

### 5.1 Port or Permission Issues

If startup fails, check the terminal logs. The script already avoids common port conflicts automatically. If your system blocks port listening, allow the terminal or adjust your security software permissions.

### 5.2 Data Write Failures

Make sure the current user can write to the `backend/data` directory.

### 5.3 Import Geocoding Failures

During Excel import, any place name that cannot be resolved automatically is shown as a warning in the import result. You can manually correct the coordinates on the map afterward.

---

## 6. API Overview

- `GET /api/admin/status`
- `GET /api/plans`
- `POST /api/plans` (admin)
- `GET /api/plans/:id`
- `PUT /api/plans/:id` (admin)
- `DELETE /api/plans/:id` (admin)
- `POST /api/plans/:id/stops` (admin)
- `DELETE /api/plans/:id/stops/:stopId` (admin)
- `GET /api/plans/import/template` (admin)
- `POST /api/plans/import` (admin)
