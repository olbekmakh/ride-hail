🚗 ride-hail-system
Learning Objectives
Service-Oriented Architecture (SOA)
Event-driven systems
Message queues (RabbitMQ)
Real-time communication (WebSocket)
PostgreSQL data modeling
Concurrency and synchronization in Go
Distributed system coordination
Abstract
This project implements a simplified backend of a ride-hailing platform. The system demonstrates how real-time distributed services communicate using message queues and WebSockets to manage ride requests, driver matching, location tracking, and ride lifecycle management.

The project is focused on backend architecture and asynchronous communication patterns rather than UI or frontend development.

Technologies
Go 1.22+
PostgreSQL
RabbitMQ
Gorilla WebSocket
JWT authentication
Docker & Docker Compose
Architecture Overview
The system consists of three services:

Ride Service
Creates and manages rides
Publishes ride events
Sends ride status updates to passengers
Driver & Location Service
Manages drivers (online/offline)
Matches drivers to rides
Tracks and publishes driver locations
Handles driver WebSocket connections
Admin Service
Provides system monitoring endpoints
Lists active rides and statistics
Services communicate via:

HTTP (REST)
WebSocket (real-time updates)
RabbitMQ (asynchronous messaging)
Project Structure
ride-hail-system/ ├── main.go ├── go.mod ├── docker-compose.yml ├── README.md ├── docs/ │ └── architecture/ │ ├── phase1.png │ ├── phase2.png │ ├── phase3.png │ ├── phase4.png │ └── phase5.png ├── internal/ │ ├── config/ │ ├── logger/ │ ├── postgres/ │ ├── rabbit/ │ ├── jwt/ │ ├── websocket/ │ ├── httpx/ │ └── geo/ ├── services/ │ ├── ride/ │ ├── driver/ │ └── admin/ └── migrations/ ├── 001_init.sql └── 002_seed.sql

Ride Lifecycle Phases
The full ride flow is divided into five phases:

Ride Request
Driver Matching
Ride Confirmation
Real-Time Tracking
Ride Completion
Sequence diagrams for each phase are located in: docs/architecture/

Installation & Run (Windows)
1. Start infrastructure
docker compose up -d
Services:

PostgreSQL → localhost:5433

RabbitMQ → localhost:5672

RabbitMQ UI → http://localhost:15672

2. Apply database migrations
docker exec -it ridehail-postgres psql -U ridehail_user ridehail_db
Execute:

migrations/001_init.sql

migrations/002_seed.sql

Exit:

\q
3. Environment variables
$env:DB_HOST="localhost"
$env:DB_PORT="5433"
$env:DB_USER="ridehail_user"
$env:DB_PASSWORD="ridehail_pass"
$env:DB_NAME="ridehail_db"

$env:RABBITMQ_HOST="localhost"
$env:RABBITMQ_PORT="5672"
$env:RABBITMQ_USER="guest"
$env:RABBITMQ_PASSWORD="guest"

$env:JWT_SECRET="dev_secret_change_me"

$env:RIDE_SERVICE_PORT="3000"
$env:DRIVER_LOCATION_SERVICE_PORT="3001"
$env:ADMIN_SERVICE_PORT="3004"
4. Build and run
go mod tidy
gofumpt -w .
go build -o ride-hail-system.exe .
.\ride-hail-system.exe
API Endpoints
Ride Service
POST /rides

POST /rides/{ride_id}/cancel

WS /ws/passengers/{passenger_id}

Driver & Location Service
POST /drivers/{id}/online

POST /drivers/{id}/offline

POST /drivers/{id}/location

POST /drivers/{id}/arrived

POST /drivers/{id}/start

POST /drivers/{id}/complete

WS /ws/drivers/{driver_id}

Admin Service
GET /admin/overview

GET /admin/rides/active

Authentication
The system uses JWT authentication with role-based access control:

PASSENGER

DRIVER

ADMIN

Notes
This project is for educational purposes only

Focus is on backend architecture and communication

Not intended for production use

AuthorAuthor
Oljas Bekmahan GitHub: https://github.com/olbekmakh

License
Educational project (Alem School)
