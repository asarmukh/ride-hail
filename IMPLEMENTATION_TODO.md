# Ride-Hailing Platform - Complete Implementation TODO List

**Current Completion:** 35-40%
**Estimated Work Remaining:** 13-17 days
**Status:** FAILING - Critical features missing

## How to Use This List

1. Tasks are organized by priority (P0 = Critical, P1 = Major, P2 = Quality)
2. Each task includes file paths, line numbers, and acceptance criteria
3. Complete all P0 tasks first to achieve minimum viable system
4. Test each task before moving to the next
5. Update checkboxes as you complete tasks

---

## PRIORITY 0 - CRITICAL (System Breaking Issues)

These must be completed for the system to function at all.

### P0.1 - Fix Root Build Configuration (0.5 day)

**Problem:** Cannot compile with `go build -o ride-hail-system .` from project root as required by README.

**Solution:**

- [ ] Create `/home/user/dev/projects/ride-hail/main.go` in project root
- [ ] Implement service selector based on environment variable or command-line flag
- [ ] Example structure:
  ```go
  package main

  import (
      "flag"
      "fmt"
      "os"
  )

  func main() {
      service := flag.String("service", "", "Service to run: ride|driver|auth|admin")
      flag.Parse()

      switch *service {
      case "ride":
          // Import and run cmd/ride-service/main.go logic
      case "driver":
          // Import and run cmd/driver-service/main.go logic
      case "auth":
          // Import and run cmd/auth-service/main.go logic
      case "admin":
          // Import and run cmd/admin-service/main.go logic
      default:
          fmt.Println("Usage: ride-hail-system -service=[ride|driver|auth|admin]")
          os.Exit(1)
      }
  }
  ```
- [ ] Test: `go build -o ride-hail-system .` succeeds
- [ ] Test: `./ride-hail-system -service=ride` starts ride service

**Acceptance Criteria:**
- ✅ Compiles from root with exact command from README
- ✅ Can run all services with single binary

---

### P0.2 - Implement Admin Service (2 days)

**Problem:** Admin service doesn't exist at all. Required by README lines 1058-1140.

**Solution:**

#### Step 1: Create Admin Service Structure (0.5 day)

- [ ] Create directory `/home/user/dev/projects/ride-hail/cmd/admin-service/`
- [ ] Create `cmd/admin-service/main.go` with basic HTTP server setup
- [ ] Create `/home/user/dev/projects/ride-hail/internal/admin/` directory structure:
  ```
  internal/admin/
  ├── api/
  │   ├── handlers.go    (HTTP handlers)
  │   └── routes.go      (Route definitions)
  ├── app/
  │   └── service.go     (Business logic)
  └── repo/
      └── postgres.go    (Database queries)
  ```
- [ ] Copy graceful shutdown pattern from `cmd/ride-service/main.go:73-87`

#### Step 2: Implement GET /admin/overview (0.75 day)

- [ ] In `internal/admin/repo/postgres.go`, create `GetSystemMetrics()` function
- [ ] Query for:
  - Active rides count: `SELECT COUNT(*) FROM rides WHERE status NOT IN ('COMPLETED', 'CANCELLED')`
  - Available drivers: `SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE'`
  - Busy drivers: `SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY', 'EN_ROUTE')`
  - Total rides today: `SELECT COUNT(*) FROM rides WHERE DATE(created_at) = CURRENT_DATE`
  - Total revenue today: `SELECT COALESCE(SUM(final_fare), 0) FROM rides WHERE status = 'COMPLETED' AND DATE(completed_at) = CURRENT_DATE`
  - Average wait time: `SELECT AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))/60) FROM rides WHERE matched_at IS NOT NULL AND DATE(requested_at) = CURRENT_DATE`
  - Average ride duration: `SELECT AVG(EXTRACT(EPOCH FROM (completed_at - started_at))/60) FROM rides WHERE status = 'COMPLETED' AND DATE(completed_at) = CURRENT_DATE`
  - Cancellation rate: `SELECT CAST(COUNT(*) FILTER (WHERE status = 'CANCELLED') AS FLOAT) / NULLIF(COUNT(*), 0) FROM rides WHERE DATE(created_at) = CURRENT_DATE`
- [ ] Query driver distribution: `SELECT vehicle_type, COUNT(*) FROM drivers WHERE status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE') GROUP BY vehicle_type`
- [ ] Create response struct matching README line 1073-1103
- [ ] In `internal/admin/api/handlers.go`, implement handler that calls repo and returns JSON
- [ ] Add JWT middleware requiring ADMIN role

#### Step 3: Implement GET /admin/rides/active (0.75 day)

- [ ] In `internal/admin/repo/postgres.go`, create `GetActiveRides(page, pageSize int)` function
- [ ] Query with pagination:
  ```sql
  SELECT
    r.id, r.ride_number, r.status, r.passenger_id, r.driver_id,
    pc.address as pickup_address, dc.address as destination_address,
    r.started_at, r.estimated_fare,
    lh.latitude as current_lat, lh.longitude as current_lng
  FROM rides r
  LEFT JOIN coordinates pc ON r.pickup_coordinate_id = pc.id
  LEFT JOIN coordinates dc ON r.destination_coordinate_id = dc.id
  LEFT JOIN LATERAL (
    SELECT latitude, longitude
    FROM location_history
    WHERE driver_id = r.driver_id
    ORDER BY recorded_at DESC
    LIMIT 1
  ) lh ON true
  WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
  ORDER BY r.requested_at DESC
  LIMIT $1 OFFSET $2
  ```
- [ ] Calculate distance completed and remaining (store in ride or calculate via PostGIS)
- [ ] Implement pagination logic: `OFFSET = (page - 1) * pageSize`
- [ ] Create response matching README line 1115-1139
- [ ] Add JWT middleware requiring ADMIN role

#### Step 4: Integration (0.25 day)

- [ ] Update `config.yaml` with admin service port (should already exist: `admin_service: ${ADMIN_SERVICE_PORT:-3004}`)
- [ ] Uncomment admin service in `docker-compose.yml` lines 131-140
- [ ] Test endpoints with curl or Postman
- [ ] Verify JSON responses match README format exactly

**Acceptance Criteria:**
- ✅ Admin service starts on port 3004
- ✅ GET /admin/overview returns all required metrics
- ✅ GET /admin/rides/active returns paginated active rides
- ✅ Only admin users can access endpoints (JWT with ADMIN role)
- ✅ All JSON fields match README specification

**Testing Commands:**
```bash
# Get admin token first
ADMIN_TOKEN=$(curl -X POST http://localhost:3005/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@test.com","password":"password"}' | jq -r .token)

# Test overview
curl -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:3004/admin/overview

# Test active rides
curl -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:3004/admin/rides/active?page=1&page_size=20
```

---

### P0.3 - Implement Driver Matching Algorithm (3 days)

**Problem:** Core feature completely missing. No consumer for ride requests, no geospatial queries, no driver selection logic.

**Reference:** README lines 836-877

#### Step 1: Enable RabbitMQ in Driver Service (0.5 day)

- [ ] In `cmd/driver-service/main.go`, uncomment lines 31-38 (RabbitMQ connection)
- [ ] Add consumer initialization before `server.ListenAndServe()`:
  ```go
  consumer := usecase.NewMatchingConsumer(driverUseCase, rmq)
  go consumer.Start()
  ```
- [ ] Add graceful consumer shutdown in shutdown block (lines 65-79)

#### Step 2: Create Driver Matching Consumer (0.5 day)

