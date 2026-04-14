# Medical Scheduling Platform — gRPC Migration

A two-service Medical Scheduling Platform where all communication is handled exclusively through **gRPC**.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    gRPC Client (grpcurl / test)              │
└────────────────┬────────────────────────┬───────────────────┘
                 │ gRPC :50051             │ gRPC :50052
                 ▼                         ▼
    ┌────────────────────┐    ┌────────────────────────────┐
    │   Doctor Service   │◄───│   Appointment Service      │
    │   port 50051       │    │   port 50052               │
    │                    │    │  (calls Doctor via gRPC)   │
    └────────────────────┘    └────────────────────────────┘
```

Each service follows Clean Architecture:
```
cmd/           → entry point
internal/
  model/       → domain models (unchanged)
  repository/  → in-memory storage (unchanged)
  usecase/     → business logic (unchanged)
  transport/
    grpc/      → gRPC server handler (delivery layer)
  client/      → gRPC client for Doctor Service (appointment only)
  app/         → wiring: creates server, registers gRPC handler
proto/         → .proto file + generated stubs (*.pb.go, *_grpc.pb.go)
```

---

## Service Responsibilities

| Service | Owns | gRPC Port |
|---|---|---|
| Doctor Service | Doctor records (CRUD) | 50051 |
| Appointment Service | Appointment records (CRUD + status) | 50052 |

The Appointment Service calls `DoctorService.GetDoctor` before creating an appointment to validate that the doctor exists.

---

## Prerequisites

- Go 1.21+
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-go` and `protoc-gen-go-grpc` plugins

### Install protoc

```bash
brew install protobuf
```

### Install Go plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

---

## Regenerating Proto Stubs

From `doctor-service/`:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/doctor.proto
```

From `appointment-service/`:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/appointment.proto
```

---

## Running Both Services Locally

**Start Doctor Service first** (it must be up before Appointment Service receives requests):

```bash
# Terminal 1
cd doctor-service
go run ./cmd/doctor-service/
# Listens on :50051
```

```bash
# Terminal 2
cd appointment-service
go run ./cmd/appointment-service/
# Listens on :50052
```

Or run both together from root:
```bash
go run main.go
```

---

## Proto Contract

### Doctor Service (`doctor-service/proto/doctor.proto`)

| RPC | Request | Response | Business Rule |
|---|---|---|---|
| `CreateDoctor` | `CreateDoctorRequest` | `DoctorResponse` | `full_name` and `email` required; email must be unique |
| `GetDoctor` | `GetDoctorRequest` | `DoctorResponse` | Returns `NOT_FOUND` if ID does not exist |
| `ListDoctors` | `ListDoctorsRequest` | `ListDoctorsResponse` | Returns all stored doctors |

### Appointment Service (`appointment-service/proto/appointment.proto`)

| RPC | Request | Response | Business Rule |
|---|---|---|---|
| `CreateAppointment` | `CreateAppointmentRequest` | `AppointmentResponse` | `title` and `doctor_id` required; doctor must exist (verified via gRPC) |
| `GetAppointment` | `GetAppointmentRequest` | `AppointmentResponse` | Returns `NOT_FOUND` if ID does not exist |
| `ListAppointments` | `ListAppointmentsRequest` | `ListAppointmentsResponse` | Returns all stored appointments |
| `UpdateAppointmentStatus` | `UpdateStatusRequest` | `AppointmentResponse` | Status must be `new`/`in_progress`/`done`; `done → new` is forbidden |

---

## gRPC Error Handling

| Situation | gRPC Status Code |
|---|---|
| Required field missing | `codes.InvalidArgument` |
| Email already in use | `codes.AlreadyExists` |
| Doctor/Appointment ID not found (local) | `codes.NotFound` |
| Doctor Service unreachable | `codes.Unavailable` |
| Doctor does not exist (remote check) | `codes.FailedPrecondition` |
| Invalid status transition (`done → new`) | `codes.InvalidArgument` |

---

## Inter-Service Communication

The Appointment Service holds a `DoctorClient` interface:

```go
type DoctorClient interface {
    DoctorExists(doctorID string) (bool, error)
}
```

`GRPCDoctorClient` implements this interface using the generated `DoctorServiceClient` stub. It is injected into the use case via dependency injection — the use case never imports any protobuf types.

**Flow for `CreateAppointment`:**
1. Validate `title` and `doctor_id` are non-empty
2. Call `DoctorService.GetDoctor(doctorID)` with a 5-second timeout
3. If unreachable → return `codes.Unavailable`
4. If `NOT_FOUND` → return `codes.FailedPrecondition`
5. If found → create and store the appointment

---

## Failure Scenario

If the Doctor Service is **unreachable** (down or network error), the Appointment Service returns:

```
Code: Unavailable
Message: doctor service is unreachable: ...
```

In production, resilience patterns would be added here:
- **Timeouts**: already implemented (5s per call)
- **Retries**: use `google.golang.org/grpc/keepalive` or a retry interceptor
- **Circuit breaker**: use a library like `sony/gobreaker` around the client call

---

## Testing with grpcurl

```bash
# Create a doctor
grpcurl -plaintext -d '{"full_name":"Dr. John Smith","specialization":"Cardiology","email":"john@hospital.com"}' \
  localhost:50051 doctor.DoctorService/CreateDoctor

# List all doctors
grpcurl -plaintext -d '{}' localhost:50051 doctor.DoctorService/ListDoctors

# Get doctor by ID
grpcurl -plaintext -d '{"id":"<DOCTOR_ID>"}' localhost:50051 doctor.DoctorService/GetDoctor

# Create an appointment
grpcurl -plaintext -d '{"title":"Heart Checkup","description":"Annual exam","doctor_id":"<DOCTOR_ID>"}' \
  localhost:50052 appointment.AppointmentService/CreateAppointment

# List appointments
grpcurl -plaintext -d '{}' localhost:50052 appointment.AppointmentService/ListAppointments

# Update status
grpcurl -plaintext -d '{"id":"<APPOINTMENT_ID>","status":"in_progress"}' \
  localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus

# Forbidden transition (done -> new)
grpcurl -plaintext -d '{"id":"<APPOINTMENT_ID>","status":"new"}' \
  localhost:50052 appointment.AppointmentService/UpdateAppointmentStatus
# → Code: InvalidArgument, Message: cannot transition from done to new
```

---

## REST vs gRPC Trade-offs

| | REST | gRPC |
|---|---|---|
| **Protocol** | HTTP/1.1 + JSON (text) | HTTP/2 + Protobuf (binary) |
| **Performance** | Slower: JSON parsing, no multiplexing | Faster: binary encoding, HTTP/2 multiplexing |
| **Contract** | Informal (OpenAPI optional) | Strict: `.proto` file is the contract |
| **Tooling** | Universal (curl, browser) | Requires gRPC clients / grpcurl |
| **Streaming** | Not native | Native bidirectional streaming |
| **Type safety** | No (JSON is dynamic) | Yes: generated typed stubs |

**Choose REST** when: building public-facing APIs consumed by browsers, third-party clients, or teams with diverse technology stacks.

**Choose gRPC** when: building internal microservice-to-microservice communication where performance, type safety, and strict contracts matter.
