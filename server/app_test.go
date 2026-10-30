package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestValidEmail(t *testing.T) {
	for _, email := range []string{"person@example.com", "first.last+tag@sub.example.org"} {
		if !validEmail(email) {
			t.Errorf("validEmail(%q) = false", email)
		}
	}
	for _, email := range []string{"", "@example.com", "person@", "person@x", "person @example.com"} {
		if validEmail(email) {
			t.Errorf("validEmail(%q) = true", email)
		}
	}
}

func TestHashToken(t *testing.T) {
	if hashToken("session-token") != hashToken("session-token") {
		t.Fatal("hashToken must be deterministic")
	}
	if hashToken("session-token") == hashToken("another-token") {
		t.Fatal("different tokens must have different hashes")
	}
}

func TestDecodeJSONRejectsTrailingValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"eventId":1} {"eventId":2}`))
	response := httptest.NewRecorder()
	var body struct {
		EventID int64 `json:"eventId"`
	}
	if decodeJSON(response, request, &body) {
		t.Fatal("decodeJSON accepted multiple JSON values")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestEventHubPublishesToSubscribers(t *testing.T) {
	hub := newEventHub()
	updates, unsubscribe := hub.subscribe()
	defer unsubscribe()

	hub.publish()
	select {
	case <-updates:
	default:
		t.Fatal("subscriber did not receive availability update")
	}
}

func TestIsPermanentConnectionError(t *testing.T) {
	for _, code := range []string{"28000", "28P01", "3D000"} {
		if !isPermanentConnectionError(&pgconn.PgError{Code: code}) {
			t.Errorf("isPermanentConnectionError(%q) = false", code)
		}
	}
	if isPermanentConnectionError(errors.New("temporary network failure")) {
		t.Fatal("temporary connection failures must remain retryable")
	}
}

func TestCreateBookingRejectsInvalidSeatSelection(t *testing.T) {
	tests := []struct {
		name       string
		eventID    int64
		seatLabels []string
	}{
		{name: "missing event", eventID: 0, seatLabels: []string{"A1"}},
		{name: "no seats", eventID: 1},
		{name: "too many seats", eventID: 1, seatLabels: []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10", "B1"}},
		{name: "empty seat label", eventID: 1, seatLabels: []string{" "}},
		{name: "duplicate seat", eventID: 1, seatLabels: []string{"A1", "A1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newService(nil, nil).createBooking(context.Background(), 1, CreateBookingRequestDTO{
				EventID: test.eventID, SeatLabels: test.seatLabels,
			})
			var expected *serviceError
			if !errors.As(err, &expected) || expected.status != http.StatusBadRequest {
				t.Fatalf("createBooking error = %v, want HTTP %d validation error", err, http.StatusBadRequest)
			}
		})
	}
}
