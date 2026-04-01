# Submission Checklist

## Assignment: AP2_Assignment1_NameSurname

This checklist ensures all assignment requirements are met before submission.

## ✅ Requirements Checklist

### Architecture Requirements

- [x] **Clean Architecture**: Both services follow clean architecture with proper layering
  - Domain layer (model) - pure business entities
  - Use case layer - business logic
  - Repository layer - data persistence
  - Transport layer - HTTP handlers (thin, no business logic)
  - Application layer - dependency wiring

- [x] **Microservices Decomposition**
  - Two separate services with bounded contexts
  - Each service owns its own data
  - No shared database or cross-service data access
  - Clear service boundaries enforced through REST APIs
  - Services can be deployed independently

- [x] **Dependency Inversion**
  - Use cases depend on interfaces, not implementations
  - Repository interface defined, implementation separate
  - Doctor client interface defined in Appointment Service
  - Dependencies point inward (outer layers depend on inner layers)

### Functional Requirements

#### Doctor Service (Port 8080)

- [x] **POST /doctors** - create a new doctor
- [x] **GET /doctors/{id}** - retrieve a doctor by ID
- [x] **GET /doctors** - list all doctors

- [x] **Business Rules**
  - full_name is required
  - email is required
  - email must be unique across all doctors

#### Appointment Service (Port 8081)

- [x] **POST /appointments** - create a new appointment
- [x] **GET /appointments/{id}** - retrieve an appointment by ID
- [x] **GET /appointments** - list all appointments
- [x] **PATCH /appointments/{id}/status** - update appointment status

- [x] **Business Rules**
  - title is required
  - doctor_id is required
  - Referenced doctor must exist (validated via REST call to Doctor Service)
  - status must be one of: new, in_progress, done
  - Cannot transition from done to new

### Communication & Failure Handling

- [x] **REST Communication**
  - Appointment Service calls Doctor Service via HTTP
  - Synchronous communication using Gin framework
  - Proper HTTP status codes returned

- [x] **Failure Scenario Handling**
  - HTTP client has 5-second timeout
  - Clear error messages returned when Doctor Service unavailable
  - Failures logged internally
  - No appointment created when validation fails

### Documentation

- [x] **README.md** includes:
  - Project overview and purpose
  - Service responsibilities
  - Folder structure and dependency flow
  - Inter-service communication explanation
  - How to run the project
  - Why shared database was not used
  - Failure scenario explanation with production considerations

- [x] **Architecture Diagram** (ARCHITECTURE.md)
  - Shows both services
  - Shows owned data for each service
  - Shows communication boundary
  - Shows clean architecture layers

- [x] **API Examples** (API_Examples.md)
  - Demonstrates all endpoints
  - Includes example payloads
  - Shows error scenarios
  - Includes complete test flow

### Project Structure

```
✓ doctor-service/
  ✓ cmd/doctor-service/main.go
  ✓ internal/model/doctor.go
  ✓ internal/usecase/doctor_usecase.go
  ✓ internal/repository/repository.go
  ✓ internal/transport/http/handler.go
  ✓ internal/app/app.go
  ✓ go.mod

✓ appointment-service/
  ✓ cmd/appointment-service/main.go
  ✓ internal/model/appointment.go
  ✓ internal/usecase/appointment_usecase.go
  ✓ internal/repository/repository.go
  ✓ internal/transport/http/handler.go
  ✓ internal/client/doctor_client.go
  ✓ internal/app/app.go
  ✓ go.mod

✓ main.go (runs both services)
✓ go.mod (root)
✓ README.md
✓ ARCHITECTURE.md
✓ API_Examples.md
```

### Technical Requirements

- [x] **Language**: Go
- [x] **Framework**: Gin for HTTP
- [x] **REST only**: No message queues or event brokers
- [x] **Compiles successfully**: `go build` works for both services
- [x] **Runs with go run .**: Main entry point works

## Testing Before Submission

### 1. Verify Compilation
```bash
cd doctor-service
go build ./cmd/doctor-service/main.go
cd ../appointment-service
go build ./cmd/appointment-service/main.go
```

### 2. Test Running Services
```bash
# Terminal 1
cd doctor-service && go run cmd/doctor-service/main.go

# Terminal 2
cd appointment-service && go run cmd/appointment-service/main.go
```

### 3. Test API Endpoints

```bash
# Create doctor
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Dr. Test","specialization":"Test","email":"test@test.com"}'

# Get doctor (use ID from response)
curl http://localhost:8080/doctors/{DOCTOR_ID}

# Create appointment
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{"title":"Test","description":"Test","doctor_id":"{DOCTOR_ID}"}'

# Update status
curl -X PATCH http://localhost:8081/appointments/{APPT_ID}/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

### 4. Test Failure Scenario
- Stop Doctor Service
- Try to create appointment
- Verify error message returned

## Submission Format

### Files to Submit

Create ZIP file named: **AP2_Assignment1_NameSurname.zip**

Include:
- doctor-service/ (entire folder)
- appointment-service/ (entire folder)
- main.go
- go.mod
- README.md
- ARCHITECTURE.md
- API_Examples.md

### DO NOT Include
- Compiled binaries (doctor-service, appointment-service executables)
- go.sum files (will be regenerated)
- .git folder
- IDE-specific folders (.vscode, .idea, etc.)
- .claude folder

### Submission Command
```bash
cd /Users/aleke/university_tasks/adv_prog2
zip -r AP2_Assignment1_NameSurname.zip \
  doctor-service/ \
  appointment-service/ \
  main.go \
  go.mod \
  README.md \
  ARCHITECTURE.md \
  API_Examples.md \
  -x "*.exe" "*.sum" "*/.git/*" "*/.idea/*" "*/.vscode/*" "*/.claude/*"
```

## Defense Preparation

Be prepared to explain:

1. **Clean Architecture**
   - Why handlers are thin
   - Where business logic lives
   - How dependency inversion works
   - Why use cases depend on interfaces

2. **Microservices vs Distributed Monolith**
   - What makes this microservices architecture
   - Data ownership principle
   - Service boundaries and independence
   - Why not shared database

3. **REST Communication**
   - How Appointment Service validates doctors
   - HTTP contract between services
   - What happens if Doctor Service is down
   - Timeout handling

4. **Failure Handling**
   - Current implementation: timeout + error response
   - Production needs: retry strategy, circuit breaker
   - When these patterns become necessary
   - Trade-offs of synchronous communication

5. **Design Decisions**
   - Why in-memory storage is acceptable here
   - How to swap storage implementation
   - Status transition validation logic location
   - Repository pattern benefits

6. **Code Walkthrough**
   - Be able to navigate and explain any file
   - Show dependency flow between layers
   - Demonstrate running and testing the system

## Final Verification

- [ ] Project compiles without errors
- [ ] Both services start successfully
- [ ] All API endpoints work correctly
- [ ] Business rules are enforced
- [ ] Failure scenario works as expected
- [ ] Documentation is complete and clear
- [ ] ZIP file created with correct name
- [ ] Submitted to Moodle before deadline (23:59 03.04.2026)

## Grading Criteria Alignment

| Criterion | Weight | Status |
|-----------|--------|--------|
| Clean Architecture inside services | 30% | ✅ Complete |
| Microservice decomposition | 20% | ✅ Complete |
| REST communication | 15% | ✅ Complete |
| Functionality | 20% | ✅ Complete |
| Documentation and explanation | 15% | ✅ Complete |

**Total**: 100% ✅
