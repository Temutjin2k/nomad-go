# 🚗 Ride-Hailing Platform

A real-time distributed ride-hailing system built with Go, implementing Service-Oriented Architecture (SOA) principles. This project demonstrates advanced microservices patterns including message queues, WebSocket communication, geospatial data processing, and distributed state management.
## 🎯 Overview

This platform simulates the backend infrastructure of modern transportation services like Uber. It handles real-time ride requests, intelligent driver matching, live location tracking, and complex ride coordination across multiple microservices.

### Key Learning Objectives

- **Advanced Message Queue Patterns** - RabbitMQ topic/fanout exchanges
- **Real-Time Communication** - WebSocket bidirectional streaming
- **Geospatial Processing** - PostGIS distance calculations and matching
- **Microservices Orchestration** - Event-driven architecture
- **High-Concurrency Programming** - Goroutines and channels
- **Distributed State Management** - Database transactions and consistency

## ✨ Features

### Core Functionality

- 🚕 **Real-time Ride Matching** - Intelligent algorithm matches passengers with nearby drivers
- 📍 **Live Location Tracking** - Real-time GPS updates with WebSocket streaming
- 💰 **Dynamic Pricing** - Distance and duration-based fare calculation
- 🔔 **Push Notifications** - Instant updates for ride status changes
- 📊 **Admin Dashboard** - System metrics and active ride monitoring
- 🔐 **JWT Authentication** - Secure role-based access control
- 📈 **Event Sourcing** - Complete audit trail for all ride events

### Business Logic

- **Multiple Vehicle Types**: Economy, Premium, XL
- **Smart Driver Selection**: Distance + rating based matching
- **Timeout Management**: Automatic fallback if drivers don't respond
- **Session Tracking**: Driver earnings and ride statistics
- **Cancellation Handling**: Refund logic and reason tracking

## 🏗️ Architecture

### Service-Oriented Architecture (SOA)

The system consists of four independent microservices:

```
┌─────────────┐         ┌──────────────────┐         ┌──────────┐
│  Passenger  │◄───────►│   Ride Service   │◄───────►│  Admin   │
│ (WebSocket) │         │  (Orchestrator)  │         │Dashboard │
└─────────────┘         └──────────────────┘         └──────────┘
                                 ▲
                                 │
                                 ▼
                ┌────────────────────────────────────┐
                │    RabbitMQ Message Broker         │
                │                                    │
                │  Exchanges:                        │
                │  • ride_topic    (topic)           │
                │  • driver_topic  (topic)           │
                │  • location_fanout (fanout)        │
                └────────────────────────────────────┘
                                 ▲
                                 │
                                 ▼
┌─────────────┐         ┌──────────────────┐
│   Driver    │◄───────►│ Driver Location  │
│ (WebSocket) │         │     Service      │
└─────────────┘         └──────────────────┘
                                 ▲
                                 │
                                 ▼
                        ┌────────────────┐
                        │   PostgreSQL   │
                        │   + PostGIS    │
                        └────────────────┘
```

### Services Overview

#### 1. **Auth Service** 🔐
- User registration and authentication
- JWT token generation and validation
- Role-based access control (Passenger, Driver, Admin)

#### 2. **Ride Service** 🚗
- Ride lifecycle orchestration
- Fare calculation and estimation
- Passenger WebSocket connections
- Ride status management
- Cancellation handling

#### 3. **Driver & Location Service** 📍
- Driver registration and availability
- Intelligent matching algorithm
- Real-time location tracking
- Driver WebSocket connections
- Session management

#### 4. **Admin Service** 📊
- System metrics and analytics
- Active ride monitoring
- Driver distribution tracking
- Revenue reporting

## 🛠️ Tech Stack

### Backend
- **Go 1.21+** - High-performance concurrent programming
- **PostgreSQL 15+** - Relational database with PostGIS extension
- **RabbitMQ 3.12+** - Message broker for service communication
- **WebSocket** - Real-time bidirectional communication

### Libraries
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/rabbitmq/amqp091-go` - AMQP client
- `github.com/gorilla/websocket` - WebSocket implementation
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `gopkg.in/yaml.v3` - Configuration management

### Infrastructure
- **Docker & Docker Compose** - Containerization
- **Make** - Build automation
- **golang-migrate** - Database migrations

## 📦 Prerequisites

Ensure you have the following installed:

- **Go 1.21 or higher**
- **Docker & Docker Compose**
- **Make** (optional, for convenience)
- **golang-migrate** (for database migrations)

Install golang-migrate:
```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Windows (using Scoop)
scoop install migrate
```

## 🚀 Installation

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/ride-hailing-platform.git
cd ride-hailing-platform
```

