# WhatsApp API — WAHA-Level, Self-Hosted

A production-ready WhatsApp API server built with **Go** and **WhatsMeow**. No browser automation — pure protocol-level communication. Multi-session, Docker-ready, with advanced anti-ban protection, **role-based access control**, and a built-in **agent chat UI**.

---

## Features

| Feature | Details |
|---|---|
| **Multi-Session** | Independent sessions, each with isolated queue and webhook |
| **Anti-Ban Layer 1** | Per-session FIFO queue with jitter delays (2–6s + ±1–3s) |
| **Anti-Ban Layer 2** | Typing simulation + Presence simulation (composing/recording/paused) |
| **Anti-Ban Layer 3** | Contextual queues: new contacts slow lane, frequent contacts fast lane; contact cooldown; duplicate detection; auto-slowdown on errors |
| **Device Fingerprint** | Random device model, OS, app version per session |
| **Network Jitter** | 50–300ms latency simulation per operation |
| **Adaptive Timing** | Slower at night, faster for active chats, longer for long messages |
| **Webhooks** | HMAC-SHA256 signed, retry with backoff, per-event filtering |
| **Messaging** | Text, Image, Video, Audio, Document, Voice Note, Sticker, Location, vCard, Reaction, Poll, Reply, Delete, Edit |
| **Groups** | Create, join, leave, add/remove/promote/demote, set name/description |
| **Contacts** | List, check existence, block/unblock, profile photo, status |
| **Chats** | List, history, mark read, mute, pin, archive |
| **Channels** | List, get info, send message |
| **Authentication** | Login-based (username + password → JWT), API Key for system routes, rate limiting (60 req/min) |
| **Roles** | Admin (full control) and Agent (chat UI only — assigned chats) |
| **Auto-Assignment** | New incoming chats are automatically assigned to a random available agent |
| **Agent Chat UI** | WhatsApp-like interface — phone number display, message bubbles, text/voice/document send |
| **Real-Time Sync** | Server-Sent Events (SSE) push new messages and new chats to agents instantly |
| **Persistence** | SQLite (default), all sessions auto-reconnect on restart |

---

## Quick Start

### 1. Clone & configure

```bash
cp .env.example .env
# Edit .env — set API_KEY, ADMIN_PASSWORD, USER_JWT_SECRET
```

### 2. Run with Docker Compose

```bash
docker compose up -d
```

The API starts on **port 3000**. The frontend starts on **port 5173**. Data is persisted in the `whatsapp_data` Docker volume.

Verify it's running:
```bash
curl http://localhost:3000/health
```

### 3. Open the Dashboard

Go to **http://localhost:5173** and log in with your admin credentials (default: `admin` / `admin123` — change in `.env`).

---

## Roles

### Admin
Logs into the full dashboard. Can:
- Manage WhatsApp sessions (connect, disconnect, QR code scan)
- Send messages, manage webhooks, test the API
- **Create/delete users** (admin or agent role)
- **Assign chats** from any session to any agent
- View all configuration settings

### Agent
Logs into a **WhatsApp-like chat UI**. Can:
- See only their assigned chats (phone numbers shown, not names)
- Read message history per chat
- Send text messages (Enter to send)
- Record and send voice notes (MediaRecorder API)
- Attach and send files (images, videos, documents — auto-detected)
- Receive new messages and new chat assignments in **real-time** via SSE

### Auto-Assignment
When a new incoming chat arrives that has no agent assigned, the system automatically picks a random agent from the agent pool and assigns it. The agent is notified instantly via SSE.

---

## Environment Variables

### Required

| Variable | Default | Description |
|---|---|---|
| `API_KEY` | `changeme` | API key for system/admin routes |
| `ADMIN_USERNAME` | `admin` | Initial admin account username |
| `ADMIN_PASSWORD` | `admin123` | Initial admin account password |
| `USER_JWT_SECRET` | `changeme-user-jwt-secret-32chars` | Secret for signing user login JWTs |

> [!IMPORTANT]
> Change `ADMIN_PASSWORD` and `USER_JWT_SECRET` before deploying. The admin account is only seeded once on first run.

