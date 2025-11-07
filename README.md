# Ride-Hail 🚗

A real-time distributed ride-hailing platform built with Go, featuring microservices architecture, message queues, WebSocket communication, and geospatial processing.

## 📋 Overview

Ride-Hail is a sophisticated backend system that simulates modern transportation platforms like Uber. Built using Service-Oriented Architecture (SOA) principles, it handles real-time ride requests, intelligent driver matching, live location tracking, and complex ride coordination across distributed microservices.

## 🎯 Key Features

### Core Capabilities
- **Real-Time Ride Matching**: Intelligent driver-passenger matching based on proximity and availability
- **Live Location Tracking**: Continuous GPS tracking with sub-second updates via WebSocket
- **Dynamic Pricing**: Distance and time-based fare calculation for multiple vehicle types
- **Ride Lifecycle Management**: Complete journey tracking from request to completion
- **Driver Session Management**: Track driver availability, earnings, and performance metrics
- **Event Sourcing**: Complete audit trail of all ride events for dispute resolution

### Advanced Features
- **Geospatial Calculations**: PostGIS-powered distance calculations and driver search
- **Message Queue Architecture**: RabbitMQ-based asynchronous communication between services
- **WebSocket Real-Time Updates**: Bidirectional communication for passengers and drivers
- **Concurrent Order Processing**: Handle thousands of simultaneous ride requests
- **Admin Dashboard**: System monitoring, analytics, and operational oversight
- **Graceful Degradation**: Circuit breakers, retries, and failover mechanisms

## 🏗️ System Architecture

### Service-Oriented Architecture (SOA)

The system consists of three independent microservices communicating via RabbitMQ and PostgreSQL:

```
┌─────────────┐        ┌───────────────────┐        ┌───────────┐
│  Passenger  │◄──────►│   Ride Service    │◄──────►│   Admin   │
│ (WebSocket) │        │  (Orchestrator)   │        │ Dashboard │
└─────────────┘        └───────────────────┘        └───────────┘
                                ▲
                                │
                                ▼
              ┌─────────────────────────────────────┐
              │    RabbitMQ Message Broker          │
              │                                     │
              │  Exchange: ride_topic               │
              │    • ride_requests                  │
              │    • ride_status                    │
              │                                     │
              │  Exchange: driver_topic             │
              │    • driver_matching                │
              │    • driver_responses               │
              │                                     │
              │  Exchange: location_fanout          │
              │    • location_updates               │
              └─────────────────────────────────────┘
                                ▲
                                │
                                ▼
┌─────────────┐        ┌───────────────────────┐
│   Driver    │◄──────►│  Driver & Location    │
│ (WebSocket) │        │      Service          │
└─────────────┘        └───────────────────────┘
```

### Microservices

1. **Ride Service** - Core orchestrator managing the complete ride lifecycle
2. **Driver & Location Service** - Handles driver operations, matching algorithms, and real-time GPS tracking
3. **Admin Service** - Provides monitoring, analytics, and system oversight

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- RabbitMQ 3.12+
- PostGIS extension

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd ride-hail
```
2. Build and run the system:
```bash
make up
```

### Configuration

The system uses environment variables for configuration:

```yaml
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=ridehail_user
DB_PASSWORD=ridehail_pass
DB_NAME=ridehail_db

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest

# Services
RIDE_SERVICE_PORT=3000
DRIVER_LOCATION_SERVICE_PORT=3001
ADMIN_SERVICE_PORT=3004
WS_PORT=8080
```

## 📡 API Documentation

### Ride Service (Port 3000)

#### Create Ride
```bash
POST /rides
Authorization: Bearer {passenger_token}
Content-Type: application/json

{
  "pickup_latitude": 43.238949,
  "pickup_longitude": 76.889709,
  "pickup_address": "Almaty Central Park",
  "destination_latitude": 43.222015,
  "destination_longitude": 76.851511,
  "destination_address": "Kok-Tobe Hill",
  "ride_type": "ECONOMY"
}
```

**Response:**
```json
{
  "ride_id": "uuid",
  "ride_number": "RIDE_20241216_001",
  "status": "REQUESTED",
  "estimated_fare": 1450.0,
  "estimated_duration_minutes": 15,
  "estimated_distance_km": 5.2
}
```

#### Cancel Ride
```bash
POST /rides/{ride_id}/cancel
Authorization: Bearer {passenger_token}
Content-Type: application/json

{
  "reason": "Changed my mind"
}
```

#### WebSocket Connection (Passengers)
```
ws://localhost:3000/ws/passengers/{passenger_id}

# Authentication message:
{
  "type": "auth",
  "token": "Bearer {passenger_token}"
}
```

### Driver & Location Service (Port 3001)

#### Go Online
```bash
POST /drivers/{driver_id}/online
Authorization: Bearer {driver_token}
Content-Type: application/json

{
  "latitude": 43.238949,
  "longitude": 76.889709
}
```

#### Update Location
```bash
POST /drivers/{driver_id}/location
Authorization: Bearer {driver_token}
Content-Type: application/json

{
  "latitude": 43.238949,
  "longitude": 76.889709,
  "accuracy_meters": 5.0,
  "speed_kmh": 45.0,
  "heading_degrees": 180.0
}
```

#### Start Ride
```bash
POST /drivers/{driver_id}/start
Authorization: Bearer {driver_token}
Content-Type: application/json

{
  "ride_id": "uuid",
  "driver_location": {
    "latitude": 43.238949,
    "longitude": 76.889709
  }
}
```

#### Complete Ride
```bash
POST /drivers/{driver_id}/complete
Authorization: Bearer {driver_token}
Content-Type: application/json