- [ ] Create `internal/driver/app/usecase/driver_matching.go`
- [ ] Implement consumer that:
  - Consumes from `driver_matching` queue
  - Uses **manual acknowledgment** (not auto-ack)
  - Calls driver matching logic
  - Sends response to `driver_topic` exchange with routing key `driver.response.{ride_id}`
  - Handles errors with `msg.Nack(false, true)` for requeue
- [ ] Example structure:
  ```go
  func (c *MatchingConsumer) Start() error {
      ch, err := c.rmq.Channel()
      msgs, err := ch.Consume("driver_matching", "", false, false, false, false, nil)

      for msg := range msgs {
          go c.handleRideRequest(msg)
      }
  }

  func (c *MatchingConsumer) handleRideRequest(msg amqp.Delivery) {
      var request RideMatchRequest
      json.Unmarshal(msg.Body, &request)

      // Find and notify drivers
      driver, err := c.useCase.MatchDriver(request)

      if err != nil {
          msg.Nack(false, true) // Requeue
          return
      }

      msg.Ack(false) // Acknowledge successful processing
  }
  ```

#### Step 3: Implement PostGIS Geospatial Query (1 day)

- [ ] In `internal/driver/adapter/psql/driver_matching.go` (create new file), implement `FindNearbyDrivers()`:
  ```sql
  SELECT
    d.id,
    u.email,
    d.rating,
    d.total_rides,
    d.vehicle_attrs,
    c.latitude,
    c.longitude,
    ST_Distance(
      ST_MakePoint(c.longitude, c.latitude)::geography,
      ST_MakePoint($1, $2)::geography
    ) / 1000 as distance_km
  FROM drivers d
  JOIN users u ON d.id = u.id
  JOIN coordinates c ON c.entity_id = d.id
    AND c.entity_type = 'driver'
    AND c.is_current = true
  WHERE d.status = 'AVAILABLE'
    AND d.vehicle_type = $3
    AND ST_DWithin(
      ST_MakePoint(c.longitude, c.latitude)::geography,
      ST_MakePoint($1, $2)::geography,
      5000  -- 5km radius
    )
  ORDER BY distance_km ASC, d.rating DESC
  LIMIT 10;
  ```
- [ ] Parameters: `$1 = pickup_lng`, `$2 = pickup_lat`, `$3 = vehicle_type`
- [ ] Return top 10 drivers sorted by distance and rating
- [ ] Calculate completion rate: `completed_rides / total_rides`

#### Step 4: Implement Driver Scoring and Ranking (0.5 day)

- [ ] In `internal/driver/app/usecase/driver_matching.go`, create scoring function:
  ```go
  type DriverScore struct {
      DriverID     uuid.UUID
      Distance     float64  // km
      Rating       float64  // 1.0-5.0
      CompletionRate float64 // 0.0-1.0
      Score        float64  // Calculated score
  }

  func calculateScore(d DriverScore) float64 {
      // Weight factors (tune as needed)
      distanceWeight := 0.5
      ratingWeight := 0.3
      completionWeight := 0.2

      // Normalize distance (inverse: closer = better)
      distanceScore := 1.0 / (1.0 + d.Distance)

      // Normalize rating (0-5 scale)
      ratingScore := d.Rating / 5.0

      return (distanceScore * distanceWeight) +
             (ratingScore * ratingWeight) +
             (d.CompletionRate * completionWeight)
  }
  ```
- [ ] Sort drivers by score descending
- [ ] Select top 3-5 drivers to send offers

#### Step 5: Implement Offer Timeout Mechanism (0.5 day)

- [ ] Create offer tracking map: `map[offerID]OfferState`
- [ ] For each driver in ranked list:
  - Generate unique `offer_id`
  - Send offer via WebSocket (will implement in P0.4)
  - Start 30-second timeout timer
  - Wait for response or timeout
  - If accepted: send match response and break
  - If rejected/timeout: try next driver
- [ ] Use channels for timeout coordination:
  ```go
  type OfferState struct {
      RideID     uuid.UUID
      DriverID   uuid.UUID
      ExpiresAt  time.Time
      ResponseCh chan OfferResponse
  }

  func (u *DriverUseCase) SendOfferWithTimeout(driver Driver, ride Ride) (*OfferResponse, error) {
      offerID := uuid.New()
      responseCh := make(chan OfferResponse, 1)

      u.offers[offerID] = OfferState{
          RideID:     ride.ID,
          DriverID:   driver.ID,
          ExpiresAt:  time.Now().Add(30 * time.Second),
          ResponseCh: responseCh,
      }

      // Send offer via WebSocket
      u.wsManager.SendOfferToDriver(driver.ID, offer)

      // Wait for response or timeout
      select {
      case response := <-responseCh:
          return &response, nil
      case <-time.After(30 * time.Second):
          return nil, errors.New("offer timeout")
      }
  }
  ```

#### Step 6: Send Match Response to Ride Service (0.25 day)

- [ ] When driver accepts, publish to RabbitMQ:
  - Exchange: `driver_topic`
  - Routing key: `driver.response.{ride_id}`
  - Message format from README lines 919-943
- [ ] Include driver info: name, rating, vehicle details, current location, ETA
- [ ] Calculate ETA: `distance_km / average_speed * 60` (assume 40 km/h average)
- [ ] Update driver status to `EN_ROUTE` in database

**Acceptance Criteria:**
- ✅ Driver service consumes from `driver_matching` queue
- ✅ Finds nearby drivers using PostGIS within 5km
- ✅ Scores drivers by distance, rating, completion rate
- ✅ Sends offers to top drivers via WebSocket
- ✅ Implements 30-second timeout per driver
- ✅ First-come-first-served: first acceptance wins
- ✅ Publishes match response to `driver_topic`
- ✅ Updates driver status to EN_ROUTE
- ✅ Uses manual message acknowledgment

**Testing:**
```bash
# Insert test driver at location
psql -d ridehail_db -c "
  INSERT INTO coordinates (entity_id, entity_type, address, latitude, longitude, is_current)
  VALUES ('driver-uuid', 'driver', 'Test Location', 43.235, 76.885, true);
"

# Create ride request (should trigger matching)
curl -X POST http://localhost:3000/rides \
  -H "Authorization: Bearer $PASSENGER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "passenger_id": "passenger-uuid",
    "pickup_latitude": 43.238949,
    "pickup_longitude": 76.889709,
    "pickup_address": "Almaty Central Park",
    "destination_latitude": 43.222015,
    "destination_longitude": 76.851511,
    "destination_address": "Kok-Tobe Hill",
    "ride_type": "ECONOMY"
  }'

# Check RabbitMQ management UI: driver should receive offer
# Check logs: should show matching algorithm execution
```

---

### P0.4 - Implement Driver WebSocket (2 days)

**Problem:** No WebSocket endpoint for drivers. Cannot send offers or receive responses.

**Reference:** README lines 972-1057

#### Step 1: Create Driver WebSocket Endpoint (0.5 day)

- [ ] Create `internal/driver/adapter/handlers/ws.go`
- [ ] Implement WebSocket upgrade handler:
  ```go
  func (h *Handler) HandleDriverWebSocket(w http.ResponseWriter, r *http.Request) {
      driverID := chi.URLParam(r, "driver_id")

      upgrader := websocket.Upgrader{
          CheckOrigin: func(r *http.Request) bool { return true },
      }

      conn, err := upgrader.Upgrade(w, r, nil)
      if err != nil {
          return
      }

      h.wsManager.RegisterDriver(driverID, conn)
      defer h.wsManager.UnregisterDriver(driverID)

      // Authentication timeout
      if !h.authenticateWithTimeout(conn, driverID) {
          conn.Close()
          return
      }

      // Start ping/pong
      go h.startPingPong(conn)

      // Handle incoming messages
      h.handleMessages(conn, driverID)
  }
  ```
