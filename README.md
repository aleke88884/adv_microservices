# Medical Scheduling Platform - Clean Architecture Microservices

## Project Overview

This project implements a two-service medical scheduling platform using Clean Architecture principles and REST-based microservices. The system consists of:

- **Doctor Service**: Manages doctor profile data
- **Appointment Service**: Manages appointment data and validates doctor existence

Both services follow Clean Architecture principles with clear separation of concerns, dependency inversion, and distinct bounded contexts. The system demonstrates proper microservices decomposition with independent data ownership and synchronous inter-service communication via REST APIs.

## Architecture

### Clean Architecture Layers

Each service is structured in layers following the Clean Architecture pattern:

1. **Domain Layer** (`internal/model/`): Contains pure business entities and domain logic
   - No dependencies on external frameworks or libraries
   - Defines core business rules (e.g., status transitions, validation)

2. **Use Case Layer** (`internal/usecase/`): Contains application business logic
   - Orchestrates data flow between layers
   - Enforces business rules
   - Depends only on interfaces, not concrete implementations

3. **Repository Layer** (`internal/repository/`): Handles data persistence
   - Implements repository interfaces defined by use cases
   - In-memory storage for this assignment
   - Can be swapped without affecting business logic

4. **Transport Layer** (`internal/transport/http/`): Handles HTTP communication
   - Thin handlers that parse requests and format responses
   - No business logic
   - Delegates all work to use cases

5. **Application Layer** (`internal/app/`): Wires dependencies together
   - Dependency injection
   - Service initialization and startup

6. **Client Layer** (`internal/client/` - Appointment Service only): Handles outbound HTTP calls
   - Abstracts communication with external services
   - Implements timeout and error handling

### Service Responsibilities

**Doctor Service**
- Manages doctor profiles (ID, name, specialization, email)
- Owns doctor data independently
- Validates email uniqueness
- Exposes REST API for doctor operations
- Runs on port 8080

**Appointment Service**
- Manages appointments (ID, title, description, doctor reference, status)
- Owns appointment data independently
- Validates appointment status transitions
- Validates doctor existence via REST call to Doctor Service
- Exposes REST API for appointment operations
- Runs on port 8081

### Dependency Flow

```
Transport Layer (HTTP Handlers)
        ↓ (depends on)
Use Case Layer (Business Logic)
        ↓ (depends on)
Repository/Client Interfaces
        ↑ (implemented by)
Repository/Client Implementations
```

Dependencies point inward: outer layers depend on inner layers, never the reverse. This enables dependency inversion and testability.

## Inter-Service Communication

The Appointment Service communicates with the Doctor Service through synchronous REST calls:

1. When creating an appointment, the Appointment Service calls `GET /doctors/{id}` on the Doctor Service
2. The Doctor Service responds with doctor data (200 OK) or not found (404)
3. Only if the doctor exists does the Appointment Service proceed with appointment creation
4. This validation also occurs when updating appointment status

**HTTP Contract:**
- Request: `GET http://localhost:8080/doctors/{doctor_id}`
- Response 200: Doctor exists
- Response 404: Doctor not found
- Timeout: 5 seconds

## Why Not a Shared Database?

This system follows the **Database per Service** pattern, a core microservices principle:

**Data Ownership**
- Each service has its own in-memory data store
- Doctor Service owns doctor data
- Appointment Service owns appointment data
- Services cannot directly access each other's data

**Benefits:**
- **Bounded Contexts**: Clear service boundaries prevent tight coupling
- **Independent Deployment**: Services can evolve independently
- **Technology Flexibility**: Each service can use different storage technologies
- **Fault Isolation**: Database issues in one service don't affect others
- **Scalability**: Services can scale independently based on load

**Trade-offs:**
- Increased latency due to network calls
- Potential consistency challenges (eventual consistency)
- More complex failure handling

## Failure Scenario: Doctor Service Unavailable

When the Doctor Service is unavailable during appointment creation or status update:

