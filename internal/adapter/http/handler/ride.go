package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler/dto"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
	wshub "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
	"github.com/gorilla/websocket"
)

type (
	RideService interface {
		Create(ctx context.Context, ride *models.Ride) (*models.Ride, error)
		Cancel(ctx context.Context, rideID uuid.UUID, reason string) (*models.Ride, error)
	}

	TokenValidator interface {
		RoleCheck(ctx context.Context, token string) (*models.User, error)
	}

	ConnectionHub interface {
		Add(newConn *wshub.Conn) error
		Delete(entityID uuid.UUID) error
	}

	Ride struct {
		l             logger.Logger
		ride          RideService
		auth          TokenValidator
		wsConnections ConnectionHub
	}
)

func NewRide(l logger.Logger, ride RideService, auth TokenValidator, wsConnections ConnectionHub) *Ride {
	return &Ride{
		l:             l,
		ride:          ride,
		auth:          auth,
		wsConnections: wsConnections,
	}
}

func (h *Ride) CreateRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "create_ride")

	var request dto.CreateRideRequest
	if err := readJSON(w, r, &request); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	request.Validate(v)

	if !v.Valid() {
		h.l.Error(ctx, "invalid request data", v)
		failedValidationResponse(w, v.Errors)
		return
	}

	domainModel, err := request.ToModel()
	if err != nil {
		h.l.Error(ctx, "failed to map request", err)
		errorResponse(w, http.StatusBadRequest, "invalid passenger_id format")
		return
	}

	createdRide, err := h.ride.Create(ctx, domainModel)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to create ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"ride_id":                    createdRide.ID,
		"ride_number":                createdRide.RideNumber,
		"status":                     createdRide.Status,
		"estimated_fare":             createdRide.EstimatedFare,
		"estimated_duration_minutes": createdRide.EstimatedDurationMin,
		"estimated_distance_km":      createdRide.EstimatedDistanceKm,
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		internalErrorResponse(w, err.Error())
	}
}

func (h *Ride) CancelRide(w http.ResponseWriter, r *http.Request) {
	ctx := wrap.WithAction(r.Context(), "cancel_ride")

	rideIDstr := r.PathValue("ride_id")
	rideID, err := uuid.Parse(rideIDstr)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid ride ID format")
		return
	}

	var request dto.CancelRideRequest
	if err := readJSON(w, r, &request); err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to read request JSON data", err)
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()
	request.Validate(v)

	if !v.Valid() {
		h.l.Warn(ctx, "invalid request data")
		failedValidationResponse(w, v.Errors)
		return
	}

	cancelledRide, err := h.ride.Cancel(ctx, rideID, request.Reason)
	if err != nil {
		h.l.Error(wrap.ErrorCtx(ctx, err), "failed to cancel ride", err)
		errorResponse(w, GetCode(err), err.Error())
		return
	}

	response := envelope{
		"ride_id":      cancelledRide.ID,
		"status":       cancelledRide.Status,
		"cancelled_at": cancelledRide.CancelledAt,
		"message":      cancelledRide.CancellationReason,
	}

	if err := writeJSON(w, http.StatusAccepted, response, nil); err != nil {
		h.l.Error(ctx, "failed to write response", err)
		internalErrorResponse(w, err.Error())
		return
	}
}