### 2. Configure Environment

Create a `config.yaml` file in the project root:

```yaml
database:
  host: localhost
  port: 5432
  user: ridehail_user
  password: ridehail_pass
  database: ridehail_db

rabbitmq:
  host: localhost
  port: 5672
  user: guest
  password: guest

websocket:
  port: 8080

services:
  ride_service: 3000
  driver_location_service: 3001
  admin_service: 3004
  auth_service: 3005
```

### 3. Start Infrastructure

Start PostgreSQL and RabbitMQ using Docker Compose:

```bash
make up
# or
docker-compose up -d
```

This will start:
- **PostgreSQL** on port `5432`
- **RabbitMQ** on port `5672` (Management UI: http://localhost:15672)

### 4. Run Database Migrations

```bash
make migrate-up
```

This will create all necessary tables and seed initial data.

### 5. Build the Application

```bash
make build-go
# or
go build -o ride-hail-system .
```

## 🎮 Running the Application

Each service must be started in a separate terminal:

### Terminal 1: Auth Service
```bash
make run-auth
# or
go run main.go --mode=auth-service
```

### Terminal 2: Ride Service
```bash
make run-ride
# or
go run main.go --mode=ride-service
```

### Terminal 3: Driver & Location Service
```bash
make run-driver
# or
go run main.go --mode=driver-service
```

### Terminal 4: Admin Service
```bash
make run-admin
# or
go run main.go --mode=admin-service
```

### Verify Services

Check that all services are running:

```bash
# Auth Service
curl http://localhost:3005/health

# Ride Service
curl http://localhost:3000/health

# Driver Service
curl http://localhost:3001/health

# Admin Service
curl http://localhost:3004/health
```

## 📚 API Documentation

### Authentication

All API requests (except registration/login) require JWT authentication:

```http
Authorization: Bearer <your_jwt_token>
```

### Auth Service (Port 3005)

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password",
  "role": "PASSENGER"
}
```

**Response (201):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "email": "user@example.com",
  "role": "PASSENGER"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "email": "user@example.com",
    "role": "PASSENGER"
  }
}
```

### Ride Service (Port 3000)

#### Create Ride Request
```http
POST /rides
Content-Type: application/json
Authorization: Bearer {passenger_token}

{
  "passenger_id": "550e8400-e29b-41d4-a716-446655440001",
  "pickup_latitude": 43.238949,
  "pickup_longitude": 76.889709,
  "pickup_address": "Almaty Central Park",
  "destination_latitude": 43.222015,
  "destination_longitude": 76.851511,
  "destination_address": "Kok-Tobe Hill",
  "ride_type": "ECONOMY"
}
```

**Response (201):**
```json
{
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "ride_number": "RIDE_20241216_001",
  "status": "REQUESTED",
  "estimated_fare": 1450.0,
  "estimated_duration_minutes": 15,
  "estimated_distance_km": 5.2
}
```

#### Cancel Ride
```http
POST /rides/{ride_id}/cancel
Content-Type: application/json
Authorization: Bearer {passenger_token}

{
  "reason": "Changed my mind"
}
```

### Driver Service (Port 3001)

#### Go Online
```http
POST /drivers/{driver_id}/online
Content-Type: application/json
Authorization: Bearer {driver_token}

{
  "latitude": 43.238949,
  "longitude": 76.889709
}
```

#### Update Location
```http
POST /drivers/{driver_id}/location
Content-Type: application/json
Authorization: Bearer {driver_token}

{
  "latitude": 43.238949,
  "longitude": 76.889709,
  "accuracy_meters": 5.0,
  "speed_kmh": 45.0,
  "heading_degrees": 180.0
}
```

#### Start Ride
```http
POST /drivers/{driver_id}/start
Content-Type: application/json
Authorization: Bearer {driver_token}

{
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "driver_location": {
    "latitude": 43.238949,
    "longitude": 76.889709
  }
}
```

#### Complete Ride
```http
POST /drivers/{driver_id}/complete
Content-Type: application/json
Authorization: Bearer {driver_token}

{
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "final_location": {
    "latitude": 43.222015,
    "longitude": 76.851511
  },
  "actual_distance_km": 5.5,
  "actual_duration_minutes": 16
}
```

### Admin Service (Port 3004)

#### Get System Overview
```http
GET /admin/overview
Authorization: Bearer {admin_token}
```