{
  "ride_id": "uuid",
  "final_location": {
    "latitude": 43.222015,
    "longitude": 76.851511
  },
  "actual_distance_km": 5.5,
  "actual_duration_minutes": 16
}
```

#### WebSocket Connection (Drivers)
```
ws://localhost:3001/ws/drivers/{driver_id}

# Authentication message:
{
  "type": "auth",
  "token": "Bearer {driver_token}"
}
```

### Admin Service (Port 3004)

#### System Overview
```bash
GET /admin/overview
Authorization: Bearer {admin_token}
```

**Response:**
```json
{
  "timestamp": "2024-12-16T10:30:00Z",
  "metrics": {
    "active_rides": 45,
    "available_drivers": 123,
    "busy_drivers": 45,
    "total_rides_today": 892,
    "total_revenue_today": 1234567.5,
    "average_wait_time_minutes": 4.2,
    "cancellation_rate": 0.05
  }
}
```

#### Active Rides
```bash
GET /admin/rides/active?page=1&page_size=20
Authorization: Bearer {admin_token}
```

## 🔄 Request Flow

### Phase 1: Ride Request Initiation
1. Passenger opens app and requests ride
2. Ride Service validates request and calculates fare
3. Ride stored with status 'REQUESTED'
4. Message published to RabbitMQ `ride_topic` exchange

### Phase 2: Driver Matching Process
1. Driver & Location Service receives match request
2. PostGIS query finds nearby available drivers (5km radius)
3. Ride offers sent to top drivers via WebSocket
4. First accepting driver is matched (30-second timeout)

### Phase 3: Ride Confirmation
1. Driver acceptance published to `driver_topic` exchange
2. Ride Service updates status to 'MATCHED'
3. Passenger receives driver details via WebSocket
4. Driver receives passenger pickup information

### Phase 4: Real-Time Tracking
1. Driver sends location updates every 3 seconds
2. Updates broadcast via `location_fanout` exchange
3. Passenger sees live driver location and ETA
4. Status transitions: EN_ROUTE → ARRIVED → IN_PROGRESS

### Phase 5: Ride Completion
1. Driver completes ride and submits final details
2. Final fare calculated based on actual distance/time
3. Ride Service updates status to 'COMPLETED'
4. Driver session updated with earnings
5. Both parties receive completion notification

## 📊 Message Queue Architecture

### Exchanges

| Exchange | Type | Purpose |
|----------|------|---------|
| `ride_topic` | Topic | Ride-related events with routing |
| `driver_topic` | Topic | Driver responses and status |
| `location_fanout` | Fanout | Broadcast location updates |

### Queues

| Queue | Exchange | Routing Key | Purpose |
|-------|----------|-------------|---------|
| `ride_requests` | ride_topic | `ride.request.*` | New ride requests |
| `ride_status` | ride_topic | `ride.status.*` | Status updates |
| `driver_matching` | ride_topic | `ride.request.*` | Match requests |
| `driver_responses` | driver_topic | `driver.response.*` | Driver acceptance |
| `driver_status` | driver_topic | `driver.status.*` | Driver availability |
| `location_updates_ride` | location_fanout | N/A | Location broadcasts |

## 💾 Database Schema

### Core Tables

- **users** - User accounts (passengers, drivers, admins)
- **drivers** - Driver profiles and vehicle information
- **rides** - Main ride records with lifecycle tracking
- **coordinates** - Pickup/destination locations with geospatial data
- **location_history** - GPS tracking history for analytics
- **ride_events** - Event sourcing audit trail
- **driver_sessions** - Driver online/offline tracking

### Key Features

- **PostGIS Extension**: Geospatial calculations and distance queries
- **JSONB Columns**: Flexible storage for vehicle attributes and custom data
- **ENUMs**: Type-safe status values
- **Timestamps with Timezone**: Accurate time tracking across regions
- **Foreign Key Constraints**: Referential integrity
- **Indexes**: Optimized queries for status, location, and temporal data

## 🛠️ Technical Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+ with PostGIS extension
- **Message Queue**: RabbitMQ 3.12+
- **Real-Time Communication**: Gorilla WebSocket
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Database Driver**: pgx/v5
- **AMQP Client**: rabbitmq/amqp091-go

## 🔒 Security

### Authentication & Authorization
- JWT-based authentication for all endpoints
- Role-based access control (Passenger, Driver, Admin)
- Service-to-service authentication tokens
- WebSocket authentication with 5-second timeout

### Data Protection
- TLS encryption for all communications
- Sensitive data encryption at rest
- Sanitized logging (no passwords, tokens, phone numbers)
- Input validation and SQL injection prevention

### Coordinate Validation
- Latitude: -90 to 90 degrees
- Longitude: -180 to 180 degrees
- Accuracy thresholds for GPS data

## 📝 Logging

All services implement structured JSON logging:

```json
{
  "timestamp": "2024-12-16T10:30:00Z",
  "level": "INFO",
  "service": "ride-service",
  "action": "ride_requested",
  "message": "New ride request created",
  "hostname": "ride-service-01",
  "request_id": "req_123456",
  "ride_id": "uuid"
}
```

## 📈 Performance Considerations

- **Location Update Rate Limiting**: Max 1 update per 3 seconds per driver
- **Driver Search Radius**: 5km for optimal matching
- **Match Timeout**: 30 seconds per driver offer
- **WebSocket Keep-Alive**: Ping every 30 seconds
- **Database Connection Pooling**: Configurable pool size
- **Message Queue Prefetch**: Optimized for throughput
