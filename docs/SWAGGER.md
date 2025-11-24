# Swagger Integration Guide

This project uses Swagger/OpenAPI for API documentation with support for multiple microservices.

## Architecture

Each service has its own Swagger documentation:
- **Auth Service** (port 3005): `/swagger/`
- **Ride Service** (port 3000): `/swagger/`
- **Driver Service** (port 3001): `/swagger/`
- **Admin Service** (port 3004): `/swagger/`

The system uses a single `main.go` entry point that spawns different services based on the `--mode` flag.

## Setup

### 1. Install Swag CLI tool (one-time setup)

```bash
make swagger-install
# or
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Generate Swagger Documentation

Generate docs for all services:
```bash
make swagger-all
```

Or generate for specific services:
```bash
make swagger-auth    # Auth service
make swagger-ride    # Ride service
make swagger-driver  # Driver service
make swagger-admin   # Admin service
```

Alternatively, use the setup script:
```bash
./scripts/setup-swagger.sh
```

## Usage

### Starting Services

Start the auth service:
```bash
make run-auth
# or
go run main.go --mode=auth-service
```

Start the ride service:
```bash
make run-ride
# or
go run main.go --mode=ride-service
```

### Accessing Swagger UI

Once a service is running, access its Swagger UI:

- Auth Service: http://localhost:3005/swagger/
- Ride Service: http://localhost:3000/swagger/
- Driver Service: http://localhost:3001/swagger/
- Admin Service: http://localhost:3004/swagger/

## Adding Swagger Annotations

### Handler Documentation

Add Swagger comments above your handler functions:

```go
// CreateRide godoc
// @Summary      Create a new ride request
// @Description  Creates a new ride request for a passenger
// @Tags         ride
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateRideRequest true "Ride request details"
// @Success      201 {object} map[string]interface{} "Created ride details"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Security     BearerAuth
// @Router       /rides [post]
func (h *Ride) CreateRide(w http.ResponseWriter, r *http.Request) {
    // handler implementation
}
```

### Model Documentation

Add field tags to your DTOs:

```go
type CreateRideRequest struct {
    PassengerID    string  `json:"passenger_id" example:"123e4567-e89b-12d3-a456-426614174000"`
    PickupLat      float64 `json:"pickup_lat" example:"40.7128"`
    PickupLon      float64 `json:"pickup_lon" example:"-74.0060"`
    DestinationLat float64 `json:"destination_lat" example:"40.7580"`
    DestinationLon float64 `json:"destination_lon" example:"-73.9855"`
}
```

## Important Notes

1. **Tag Filtering**: Each service uses tag filtering (`--tags`) to only include relevant endpoints in its documentation
2. **Instance Names**: Each service has a unique instance name to avoid conflicts when all docs are imported
3. **Regeneration**: Run `make swagger-all` after modifying handler annotations or adding new endpoints
4. **Authentication**: Protected endpoints show a lock icon in Swagger UI. Use the "Authorize" button to add your Bearer token

## File Structure

```
docs/
├── swagger.go              # Main Swagger annotations for all services
├── auth/
│   ├── auth_docs.go
│   ├── auth_swagger.json
│   └── auth_swagger.yaml
├── ride/
│   ├── ride_docs.go
│   ├── ride_swagger.json
│   └── ride_swagger.yaml
├── driver/
│   ├── driver_docs.go
│   ├── driver_swagger.json
│   └── driver_swagger.yaml
└── admin/
    ├── admin_docs.go
    ├── admin_swagger.json
    └── admin_swagger.yaml
```

## Development Workflow

1. Add/modify handler endpoints
2. Add Swagger annotations to handlers
3. Run `make swagger-all` to regenerate docs
4. Start your service
5. Test via Swagger UI at `http://localhost:<port>/swagger/`

## Troubleshooting

**Issue**: Swagger UI shows "Failed to load API definition"
- Solution: Ensure you've run `make swagger-<service>` for that service

**Issue**: New endpoints not showing
- Solution: Check that you've added the correct `@Tags` annotation matching the service

**Issue**: Import errors
- Solution: Run `./scripts/setup-swagger.sh` to regenerate all docs and imports