- [ ] Add route in `internal/driver/adapter/handlers/init.go`:
  ```go
  r.Get("/ws/drivers/{driver_id}", h.HandleDriverWebSocket)
  ```

#### Step 2: Implement Authentication with 5-Second Timeout (0.25 day)

- [ ] Copy pattern from `internal/ride/api/ws.go:39-76`
- [ ] Set read deadline: `conn.SetReadDeadline(time.Now().Add(5 * time.Second))`
- [ ] Wait for auth message: `{"type":"auth","token":"Bearer <jwt>"}`
- [ ] Validate JWT token using `internal/shared/jwt/jwt.go`
- [ ] Verify token belongs to specified driver_id
- [ ] Verify user has DRIVER role
- [ ] If timeout or invalid: close connection
- [ ] Remove deadline after successful auth: `conn.SetReadDeadline(time.Time{})`

#### Step 3: Implement Ping/Pong Keep-Alive (0.25 day)

- [ ] Create goroutine that sends ping every 30 seconds
- [ ] Set pong handler to track last pong time:
  ```go
  conn.SetPongHandler(func(string) error {
      conn.SetReadDeadline(time.Now().Add(60 * time.Second))
      return nil
  })
  ```
- [ ] Close connection if no pong received within 60 seconds
- [ ] Note: Fix passenger WebSocket too - `internal/ride/api/ws.go` doesn't handle pong timeout

#### Step 4: Implement Message Handlers (0.5 day)

- [ ] Handle incoming `ride_response` messages (README lines 1014-1026):
  ```go
  type RideResponse struct {
      Type            string   `json:"type"`
      OfferID         string   `json:"offer_id"`
      RideID          string   `json:"ride_id"`
      Accepted        bool     `json:"accepted"`
      CurrentLocation Location `json:"current_location"`
  }
  ```
- [ ] When accepted:
  - Look up offer in offer tracking map
  - Send response to offer's response channel
  - Update driver status to EN_ROUTE
  - Publish match response to RabbitMQ
- [ ] When rejected:
  - Mark offer as rejected
  - Continue to next driver in matching algorithm
- [ ] Handle incoming `location_update` messages (README lines 1046-1056):
  - Update coordinates table
  - Publish to `location_fanout` exchange (see P0.5)

#### Step 5: Implement Outgoing Message Senders (0.5 day)

- [ ] Create WebSocket manager with connection map:
  ```go
  type WSManager struct {
      drivers map[string]*websocket.Conn
      mu      sync.RWMutex
  }

  func (m *WSManager) SendOfferToDriver(driverID string, offer RideOffer) error {
      m.mu.RLock()
      conn, ok := m.drivers[driverID]
      m.mu.RUnlock()

      if !ok {
          return errors.New("driver not connected")
      }

      return conn.WriteJSON(offer)
  }
  ```
- [ ] Implement `SendOfferToDriver()` - sends `ride_offer` message (README lines 988-1011)
- [ ] Implement `SendRideDetailsToDriver()` - sends `ride_details` after acceptance (README lines 1028-1043)
- [ ] Include passenger name, phone, pickup location with notes

**Acceptance Criteria:**
- ✅ WebSocket endpoint at `ws://localhost:3001/ws/drivers/{driver_id}`
- ✅ Authentication required within 5 seconds
- ✅ JWT token validated with DRIVER role
- ✅ Ping sent every 30 seconds, connection closed if no pong within 60s
- ✅ Receives and processes `ride_response` messages
- ✅ Receives and processes `location_update` messages
- ✅ Sends `ride_offer` messages to drivers
- ✅ Sends `ride_details` after acceptance
- ✅ Connection cleanup on disconnect

**Testing:**
```bash
# Use wscat or websocat for testing
wscat -c "ws://localhost:3001/ws/drivers/driver-uuid-here"

# Send auth (within 5 seconds)
{"type":"auth","token":"Bearer <driver-jwt>"}

# Should receive ping every 30 seconds
# Send pong response

# Accept ride offer
{"type":"ride_response","offer_id":"offer-123","ride_id":"ride-uuid","accepted":true,"current_location":{"latitude":43.235,"longitude":76.885}}

# Send location update
{"type":"location_update","latitude":43.236,"longitude":76.886,"accuracy_meters":5.0,"speed_kmh":45.0,"heading_degrees":180.0}
```

---

### P0.5 - Implement Location Broadcasting (1 day)

**Problem:** Location updates stored in DB but not broadcast to other services via `location_fanout` exchange.

**Reference:** README lines 869-876, 956-970

#### Step 1: Add Location Publishing in Driver Service (0.25 day)

- [ ] In `internal/driver/adapter/handlers/driver_ride.go`, after line 37 (location stored), add:
  ```go
  // Publish to location_fanout exchange
  locationUpdate := map[string]interface{}{
      "driver_id": driverID,
      "ride_id": rideID, // Get from current driver session
      "location": map[string]float64{
          "lat": req.Latitude,
          "lng": req.Longitude,
      },
      "speed_kmh": req.SpeedKmh,
      "heading_degrees": req.HeadingDegrees,
      "timestamp": time.Now().UTC().Format(time.RFC3339),
  }

  if err := h.rmq.PublishFanout("location_fanout", locationUpdate); err != nil {
      // Log error but don't fail request
      h.logger.Error("Failed to broadcast location", err)
  }
  ```
- [ ] Add rate limiting: reject updates faster than 3 seconds apart (store last update time in memory or Redis)

#### Step 2: Add Location Publishing via WebSocket (0.25 day)

- [ ] In driver WebSocket handler (`internal/driver/adapter/handlers/ws.go`), when receiving `location_update` message:
  - Store in database (call existing handler)
  - Publish to `location_fanout` exchange
  - Same rate limiting: max 1 update per 3 seconds per driver

#### Step 3: Create Location Consumer in Ride Service (0.5 day)

- [ ] Create `internal/ride/consumer/location_consumer.go`
- [ ] Implement consumer for `location_updates_ride` queue:
  ```go
  func (c *LocationConsumer) Start() error {
      ch, err := c.rmq.Channel()
      msgs, err := ch.Consume("location_updates_ride", "", false, false, false, false, nil)

      for msg := range msgs {
          go c.handleLocationUpdate(msg)
      }
  }

  func (c *LocationConsumer) handleLocationUpdate(msg amqp.Delivery) {
      var update LocationUpdate
      json.Unmarshal(msg.Body, &update)

      // Calculate ETA if driver is en route
      ride, err := c.repo.GetRideByID(update.RideID)
      if ride.Status == "EN_ROUTE" || ride.Status == "MATCHED" {
          eta := c.calculateETA(update.Location, ride.PickupLocation, update.SpeedKmh)

          // Send update to passenger via WebSocket
          c.wsManager.SendToPassenger(ride.PassengerID, map[string]interface{}{
              "type": "driver_location_update",
              "ride_id": ride.ID,
              "driver_location": update.Location,
              "estimated_arrival": eta,
              "distance_to_pickup_km": calculateDistance(update.Location, ride.PickupLocation),
          })
      }

      msg.Ack(false)
  }
  ```