**Current Implementation:**
1. HTTP client has a 5-second timeout
2. If the call fails, the Appointment Service returns HTTP 400 with error message
3. Error is logged internally: `"failed to validate doctor: [error details]"`
4. No appointment is created/updated
5. Client receives clear error response

**Production Considerations:**

For a production system at scale, the following patterns would be necessary:

1. **Timeout Policy**
   - Already implemented: 5-second timeout on HTTP client
   - Should be tunable via configuration
   - Different timeouts for different operations

2. **Retry Strategy**
   - Implement exponential backoff for transient failures
   - Distinguish between retryable (503, timeout) and non-retryable (404, 400) errors
   - Set maximum retry attempts (e.g., 3 retries)
   - Example: retry after 100ms, 200ms, 400ms

3. **Circuit Breaker**
   - Track failure rate of Doctor Service calls
   - If failure rate exceeds threshold (e.g., 50% over 10 requests), open circuit
   - When open, fail fast without calling Doctor Service
   - Periodically test if service recovered (half-open state)
   - Close circuit when service is healthy again
   - Prevents cascading failures and gives failing service time to recover

4. **Fallback Strategies**
   - Cache doctor existence results with TTL
   - Allow appointment creation with "pending validation" status
   - Process validation asynchronously when service recovers

5. **Observability**
   - Metrics: request latency, error rates, circuit breaker state
   - Distributed tracing across service calls
   - Structured logging with correlation IDs

## Microservices vs Distributed Monolith

**Why This Is a Microservices Architecture:**
- Each service owns its data independently
- Services communicate only through well-defined REST APIs
- No shared database or direct table access
- Services can be deployed, scaled, and developed independently
- Clear bounded contexts around business capabilities

**What Would Make It a Distributed Monolith:**
- Shared database with both services accessing same tables
- Direct database queries across service boundaries
- Tight coupling through shared code/libraries
- Services that cannot function independently
- Synchronous call chains that span multiple services

## How to Run the Project

### Prerequisites
- Go 1.21 or higher
- No additional dependencies required (uses in-memory storage)

### Starting the Services

**Option 1: Run both services (requires two terminals)**

Terminal 1 - Doctor Service:
```bash
cd doctor-service
go run cmd/doctor-service/main.go
```

Terminal 2 - Appointment Service:
```bash
cd appointment-service
go run cmd/appointment-service/main.go
```

**Option 2: Run from project root**

Terminal 1:
```bash
cd doctor-service && go run cmd/doctor-service/main.go
```

Terminal 2:
```bash
cd appointment-service && go run cmd/appointment-service/main.go
```

The Doctor Service will start on `http://localhost:8080`
The Appointment Service will start on `http://localhost:8081`

## API Endpoints

### Doctor Service (Port 8080)

**Create Doctor**
```bash
POST http://localhost:8080/doctors
Content-Type: application/json

{
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

**Get Doctor by ID**
```bash
GET http://localhost:8080/doctors/{id}
```

**Get All Doctors**
```bash
GET http://localhost:8080/doctors
```

### Appointment Service (Port 8081)

**Create Appointment**
```bash
POST http://localhost:8081/appointments
Content-Type: application/json

{
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "doctor-id-here"
}
```

**Get Appointment by ID**
```bash
GET http://localhost:8081/appointments/{id}
```

**Get All Appointments**
```bash
GET http://localhost:8081/appointments
```

**Update Appointment Status**
```bash
PATCH http://localhost:8081/appointments/{id}/status
Content-Type: application/json

{
  "status": "in_progress"
}
```

Valid status values: `new`, `in_progress`, `done`

## Testing the System

### Example Workflow

1. Start both services
2. Create a doctor:
```bash
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }'
```

3. Copy the doctor ID from the response

4. Create an appointment using that doctor ID:
```bash
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "YOUR_DOCTOR_ID"
  }'
```

5. Update appointment status:
```bash
curl -X PATCH http://localhost:8081/appointments/YOUR_APPOINTMENT_ID/status \
  -H "Content-Type: application/json" \
  -d '{"status": "in_progress"}'
