package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const sessionLifetime = 30 * 24 * time.Hour

type serviceError struct {
	status  int
	message string
}

func (err *serviceError) Error() string {
	return err.message
}

type authResult struct {
	User      UserDTO
	Token     string
	ExpiresAt time.Time
}

type service struct {
	dao *dao
	hub *eventHub
}

func newService(data *dao, hub *eventHub) *service {
	return &service{dao: data, hub: hub}
}

func (s *service) register(ctx context.Context, input RegisterRequestDTO) (authResult, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Name == "" || len(input.Name) > 80 || !validEmail(input.Email) || len(input.Password) < 8 || len(input.Password) > 72 {
		return authResult{}, &serviceError{status: http.StatusBadRequest, message: "Enter a name, valid email, and password between 8 and 72 characters."}
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return authResult{}, fmt.Errorf("hash password: %w", err)
	}
	current, err := s.dao.createUser(ctx, input.Name, input.Email, string(passwordHash))
	if errors.Is(err, errEmailExists) {
		return authResult{}, &serviceError{status: http.StatusConflict, message: "An account with that email already exists."}
	}
	if err != nil {
		return authResult{}, err
	}
	return s.createSession(ctx, current)
}

func (s *service) login(ctx context.Context, input LoginRequestDTO) (authResult, error) {
	current, passwordHash, err := s.dao.findUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil) {
		return authResult{}, &serviceError{status: http.StatusUnauthorized, message: "Email or password is incorrect."}
	}
	if err != nil {
		return authResult{}, fmt.Errorf("load user for sign in: %w", err)
	}
	return s.createSession(ctx, current)
}

func (s *service) createSession(ctx context.Context, current UserDTO) (authResult, error) {
	token, err := newSessionToken()
	if err != nil {
		return authResult{}, fmt.Errorf("generate session token: %w", err)
	}
	expires := time.Now().Add(sessionLifetime)
	if err := s.dao.createSession(ctx, hashToken(token), current.ID, expires); err != nil {
		return authResult{}, err
	}
	return authResult{User: current, Token: token, ExpiresAt: expires}, nil
}

func (s *service) logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.dao.deleteSession(ctx, hashToken(token))
}

func (s *service) currentUser(ctx context.Context, token string) (*UserDTO, error) {
	if token == "" {
		return nil, nil
	}
	current, err := s.dao.findUserBySession(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load current user: %w", err)
	}
	return &current, nil
}

func (s *service) listEvents(ctx context.Context) ([]EventDTO, error) {
	return s.dao.listEvents(ctx)
}

func (s *service) listEventSeats(ctx context.Context, eventID int64) ([]SeatDTO, error) {
	if eventID < 1 {
		return nil, &serviceError{status: http.StatusBadRequest, message: "Invalid event."}
	}
	seats, err := s.dao.listEventSeats(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if len(seats) == 0 {
		return nil, &serviceError{status: http.StatusNotFound, message: "That event could not be found."}
	}
	return seats, nil
}

func (s *service) listBookings(ctx context.Context, userID int64) ([]BookingDTO, error) {
	return s.dao.listBookings(ctx, userID)
}

func (s *service) createBooking(ctx context.Context, userID int64, input CreateBookingRequestDTO) (BookingDTO, error) {
	if input.EventID < 1 || len(input.SeatLabels) < 1 || len(input.SeatLabels) > 10 {
		return BookingDTO{}, &serviceError{status: http.StatusBadRequest, message: "Choose an event and between 1 and 10 seats."}
	}
	seen := make(map[string]struct{}, len(input.SeatLabels))
	for _, seatLabel := range input.SeatLabels {
		if strings.TrimSpace(seatLabel) == "" {
			return BookingDTO{}, &serviceError{status: http.StatusBadRequest, message: "Selected seats are invalid."}
		}
		if _, exists := seen[seatLabel]; exists {
			return BookingDTO{}, &serviceError{status: http.StatusBadRequest, message: "A seat can only be selected once."}
		}
		seen[seatLabel] = struct{}{}
	}
	created, err := s.dao.createBooking(ctx, userID, input.EventID, input.SeatLabels)
	switch {
	case errors.Is(err, errEventNotFound):
		return BookingDTO{}, &serviceError{status: http.StatusNotFound, message: "That event could not be found."}
	case errors.Is(err, errEventStarted):
		return BookingDTO{}, &serviceError{status: http.StatusConflict, message: "This event has already started."}
	case errors.Is(err, errSeatsUnavailable):
		return BookingDTO{}, &serviceError{status: http.StatusConflict, message: "One or more selected seats are no longer available. Please choose again."}
	case err != nil:
		return BookingDTO{}, err
	}
	s.hub.publish()
	return created, nil
}

func (s *service) cancelBooking(ctx context.Context, bookingID, userID int64) error {
	if err := s.dao.cancelBooking(ctx, bookingID, userID); errors.Is(err, errBookingNotFound) {
		return &serviceError{status: http.StatusNotFound, message: "That active booking could not be found."}
	} else if err != nil {
		return err
	}
	s.hub.publish()
	return nil
}

func (s *service) health(ctx context.Context) error {
	return s.dao.ping(ctx)
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	at := strings.LastIndexByte(email, '@')
	return at > 0 && at < len(email)-3 && !strings.ContainsAny(email, " \t\r\n")
}

func newSessionToken() (string, error) {
	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return "", err
	}
	return hex.EncodeToString(rawToken), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