### Optional

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | Server port |
| `DB_DRIVER` | `sqlite3` | Database driver |
| `DB_DSN` | `/app/data/whatsapp.db` | Database path |
| `JWT_ENABLED` | `false` | Enable JWT for legacy API-key routes |
| `JWT_SECRET` | `changeme-jwt-secret` | Secret for legacy JWT |
| `MEDIA_STORAGE_PATH` | `/app/data/media` | Media file storage path |

### Anti-Ban Tuning

| Variable | Default | Description |
|---|---|---|
| `QUEUE_BASE_DELAY` | `4s` | Base delay between messages |
| `QUEUE_JITTER_MIN` | `1s` | Minimum extra jitter |
| `QUEUE_JITTER_MAX` | `3s` | Maximum extra jitter |
| `MAX_MESSAGES_PER_MINUTE` | `20` | Rate cap per session |
| `TYPING_ENABLED` | `true` | Send composing presence before messages |
| `TYPING_CHARS_PER_SECOND` | `12` | Typing speed (affects delay length) |
| `PRESENCE_ENABLED` | `true` | Presence state simulation |

---

## API Reference

### Authentication (Login)

User-facing login — no API key required.

```bash
POST /auth/login
{ "username": "admin", "password": "admin123" }
# Returns: { "token": "<jwt>", "role": "admin", "user_id": "...", "username": "admin" }
```

Use the returned `token` as a `Bearer` token on all admin/agent endpoints:
```
Authorization: Bearer <token>
```

---

### System Routes

These use the `X-API-Key` header (the existing API key).

```bash
GET /health      # { "status": "ok" }
GET /metrics     # Prometheus metrics
```

---

### Sessions (Admin — API Key)

| Method | Endpoint | Description |
|---|---|---|
| POST | `/sessions` | Create session `{"id":"my-session"}` |
| GET | `/sessions` | List all sessions |
| GET | `/sessions/{id}` | Get session status |
| POST | `/sessions/{id}/login` | Generate QR code (base64 PNG) |
| POST | `/sessions/{id}/logout` | Logout session |
| DELETE | `/sessions/{id}` | Delete session |

```bash
# Create and link a session
curl -X POST http://localhost:3000/sessions \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"id": "session1"}'

# Get QR code
curl -X POST http://localhost:3000/sessions/session1/login \
  -H "X-API-Key: your-key"
# Returns: { "qr_code": "<base64-png>", "qr_raw": "<qr-string>" }
# Decode qr_code from base64 and scan it with WhatsApp
```

---

### Messaging (Admin / API Key)

| Method | Endpoint | Description |
|---|---|---|
| POST | `/messages/text` | Send text message |
| POST | `/messages/media` | Send image/video/audio/document/sticker |
| POST | `/messages/location` | Send location |
| POST | `/messages/contact` | Send vCard contact |
| POST | `/messages/reaction` | Send emoji reaction |
| POST | `/messages/poll` | Send poll |
| DELETE | `/messages` | Delete/revoke message |
| PATCH | `/messages` | Edit sent message |
| GET | `/messages/{tracking_id}/status` | Check delivery status |

```bash
curl -X POST http://localhost:3000/messages/text \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"session":"session1","to":"15551234567@s.whatsapp.net","text":"Hello!"}'
# Returns: { "tracking_id": "uuid", "status": "queued" }
```

---

### User Management (Admin — JWT)

All routes require `Authorization: Bearer <admin-token>`.

| Method | Endpoint | Description |
|---|---|---|
| GET | `/admin/users` | List all users |
| POST | `/admin/users` | Create user `{username, password, role}` |
| DELETE | `/admin/users/:id` | Delete user |
| POST | `/admin/users/:id/chats` | Assign chats to agent `{session_id, jids:[...]}` |
| GET | `/admin/users/:id/chats` | List assigned chats |
| DELETE | `/admin/users/:id/chats/:chatid` | Remove a chat assignment |

```bash
# Create an agent
curl -X POST http://localhost:3000/admin/users \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"agent1","password":"pass123","role":"agent"}'

# Assign chats to that agent
curl -X POST http://localhost:3000/admin/users/<agent-id>/chats \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"session_id":"session1","jids":["15551234567@s.whatsapp.net"]}'
```

---

### Agent Routes (Agent — JWT)

All routes require `Authorization: Bearer <agent-token>`. Agents can only access their own assigned chats.

