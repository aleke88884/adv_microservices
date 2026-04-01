# Architecture Diagram

## System Overview

```
                    ┌─────────────────────────┐
                    │    Client/User          │
                    │  (API Consumer)         │
                    └───────────┬─────────────┘
                                │
                    ┌───────────┴───────────┐
                    │   HTTP Requests       │
                    │   (REST APIs)         │
                    └───────────┬───────────┘
                                │
            ┌───────────────────┼───────────────────┐
            │                   │                   │
            ▼                   ▼                   ▼
    ┌───────────────┐   ┌───────────────┐   ┌──────────┐
    │  POST/GET     │   │  POST/GET     │   │  PATCH   │
    │  /doctors     │   │ /appointments │   │/appt/*/  │
    │               │   │               │   │  status  │
    └───────┬───────┘   └───────┬───────┘   └────┬─────┘
            │                   │                 │
            │                   │                 │
            ▼                   ▼                 │
    ┌────────────────────────────────────────────┴────┐
    │                                                  │
    │                                                  │
┌───┴───────────────────┐          ┌──────────────────┴──────┐
│  Doctor Service       │          │  Appointment Service    │
│  Port: 8080           │◄─────────┤  Port: 8081             │
│                       │  REST    │                         │
│  ┌─────────────────┐  │  Call    │  ┌───────────────────┐  │
│  │ HTTP Handler    │  │          │  │ HTTP Handler      │  │
│  └────────┬────────┘  │          │  └─────────┬─────────┘  │
│           │           │          │            │            │
│  ┌────────▼────────┐  │          │  ┌─────────▼─────────┐  │
│  │ Use Case Layer  │  │          │  │ Use Case Layer    │  │
│  │ (Business Logic)│  │          │  │ (Business Logic)  │  │
│  └────────┬────────┘  │          │  └─────┬──────┬──────┘  │
│           │           │          │        │      │         │
│  ┌────────▼────────┐  │          │  ┌─────▼──┐ ┌▼───────┐ │
│  │ Repository      │  │          │  │ Repo   │ │Doctor  │ │
│  │ Interface       │  │          │  │Interface│ │Client  │ │
│  └────────┬────────┘  │          │  └─────┬──┘ └┬───────┘ │
│           │           │          │        │     │         │
│  ┌────────▼────────┐  │          │  ┌─────▼──┐ └─────────┘ │
│  │ In-Memory       │  │          │  │In-Memory│            │
│  │ Doctor Storage  │  │          │  │Appt     │            │
│  │                 │  │          │  │Storage  │            │
│  └─────────────────┘  │          │  └────────┘            │
└───────────────────────┘          └─────────────────────────┘
         Owns                               Owns
      Doctor Data                      Appointment Data


HTTP Call Flow for Creating Appointment:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. Client → Appointment Service:
   POST /appointments {"doctor_id": "123", ...}

2. Appointment Service → Doctor Service:
   GET /doctors/123

3. Doctor Service → Appointment Service:
   200 OK {doctor data} OR 404 Not Found

4. Appointment Service → Client:
   201 Created {appointment} OR 400 Bad Request
```

## Clean Architecture Layers

### Doctor Service

```
┌─────────────────────────────────────────────────────┐
│                 Doctor Service                      │
├─────────────────────────────────────────────────────┤
│  Layer 1: Transport (HTTP Handlers)                 │
│  - Parse incoming HTTP requests                     │
│  - Call use case methods                            │
│  - Format HTTP responses                            │
│  File: internal/transport/http/handler.go           │
├─────────────────────────────────────────────────────┤
│  Layer 2: Use Case (Business Logic)                 │
│  - Validate business rules                          │
│  - Orchestrate data flow                            │
│  - Enforce: email uniqueness, required fields       │
│  File: internal/usecase/doctor_usecase.go           │
├─────────────────────────────────────────────────────┤
│  Layer 3: Repository Interface                      │
│  - Abstract storage operations                      │
│  - Depend on abstraction, not concrete              │
│  File: internal/repository/repository.go            │
├─────────────────────────────────────────────────────┤
│  Layer 4: Repository Implementation                 │
│  - In-memory map storage                            │
│  - Thread-safe with sync.RWMutex                    │
│  File: internal/repository/repository.go            │
├─────────────────────────────────────────────────────┤
│  Core: Domain Model                                 │
│  - Pure business entity                             │
│  - No external dependencies                         │
│  File: internal/model/doctor.go                     │
└─────────────────────────────────────────────────────┘
```

### Appointment Service