func (h *Ride) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	passengerIdStr := r.PathValue("passenger_id")

	ctx := wrap.WithAction(wrap.WithPassengerID(r.Context(), passengerIdStr), "ws_handle_ride")

	passengerID, err := uuid.Parse(passengerIdStr)
	if err != nil {
		h.l.Error(ctx, "invalid passenger id", err)
		errorResponse(w, http.StatusBadRequest, "invalid passenger ID format")
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.l.Error(ctx, "failed to upgrade to websocket", err)
		errorResponse(w, http.StatusBadRequest, "upgrade failed")
		return
	}

	// Authenticate the WebSocket connection
	passenger, err := h.wsAuthenticate(ctx, wsConn, passengerID)
	if err != nil {
		h.l.Error(ctx, "websocket authentication failed", err)
		return
	}

	if passenger.Role != types.RolePassenger.String() {
		h.l.Warn(wrap.WithUserID(ctx, passenger.ID.String()), "attempt to start websocket with invalid role(must be passenger)", "role", passenger.Role)
		_ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "access denied: invalid role"),
			time.Now().Add(time.Second),
		)
		_ = wsConn.Close()
		return
	}

	conn := wshub.NewConn(ctx, passenger.ID, wsConn, h.l)
	if err := h.wsConnections.Add(conn); err != nil {
		h.l.Error(ctx, "failed to register WS connection", err)
		wsConn.WriteJSON(map[string]any{"error": "failed to register"})
		wsConn.Close()
		return
	}
	defer h.wsConnections.Delete(passenger.ID)

	h.l.Info(ctx, "websocket connection registered")
	// Heartbeat
	go func() {
		if err := conn.HeartbeatLoop(time.Second*60, time.Second*30); err != nil {
			h.l.Error(ctx, "heartbeat loop failed", err)
		}
	}()

	// Listen for messages
	if err := conn.Listen(); err != nil {
		h.l.Error(ctx, "websocket listen failed", err)
		_ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "websocket listen failed"),
			time.Now().Add(time.Second),
		)
		_ = wsConn.Close()
	}
}

// wsAuthenticate enforces a 5s auth window, expects a JSON text message:
//
//	{"type":"auth","token":"Bearer <jwt>"}
//
// It validates the JWT via RideService and returns the passenger UUID.
// On any error, it sends an appropriate WebSocket close frame and closes the connection.
func (h *Ride) wsAuthenticate(ctx context.Context, conn *websocket.Conn, passengerID uuid.UUID) (*models.User, error) {
	const authTimeout = 5 * time.Second

	// Enforce "client must authenticate within 5 seconds".
	if err := conn.SetReadDeadline(time.Now().Add(authTimeout)); err != nil {
		h.l.Error(ctx, "failed to set read deadline", err)
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "internal error"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, err
	}

	msgType, payload, err := conn.ReadMessage()
	if err != nil {
		h.l.Error(ctx, "failed to read initial auth message", err)
		closeCode := websocket.ClosePolicyViolation
		closeReason := "must send auth message within 5 seconds"
		// If it was a timeout, clarify reason.
		if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
			closeReason = "authentication timeout (5s)"
		}
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(closeCode, closeReason),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, err
	}

	if msgType != websocket.TextMessage {
		h.l.Error(ctx, "first message must be text", errors.New("non-text first frame"))
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "first message must be JSON text"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, errors.New("first message must be text")
	}

	var req dto.AuthWebSocketReq
	if err := json.Unmarshal(payload, &req); err != nil {
		h.l.Error(ctx, "invalid auth JSON", err)
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid auth JSON"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, err
	}

	if req.Type != "auth" {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "first message must be type=auth"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, errors.New("unexpected message type")
	}

	// Validate the token and get the passenger info
	passenger, err := h.auth.RoleCheck(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	if passenger.ID != passengerID {
		h.l.Error(ctx, "passenger ID mismatch", errors.New("passenger ID does not match token"))
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "passenger ID does not match token"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, errors.New("passenger ID mismatch")
	}

	// Auth succeeded; clear the read deadline for normal operation.
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		h.l.Error(ctx, "failed to clear read deadline", err)
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "internal error"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, err
	}

	// Send an explicit acknowledgment so the client can transition its state machine.
	ack := dto.AuthWebSocketResp{
		Type:        "auth_ok",
		PassengerID: passenger.ID.String(),
	}
	if err := conn.WriteJSON(ack); err != nil {
		h.l.Error(ctx, "failed to send auth_ok", err)
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "failed to ack authentication"),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return nil, err
	}

	return passenger, nil
}
