# WhatsApp API — WAHA-Level, Self-Hosted

A production-ready WhatsApp API server built with **Go** and **WhatsMeow**. No browser automation — pure protocol-level communication. Multi-session, Docker-ready, with advanced anti-ban protection.

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
| **Security** | API Key required, optional JWT, rate limiting (60 req/min) |
| **Observability** | Structured logs (zerolog), Prometheus metrics at `/metrics` |
| **Persistence** | SQLite (default), all sessions auto-reconnect on restart |
| **GUI Dashboard** | Premium React + Vite frontend to manage sessions, webhooks, and test messaging |

---

## Quick Start

### 1. Clone & configure

```bash
cp .env.example .env
# Edit .env and set API_KEY to a strong secret
```

### 2. Run with Docker Compose

```bash
docker compose up -d
```

The server starts on **port 3000**. Data is persisted in the `whatsapp_data` Docker volume.

Verify it's running:
```bash
curl http://localhost:3000/health
```

### 3. Run the GUI Dashboard (Optional)

The project includes a sleek, modern React frontend to easily manage your API visually.

```bash
cd frontend
npm install
npm run dev
```

The GUI will start on `http://localhost:5173`.
Open it in your browser, enter your `API_KEY`, and you can:

- **Manage Sessions:** Generate QR codes, connect devices, and disconnect.
- **Messaging:** Send text and media using the built-in anti-ban queue.
- **Webhooks:** Register and monitor webhook endpoints.
- **API Tester:** Dispatch custom JSON payloads to any server endpoint.

---

## API Reference

All endpoints require the header:
```
X-API-Key: your-api-key
```

### Sessions

| Method | Endpoint | Description |
|---|---|---|
| POST | `/sessions` | Create session `{"id":"my-session"}` |
| GET | `/sessions` | List all sessions |
| GET | `/sessions/{id}` | Get session status |
| POST | `/sessions/{id}/login` | Generate QR code (base64 PNG) |
| POST | `/sessions/{id}/logout` | Logout session |
| DELETE | `/sessions/{id}` | Delete session |

**Create and link a session:**
```bash
# Create
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

### Messaging

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
| GET | `/messages/{tracking_id}/status` | Check message delivery status |

**Send a text message (queued with anti-ban delays):**
```bash
curl -X POST http://localhost:3000/messages/text \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "session": "session1",
    "to": "15551234567@s.whatsapp.net",
    "text": "Hello from WhatsApp API!"
  }'
# Returns: { "tracking_id": "uuid", "status": "queued" }
```

**Send an image from URL:**
```bash
curl -X POST http://localhost:3000/messages/media \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "session": "session1",
    "to": "15551234567@s.whatsapp.net",
    "media_type": "image",
    "url": "https://example.com/photo.jpg",
    "caption": "Check this out!"
  }'
```

**Send media via multipart upload:**
```bash
curl -X POST http://localhost:3000/messages/media \
  -H "X-API-Key: your-key" \
  -F "session=session1" \
  -F "to=15551234567@s.whatsapp.net" \
  -F "media_type=image" \
  -F "caption=Hello" \
  -F "file=@/path/to/image.jpg"
```

---

### Queue

```bash
# All session queues
GET /queue?api_key=your-key

# Specific session
GET /queue/session1?api_key=your-key
# Returns: { "session_id": "session1", "pending": 3, "workers": 1 }
```

---

### Groups

```bash
# Create group
POST /groups
{ "session": "session1", "name": "My Group", "participants": ["15551234567@s.whatsapp.net"] }

# Add/remove/promote/demote
POST /groups/{jid}/participants
{ "session": "session1", "action": "add", "participants": ["..."] }

# Leave group
DELETE /groups/{jid}/leave?session=session1
```

---

### Contacts

```bash
# Check if numbers are on WhatsApp
POST /contacts/exists?session=session1
{ "phones": ["+15551234567", "+15557654321"] }

# Block/unblock
POST /contacts/{jid}/block?session=session1
POST /contacts/{jid}/unblock?session=session1

