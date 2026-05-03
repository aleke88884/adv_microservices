# AP2 — Assignment 3: Message Queue & Database Migrations

## 1. Project Overview

This is the Medical Scheduling Platform built across Assignments 1–3.

**What changed compared to Assignment 2:**

| Area | Change |
|---|---|
| Storage | Both services now persist data in separate PostgreSQL databases instead of in-memory maps |
| Schema management | All DDL lives exclusively in versioned `migrations/` files managed by `golang-migrate` |
| Event publishing | After every successful write operation each service publishes a domain event to NATS |
| Notification Service | New third service subscribes to all three event subjects and logs structured JSON to stdout |

What did **not** change: domain models, use-case business rules, gRPC contracts, generated proto stubs, and Clean Architecture layering.

---

## 2. Broker Choice — NATS (Core)

**Chosen broker: NATS Core**

**Reason:** NATS Core offers the simplest possible setup (single binary, zero configuration) while fully satisfying the stateless notification requirement. Because the Notification Service only needs to log events and requires no replay or guaranteed delivery, NATS fire-and-forget Pub/Sub is a perfect fit and avoids the operational overhead of RabbitMQ exchanges, queues, and bindings.

### NATS vs RabbitMQ — two concrete differences

| | NATS Core | RabbitMQ |
|---|---|---|
| Persistence | None — messages are lost if no subscriber is connected at publish time | Queue-level durability — messages survive broker restart and consumer downtime |
| Delivery guarantee | Fire-and-forget (at-most-once) | At-least-once delivery with publisher confirms and consumer acks |

**When to choose RabbitMQ:** when guaranteed delivery is required (e.g. billing events, audit logs), when consumers may be transiently offline, or when complex routing (topic/header exchanges) is needed.

**When to choose NATS:** stateless fan-out notifications, low-latency telemetry, or scenarios where losing an occasional event is acceptable and operational simplicity matters more.

### What would change if you switched to RabbitMQ?

- Replace `github.com/nats-io/nats.go` with `github.com/rabbitmq/amqp091-go`.
- In each publisher: declare a `fanout` exchange named `ap2.events`; publish to that exchange instead of a subject string.
- In the Notification Service: declare the exchange, create an exclusive auto-delete queue, bind it to the exchange, and call `ch.Ack(tag, false)` for each delivery.
- Change env var `NATS_URL` → `AMQP_URL`.

### Where durable delivery would be necessary in production

If the service crashes between the PostgreSQL commit and the NATS publish, the event is silently lost. The **Outbox pattern** (write the event into the DB in the same transaction, then relay it) or **NATS JetStream** (durable subjects with acknowledgement) would close this gap.

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                         Medical Scheduling Platform                          │
│                                                                              │
│  ┌─────────────────────┐    gRPC (50051)    ┌──────────────────────────┐    │
│  │   Doctor Service    │◄──────────────────►│  Appointment Service     │    │
│  │                     │                    │                          │    │
│  │  PostgreSQL DB      │                    │  PostgreSQL DB           │    │
│  │  (doctors)          │                    │  (appointments)          │    │
│  │                     │  doctors.created   │                          │    │
│  │                     │──────────────────► │  appointments.created    │    │
│  └─────────────────────┘       NATS         │  appointments.status_    │    │
│                                             │  updated                 │    │
│                                             └──────────────────────────┘    │
│                                                         │                   │
│                                  NATS subjects ─────────┘                   │
│                                       │                                     │
│                          ┌────────────▼──────────────┐                     │
│                          │   Notification Service     │                     │
│                          │   (stdout JSON logger)     │                     │
│                          └───────────────────────────┘                     │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Environment Variables

### Doctor Service
| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string, e.g. `postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable` |
| `NATS_URL` | *(optional)* | NATS connection URL, e.g. `nats://localhost:4222`. If unset, events are silently discarded. |
| `GRPC_PORT` | `50051` | Port for the gRPC server |

### Appointment Service
| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string, e.g. `postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable` |
| `NATS_URL` | *(optional)* | NATS connection URL. If unset, events are silently discarded. |
| `GRPC_PORT` | `50052` | Port for the gRPC server |
| `DOCTOR_SERVICE_ADDR` | `localhost:50051` | Address of the Doctor Service gRPC server |

### Notification Service
| Variable | Default | Description |
|---|---|---|
| `NATS_URL` | `nats://localhost:4222` | NATS connection URL. Required for operation. |

---

## 5. Infrastructure Setup

### PostgreSQL (Docker)

```bash
# Separate database for Doctor Service
docker run -d --name postgres-doctors \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=doctors \
  -p 5432:5432 \
  postgres:16-alpine

# Separate database for Appointment Service
docker run -d --name postgres-appointments \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=appointments \
  -p 5433:5432 \
  postgres:16-alpine
```

### NATS (Docker)

```bash
docker run -d --name nats \
  -p 4222:4222 \
  nats:latest
```

---

## 6. Migration Instructions

Migrations run **automatically on service startup** before the gRPC server accepts requests.

To run or roll back manually using the `golang-migrate` CLI:

```bash
# Install CLI (once)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Apply migrations — Doctor Service
migrate -path doctor-service/migrations \
        -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" up

# Roll back one version — Doctor Service
migrate -path doctor-service/migrations \
        -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" down 1

# Apply migrations — Appointment Service
migrate -path appointment-service/migrations \
        -database "postgres://postgres:postgres@localhost:5433/appointments?sslmode=disable" up

# Roll back one version — Appointment Service
migrate -path appointment-service/migrations \
        -database "postgres://postgres:postgres@localhost:5433/appointments?sslmode=disable" down 1
```