- [ ] Initialize consumer in `cmd/ride-service/main.go` after line 72:
  ```go
  locationConsumer := consumer.NewLocationConsumer(rideService, rmq, wsManager)
  go locationConsumer.Start()
  ```

#### Step 4: Implement ETA Calculation (0.25 day)

- [ ] Create `internal/shared/util/eta.go`:
  ```go
  func CalculateETA(from Location, to Location, currentSpeedKmh float64) time.Time {
      distanceKm := CalculateDistance(from, to)

      // Use current speed if available, otherwise assume 40 km/h average
      speed := currentSpeedKmh
      if speed < 10 { // Too slow or stopped
          speed = 40.0
      }

      durationHours := distanceKm / speed
      durationMinutes := durationHours * 60

      return time.Now().Add(time.Duration(durationMinutes) * time.Minute)
  }
  ```
- [ ] Use existing Haversine formula from `internal/shared/util/utils.go` for distance

**Acceptance Criteria:**
- ✅ Location updates published to `location_fanout` exchange
- ✅ Published from both HTTP POST endpoint and WebSocket
- ✅ Rate limited to max 1 update per 3 seconds per driver
- ✅ Ride service consumes from `location_updates_ride` queue
- ✅ Calculates ETA based on current location and speed
- ✅ Sends updates to passenger via WebSocket
- ✅ Uses manual message acknowledgment

**Testing:**
```bash
# Update driver location
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/location \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 43.236,
    "longitude": 76.886,
    "accuracy_meters": 5.0,
    "speed_kmh": 45.0,
    "heading_degrees": 180.0
  }'

# Check RabbitMQ management UI: message should appear in location_updates_ride queue
# Check passenger WebSocket: should receive driver_location_update message
# Check rate limiting: sending 2 updates within 3 seconds should reject second
```

---

### P0.6 - Fix Ride Service Consumer Acknowledgment (0.25 day)

**Problem:** `internal/ride/consumer/consumer.go:30` uses auto-ack, risking message loss.

**Solution:**

- [ ] Change line 30 from:
  ```go
  msgs, err := ch.Consume("driver_responses", "", true, false, false, false, nil)
  ```
  to:
  ```go
  msgs, err := ch.Consume("driver_responses", "", false, false, false, false, nil)
  ```
- [ ] In `handleDriverResponse()` function (lines 46-68), add error handling:
  ```go
  func (c *Consumer) handleDriverResponse(msg amqp.Delivery) {
      var response DriverResponse
      if err := json.Unmarshal(msg.Body, &response); err != nil {
          msg.Nack(false, false) // Don't requeue malformed messages
          return
      }

      if err := c.service.ProcessDriverResponse(response); err != nil {
          // Log error
          c.logger.Error("Failed to process driver response", err)
          msg.Nack(false, true) // Requeue for retry
          return
      }

      msg.Ack(false) // Success
  }
  ```
- [ ] Remove `log.Println` calls, use structured logger instead

**Acceptance Criteria:**
- ✅ Consumer uses manual acknowledgment
- ✅ Messages acknowledged on success
- ✅ Messages nacked with requeue on processing error
- ✅ Malformed messages rejected without requeue

---

## PRIORITY 1 - MAJOR (Core Features Missing)

These are required for complete business logic and user experience.

### P1.1 - Complete Ride Lifecycle State Transitions (2 days)

**Problem:** Only REQUESTED → MATCHED implemented. Missing 4 more transitions.

**Reference:** README lines 106-109, 296-308

#### Current State
- ✅ REQUESTED (ride created)
- ✅ MATCHED (driver assigned) - `internal/ride/app/services.go:175-206`
- ❌ EN_ROUTE (driver heading to pickup)
- ❌ ARRIVED (driver at pickup)
- ❌ IN_PROGRESS (ride started)
- ❌ COMPLETED (ride finished)
- ⚠️ CANCELLED (exists but incomplete) - `internal/ride/api/handlers.go:75-153`

#### Step 1: Implement Driver Start Ride Endpoint (0.5 day)

- [ ] In `internal/driver/adapter/handlers/driver_ride.go`, add `StartRide()` handler:
  ```go
  func (h *Handler) StartRide(w http.ResponseWriter, r *http.Request) {
      driverID := chi.URLParam(r, "driver_id")

      var req struct {
          RideID         string `json:"ride_id"`
          DriverLocation struct {
              Latitude  float64 `json:"latitude"`
              Longitude float64 `json:"longitude"`
          } `json:"driver_location"`
      }

      // Verify driver is assigned to this ride
      // Update ride status to IN_PROGRESS
      // Update driver status to BUSY
      // Store start time (started_at)
      // Publish ride.status.in_progress event
      // Send WebSocket update to passenger

      json.NewEncoder(w).Encode(response)
  }
  ```
- [ ] Add route: `r.Post("/drivers/{driver_id}/start", h.StartRide)`
- [ ] Response format from README lines 795-804

#### Step 2: Implement Driver Complete Ride Endpoint (0.5 day)

- [ ] In `internal/driver/adapter/handlers/driver_ride.go`, add `CompleteRide()` handler:
  ```go
  func (h *Handler) CompleteRide(w http.ResponseWriter, r *http.Request) {
      driverID := chi.URLParam(r, "driver_id")

      var req struct {
          RideID              string `json:"ride_id"`
          FinalLocation       Location `json:"final_location"`
          ActualDistanceKm    float64 `json:"actual_distance_km"`
          ActualDurationMin   int `json:"actual_duration_minutes"`
      }

      // Calculate final fare (may differ from estimate)
      finalFare := calculateFinalFare(req.ActualDistanceKm, req.ActualDurationMin, ride.VehicleType)

      // Update ride status to COMPLETED
      // Store completed_at, final_fare, actual distance/duration
      // Update driver status to AVAILABLE
      // Update driver stats (total_rides++, total_earnings += fare)
      // Publish ride.status.completed event
      // Send WebSocket update to passenger

      json.NewEncoder(w).Encode(response)
  }
  ```
- [ ] Add route: `r.Post("/drivers/{driver_id}/complete", h.CompleteRide)`
- [ ] Response format from README lines 825-834
- [ ] Calculate driver earnings: 80% of final_fare (20% platform fee)

#### Step 3: Implement EN_ROUTE Transition (0.25 day)

- [ ] Automatically transition to EN_ROUTE when driver accepts offer
- [ ] In `internal/driver/app/usecase/driver_matching.go`, after sending match response:
  ```go
  // Update ride status to EN_ROUTE
  if err := repo.UpdateRideStatus(rideID, "EN_ROUTE", driverID); err != nil {
      return err
  }

  // Publish event
  rmq.Publish("ride_topic", "ride.status.en_route", RideStatusUpdate{
      RideID:    rideID,
      Status:    "EN_ROUTE",
      DriverID:  driverID,
      Timestamp: time.Now(),
  })

  // Send WebSocket to passenger
  wsManager.SendToPassenger(passengerID, map[string]interface{}{
      "type": "ride_status_update",
      "ride_id": rideID,
      "status": "EN_ROUTE",
      "message": "Your driver is on the way",
      "driver_info": driverInfo,
  })
  ```

#### Step 4: Implement ARRIVED Transition (0.25 day)

