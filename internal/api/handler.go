package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"reservation-system/internal/reservation"
)

const maxRequestBodyBytes = 1 << 20

type ReservationService interface {
	Create(ctx context.Context, input reservation.CreateInput) (reservation.Reservation, error)
	List(ctx context.Context) ([]reservation.Reservation, error)
	GetByID(ctx context.Context, id int64) (reservation.Reservation, error)
	Cancel(ctx context.Context, id int64) (reservation.Reservation, error)
	IsAvailable(ctx context.Context, equipmentID, startsAt, endsAt string) (bool, error)
}

type EquipmentLister interface {
	List(ctx context.Context) ([]reservation.Equipment, error)
}

type handler struct {
	reservations ReservationService
	equipment    EquipmentLister
}

// NewHandler builds the API handler from service and inventory dependencies.
// It does not access SQLite directly.
func NewHandler(reservations ReservationService, equipment EquipmentLister) http.Handler {
	return &handler{reservations: reservations, equipment: equipment}
}

type envelope struct {
	Data  any          `json:"data,omitempty"`
	Error *errorDetail `json:"error,omitempty"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type equipmentResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type reservationResponse struct {
	ID          int64              `json:"id"`
	EquipmentID string             `json:"equipment_id"`
	Requester   string             `json:"requester"`
	StartsAt    time.Time          `json:"starts_at"`
	EndsAt      time.Time          `json:"ends_at"`
	Status      reservation.Status `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	CancelledAt *time.Time         `json:"cancelled_at"`
}

type createReservationRequest struct {
	EquipmentID string `json:"equipment_id"`
	Requester   string `json:"requester"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
}

type availabilityResponse struct {
	EquipmentID string `json:"equipment_id"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	Available   bool   `json:"available"`
}

type requestError struct {
	status  int
	code    string
	message string
}

func (e *requestError) Error() string { return e.message }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	switch r.URL.Path {
	case "/api/equipment":
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		if !h.requireNoBody(w, r) {
			return
		}
		h.listEquipment(w, r)
		return
	case "/api/reservations":
		switch r.Method {
		case http.MethodGet:
			if !h.requireNoBody(w, r) {
				return
			}
			h.listReservations(w, r)
		case http.MethodPost:
			h.createReservation(w, r)
		default:
			h.methodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "equipment" && parts[4] == "availability" && parts[3] != "" {
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		if !h.requireNoBody(w, r) {
			return
		}
		h.equipmentAvailability(w, r, parts[3])
		return
	}
	if len(parts) == 4 && parts[1] == "api" && parts[2] == "reservations" && parts[3] != "" {
		if r.Method != http.MethodGet {
			h.methodNotAllowed(w, http.MethodGet)
			return
		}
		if !h.requireNoBody(w, r) {
			return
		}
		h.getReservation(w, r, parts[3])
		return
	}
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "reservations" && parts[3] != "" && parts[4] == "cancel" {
		if r.Method != http.MethodPatch {
			h.methodNotAllowed(w, http.MethodPatch)
			return
		}
		if !h.requireNoBody(w, r) {
			return
		}
		h.cancelReservation(w, r, parts[3])
		return
	}

	h.writeError(w, &requestError{status: http.StatusNotFound, code: "not_found", message: "resource not found"})
}

func (h *handler) listEquipment(w http.ResponseWriter, r *http.Request) {
	items, err := h.equipment.List(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	data := make([]equipmentResponse, 0, len(items))
	for _, item := range items {
		data = append(data, equipmentResponse{ID: item.ID, Name: item.Name})
	}
	h.writeJSON(w, http.StatusOK, envelope{Data: data})
}

func (h *handler) equipmentAvailability(w http.ResponseWriter, r *http.Request, equipmentID string) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.writeError(w, &requestError{status: http.StatusBadRequest, code: "invalid_query", message: "invalid query string"})
		return
	}
	for key := range query {
		if key != "starts_at" && key != "ends_at" {
			h.writeError(w, &requestError{status: http.StatusBadRequest, code: "invalid_query", message: "only starts_at and ends_at are accepted"})
			return
		}
	}
	startsAt, err := singleQueryValue(query, "starts_at")
	if err != nil {
		h.writeError(w, err)
		return
	}
	endsAt, err := singleQueryValue(query, "ends_at")
	if err != nil {
		h.writeError(w, err)
		return
	}
	available, err := h.reservations.IsAvailable(r.Context(), equipmentID, startsAt, endsAt)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, envelope{Data: availabilityResponse{
		EquipmentID: equipmentID,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		Available:   available,
	}})
}

func (h *handler) createReservation(w http.ResponseWriter, r *http.Request) {
	var input createReservationRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.writeError(w, err)
		return
	}
	created, err := h.reservations.Create(r.Context(), reservation.CreateInput{
		EquipmentID: input.EquipmentID,
		Requester:   input.Requester,
		StartsAt:    input.StartsAt,
		EndsAt:      input.EndsAt,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, envelope{Data: toReservationResponse(created)})
}

