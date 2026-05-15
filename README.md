# AP2 — Assignment 4: Caching Strategies & Background Jobs

## 1. Project Overview

Medical Scheduling Platform — four-service system.

**What changed compared to Assignment 3:**

| Area | Change |
|---|---|
| Redis cache | Doctor Service and Appointment Service now use a Redis-backed caching layer for read operations (Cache-Aside) with Write-Through / Write-Around invalidation on writes |
| Rate limiting | Both gRPC services enforce a per-client-IP sliding-window rate limit via a `UnaryServerInterceptor` backed by Redis |
| Background job queue | Notification Service extended with a worker-pool-based job queue that calls the Mock Notification Gateway on `appointments.status_updated` events with `new_status="done"` |
| Mock Notification Gateway | New fourth binary (`mock-gateway`) simulates an external email/SMS API with idempotency and 20% transient failure rate |
| Event payload | `appointments.status_updated` now includes `doctor_id` (needed by the job contract) |

What did **not** change: domain models, use-case business rules, gRPC contracts, generated proto stubs, PostgreSQL schemas, migration files, NATS integration, Clean Architecture layering.

---

## 2. Architecture Diagram

```
┌──────────────────┐        gRPC         ┌──────────────────────┐
│  Doctor Service  │◄────────────────────│  Appointment Service │
│  :50051          │                     │  :50052              │
│                  │                     │                      │
│  PostgreSQL DB   │                     │  PostgreSQL DB       │
│  Redis (cache,   │                     │  Redis (cache,       │
│  rate limit)     │                     │  rate limit)         │
└────────┬─────────┘                     └──────────┬───────────┘
         │ NATS publish                             │ NATS publish
         │ doctors.created                          │ appointments.created
         │                                          │ appointments.status_updated
         └──────────────────┬───────────────────────┘
                            ▼
                   ┌─────────────────┐
                   │  NATS Broker    │
                   │  :4222          │
                   └────────┬────────┘
                            │ subscribe (all 3 subjects)
                            ▼
               ┌────────────────────────┐       HTTP POST /notify
               │  Notification Service  │──────────────────────────►┌──────────────────┐
               │                        │                            │  Mock Gateway    │
               │  internal/subscriber   │                            │  :8080           │
               │  internal/logger       │◄───────────────────────────│                  │
               │  internal/jobqueue     │    HTTP 200 / 503 (20%)    └──────────────────┘
               │  Redis (idempotency)   │
               └────────────────────────┘
```

**Communication labels:**
- Doctor Service ↔ Appointment Service: gRPC (synchronous, doctor existence check)
- Doctor/Appointment Service → NATS: publish domain events (fire-and-forget)
- NATS → Notification Service: push subscriptions
- Notification Service → Mock Gateway: HTTP POST `/notify` (with retry + backoff)
- All services → Redis: cache reads/writes, rate-limit counters, idempotency keys

---

## 3. Cache Strategy per Endpoint

### Doctor Service

| Operation | Strategy | Key | TTL |
|---|---|---|---|
| `GetDoctor` | **Cache-Aside** — check Redis first; on miss load from PG and populate cache | `doctor:<id>` | `CACHE_TTL_SECONDS` |
| `ListDoctors` | **Cache-Aside** | `doctors:list` | `CACHE_TTL_SECONDS` |
| `CreateDoctor` | **Write-Through (invalidate)** — write to PG, then `DEL doctors:list` | `doctors:list` | immediate eviction |

**Why Cache-Aside for reads:** Read patterns dominate; lazy population avoids caching data that is never requested. The cache is optional — a Redis failure transparently falls through to PostgreSQL.

**Why Write-Through (invalidate) for CreateDoctor:** The list is stale the moment a new doctor is created. Immediate eviction forces the next list read to repopulate from the authoritative DB, keeping the stale-read window at zero.

### Appointment Service

| Operation | Strategy | Key | TTL |
|---|---|---|---|
| `GetAppointment` | **Cache-Aside** | `appointment:<id>` | `CACHE_TTL_SECONDS` |
| `ListAppointments` | **Cache-Aside** | `appointments:list` | `CACHE_TTL_SECONDS` |
| `CreateAppointment` | **Write-Around** — write only to PG, evict list | `appointments:list` | immediate eviction |
| `UpdateAppointmentStatus` | **Write-Through** — update PG, then SET `appointment:<id>` and DEL `appointments:list` | `appointment:<id>`, `appointments:list` | immediate eviction |