**Response (200):**
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
    "average_ride_duration_minutes": 18.5,
    "cancellation_rate": 0.05
  }
}
```

#### Get Active Rides
```http
GET /admin/rides/active?page=1&page_size=20
Authorization: Bearer {admin_token}
```

## 🔌 WebSocket Protocol

### Passenger Connection

**Connect:**
```javascript
const ws = new WebSocket('ws://localhost:3000/ws/passengers/{passenger_id}');
```

**Authenticate:**
```json
{
  "type": "auth",
  "token": "Bearer eyJhbGciOiJIUzI1NiIs..."
}
```

**Receive Events:**

```json
{
  "type": "ride_status_update",
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "MATCHED",
  "driver_info": {
    "driver_id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "Aidar Nurlan",
    "rating": 4.8,
    "vehicle": {
      "make": "Toyota",
      "model": "Camry",
      "color": "White",
      "plate": "KZ 123 ABC"
    }
  }
}
```

### Driver Connection

**Connect:**
```javascript
const ws = new WebSocket('ws://localhost:3001/ws/drivers/{driver_id}');
```

**Receive Ride Offers:**
```json
{
  "type": "ride_offer",
  "offer_id": "offer_123456",
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "pickup_location": {
    "latitude": 43.238949,
    "longitude": 76.889709,
    "address": "Almaty Central Park"
  },
  "estimated_fare": 1500.0,
  "driver_earnings": 1200.0,
  "expires_at": "2024-12-16T10:32:00Z"
}
```

**Accept/Reject Ride:**
```json
{
  "type": "ride_response",
  "offer_id": "offer_123456",
  "ride_id": "550e8400-e29b-41d4-a716-446655440000",
  "accepted": true,
  "current_location": {
    "latitude": 43.235,
    "longitude": 76.885
  }
}
```

## 🔄 Request Flow - Step by Step

### PHASE 1: RIDE REQUEST INITIATION

<div align="center">
  <img src="assets/images/phase1.png" alt="Phase 1: Ride Request Initiation" width="800"/>
</div>

**What happens:**
1. **Passenger opens the app** and enters pickup and destination locations
2. **Ride Service receives** the ride request via REST API
3. **Fare calculation** is performed based on distance, duration, and vehicle type
4. **Ride record is created** in the database with status `REQUESTED`
5. **Request is published** to RabbitMQ `ride_topic` exchange with routing key `ride.request.{ride_type}`
6. **Passenger WebSocket connection** receives confirmation of request submission

**Key Components:**
- REST API endpoint: `POST /rides`
- Database: Insert into `rides` and `coordinates` tables
- Message Queue: Publish to `ride_topic` exchange
- Response: Estimated fare and ride details

---

### PHASE 2: DRIVER MATCHING PROCESS

<div align="center">
  <img src="assets/images/phase2.png" alt="Phase 2: Driver Matching Process" width="800"/>
</div>

**What happens:**
1. **Driver Service consumes** the ride request from `driver_matching` queue
2. **Geospatial query** finds available drivers within 5km radius using PostGIS:
```sql
   SELECT d.id, ST_Distance(...) as distance_km
   FROM drivers d
   JOIN coordinates c ON c.entity_id = d.id
   WHERE d.status = 'AVAILABLE'
     AND d.vehicle_type = 'ECONOMY'
     AND ST_DWithin(geography_point, pickup_point, 5000)
   ORDER BY distance_km, d.rating DESC
   LIMIT 10