- [ ] Create new endpoint or use existing location update logic
- [ ] When driver is within 100 meters of pickup location:
  ```go
  if ride.Status == "EN_ROUTE" {
      distanceToPickup := CalculateDistance(driverLocation, pickupLocation)

      if distanceToPickup <= 0.1 { // 100 meters
          // Update status to ARRIVED
          repo.UpdateRideStatus(rideID, "ARRIVED", driverID)
          repo.UpdateRideArrivedAt(rideID, time.Now())

          // Publish event
          rmq.Publish("ride_topic", "ride.status.arrived", ...)

          // Send WebSocket to passenger
          wsManager.SendToPassenger(passengerID, map[string]interface{}{
              "type": "ride_status_update",
              "status": "ARRIVED",
              "message": "Your driver has arrived at the pickup location",
          })
      }
  }
  ```
- [ ] Implement in location update handler (`internal/driver/adapter/handlers/driver_ride.go`)

#### Step 5: Add Event Sourcing to ride_events Table (0.25 day)

- [ ] For each status transition, insert event to `ride_events` table:
  ```go
  func (r *RideRepo) RecordEvent(ctx context.Context, rideID uuid.UUID, eventType string, eventData map[string]interface{}) error {
      query := `
          INSERT INTO ride_events (ride_id, event_type, event_data)
          VALUES ($1, $2, $3)
      `

      data, _ := json.Marshal(eventData)
      _, err := r.db.Exec(ctx, query, rideID, eventType, data)
      return err
  }
  ```
- [ ] Call after each transition with relevant data:
  - RIDE_REQUESTED: pickup, destination, fare estimate
  - DRIVER_MATCHED: driver_id, ETA
  - DRIVER_ARRIVED: arrival time
  - RIDE_STARTED: start time, location
  - RIDE_COMPLETED: end time, final fare, distance
  - RIDE_CANCELLED: reason, refund amount

#### Step 6: Update ride_status Consumer in Ride Service (0.25 day)

- [ ] Create consumer for `ride_status` queue (if not exists)
- [ ] Listen for status updates from driver service
- [ ] Update local ride status and send WebSocket notifications to passenger

**Acceptance Criteria:**
- ✅ POST /drivers/{id}/start transitions to IN_PROGRESS
- ✅ POST /drivers/{id}/complete transitions to COMPLETED
- ✅ Auto-transition to EN_ROUTE on driver acceptance
- ✅ Auto-transition to ARRIVED when within 100m of pickup
- ✅ All transitions recorded in ride_events table
- ✅ All transitions publish RabbitMQ events
- ✅ All transitions send WebSocket updates to passenger
- ✅ Final fare calculated based on actual distance/duration
- ✅ Driver earnings and stats updated on completion

**Testing:**
```bash
# Start ride
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/start \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -d '{"ride_id":"'$RIDE_ID'","driver_location":{"latitude":43.238,"longitude":76.889}}'

# Complete ride
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/complete \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -d '{"ride_id":"'$RIDE_ID'","final_location":{"latitude":43.222,"longitude":76.851},"actual_distance_km":5.5,"actual_duration_minutes":16}'

# Verify in DB
psql -d ridehail_db -c "SELECT * FROM rides WHERE id = '$RIDE_ID';"
psql -d ridehail_db -c "SELECT * FROM ride_events WHERE ride_id = '$RIDE_ID' ORDER BY created_at;"
```

---

### P1.2 - Implement WebSocket Message Broadcasting (1 day)

**Problem:** `SendToPassenger()` function exists but never called from business logic.

**Reference:** `internal/ride/api/ws.go:115-121`

#### Step 1: Fix All Status Transition Notifications (0.5 day)

- [ ] In `internal/ride/app/services.go`, after line 206 (MATCHED status), add:
  ```go
  // Send WebSocket notification
  h.wsManager.SendToPassenger(ride.PassengerID.String(), map[string]interface{}{
      "type": "ride_status_update",
      "ride_id": ride.ID.String(),
      "ride_number": ride.RideNumber,
      "status": "MATCHED",
      "driver_info": map[string]interface{}{
          "driver_id": response.DriverID,
          "name": response.DriverInfo.Name,
          "rating": response.DriverInfo.Rating,
          "vehicle": response.DriverInfo.Vehicle,
      },
      "correlation_id": correlationID,
  })
  ```
- [ ] Similarly add after each status change:
  - EN_ROUTE notification
  - ARRIVED notification ("Your driver has arrived")
  - IN_PROGRESS notification ("Your ride has started")
  - COMPLETED notification with final fare
  - CANCELLED notification with refund info

#### Step 2: Integrate Location Updates (0.25 day)

- [ ] In location consumer (from P0.5), ensure `SendToPassenger()` is called
- [ ] Format from README lines 606-618:
  ```go
  wsManager.SendToPassenger(passengerID, map[string]interface{}{
      "type": "driver_location_update",
      "ride_id": rideID,
      "driver_location": map[string]float64{
          "lat": location.Lat,
          "lng": location.Lng,
      },
      "estimated_arrival": eta.Format(time.RFC3339),
      "distance_to_pickup_km": distance,
  })
  ```

#### Step 3: Add Connection State Verification (0.25 day)

- [ ] In `internal/ride/api/ws.go`, modify `SendToPassenger()`:
  ```go
  func (m *WSManager) SendToPassenger(passengerID string, message interface{}) error {
      m.mu.RLock()
      conn, ok := m.passengers[passengerID]
      m.mu.RUnlock()

      if !ok {
          // Passenger not connected - this is OK, just log at debug level
          logger.Debug("Passenger not connected", "passenger_id", passengerID)
          return nil
      }

      if err := conn.WriteJSON(message); err != nil {
          // Connection dead, remove from map
          m.UnregisterPassenger(passengerID)
          return err
      }

      return nil
  }
  ```
- [ ] Handle stale connections gracefully

**Acceptance Criteria:**
- ✅ Match notification sent to passenger on driver match
- ✅ EN_ROUTE notification sent when driver starts journey
- ✅ ARRIVED notification sent when driver reaches pickup
- ✅ IN_PROGRESS notification sent when ride starts
- ✅ COMPLETED notification sent with final fare
- ✅ Location updates sent every 3-10 seconds during active ride
- ✅ All messages match README WebSocket event formats
- ✅ Gracefully handles disconnected passengers

**Testing:**
```bash
# Connect passenger WebSocket
wscat -c "ws://localhost:3000/ws/passengers/$PASSENGER_ID"

# Authenticate
{"type":"auth","token":"Bearer $PASSENGER_TOKEN"}

# Create ride request (in another terminal)
# Should receive: ride_status_update with status MATCHED
# Should receive: driver_location_update messages periodically
# Driver starts ride -> should receive IN_PROGRESS update
# Driver completes -> should receive COMPLETED update
```

---

### P1.3 - Implement Structured JSON Logging (1 day)

**Problem:** Current logger outputs colored text, not JSON. Missing required fields.

**Reference:** README lines 44-59

#### Step 1: Rewrite Logger to Output JSON (0.5 day)