**Why Write-Around for CreateAppointment:** Newly created appointments are unlikely to be immediately read individually; writing to the cache on creation wastes memory and adds latency. The list is evicted so the next list call gets fresh data.

**Why Write-Through for UpdateAppointmentStatus:** Status changes (especially to `done`) are immediately observable. Updating the individual cache entry ensures the next `GetAppointment` call returns current data without a round-trip to PG.

---

## 4. Cache Invalidation

- Invalidation always happens **after** the database write succeeds and **before** the gRPC response is returned.
- A cache miss never returns an error — the repository falls through to PostgreSQL transparently.
- A cache write failure is logged at `WARNING` level but does not block the gRPC response (best-effort caching).
- Stale-read window: for `CreateDoctor` and `CreateAppointment`, the list is evicted immediately → zero stale window. For TTL-based expiry on individual records, a stale read can persist for up to `CACHE_TTL_SECONDS` (default 60 s) after a missed invalidation (e.g., if the invalidation `DEL` fails).

---

## 5. Rate-Limiting Algorithm

**Algorithm chosen: Sliding-window counter using a Redis Sorted Set**

**Redis data structure:** `ZSET` with key `ratelimit:{service}:{client_ip}`
- **Member:** monotonically increasing integer (unique per request, per service instance)
- **Score:** request timestamp in nanoseconds (`time.Now().UnixNano()`)

**Per-request logic (executed as a pipeline):**
1. `ZREMRANGEBYSCORE key 0 <(now - 60s) in ns>` — remove entries outside the 1-minute window
2. `ZCARD key` — count remaining entries
3. `EXPIRE key 2m` — keep key alive
4. If count ≥ limit → return `codes.ResourceExhausted` with retry-after message
5. `ZADD key <nowNs> <uniqueMember>` — record this request

**Why sliding window over fixed window:** Avoids the burst problem at window boundaries (a fixed-window allows 2× the limit in two consecutive windows). Sliding window gives a smoother limit.

**Implementation:** `doctor-service/internal/middleware/ratelimit.go` and `appointment-service/internal/middleware/ratelimit.go`. Applied as `grpc.UnaryInterceptor(rl.UnaryServerInterceptor())` — no handler code is modified.

**Rate-limiting trade-offs (per-instance vs centralised):**
1. **Per-instance counting:** Each service replica maintains its own ZSET key per client IP. With N replicas, a client can make `N × RATE_LIMIT_RPM` requests total. This is solved by using a **shared Redis instance** — all replicas share the same key namespace, so the limit is enforced globally.
2. **Clock skew between replicas:** Sorted-set scores are nanosecond timestamps from each host's clock. Minor clock differences can cause slight over/under-counting at window edges. A centralised Redis Lua script that uses `TIME` (Redis server clock) eliminates this issue entirely.

---

## 6. Job Queue Design

**Architecture:** Single in-process worker pool using Go channels and goroutines.

```
NATS message
    │
    ▼
subscriber.handleMessage()
    │  (only for appointments.status_updated where new_status="done")
    ▼
Queue.Enqueue(job)
    │  check Redis idempotency key → drop if "done"
    │  log status="enqueued"
    ▼
buffered channel  chan Job  (capacity = WORKER_POOL_SIZE × 10)
    │
    ▼  (WORKER_POOL_SIZE goroutines reading from channel)
Queue.processJob(job)
    │  log status="processing"
    │  callGateway(job) → POST /notify
    │  ├─ HTTP 200 → SET idempotency key "done" (TTL 24h), log status="success"
    │  ├─ HTTP 503 / network error → log status="retry", sleep backoff, retry
    │  └─ after maxRetries failures → log status="dead_letter" to stderr
    ▼
done
```

**Buffer size:** `WORKER_POOL_SIZE × 10` (configurable indirectly via `WORKER_POOL_SIZE`).
**Backpressure:** When the channel is full, `Enqueue` logs a warning and drops the job (fail-fast). In production, you would block with a timeout or route to a persistent DLQ.

---

## 7. Idempotency

**Derivation:** `SHA-256(event_type + appointment_id + occurred_at)` — deterministic from the event payload, so replaying the same NATS message always produces the same key.

**Storage:** Redis key `notification:idem:<hex>`, value `"done"`, TTL **24 hours**.