```
3. **Ride offers sent** to selected drivers via WebSocket
4. **30-second timeout** starts for each driver to respond
5. **First driver to accept** wins the ride match

**Key Components:**
- Queue: `driver_matching` bound to `ride.request.*`
- Database: PostGIS geospatial queries on `coordinates` table
- WebSocket: Push ride offers to drivers
- Logic: Timeout management and offer expiration

---

### PHASE 3: RIDE CONFIRMATION AND SETUP

<div align="center">
  <img src="assets/images/phase3.png" alt="Phase 3: Ride Confirmation" width="800"/>
</div>

**What happens:**
1. **Driver accepts** the ride offer via WebSocket
2. **Driver Service publishes** acceptance to `driver_topic` with routing key `driver.response.{ride_id}`
3. **Ride Service consumes** the driver response and updates ride status to `MATCHED`
4. **Driver status updated** to `BUSY` in the database
5. **Passenger notified** via WebSocket with driver details (name, rating, vehicle info, ETA)
6. **Other pending offers** are automatically cancelled
7. **Ride event logged** to `ride_events` table for audit trail

**Key Components:**
- WebSocket: Driver acceptance message
- Message Queue: `driver.response.{ride_id}` routing
- Database: Update `rides.status` to `MATCHED`, `rides.driver_id`, `drivers.status` to `BUSY`
- Notification: Passenger receives driver information and estimated arrival

---

### PHASE 4: REAL-TIME TRACKING AND UPDATES

<div align="center">
  <img src="assets/images/phase4.png" alt="Phase 4: Real-time Tracking" width="800"/>
</div>

**What happens:**
1. **Driver updates location** every 3-5 seconds via `POST /drivers/{driver_id}/location`
2. **Location stored** in `coordinates` table (previous location marked as `is_current=false`)
3. **Location broadcast** to `location_fanout` exchange (fanout type - all subscribers receive)
4. **Ride Service consumes** location updates and forwards to passenger via WebSocket
5. **ETA recalculated** based on current distance and speed
6. **Status transitions:**
   - `MATCHED` → `EN_ROUTE` (driver heading to pickup)
   - `EN_ROUTE` → `ARRIVED` (driver at pickup location)
   - `ARRIVED` → `IN_PROGRESS` (ride started)

**Key Components:**
- REST API: `POST /drivers/{driver_id}/location`
- Database: Real-time updates to `coordinates` and `location_history`
- Message Queue: Fanout exchange broadcasts to all interested services
- WebSocket: Continuous location stream to passenger
- Rate Limiting: Max 1 update per 3 seconds

---

### PHASE 5: RIDE EXECUTION AND COMPLETION

<div align="center">
  <img src="assets/images/phase5.png" alt="Phase 5: Ride Completion" width="800"/>
</div>

**What happens:**
1. **Driver starts ride** via `POST /drivers/{driver_id}/start`
   - Ride status: `IN_PROGRESS`
   - `started_at` timestamp recorded
2. **Continuous location tracking** during the ride
3. **Driver completes ride** via `POST /drivers/{driver_id}/complete`
   - Final location, distance, and duration submitted
4. **Final fare calculated:**
```
   final_fare = base_fare + (actual_distance_km × rate_per_km) + (actual_duration_min × rate_per_min)