func (h *handler) listReservations(w http.ResponseWriter, r *http.Request) {
	items, err := h.reservations.List(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	data := make([]reservationResponse, 0, len(items))
	for _, item := range items {
		data = append(data, toReservationResponse(item))
	}
	h.writeJSON(w, http.StatusOK, envelope{Data: data})
}

func (h *handler) getReservation(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	item, err := h.reservations.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, envelope{Data: toReservationResponse(item)})
}

func (h *handler) cancelReservation(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	item, err := h.reservations.Cancel(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, envelope{Data: toReservationResponse(item)})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &requestError{status: http.StatusUnsupportedMediaType, code: "unsupported_media_type", message: "Content-Type must be application/json"}
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return decodeError(err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return &requestError{status: http.StatusBadRequest, code: "invalid_json", message: "request must contain exactly one JSON value"}
		}
		return decodeError(err)
	}
	return nil
}

func decodeError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return &requestError{status: http.StatusRequestEntityTooLarge, code: "body_too_large", message: "request body exceeds the 1 MiB limit"}
	}
	return &requestError{status: http.StatusBadRequest, code: "invalid_json", message: "request body must contain a valid JSON object with only supported fields"}
}

func requireEmptyBody(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &requestError{status: http.StatusRequestEntityTooLarge, code: "body_too_large", message: "request body exceeds the 1 MiB limit"}
		}
		return &requestError{status: http.StatusBadRequest, code: "invalid_body", message: "could not read request body"}
	}
	if len(body) != 0 {
		return &requestError{status: http.StatusBadRequest, code: "unexpected_body", message: "this endpoint does not accept a request body"}
	}
	return nil
}

func singleQueryValue(query url.Values, key string) (string, error) {
	values, ok := query[key]
	if !ok || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", &requestError{status: http.StatusBadRequest, code: "invalid_query", message: key + " must be provided exactly once"}
	}
	return values[0], nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, &requestError{status: http.StatusBadRequest, code: "invalid_id", message: "reservation ID must be a positive integer"}
	}
	return id, nil
}

func toReservationResponse(item reservation.Reservation) reservationResponse {
	return reservationResponse{
		ID:          item.ID,
		EquipmentID: item.EquipmentID,
		Requester:   item.Requester,
		StartsAt:    item.StartsAt,
		EndsAt:      item.EndsAt,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		CancelledAt: item.CancelledAt,
	}
}

func (h *handler) methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	h.writeError(w, &requestError{status: http.StatusMethodNotAllowed, code: "method_not_allowed", message: "method not allowed for this resource"})
}

func (h *handler) requireNoBody(w http.ResponseWriter, r *http.Request) bool {
	if err := requireEmptyBody(r); err != nil {
		h.writeError(w, err)
		return false
	}
	return true
}

func (h *handler) writeError(w http.ResponseWriter, err error) {
	var requestErr *requestError
	if errors.As(err, &requestErr) {
		h.writeJSON(w, requestErr.status, envelope{Error: &errorDetail{Code: requestErr.code, Message: requestErr.message}})
		return
	}

	switch {
	case errors.Is(err, reservation.ErrInvalidInput):
		message := strings.TrimSpace(strings.TrimPrefix(err.Error(), reservation.ErrInvalidInput.Error()+":"))
		h.writeJSON(w, http.StatusBadRequest, envelope{Error: &errorDetail{Code: "invalid_request", Message: message}})
	case errors.Is(err, reservation.ErrEquipmentNotFound):
		h.writeJSON(w, http.StatusNotFound, envelope{Error: &errorDetail{Code: "equipment_not_found", Message: "equipment not found"}})
	case errors.Is(err, reservation.ErrReservationNotFound):
		h.writeJSON(w, http.StatusNotFound, envelope{Error: &errorDetail{Code: "reservation_not_found", Message: "reservation not found"}})
	case errors.Is(err, reservation.ErrReservationConflict):
		h.writeJSON(w, http.StatusConflict, envelope{Error: &errorDetail{Code: "reservation_conflict", Message: "equipment is already reserved for part of that interval"}})
	case errors.Is(err, reservation.ErrReservationAlreadyCancelled):
		h.writeJSON(w, http.StatusConflict, envelope{Error: &errorDetail{Code: "reservation_already_cancelled", Message: "reservation is already cancelled"}})
	case errors.Is(err, reservation.ErrReservationNotActive):
		h.writeJSON(w, http.StatusConflict, envelope{Error: &errorDetail{Code: "reservation_not_active", Message: "reservation is not active"}})
	default:
		log.Printf("API internal error: %v", err)
		h.writeJSON(w, http.StatusInternalServerError, envelope{Error: &errorDetail{Code: "internal_error", Message: "internal server error"}})
	}
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("API response encoding error: %v", err)
	}
}