**Flow:**
1. `Enqueue` checks Redis before placing the job on the channel. If key = `"done"` → drop silently (logged at info).
2. After a successful gateway call the worker sets the key to `"done"` in Redis.
3. If the service restarts within 24 h, the key is still present → replayed events are dropped. After 24 h the key expires and the job would be re-processed if replayed (acceptable for notifications).

---

## 8. Dead-Letter Strategy

After **3 consecutive failures** (HTTP 503 or network error, backoff 1s → 2s → 4s), the worker writes a structured JSON line to **stderr**:

```json
{"time":"...","level":"error","job_id":"<sha256>","attempt":3,"status":"dead_letter","error":"exceeded max retries"}
```

The worker then drops the job and continues processing subsequent jobs (no crash).

**Inspecting dead-letter entries:** `docker logs <notification-container> 2>&1 | grep dead_letter` or redirect stderr to a file.

**Production approach:**
- Publish the dead-letter job back to a dedicated NATS subject (`notifications.dead_letter`) or a RabbitMQ DLX queue for human review.
- Emit a metric/alert (Prometheus counter `job_dead_letter_total`).
- Store the full job payload for manual replay after the gateway issue is resolved.

---

## 9. Cache Consistency Trade-offs

**When Redis is unavailable:**
Both services detect the failure at startup (`Ping` timeout) and fall back to `NoOpCache`. All reads go directly to PostgreSQL — no staleness, just higher latency. The service never crashes.

**Which reads become eventually consistent:**
When Redis is available, `GetDoctor` / `GetAppointment` and their list equivalents are eventually consistent for up to `CACHE_TTL_SECONDS` after a write whose cache invalidation failed (e.g., Redis network blip between DB write and `DEL`). The write itself is always durable in PostgreSQL.

**Redis Cluster and consistency:**
Redis Cluster uses hash slots — a key always lives on one primary shard, so single-key reads/writes remain strongly consistent within that shard. However, `DEL` of multiple keys that hash to different slots loses atomicity unless grouped via hash tags (e.g., `{doctor}:list`). For this assignment, keys are simple strings and single-shard operations are atomic.

---

## 10. Infrastructure Setup

```bash
# PostgreSQL (two separate databases)
docker run -d --name pg-doctor   -e POSTGRES_DB=doctors_db       -e POSTGRES_PASSWORD=secret -p 5432:5432 postgres:15-alpine
docker run -d --name pg-appt     -e POSTGRES_DB=appointments_db  -e POSTGRES_PASSWORD=secret -p 5433:5432 postgres:15-alpine

# NATS
docker run -d --name nats -p 4222:4222 nats:latest

# Redis (shared by all services)
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Mock Notification Gateway
cd mock-gateway
GATEWAY_PORT=8080 go run .
```

---

## 11. Service Startup Order

Start services in this order (each in its own terminal):

```bash
# 1. Infrastructure (PostgreSQL, NATS, Redis, Mock Gateway) — see above

# 2. Doctor Service
cd doctor-service
DATABASE_URL="postgres://postgres:secret@localhost:5432/doctors_db?sslmode=disable" \
NATS_URL=nats://localhost:4222 \
REDIS_URL=redis://localhost:6379 \
CACHE_TTL_SECONDS=60 \
RATE_LIMIT_RPM=100 \
GRPC_PORT=50051 \
go run .

# 3. Appointment Service
cd appointment-service
DATABASE_URL="postgres://postgres:secret@localhost:5433/appointments_db?sslmode=disable" \
NATS_URL=nats://localhost:4222 \
REDIS_URL=redis://localhost:6379 \
CACHE_TTL_SECONDS=60 \
RATE_LIMIT_RPM=100 \
GRPC_PORT=50052 \
DOCTOR_SERVICE_ADDR=localhost:50051 \
go run .

# 4. Notification Service
cd notification-service
NATS_URL=nats://localhost:4222 \
REDIS_URL=redis://localhost:6379 \
GATEWAY_URL=http://localhost:8080 \
WORKER_POOL_SIZE=3 \
go run .
```

---

## 12. Environment Variables