- [ ] In `internal/shared/util/logger.go`, replace entire implementation:
  ```go
  package util

  import (
      "encoding/json"
      "os"
      "time"
  )

  type Logger struct {
      service  string
      hostname string
  }

  type LogEntry struct {
      Timestamp string                 `json:"timestamp"`
      Level     string                 `json:"level"`
      Service   string                 `json:"service"`
      Action    string                 `json:"action,omitempty"`
      Message   string                 `json:"message"`
      Hostname  string                 `json:"hostname"`
      RequestID string                 `json:"request_id,omitempty"`
      RideID    string                 `json:"ride_id,omitempty"`
      Error     *ErrorDetails          `json:"error,omitempty"`
      Extra     map[string]interface{} `json:"-"`
  }

  type ErrorDetails struct {
      Msg   string `json:"msg"`
      Stack string `json:"stack,omitempty"`
  }

  func NewLogger(serviceName string) *Logger {
      hostname, _ := os.Hostname()
      return &Logger{
          service:  serviceName,
          hostname: hostname,
      }
  }

  func (l *Logger) Info(action, message string, fields ...map[string]interface{}) {
      l.log("INFO", action, message, nil, fields...)
  }

  func (l *Logger) Debug(action, message string, fields ...map[string]interface{}) {
      l.log("DEBUG", action, message, nil, fields...)
  }

  func (l *Logger) Error(action, message string, err error, fields ...map[string]interface{}) {
      errorDetails := &ErrorDetails{
          Msg: err.Error(),
      }
      l.log("ERROR", action, message, errorDetails, fields...)
  }

  func (l *Logger) log(level, action, message string, errDetails *ErrorDetails, fields ...map[string]interface{}) {
      entry := LogEntry{
          Timestamp: time.Now().UTC().Format(time.RFC3339),
          Level:     level,
          Service:   l.service,
          Action:    action,
          Message:   message,
          Hostname:  l.hostname,
          Error:     errDetails,
      }

      // Merge extra fields if provided
      if len(fields) > 0 {
          for k, v := range fields[0] {
              switch k {
              case "request_id":
                  entry.RequestID = v.(string)
              case "ride_id":
                  entry.RideID = v.(string)
              // Add other known fields
              }
          }
      }

      json.NewEncoder(os.Stdout).Encode(entry)
  }
  ```

#### Step 2: Update All Services to Use New Logger (0.25 day)

- [ ] Update `cmd/ride-service/main.go` line 25:
  ```go
  logger := util.NewLogger("ride-service")
  ```
- [ ] Update `cmd/driver-service/main.go` similarly
- [ ] Update `cmd/auth-service/main.go` similarly
- [ ] Update `cmd/admin-service/main.go` (when created)

#### Step 3: Add Logger to All Handlers and Use Cases (0.25 day)

- [ ] Pass logger instance to all handlers, use cases, repos
- [ ] Replace all `log.Println()` and `fmt.Println()` calls with structured logger:
  ```go
  // Before
  log.Println("Error binding JSON:", err)

  // After
  logger.Error("request_validation_failed", "Failed to parse request body", err, map[string]interface{}{
      "request_id": requestID,
  })
  ```
- [ ] Replace all `util.LogInfo()`, `util.LogError()` calls with new logger methods

**Acceptance Criteria:**
- ✅ All logs output valid JSON to stdout
- ✅ All required fields present: timestamp (ISO 8601), level, service, action, message, hostname
- ✅ request_id included when available
- ✅ ride_id included for ride-related logs
- ✅ ERROR logs include error object with msg and stack
- ✅ No more console.log, fmt.Println, or colored text output
- ✅ All three services use consistent format

**Testing:**
```bash
# Start service and verify JSON output
./ride-hail-system -service=ride 2>&1 | head -10

# Should see valid JSON like:
# {"timestamp":"2024-10-29T19:00:00Z","level":"INFO","service":"ride-service","action":"service_start","message":"Starting Ride Service","hostname":"localhost"}

# Verify with jq
./ride-hail-system -service=ride 2>&1 | jq .

# Should parse without errors
```

---

## PRIORITY 2 - QUALITY (Polish and Compliance)

These improve code quality, security, and full spec compliance.

### P2.1 - Implement Correlation IDs (1 day)

**Problem:** No distributed tracing. Cannot follow requests across services.

**Reference:** README lines 189-192

#### Step 1: Create Request ID Middleware (0.25 day)

- [ ] Create `internal/shared/middleware/request_id.go`:
  ```go
  package middleware

  import (
      "context"
      "net/http"
      "github.com/google/uuid"
  )

  type contextKey string

  const RequestIDKey contextKey = "request_id"

  func RequestID(next http.Handler) http.Handler {
      return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          // Check for existing correlation ID in header
          requestID := r.Header.Get("X-Request-ID")
          if requestID == "" {
              requestID = uuid.New().String()
          }

          // Add to context
          ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

          // Add to response header
          w.Header().Set("X-Request-ID", requestID)

          next.ServeHTTP(w, r.WithContext(ctx))
      })
  }

  func GetRequestID(ctx context.Context) string {
      if id, ok := ctx.Value(RequestIDKey).(string); ok {
          return id
      }
      return ""
  }
  ```

#### Step 2: Add Middleware to All Services (0.25 day)

- [ ] In `internal/ride/api/routes.go`, add middleware before routes:
  ```go
  r.Use(middleware.RequestID)
  ```
- [ ] In `internal/driver/adapter/handlers/init.go`, add similarly
- [ ] In `internal/auth/api/routes.go`, add similarly

#### Step 3: Propagate Correlation ID Through Logs (0.25 day)

- [ ] In all handlers, extract request ID from context:
  ```go
  requestID := middleware.GetRequestID(r.Context())
  ```
- [ ] Pass to logger in all log calls:
  ```go
  logger.Info("ride_requested", "New ride request received", map[string]interface{}{
      "request_id": requestID,
      "ride_id": ride.ID.String(),
  })
  ```

#### Step 4: Propagate Through RabbitMQ Messages (0.25 day)

- [ ] Add `correlation_id` field to all message payloads:
  ```go
  message := map[string]interface{}{
      "ride_id": rideID,
      "correlation_id": requestID,
      // ... other fields
  }
  ```
- [ ] In consumers, extract correlation ID and use in logs:
  ```go
  var msg struct {
      CorrelationID string `json:"correlation_id"`
      // ... other fields
  }
  json.Unmarshal(delivery.Body, &msg)

  logger.Info("driver_response_received", "Processing driver response", map[string]interface{}{
      "request_id": msg.CorrelationID,
  })
  ```

**Acceptance Criteria:**
- ✅ All HTTP requests generate or propagate X-Request-ID header
- ✅ All logs include request_id field
- ✅ All RabbitMQ messages include correlation_id
- ✅ Can trace single ride request across all services using correlation ID
- ✅ README examples with correlation_id are now accurate

**Testing:**
```bash
# Create ride with correlation ID
curl -X POST http://localhost:3000/rides \
  -H "X-Request-ID: test-correlation-123" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{...}'

# Grep logs across all services for correlation ID
docker-compose logs | grep "test-correlation-123"

# Should see logs from:
# - Ride service: request received
# - Driver service: matching algorithm
# - Ride service: driver matched
# All with same request_id
```

---

### P2.2 - Fix Ride Number Format (0.5 day)

**Problem:** Current format `RIDE_YYYYMMDD_XXXXXX` doesn't match spec `RIDE_YYYYMMDD_HHMMSS_XXX`.

**File:** `internal/ride/app/services.go:62`

**Solution:**

- [ ] Replace ride number generation:
  ```go
  // Current (wrong)
  rideNumber := fmt.Sprintf("RIDE_%s_%06d", time.Now().Format("20060102"), time.Now().Unix()%1000000)

  // Correct
  now := time.Now()
  rideNumber := fmt.Sprintf("RIDE_%s_%s_%03d",
      now.Format("20060102"),      // YYYYMMDD
      now.Format("150405"),         // HHMMSS
      now.Nanosecond()/1000000%1000 // XXX (0-999)
  )
  ```