```
┌─────────────────────────────────────────────────────┐
│              Appointment Service                    │
├─────────────────────────────────────────────────────┤
│  Layer 1: Transport (HTTP Handlers)                 │
│  - Parse incoming HTTP requests                     │
│  - Call use case methods                            │
│  - Format HTTP responses                            │
│  File: internal/transport/http/handler.go           │
├─────────────────────────────────────────────────────┤
│  Layer 2: Use Case (Business Logic)                 │
│  - Validate business rules                          │
│  - Validate doctor via client                       │
│  - Enforce status transition rules                  │
│  File: internal/usecase/appointment_usecase.go      │
├─────────────────────────────────────────────────────┤
│  Layer 3a: Repository Interface                     │
│  - Abstract storage operations                      │
│  File: internal/repository/repository.go            │
│                                                     │
│  Layer 3b: Doctor Client Interface                  │
│  - Abstract external service calls                  │
│  - Timeout handling (5 seconds)                     │
│  File: internal/client/doctor_client.go             │
├─────────────────────────────────────────────────────┤
│  Layer 4: Repository Implementation                 │
│  - In-memory map storage                            │
│  - Thread-safe with sync.RWMutex                    │
│  File: internal/repository/repository.go            │
├─────────────────────────────────────────────────────┤
│  Core: Domain Model                                 │
│  - Pure business entity                             │
│  - Status validation logic                          │
│  File: internal/model/appointment.go                │
└─────────────────────────────────────────────────────┘
```

## Dependency Direction

```
Outer Layers ──────► Inner Layers
(Infrastructure)     (Business Logic)

HTTP Handler ──► Use Case ──► Domain Model
Repository   ──► Interface ─┘
Client       ──► Interface ─┘

Key Principle: Dependencies point INWARD
- Inner layers know nothing about outer layers
- Use cases depend on interfaces, not implementations
- Domain models have zero external dependencies
```

## Service Communication

```
Synchronous REST Communication:

┌──────────────────────┐
│ Appointment Service  │
│                      │
│  1. Client Request   │
│  2. Validate inputs  │
│  3. Call Doctor API ─┼────────┐
│  4. Wait for response│        │
│  5. Process result   │        │ HTTP GET
│  6. Save appointment │        │ Timeout: 5s
│  7. Return response  │        │
└──────────────────────┘        │
                                │
                                ▼
                    ┌───────────────────┐
                    │ Doctor Service    │
                    │                   │
                    │ 1. Receive request│
                    │ 2. Query storage  │
                    │ 3. Return doctor  │
                    │    or 404         │
                    └───────────────────┘

Failure Scenario:
- Doctor Service down/unreachable
- HTTP client times out after 5s
- Appointment Service returns error
- No appointment is created
```

## Data Ownership

```
┌──────────────────────────────────────────────────┐
│                                                  │
│  Microservices Principle: Database per Service  │
│                                                  │
└──────────────────────────────────────────────────┘

Doctor Service owns:               Appointment Service owns:
- Doctor ID                        - Appointment ID
- Doctor Full Name                 - Appointment Title
- Doctor Specialization            - Appointment Description
- Doctor Email                     - Doctor ID (reference only)
                                  - Appointment Status
                                  - Created/Updated timestamps

✓ Each service has its own storage
✓ No shared database
✓ No cross-service table access
✓ Communication only via REST APIs
✗ Cannot directly query other service's data
```

## Status Transition Flow

```
Appointment Status State Machine:

    ┌─────┐
    │ new │ ◄── Initial state
    └──┬──┘
       │
       ├──────► ┌─────────────┐
       │        │ in_progress │
       │        └──────┬──────┘
       │               │
       │               ▼
       └──────────► ┌──────┐
                    │ done │ ◄── Terminal state
                    └──────┘

Valid Transitions:
✓ new → in_progress
✓ new → done
✓ in_progress → done
✗ done → new (NOT ALLOWED)
✗ done → in_progress (allowed by current rules)

Validation Logic Location:
- model/appointment.go: ValidateStatusTransition()
- usecase/appointment_usecase.go: UpdateAppointmentStatus()
```

## Request/Response Flow

```
Complete Flow Example: Create Appointment

1. Client Request
   ↓
   POST http://localhost:8081/appointments
   {
     "title": "Cardiac consultation",
     "doctor_id": "doctor-123"
   }

2. Appointment HTTP Handler
   ↓
   - Parse JSON request
   - Extract fields

3. Appointment Use Case
   ↓
   - Validate title (required)
   - Validate doctor_id (required)
   - Call Doctor Client

4. Doctor Client (HTTP)
   ↓
   GET http://localhost:8080/doctors/doctor-123
   Timeout: 5 seconds

5. Doctor Service
   ↓
   - Receive request
   - Query repository
   - Return doctor data (200) or not found (404)

6. Appointment Use Case (continued)
   ↓
   - Process doctor validation result
   - Create appointment entity
   - Generate UUID
   - Set status = "new"
   - Save to repository

7. Appointment Repository
   ↓
   - Store in memory map
   - Thread-safe operation

8. Response to Client
   ↓
   201 Created
   {
     "id": "appt-uuid",
     "title": "Cardiac consultation",
     "doctor_id": "doctor-123",
     "status": "new",
     "created_at": "2026-03-31T10:00:00Z",
     "updated_at": "2026-03-31T10:00:00Z"
   }
```