| Service | Variable | Default | Purpose |
|---|---|---|---|
| All | `REDIS_URL` | — | Redis connection string (e.g. `redis://localhost:6379`) |
| All | `CACHE_TTL_SECONDS` | `60` | Cache entry TTL in seconds |
| Doctor / Appt | `RATE_LIMIT_RPM` | `100` | Max requests per minute per client IP |
| Doctor Service | `DATABASE_URL` | required | PostgreSQL connection string |
| Doctor Service | `NATS_URL` | — | NATS broker URL |
| Doctor Service | `GRPC_PORT` | `50051` | gRPC listen port |
| Appt Service | `DATABASE_URL` | required | PostgreSQL connection string |
| Appt Service | `NATS_URL` | — | NATS broker URL |
| Appt Service | `GRPC_PORT` | `50052` | gRPC listen port |
| Appt Service | `DOCTOR_SERVICE_ADDR` | `localhost:50051` | Doctor Service gRPC endpoint |
| Notification | `NATS_URL` | `nats://localhost:4222` | NATS broker URL |
| Notification | `GATEWAY_URL` | `http://localhost:8080` | Mock Gateway URL |
| Notification | `WORKER_POOL_SIZE` | `3` | Worker goroutine count |
| Mock Gateway | `GATEWAY_PORT` | `8080` | HTTP listen port |

---

## 13. grpcurl Testing Commands

```bash
# ── Doctor Service ────────────────────────────────────────────────────────────

# Create a doctor
grpcurl -plaintext -d '{"full_name":"Alice Smith","specialization":"Cardiology","email":"alice@clinic.kz"}' \
  localhost:50051 doctor.DoctorService/CreateDoctor

# GetDoctor (call twice — second call hits Redis cache)
grpcurl -plaintext -d '{"id":"<doctor-id>"}' localhost:50051 doctor.DoctorService/GetDoctor
grpcurl -plaintext -d '{"id":"<doctor-id>"}' localhost:50051 doctor.DoctorService/GetDoctor
# Verify with: redis-cli MONITOR  (first call: GET miss + SET; second call: GET hit only)

# ListDoctors
grpcurl -plaintext -d '{}' localhost:50051 doctor.DoctorService/ListDoctors

# ── Appointment Service ───────────────────────────────────────────────────────

# Create an appointment
grpcurl -plaintext -d '{"title":"Checkup","doctor_id":"<doctor-id>"}' \
  localhost:50052 appointment.AppointmentService/CreateAppointment

# GetAppointment (first = DB + cache SET; second = cache hit)
grpcurl -plaintext -d '{"id":"<appt-id>"}' localhost:50052 appointment.AppointmentService/GetAppointment
grpcurl -plaintext -d '{"id":"<appt-id>"}' localhost:50052 appointment.AppointmentService/GetAppointment

# UpdateAppointmentStatus to "done" — triggers job queue
grpcurl -plaintext -d '{"id":"<appt-id>","new_status":"done"}' \
  localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus

# Expected Notification Service stdout (within seconds):
# {"time":"...","subject":"appointments.status_updated","event":{...}}
# {"time":"...","level":"info","job_id":"<sha256>","attempt":1,"status":"enqueued","error":""}
# {"time":"...","level":"info","job_id":"<sha256>","attempt":1,"status":"processing","error":""}
# {"time":"...","level":"info","job_id":"<sha256>","attempt":1,"status":"success","error":""}
# (or "retry" if mock-gateway returned 503, followed by eventual "success")

# Expected Mock Gateway stdout:
# {"time":"...","method":"POST","path":"/notify","idempotency_key":"<sha256>","channel":"email","recipient":"patient@clinic.kz","status_code":200}
```

---

## 14. Defense Checkpoint Notes

| Checkpoint | How to verify |
|---|---|
| Cache Hit (CP1) | `redis-cli MONITOR` — 1st `GetDoctor` shows `GET` miss + `SET`; 2nd shows `GET` hit only |
| Rate Limiter (CP2) | Send > `RATE_LIMIT_RPM` requests/min to any endpoint; service returns `ResourceExhausted` |
| Job Queue & Gateway (CP3) | `UpdateAppointmentStatus` to `done`; observe logs in Notification Service and Mock Gateway terminals |
| Idempotency (CP4) | Replay the same NATS message; Notification Service logs `status="dropped"` and makes no second gateway call |
| Dead Letter (CP5) | Stop Mock Gateway; trigger `done` transition; Notification Service logs 3× `status="retry"` then `status="dead_letter"` to stderr; service keeps running |