- [ ] Or use atomic counter for XXX to ensure uniqueness:
  ```go
  var rideCounter int32

  func generateRideNumber() string {
      now := time.Now()
      counter := atomic.AddInt32(&rideCounter, 1) % 1000
      return fmt.Sprintf("RIDE_%s_%s_%03d",
          now.Format("20060102"),
          now.Format("150405"),
          counter,
      )
  }
  ```

**Acceptance Criteria:**
- ✅ Ride numbers match format `RIDE_20241029_143052_001`
- ✅ Contains date, time, and 3-digit counter
- ✅ Still unique across concurrent requests

**Testing:**
```bash
# Create multiple rides quickly
for i in {1..5}; do
  curl -X POST http://localhost:3000/rides \
    -H "Authorization: Bearer $TOKEN" \
    -d '{...}' &
done
wait

# Check ride numbers in DB
psql -d ridehail_db -c "SELECT ride_number FROM rides ORDER BY created_at DESC LIMIT 5;"

# Should show:
# RIDE_20241029_143052_001
# RIDE_20241029_143052_002
# RIDE_20241029_143053_001
# etc.
```

---

### P2.3 - Add RabbitMQ Reconnection Logic (0.5 day)

**Problem:** Services crash if RabbitMQ connection drops during runtime.

**File:** `internal/shared/mq/rabbitmq.go`

**Solution:**

- [ ] Add connection monitoring:
  ```go
  func (r *RabbitMQ) Start() {
      go r.reconnectLoop()
  }

  func (r *RabbitMQ) reconnectLoop() {
      for {
          reason, ok := <-r.conn.NotifyClose(make(chan *amqp.Error))
          if !ok {
              // Connection closed cleanly
              return
          }

          log.Printf("RabbitMQ connection lost: %v. Reconnecting...", reason)

          for {
              time.Sleep(5 * time.Second)

              conn, err := amqp.Dial(r.url)
              if err != nil {
                  log.Printf("Reconnect failed: %v", err)
                  continue
              }

              r.conn = conn
              r.channel, _ = conn.Channel()
              log.Println("Reconnected to RabbitMQ")
              break
          }
      }
  }
  ```
- [ ] Add exponential backoff (5s, 10s, 20s, 40s, max 60s)
- [ ] Recreate consumers after reconnection
- [ ] Add health check endpoint that verifies RabbitMQ connectivity

**Acceptance Criteria:**
- ✅ Services continue running if RabbitMQ restarts
- ✅ Automatically reconnects with exponential backoff
- ✅ Consumers restart after reconnection
- ✅ Logs reconnection attempts

**Testing:**
```bash
# Start services
docker-compose up -d

# Stop RabbitMQ
docker-compose stop rabbitmq

# Services should log connection lost
# Services should NOT crash

# Restart RabbitMQ
docker-compose start rabbitmq

# Services should log reconnection success
# Functionality should resume
```

---

### P2.4 - Enable Driver Authorization Middleware (0.25 day)

**Problem:** Driver endpoints have no authentication - security vulnerability.

**File:** `internal/driver/adapter/handlers/middleware.go:7-34`

**Solution:**

- [ ] Uncomment lines 7-34
- [ ] Fix any compilation errors
- [ ] Ensure JWT validation works correctly
- [ ] Verify driver_id from token matches driver_id in URL
- [ ] Add middleware to all driver routes:
  ```go
  r.Use(middleware.AuthorizeDriver)
  r.Post("/drivers/{driver_id}/online", h.GoOnline)
  r.Post("/drivers/{driver_id}/offline", h.GoOffline)
  r.Post("/drivers/{driver_id}/location", h.UpdateLocation)
  r.Post("/drivers/{driver_id}/start", h.StartRide)
  r.Post("/drivers/{driver_id}/complete", h.CompleteRide)
  ```

**Acceptance Criteria:**
- ✅ All driver endpoints require JWT token with DRIVER role
- ✅ Driver can only access their own driver_id routes
- ✅ Returns 401 for missing/invalid token
- ✅ Returns 403 for wrong driver_id or missing DRIVER role

**Testing:**
```bash
# Without token - should fail
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/online
# Expected: 401 Unauthorized

# With passenger token - should fail
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/online \
  -H "Authorization: Bearer $PASSENGER_TOKEN"
# Expected: 403 Forbidden

# With correct driver token - should succeed
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/online \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -d '{"latitude":43.238,"longitude":76.889}'
# Expected: 200 OK

# With different driver's token - should fail
curl -X POST http://localhost:3001/drivers/$DRIVER_ID/online \
  -H "Authorization: Bearer $OTHER_DRIVER_TOKEN"
# Expected: 403 Forbidden
```

---

### P2.5 - Add Health Check Endpoints (0.5 day)

**Problem:** No health check endpoints for monitoring.

**Reference:** README lines 176-180

**Solution:**

#### Step 1: Create Health Check Handler (0.25 day)

- [ ] In each service, add `internal/{service}/api/health.go`:
  ```go
  func HealthCheck(db *pgxpool.Pool, rmq *RabbitMQ) http.HandlerFunc {
      return func(w http.ResponseWriter, r *http.Request) {
          health := map[string]interface{}{
              "status": "healthy",
              "service": "ride-service",
              "timestamp": time.Now().UTC().Format(time.RFC3339),
              "checks": map[string]string{},
          }

          // Check database
          if err := db.Ping(r.Context()); err != nil {
              health["status"] = "unhealthy"
              health["checks"].(map[string]string)["database"] = "down"
          } else {
              health["checks"].(map[string]string)["database"] = "up"
          }

          // Check RabbitMQ
          if rmq.conn.IsClosed() {
              health["status"] = "unhealthy"
              health["checks"].(map[string]string)["rabbitmq"] = "down"
          } else {
              health["checks"].(map[string]string)["rabbitmq"] = "up"
          }

          status := 200
          if health["status"] == "unhealthy" {
              status = 503
          }

          w.Header().Set("Content-Type", "application/json")
          w.WriteHeader(status)
          json.NewEncoder(w).Encode(health)
      }
  }
  ```

#### Step 2: Add Routes (0.25 day)

- [ ] In each service's route file, add:
  ```go
  r.Get("/health", HealthCheck(db, rmq))
  ```
- [ ] Add to:
  - Ride Service: `internal/ride/api/routes.go`
  - Driver Service: `internal/driver/adapter/handlers/init.go`
  - Auth Service: `internal/auth/api/routes.go`
  - Admin Service: `internal/admin/api/routes.go`

**Acceptance Criteria:**
- ✅ All services have /health endpoint
- ✅ Returns 200 when healthy, 503 when unhealthy
- ✅ Checks database and RabbitMQ connectivity
- ✅ Returns JSON with status and checks

**Testing:**
```bash
# Check all services
curl http://localhost:3000/health  # Ride
curl http://localhost:3001/health  # Driver
curl http://localhost:3004/health  # Admin
curl http://localhost:3005/health  # Auth

# Should all return:
# {"status":"healthy","service":"ride-service","timestamp":"...","checks":{"database":"up","rabbitmq":"up"}}

# Stop RabbitMQ
docker-compose stop rabbitmq

# Health checks should return 503 with rabbitmq: down
```

---

### P2.6 - Fix Passenger WebSocket Pong Timeout (0.25 day)

**Problem:** Passenger WebSocket sends ping but never closes connection if no pong received.

**File:** `internal/ride/api/ws.go:82-94`

**Solution:**