---

## 7. Service Startup Order

Start infrastructure first, then services in this order:

```
1. PostgreSQL containers (both)
2. NATS container
3. Doctor Service        (provides gRPC endpoint validated by Appointment Service)
4. Appointment Service   (depends on Doctor Service gRPC)
5. Notification Service  (depends on NATS)
```

```bash
# Terminal 1 — Doctor Service
cd doctor-service
DATABASE_URL="postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
go run ./cmd/doctor-service

# Terminal 2 — Appointment Service
cd appointment-service
DATABASE_URL="postgres://postgres:postgres@localhost:5433/appointments?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
go run ./cmd/appointment-service

# Terminal 3 — Notification Service
cd notification-service
NATS_URL="nats://localhost:4222" \
go run ./cmd/notification-service
```

---

## 8. Event Contract

| Subject | Published by | Trigger | JSON Fields |
|---|---|---|---|
| `doctors.created` | Doctor Service | `CreateDoctor` succeeds | `event_type`, `occurred_at`, `id`, `full_name`, `specialization`, `email` |
| `appointments.created` | Appointment Service | `CreateAppointment` succeeds | `event_type`, `occurred_at`, `id`, `title`, `doctor_id`, `status` |
| `appointments.status_updated` | Appointment Service | `UpdateAppointmentStatus` succeeds | `event_type`, `occurred_at`, `id`, `old_status`, `new_status` |

### Example payloads

```json
// doctors.created
{
  "event_type": "doctors.created",
  "occurred_at": "2026-05-01T10:23:44Z",
  "id": "abc-123",
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}

// appointments.created
{
  "event_type": "appointments.created",
  "occurred_at": "2026-05-01T10:24:01Z",
  "id": "appt-1",
  "title": "Initial cardiac consultation",
  "doctor_id": "abc-123",
  "status": "new"
}

// appointments.status_updated
{
  "event_type": "appointments.status_updated",
  "occurred_at": "2026-05-01T10:25:10Z",
  "id": "appt-1",
  "old_status": "new",
  "new_status": "in_progress"
}
```

---

## 9. Notification Service

The Notification Service is a pure subscriber: no gRPC server, no HTTP server, no database.

On startup it:
1. Connects to NATS with exponential backoff (1 s, 2 s, 4 s … up to 7 attempts ≈ 127 s total). Exits with code 1 if all attempts fail.
2. Subscribes to `doctors.created`, `appointments.created`, and `appointments.status_updated`.
3. Waits for SIGTERM / SIGINT.

On each received message it prints **one JSON line** to stdout:

```json
{"time":"2026-05-01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"2026-05-01T10:23:44Z","id":"doc-1","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

To verify during a live demo:
```bash
grpcurl -plaintext -d '{"full_name":"Dr. Test","specialization":"Gen","email":"test@example.com"}' \
  localhost:50051 doctor.DoctorService/CreateDoctor
# → Notification Service terminal immediately prints a doctors.created JSON line
```

On shutdown (Ctrl-C or SIGTERM) it drains in-flight messages, closes the NATS connection, and exits with code 0.

---

## 10. grpcurl Test Commands & Expected Notification Output

```bash
# 1. Create a doctor
grpcurl -plaintext -d '{"full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}' \
  localhost:50051 doctor.DoctorService/CreateDoctor

# Notification Service stdout:
# {"time":"...","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"...","id":"<id>","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}

# 2. Get the doctor
grpcurl -plaintext -d '{"id":"<id>"}' localhost:50051 doctor.DoctorService/GetDoctor

# 3. List doctors
grpcurl -plaintext -d '{}' localhost:50051 doctor.DoctorService/ListDoctors

# 4. Create an appointment (replace <doctor_id> with the id from step 1)
grpcurl -plaintext -d '{"title":"Initial cardiac consultation","description":"First visit","doctor_id":"<doctor_id>"}' \
  localhost:50052 appointment.AppointmentService/CreateAppointment

# Notification Service stdout:
# {"time":"...","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"...","id":"<appt_id>","title":"Initial cardiac consultation","doctor_id":"<doctor_id>","status":"new"}}

# 5. Get the appointment
grpcurl -plaintext -d '{"id":"<appt_id>"}' localhost:50052 appointment.AppointmentService/GetAppointment

# 6. Update appointment status
grpcurl -plaintext -d '{"id":"<appt_id>","status":"in_progress"}' \
  localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus

# Notification Service stdout:
# {"time":"...","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"...","id":"<appt_id>","old_status":"new","new_status":"in_progress"}}

# 7. List appointments
grpcurl -plaintext -d '{}' localhost:50052 appointment.AppointmentService/ListAppointments
```

---

## 11. Consistency Trade-offs

Because broker publishing is **best-effort** (fire-and-forget after a successful DB commit), there is a race condition: if the service process crashes between the `COMMIT` and the `nc.Publish(...)` call, the event is silently lost even though the database change is durable.

**Mitigations:**
- **Outbox pattern:** Write the event into an `outbox` table in the same DB transaction; a background relay process reads unsent rows and publishes them, deleting each row after confirmation.
- **NATS JetStream:** Upgrade from Core NATS to JetStream, which provides durable subjects with publisher acknowledgement (`js.Publish` waits for a server ack before returning).
- **RabbitMQ publisher confirms:** The channel's `Confirm` mode causes `ch.Publish` to block until the broker writes the message to disk.

In this assignment the trade-off is acceptable because notifications are informational; a lost notification does not corrupt system state.