| Method | Endpoint | Description |
|---|---|---|
| GET | `/agent/chats` | List assigned chats |
| GET | `/agent/chats/:jid/messages?session=` | Get message history |
| POST | `/agent/chats/:jid/send/text` | Send text `{session_id, text}` |
| POST | `/agent/chats/:jid/send/media` | Send file (multipart: session_id, media_type, file) |
| GET | `/agent/events?token=` | SSE stream — real-time events (`new_chat`, `new_message`) |

---

### Webhooks (Admin — API Key)

```bash
# Register a webhook
curl -X POST http://localhost:3000/webhooks \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session1",
    "url": "https://your-server.com/webhook",
    "secret": "your-hmac-secret",
    "events": ["message.received", "session.connected"]
  }'

# Delete a webhook
DELETE /webhooks/{id}
```

Webhooks include an `X-WhatsApp-Signature: sha256=<hmac>` header for validation.

**Supported events:** `message.received`, `message.sent`, `message.delivered`, `message.read`, `message.reaction`, `session.connected`, `session.disconnected`, `media.received`, `group.updated`

---

## Directory Structure

```
whatsapp-api/
├── cmd/server/main.go              # Entry point (seeds admin account)
├── internal/
│   ├── config/config.go            # Configuration loader
│   ├── db/                         # SQLite schema + queries (users, agent_chats, sessions…)
│   ├── fingerprint/                # Device & network randomization
│   ├── antiban/                    # Jitter, typing, presence, adaptive timing
│   ├── queue/                      # Per-session FIFO queue + contextual queue
│   ├── session/                    # WhatsMeow multi-session manager
│   ├── webhook/                    # Dispatcher with HMAC + retry
│   ├── messaging/                  # Message builders (text/media/special)
│   ├── groups/                     # Group operations
│   ├── contacts/                   # Contact operations
│   ├── channels/                   # Channel (newsletter) operations
│   ├── chats/                      # Chat management + auto-assignment logic
│   ├── auth/                       # API key, JWT (system + user), bcrypt, middleware
│   └── api/                        # Gin router + all HTTP handlers + SSE hub
├── Dockerfile
├── docker-compose.yml
├── .env
└── frontend/                       # React + Vite frontend
    ├── src/
    │   ├── api.js                  # Centralized API fetcher (API key + JWT)
    │   ├── App.jsx                 # Role-based routing (Login → Admin or Agent)
    │   ├── index.css               # Premium vanilla CSS styling
    │   ├── context/
    │   │   └── AuthContext.jsx     # Auth state (login, logout, JWT storage)
    │   └── components/
    │       ├── Login.jsx           # Login page (glassmorphism)
    │       ├── Dashboard.jsx       # Admin overview
    │       ├── SessionsManager.jsx # Session management UI
    │       ├── Messaging.jsx       # Messaging test UI
    │       ├── WebhooksManager.jsx # Webhooks management UI
    │       ├── ApiTester.jsx       # Raw API request tester
    │       ├── ApiSettings.jsx     # API key settings
    │       ├── admin/
    │       │   ├── UserManagement.jsx   # Create/delete admin & agent users
    │       │   └── ChatAssignment.jsx   # Assign chats to agents
    │       └── agent/
    │           ├── ChatUI.jsx           # WhatsApp-like agent interface
    │           ├── MessageThread.jsx    # Message bubbles (in/out)
    │           └── MessageInput.jsx     # Text / voice / file send bar
    ├── package.json
    └── vite.config.js
```

---

## Example Webhook Receiver (Node.js)

```js
const express = require("express");
const crypto = require("crypto");
const app = express();

app.use(express.json());

app.post("/webhook", (req, res) => {
  const sig = req.headers["x-whatsapp-signature"];
  const expected = "sha256=" + crypto
    .createHmac("sha256", "your-hmac-secret")
    .update(JSON.stringify(req.body))
    .digest("hex");

  if (sig !== expected) return res.status(401).send("Invalid signature");

  const { event, session, data } = req.body;
  console.log(`[${session}] ${event}:`, data);
  res.sendStatus(200);
});

app.listen(4000, () => console.log("Webhook receiver on :4000"));
```

---

## License

MIT