```
5. **Database updates:**
   - `rides.status` → `COMPLETED`
   - `rides.final_fare` calculated
   - `rides.completed_at` timestamp
   - `drivers.status` → `AVAILABLE`
   - `drivers.total_rides` incremented
   - `drivers.total_earnings` updated
6. **Ride event logged** with completion details
7. **Both parties notified** via WebSocket
8. **Driver session updated** with earnings

**Key Components:**
- REST API: Start and complete endpoints
- Database: Transaction ensuring ride completion and driver availability
- Fare Logic: Distance and duration-based calculation
- WebSocket: Completion notifications
- Analytics: Session tracking and driver statistics

**Fare Rates:**
| Vehicle Type | Base Fare | Per KM | Per Minute |
|--------------|-----------|--------|------------|
| ECONOMY      | 500₸     | 100₸   | 50₸        |
| PREMIUM      | 800₸     | 120₸   | 60₸        |
| XL           | 1000₸    | 150₸   | 75₸        |

---

## 🔄 Cancellation Flow

**At any phase, either party can cancel:**

**Passenger Cancellation:**
```http
POST /rides/{ride_id}/cancel
{
  "reason": "Changed my mind"
}
```

**What happens:**
1. Ride status → `CANCELLED`
2. If driver matched → driver status → `AVAILABLE`
3. Cancellation event logged with reason
4. Both parties notified via WebSocket
5. Refund logic applied based on cancellation timing

**Driver Rejection:**
- If driver rejects offer → next driver in queue gets the offer
- After 2 minutes with no acceptance → ride request expires
- Passenger notified to try again or adjust pickup location


## 📨 Message Queue Architecture

### Exchanges

| Exchange | Type | Purpose |
|----------|------|---------|
| `ride_topic` | Topic | Ride-related messages with routing |
| `driver_topic` | Topic | Driver-related messages with routing |
| `location_fanout` | Fanout | Broadcast location updates |

### Routing Keys

**Ride Topic:**
- `ride.request.ECONOMY`
- `ride.request.PREMIUM`
- `ride.request.XL`
- `ride.status.MATCHED`
- `ride.status.COMPLETED`

**Driver Topic:**
- `driver.response.{ride_id}`
- `driver.status.{driver_id}`

### Message Flow Example

1. **Passenger requests ride** → Ride Service publishes to `ride_topic` with key `ride.request.ECONOMY`
2. **Driver Service consumes** from `driver_matching` queue
3. **Finds nearby drivers** using PostGIS
4. **Sends offers via WebSocket** to selected drivers
5. **Driver accepts** → Publishes to `driver_topic` with key `driver.response.{ride_id}`
6. **Ride Service updates** ride status to `MATCHED`
7. **Notifies passenger** via WebSocket

## 💾 Database Schema

### Key Tables

**users** - Passenger, driver, and admin accounts
**drivers** - Driver-specific information
**rides** - Core ride records
**coordinates** - Location tracking
**ride_events** - Event sourcing audit trail
**location_history** - GPS history for analytics

### Entity Relationships

```
users (1) ──── (N) rides
users (1) ──── (1) drivers
rides (1) ──── (N) ride_events
rides (1) ──── (2) coordinates (pickup & destination)
drivers (1) ──── (N) location_history
```

## 🔧 Development

### Code Formatting

This project uses `gofumpt` for code formatting:

```bash
make format
# or
gofumpt -l -w .
```

**⚠️ All code must be formatted with gofumpt before submission.**

### Database Migrations

Create a new migration:
```bash
make migrate-create name=add_ratings_table
```

Apply migrations:
```bash
make migrate-up
```

Rollback last migration:
```bash
make migrate-down1
```

Check migration version:
```bash
make migrate-version
```

### Logging

All services use structured JSON logging:

```json
{
  "timestamp": "2024-12-16T10:30:00Z",
  "level": "INFO",
  "service": "ride-service",
  "action": "ride_requested",
  "message": "New ride request created",
  "hostname": "ride-service-1",
  "request_id": "req_123456",
  "ride_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## 🧪 Testing

### Manual Testing Flow

1. **Register users:**
```bash
# Register passenger
curl -X POST http://localhost:3005/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"passenger@test.com","password":"pass123","role":"PASSENGER"}'

# Register driver
curl -X POST http://localhost:3005/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"driver@test.com","password":"pass123","role":"DRIVER"}'
```

2. **Login and get tokens**

3. **Driver goes online:**
```bash
curl -X POST http://localhost:3001/drivers/{driver_id}/online \
  -H "Authorization: Bearer {driver_token}" \
  -H "Content-Type: application/json" \
  -d '{"latitude":43.238949,"longitude":76.889709}'
```

4. **Passenger requests ride:**
```bash
curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer {passenger_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "passenger_id":"550e8400-e29b-41d4-a716-446655440001",
    "pickup_latitude":43.238949,
    "pickup_longitude":76.889709,
    "pickup_address":"Almaty Central Park",
    "destination_latitude":43.222015,
    "destination_longitude":76.851511,
    "destination_address":"Kok-Tobe Hill",
    "ride_type":"ECONOMY"
  }'
```

5. **Monitor WebSocket connections** for real-time updates

## 🐛 Troubleshooting

### Services won't start

**Check if ports are available:**
```bash
lsof -i :3000  # Ride Service
lsof -i :3001  # Driver Service
lsof -i :3004  # Admin Service
lsof -i :3005  # Auth Service
```

### RabbitMQ connection issues

**Verify RabbitMQ is running:**
```bash
docker ps | grep rabbitmq
```

**Check Management UI:**
- URL: http://localhost:15672
- Username: `guest`
- Password: `guest`

### Database connection errors

**Verify PostgreSQL is running:**
```bash
docker ps | grep postgres
```

**Test connection:**
```bash
psql -h localhost -p 5432 -U ridehail_user -d ridehail_db
```

### WebSocket authentication fails

- Ensure token is prefixed with `Bearer `
- Check token expiration
- Verify user role matches endpoint

### Messages not flowing between services

1. Check RabbitMQ Management UI for queue depths
2. Verify exchange bindings are correct
3. Check service logs for correlation IDs
4. Ensure routing keys match expected patterns

## 🛑 Stopping the Application

### Stop all services
```bash
# Stop infrastructure only
make down

# Stop and remove volumes (clean slate)
make nuke
```

### Manual cleanup
```bash
# Stop Docker containers
docker-compose down -v

# Kill running Go processes
pkill -f "ride-hail-system"
```

## 📁 Project Structure

```
ride-hailing-platform/
├── cmd/
│   ├── auth/           # Auth service
│   ├── ride/           # Ride service
│   ├── driver/         # Driver & location service
│   └── admin/          # Admin service
├── internal/
│   ├── config/         # Configuration
│   ├── database/       # Database utilities
│   ├── messaging/      # RabbitMQ helpers
│   ├── models/         # Data models
│   └── websocket/      # WebSocket handlers
├── migrations/         # Database migrations
├── docker-compose.yml  # Infrastructure setup
├── Makefile           # Build automation
├── config.yaml        # Configuration file
└── main.go            # Entry point
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Format code with gofumpt (`make format`)
4. Commit changes (`git commit -m 'Add amazing feature'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.


---

**Built with ❤️ using Go, PostgreSQL, and RabbitMQ**