# Profile photo
GET /contacts/{jid}/photo?session=session1
```

---

### Webhooks

```bash
# Register a webhook
curl -X POST http://localhost:3000/webhooks \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session1",
    "url": "https://your-server.com/webhook",
    "secret": "your-hmac-secret",
    "events": ["message.received", "session.connected", "message.delivered"]
  }'

# Delete a webhook
DELETE /webhooks/{id}
```

**Incoming webhook payload:**
```json
{
  "event": "message.received",
  "session": "session1",
  "ts": 1710000000,
  "data": {
    "id": "msg-id",
    "from": "15551234567@s.whatsapp.net",
    "chat": "15551234567@s.whatsapp.net",
    "timestamp": "2024-03-10T12:00:00Z",
    "is_from_me": false,
    "type": "text"
  }
}
```

Webhooks include the `X-WhatsApp-Signature: sha256=<hmac>` header for validation.

**Supported events:**
- `message.received`, `message.sent`, `message.delivered`, `message.read`
- `message.reaction`, `session.connected`, `session.disconnected`
- `media.received`, `group.updated`

---

### System

```bash
GET /health      # { "status": "ok", "version": "1.0.0" }
GET /metrics     # Prometheus metrics
```

---

## Anti-Ban Configuration

Tune these environment variables to balance speed vs. safety:

| Variable | Default | Description |
|---|---|---|
| `QUEUE_BASE_DELAY` | `4s` | Base delay between messages |
| `QUEUE_JITTER_MIN` | `1s` | Minimum extra jitter added |
| `QUEUE_JITTER_MAX` | `3s` | Maximum extra jitter added |
| `MAX_MESSAGES_PER_MINUTE` | `20` | Rate cap per session |
| `TYPING_ENABLED` | `true` | Send composing presence before messages |
| `TYPING_CHARS_PER_SECOND` | `12` | Typing speed (affects delay length) |
| `PRESENCE_ENABLED` | `true` | Presence state simulation |

**New contact behavior:** contacts with 0 prior messages get an extra 6s delay + slow queue lane automatically.

**Auto-slowdown:** when send errors occur, the base delay is increased by 1s per error to back off.

---

## Example Webhook Receiver (Node.js)

```js
const express = require("express");
const crypto = require("crypto");
const app = express();

app.use(express.json());

app.post("/webhook", (req, res) => {
  // Validate HMAC signature
  const sig = req.headers["x-whatsapp-signature"];
  const expected = "sha256=" + crypto
    .createHmac("sha256", "your-hmac-secret")
    .update(JSON.stringify(req.body))
    .digest("hex");

  if (sig !== expected) {
    return res.status(401).send("Invalid signature");
  }

  const { event, session, data } = req.body;
  console.log(`[${session}] ${event}:`, data);
  res.sendStatus(200);
});

app.listen(4000, () => console.log("Webhook receiver on :4000"));
```

---

## Directory Structure

```
whatsapp-api/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── config/config.go            # Configuration loader
│   ├── db/                         # SQLite schema + queries
│   ├── fingerprint/                # Device & network randomization
│   ├── antiban/                    # Jitter, typing, presence, adaptive timing
│   ├── queue/                      # Per-session FIFO queue + contextual queue
│   ├── session/                    # WhatsMeow multi-session manager
│   ├── webhook/                    # Dispatcher with HMAC + retry
│   ├── messaging/                  # Message builders (text/media/special)
│   ├── groups/                     # Group operations
│   ├── contacts/                   # Contact operations
│   ├── channels/                   # Channel (newsletter) operations
│   ├── chats/                      # Chat management
│   ├── auth/                       # API key + JWT middleware
│   └── api/                        # Gin router + all HTTP handlers
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── frontend/                   # Modern React + Vite Dashboard
    ├── src/
    │   ├── api.js              # Centralized API fetcher
    │   ├── App.jsx             # Main layout and routing
    │   ├── index.css           # Premium vanilla CSS styling
    │   └── components/         # GUI Components (Sessions, Messaging, Webhooks, Tester)
    ├── package.json
    └── vite.config.js
```

---

## License

MIT