```

### Testing Failure Scenario

1. Stop the Doctor Service (Ctrl+C)
2. Try to create an appointment
3. Observe the error response: `"failed to validate doctor: ..."`
4. Check appointment service logs for failure details

## Business Rules

### Doctor Service
- `full_name` is required
- `email` is required
- `email` must be unique across all doctors

### Appointment Service
- `title` is required
- `doctor_id` is required
- Referenced doctor must exist in Doctor Service
- `status` must be one of: `new`, `in_progress`, `done`
- Cannot transition from `done` back to `new`
- Valid transitions:
  - `new` → `in_progress` ✓
  - `new` → `done` ✓
  - `in_progress` → `done` ✓
  - `done` → `new` ✗

## Project Structure

```
.
├── doctor-service/
│   ├── cmd/
│   │   └── doctor-service/
│   │       └── main.go                 # Entry point
│   ├── internal/
│   │   ├── model/
│   │   │   └── doctor.go               # Domain model
│   │   ├── usecase/
│   │   │   └── doctor_usecase.go       # Business logic
│   │   ├── repository/
│   │   │   └── repository.go           # Data persistence
│   │   ├── transport/
│   │   │   └── http/
│   │   │       └── handler.go          # HTTP handlers
│   │   └── app/
│   │       └── app.go                  # Dependency wiring
│   ├── go.mod
│   └── go.sum
│
├── appointment-service/
│   ├── cmd/
│   │   └── appointment-service/
│   │       └── main.go                 # Entry point
│   ├── internal/
│   │   ├── model/
│   │   │   └── appointment.go          # Domain model
│   │   ├── usecase/
│   │   │   └── appointment_usecase.go  # Business logic
│   │   ├── repository/
│   │   │   └── repository.go           # Data persistence
│   │   ├── transport/
│   │   │   └── http/
│   │   │       └── handler.go          # HTTP handlers
│   │   ├── client/
│   │   │   └── doctor_client.go        # Doctor Service client
│   │   └── app/
│   │       └── app.go                  # Dependency wiring
│   ├── go.mod
│   └── go.sum
│
└── README.md
```

## Design Decisions

### Why Clean Architecture?
- **Testability**: Business logic can be tested without HTTP framework
- **Flexibility**: Easy to swap storage or transport mechanisms
- **Maintainability**: Clear separation of concerns
- **Independence**: Framework-agnostic core business logic

### Why In-Memory Storage?
- Simplifies assignment demonstration
- Focuses on architecture rather than database setup
- Easy to swap with real database implementation (just implement repository interface)

### Why Synchronous REST?
- Assignment requirement
- Simpler to implement and understand
- Appropriate for real-time validation requirements
- Trade-off: higher coupling and potential cascading failures

### Why Interface-Based Dependencies?
- Enables dependency inversion principle
- Facilitates testing with mocks
- Allows swapping implementations without changing business logic
- Example: Can swap HTTPDoctorClient with mock client for testing

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     Client (API Consumer)                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
         ▼                               ▼
┌─────────────────┐              ┌─────────────────┐
│ Doctor Service  │              │ Appointment     │
│   Port 8080     │◄────REST─────┤   Service       │
│                 │              │   Port 8081     │
└────────┬────────┘              └────────┬────────┘
         │                                │
    ┌────┴────┐                      ┌────┴────┐
    │ Doctor  │                      │Appoint- │
    │  Data   │                      │ment Data│
    │(In-mem) │                      │(In-mem) │
    └─────────┘                      └─────────┘

Service Boundary:
- Each service owns its data
- Communication only via REST APIs
- No shared database
- Independent deployment units
```

## Technologies Used

- **Language**: Go 1.21+
- **HTTP Framework**: Gin (github.com/gin-gonic/gin)
- **UUID Generation**: Google UUID (github.com/google/uuid)
- **Storage**: In-memory (map with sync.RWMutex)

## Conclusion

This project demonstrates Clean Architecture principles applied to a microservices system. Each service maintains clear boundaries, owns its data independently, and communicates through well-defined REST APIs. The architecture supports independent development, deployment, and scaling while maintaining separation of concerns and testability.
