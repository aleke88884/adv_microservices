# API Examples

## Doctor Service (Port 8080)

### 1. Create Doctor
```bash
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }'
```

Response (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

### 2. Get Doctor by ID
```bash
curl -X GET http://localhost:8080/doctors/550e8400-e29b-41d4-a716-446655440000
```

Response (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

### 3. Get All Doctors
```bash
curl -X GET http://localhost:8080/doctors
```

Response (200 OK):
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }
]
```

### 4. Error: Duplicate Email
```bash
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Dr. Another Doctor",
    "specialization": "Neurology",
    "email": "a.seitkali@clinic.kz"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "email must be unique"
}
```

### 5. Error: Missing Required Field
```bash
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{
    "specialization": "Cardiology"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "full_name is required"
}
```

## Appointment Service (Port 8081)

### 1. Create Appointment
```bash
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

Response (201 Created):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "new",
  "created_at": "2026-03-31T10:30:00Z",
  "updated_at": "2026-03-31T10:30:00Z"
}
```

### 2. Get Appointment by ID
```bash
curl -X GET http://localhost:8081/appointments/660e8400-e29b-41d4-a716-446655440001
```

Response (200 OK):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "new",
  "created_at": "2026-03-31T10:30:00Z",
  "updated_at": "2026-03-31T10:30:00Z"
}
```

### 3. Get All Appointments
```bash
curl -X GET http://localhost:8081/appointments
```

Response (200 OK):
```json
[
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "new",
    "created_at": "2026-03-31T10:30:00Z",
    "updated_at": "2026-03-31T10:30:00Z"
  }
]
```

### 4. Update Appointment Status to in_progress
```bash
curl -X PATCH http://localhost:8081/appointments/660e8400-e29b-41d4-a716-446655440001/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress"
  }'
```

Response (200 OK):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "in_progress",
  "created_at": "2026-03-31T10:30:00Z",
  "updated_at": "2026-03-31T10:35:00Z"
}
```

### 5. Update Appointment Status to done
```bash
curl -X PATCH http://localhost:8081/appointments/660e8400-e29b-41d4-a716-446655440001/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "done"
  }'
```

Response (200 OK):
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctor_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "done",
  "created_at": "2026-03-31T10:30:00Z",
  "updated_at": "2026-03-31T10:40:00Z"
}
```

### 6. Error: Invalid Status Transition (done to new)
```bash
curl -X PATCH http://localhost:8081/appointments/660e8400-e29b-41d4-a716-446655440001/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "new"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "cannot transition from done to new"
}
```

### 7. Error: Doctor Does Not Exist
```bash
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Follow-up consultation",
    "description": "Check test results",
    "doctor_id": "non-existent-doctor-id"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "doctor does not exist"
}
```

### 8. Error: Doctor Service Unavailable
Stop the Doctor Service and try to create an appointment:

```bash
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Emergency consultation",
    "description": "Urgent case",
    "doctor_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

Response (400 Bad Request):
```json
{
  "error": "failed to validate doctor: failed to call doctor service: Get \"http://localhost:8080/doctors/550e8400-e29b-41d4-a716-446655440000\": dial tcp [::1]:8080: connect: connection refused"
}
```

## Complete Test Flow

Run these commands in order to test the complete system:

```bash
# 1. Create a doctor
curl -X POST http://localhost:8080/doctors \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }'

# Copy the doctor ID from response, then use it below

# 2. List all doctors
curl -X GET http://localhost:8080/doctors

# 3. Create an appointment with valid doctor ID
curl -X POST http://localhost:8081/appointments \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "YOUR_DOCTOR_ID_HERE"
  }'

# Copy the appointment ID from response

# 4. Get the appointment
curl -X GET http://localhost:8081/appointments/YOUR_APPOINTMENT_ID_HERE

# 5. Update status to in_progress
curl -X PATCH http://localhost:8081/appointments/YOUR_APPOINTMENT_ID_HERE/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress"
  }'

# 6. Update status to done
curl -X PATCH http://localhost:8081/appointments/YOUR_APPOINTMENT_ID_HERE/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "done"
  }'

# 7. Try invalid transition (should fail)
curl -X PATCH http://localhost:8081/appointments/YOUR_APPOINTMENT_ID_HERE/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "new"
  }'

# 8. List all appointments
curl -X GET http://localhost:8081/appointments
```
