// Example Swagger Annotations for Different Handler Types
// Copy these patterns when adding Swagger docs to your handlers

package docs

// ============================================
// AUTH SERVICE EXAMPLES (@Tags auth)
// ============================================

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user account (driver or passenger)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterUserRequest true "User registration details"
// @Success      201 {object} map[string]interface{} "User ID"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      422 {object} map[string]interface{} "Validation error"
// @Failure      500 {object} map[string]interface{} "Internal server error"
// @Router       /auth/register [post]

// Login godoc
// @Summary      User login
// @Description  Authenticate user and receive JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200 {object} map[string]interface{} "Access and refresh tokens"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Router       /auth/login [post]

// Refresh godoc
// @Summary      Refresh access token
// @Description  Get a new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body map[string]string true "Refresh token"
// @Success      200 {object} map[string]interface{} "New token pair"
// @Failure      401 {object} map[string]interface{} "Invalid refresh token"
// @Router       /auth/refresh [post]

// Profile godoc
// @Summary      Get user profile
// @Description  Get current user profile information
// @Tags         auth
// @Produce      json
// @Success      200 {object} models.User "User profile"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Security     BearerAuth
// @Router       /auth/me [get]

// ============================================
// RIDE SERVICE EXAMPLES (@Tags ride)
// ============================================

// CreateRide godoc
// @Summary      Create a new ride request
// @Description  Creates a new ride request for a passenger
// @Tags         ride
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateRideRequest true "Ride request details"
// @Success      201 {object} map[string]interface{} "Created ride details"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden"
// @Failure      422 {object} map[string]interface{} "Validation error"
// @Security     BearerAuth
// @Router       /rides [post]

// CancelRide godoc
// @Summary      Cancel a ride
// @Description  Cancel an existing ride request
// @Tags         ride
// @Accept       json
// @Produce      json
// @Param        ride_id path string true "Ride ID"
// @Param        request body map[string]string true "Cancel reason"
// @Success      200 {object} models.Ride "Updated ride"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      404 {object} map[string]interface{} "Ride not found"
// @Security     BearerAuth
// @Router       /rides/{ride_id}/cancel [post]

// ============================================
// DRIVER SERVICE EXAMPLES (@Tags driver)
// ============================================

// Register godoc
// @Summary      Register a new driver
// @Description  Register a new driver with vehicle information
// @Tags         driver
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterDriverRequest true "Driver registration details"
// @Success      201 {object} map[string]interface{} "Driver ID"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Failure      422 {object} map[string]interface{} "Validation error"
// @Router       /drivers [post]

// GoOnline godoc
// @Summary      Driver goes online
// @Description  Set driver status to online and available for rides
// @Tags         driver
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} map[string]interface{} "Success message"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      404 {object} map[string]interface{} "Driver not found"
// @Security     BearerAuth
// @Router       /drivers/{driver_id}/online [post]

// GoOffline godoc
// @Summary      Driver goes offline
// @Description  Set driver status to offline
// @Tags         driver
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} map[string]interface{} "Success message"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Security     BearerAuth
// @Router       /drivers/{driver_id}/offline [post]

// UpdateLocation godoc
// @Summary      Update driver location
// @Description  Update driver's current GPS location
// @Tags         driver
// @Accept       json
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Param        request body dto.LocationUpdate true "Location coordinates"
// @Success      200 {object} map[string]interface{} "Success message"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Security     BearerAuth
// @Router       /drivers/{driver_id}/location [post]

// StartRide godoc
// @Summary      Start a ride
// @Description  Driver starts the assigned ride
// @Tags         driver
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} models.Ride "Updated ride"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Security     BearerAuth
// @Router       /drivers/{driver_id}/start [post]

// CompleteRide godoc
// @Summary      Complete a ride
// @Description  Driver marks the ride as completed
// @Tags         driver
// @Produce      json
// @Param        driver_id path string true "Driver ID"
// @Success      200 {object} models.Ride "Completed ride"
// @Failure      400 {object} map[string]interface{} "Bad request"
// @Security     BearerAuth
// @Router       /drivers/{driver_id}/complete [post]

// ============================================
// ADMIN SERVICE EXAMPLES (@Tags admin)
// ============================================

// GetOverview godoc
// @Summary      Get system overview
// @Description  Get system metrics and statistics
// @Tags         admin
// @Produce      json
// @Success      200 {object} map[string]interface{} "System overview data"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden - Admin only"
// @Security     BearerAuth
// @Router       /admin/overview [get]

// GetActiveRides godoc
// @Summary      Get active rides
// @Description  Get list of all currently active rides
// @Tags         admin
// @Produce      json
// @Success      200 {array} models.Ride "List of active rides"
// @Failure      401 {object} map[string]interface{} "Unauthorized"
// @Failure      403 {object} map[string]interface{} "Forbidden - Admin only"
// @Security     BearerAuth
// @Router       /admin/rides/active [get]

// ============================================
// COMMON PATTERNS
// ============================================

// For path parameters:
// @Param        id path string true "Entity ID"

// For query parameters:
// @Param        limit query int false "Limit results" default(10)
// @Param        offset query int false "Offset results" default(0)

// For request body with specific model:
// @Param        request body dto.YourRequest true "Request description"

// For responses with models:
// @Success      200 {object} models.YourModel "Description"
// @Success      200 {array} models.YourModel "List description"

// For authentication:
// @Security     BearerAuth

// Multiple tags (for shared endpoints):
// @Tags         ride,driver

// ============================================
// IMPORTANT NOTES
// ============================================
// 1. Always use the correct @Tags matching your service (ride, driver, admin, auth)
// 2. Add @Security BearerAuth for protected endpoints
// 3. HTTP methods in @Router should match your actual route (get, post, put, delete)
// 4. Path parameters in {braces} should match @Param definitions
// 5. Run `make swagger-<service>` after adding/modifying annotations