- [ ] Add pong handler in `authenticateWithTimeout` or connection setup:
  ```go
  func (m *WSManager) RegisterPassenger(passengerID string, conn *websocket.Conn) {
      m.mu.Lock()
      m.passengers[passengerID] = conn
      m.mu.Unlock()

      // Set initial read deadline
      conn.SetReadDeadline(time.Now().Add(60 * time.Second))

      // Set pong handler
      conn.SetPongHandler(func(string) error {
          // Reset deadline when pong received
          conn.SetReadDeadline(time.Now().Add(60 * time.Second))
          return nil
      })
  }
  ```
- [ ] In ping loop (lines 82-94), if write fails, close connection
- [ ] Add goroutine that reads messages (even if just pongs) to trigger pong handler

**Acceptance Criteria:**
- ✅ Connection closed if no pong within 60 seconds
- ✅ Connection stays alive if pongs received
- ✅ Matches driver WebSocket behavior (once implemented)

---

### P2.7 - Add Input Validation for All Endpoints (0.5 day)

**Problem:** Only ride creation validates inputs. Other endpoints need validation.

**Solution:**

- [ ] Create validation utility in `internal/shared/util/validation.go`:
  ```go
  func ValidateCoordinates(lat, lng float64) error {
      if lat < -90 || lat > 90 {
          return errors.New("latitude must be between -90 and 90")
      }
      if lng < -180 || lng > 180 {
          return errors.New("longitude must be between -180 and 180")
      }
      return nil
  }

  func ValidateUUID(id string) error {
      _, err := uuid.Parse(id)
      return err
  }

  func ValidateRideType(t string) error {
      valid := []string{"ECONOMY", "PREMIUM", "XL"}
      for _, v := range valid {
          if t == v {
              return nil
          }
      }
      return errors.New("invalid ride type")
  }
  ```
- [ ] Add validation to:
  - Driver location updates
  - Driver online/offline endpoints
  - Ride cancellation
  - All admin endpoints (pagination params)

**Acceptance Criteria:**
- ✅ Invalid coordinates rejected with 400
- ✅ Invalid UUIDs rejected with 400
- ✅ Invalid enums rejected with 400
- ✅ Clear error messages returned

---

### P2.8 - Verify gofumpt Formatting (0.25 day)

**Problem:** Cannot verify compliance without running gofumpt.

**Solution:**

- [ ] Install gofumpt:
  ```bash
  go install mvdan.cc/gofumpt@latest
  ```
- [ ] Format all code:
  ```bash
  gofumpt -l -w .
  ```
- [ ] Check for differences:
  ```bash
  gofumpt -l .
  # Should output nothing if all files formatted
  ```
- [ ] Add to Makefile:
  ```makefile
  fmt:
      gofumpt -l -w .

  fmt-check:
      gofumpt -l .
  ```
- [ ] Run before final submission

**Acceptance Criteria:**
- ✅ `gofumpt -l .` outputs nothing
- ✅ All code follows gofumpt standards
- ✅ No formatting-related rejection

---

### P2.9 - Add Rate Limiting to Location Updates (0.25 day)

**Problem:** No rate limiting - drivers could spam location updates.

**Reference:** README line 876 - "max 1 update per 3 seconds"

**Solution:**

- [ ] In driver service, create rate limiter:
  ```go
  type RateLimiter struct {
      lastUpdate map[string]time.Time
      mu         sync.RWMutex
  }

  func (rl *RateLimiter) Allow(driverID string) bool {
      rl.mu.Lock()
      defer rl.mu.Unlock()

      last, exists := rl.lastUpdate[driverID]
      if exists && time.Since(last) < 3*time.Second {
          return false
      }

      rl.lastUpdate[driverID] = time.Now()
      return true
  }
  ```
- [ ] In location update handlers (HTTP and WebSocket), check rate limit:
  ```go
  if !h.rateLimiter.Allow(driverID) {
      http.Error(w, "Rate limit exceeded. Max 1 update per 3 seconds", 429)
      return
  }
  ```

**Acceptance Criteria:**
- ✅ Max 1 location update per 3 seconds per driver
- ✅ Returns 429 Too Many Requests on rate limit
- ✅ Applies to both HTTP POST and WebSocket updates

---

## FINAL CHECKLIST

Before marking the project complete, verify:

### Compilation and Setup
- [ ] `go build -o ride-hail-system .` succeeds
- [ ] `gofumpt -l .` outputs nothing
- [ ] All services start without errors
- [ ] Docker compose brings up all services

### Database
- [ ] All migrations run successfully
- [ ] All tables have proper constraints
- [ ] PostGIS extension enabled and working
- [ ] Coordinate validations in place

### Services
- [ ] Ride Service running on port 3000
- [ ] Driver Service running on port 3001
- [ ] Admin Service running on port 3004
- [ ] Auth Service running on port 3005

### RabbitMQ
- [ ] All exchanges created (ride_topic, driver_topic, location_fanout)
- [ ] All queues bound correctly
- [ ] Messages flow between services
- [ ] Manual acknowledgment working
- [ ] Reconnection logic works

### Core Features
- [ ] Ride creation with fare calculation
- [ ] Driver matching algorithm with PostGIS
- [ ] Real-time location broadcasting
- [ ] Complete ride lifecycle (all 6+ statuses)
- [ ] Ride cancellation with refunds

### WebSockets
- [ ] Passenger WebSocket working
- [ ] Driver WebSocket working
- [ ] Authentication within 5 seconds
- [ ] Ping/pong keep-alive
- [ ] All message types implemented

### Admin Service
- [ ] GET /admin/overview returns metrics
- [ ] GET /admin/rides/active returns paginated rides
- [ ] Only accessible by admin users

### Logging
- [ ] All logs in JSON format
- [ ] All required fields present
- [ ] Correlation IDs working
- [ ] Can trace requests across services

### Security
- [ ] JWT authentication on all endpoints
- [ ] Role-based access control
- [ ] Input validation everywhere
- [ ] Driver authorization enabled

### Testing
- [ ] Can create ride end-to-end
- [ ] Can match driver
- [ ] Can track location
- [ ] Can complete ride
- [ ] Can cancel ride
- [ ] Can view admin metrics

### Documentation
- [ ] README accurate
- [ ] QUESTIONS.md can be answered "Yes" to all
- [ ] Code comments where needed
- [ ] API matches specification exactly

---

## ESTIMATED TIMELINE

| Priority | Tasks | Time | Cumulative |
|----------|-------|------|------------|
| P0 | Build config, Admin service, Driver matching, Driver WS, Location broadcast, Consumer fixes | 8.5 days | 8.5 days |
| P1 | Ride lifecycle, WS broadcasting, JSON logging | 4 days | 12.5 days |
| P2 | Correlation IDs, Ride number fix, Reconnection, Auth, Health checks, Rate limiting, Validation, Formatting | 4 days | 16.5 days |

**Total: 16.5 days of focused development**

---

## PRIORITY RECOMMENDATIONS

If time is limited, focus on priorities in this order:

1. **P0.1-P0.5** (Critical path for basic functionality)
2. **P1.1** (Complete ride lifecycle)
3. **P1.3** (Logging compliance - will be checked immediately)
4. **P1.2** (WebSocket notifications - user-facing)
5. **P2.2, P2.4, P2.8** (Quick wins for compliance)
6. **P2.1** (Correlation IDs - demonstrates system design understanding)
7. **Remaining P2 tasks** (Quality improvements)

Good luck! The foundation is solid - you just need to connect all the pieces together and ensure compliance with the specification.
