package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const sessionCookie = "seatflow_session"

type controller struct {
	service      *service
	hub          *eventHub
	cookieSecure bool
	webDist      string
}

func newController(service *service, hub *eventHub, cookieSecure bool, webDist string) *controller {
	return &controller{
		service: service, hub: hub, cookieSecure: cookieSecure, webDist: webDist,
	}
}

func (c *controller) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", c.register)
	mux.HandleFunc("POST /api/auth/login", c.login)
	mux.HandleFunc("POST /api/auth/logout", c.logout)
	mux.HandleFunc("GET /api/healthz", c.health)
	mux.HandleFunc("GET /api/me", c.me)
	mux.HandleFunc("GET /api/events", c.listEvents)
	mux.HandleFunc("GET /api/events/{id}/seats", c.listEventSeats)
	mux.HandleFunc("GET /api/bookings", c.listBookings)
	mux.HandleFunc("POST /api/bookings", c.createBooking)
	mux.HandleFunc("DELETE /api/bookings/{id}", c.cancelBooking)
	mux.HandleFunc("GET /api/availability/stream", c.availabilityStream)
	mux.HandleFunc("/", c.serveFrontend)
	return c.cors(c.securityHeaders(mux))
}

func (c *controller) register(w http.ResponseWriter, r *http.Request) {
	var input RegisterRequestDTO
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := c.service.register(r.Context(), input)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	c.setSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusCreated, map[string]any{"user": result.User})
}

func (c *controller) login(w http.ResponseWriter, r *http.Request) {
	var input LoginRequestDTO
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := c.service.login(r.Context(), input)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	c.setSessionCookie(w, result.Token, result.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User})
}

func (c *controller) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err == nil {
		if err := c.service.logout(r.Context(), cookie.Value); err != nil {
			c.writeServiceError(w, err)
			return
		}
	} else if !errors.Is(err, http.ErrNoCookie) {
		writeError(w, http.StatusBadRequest, "Invalid session cookie.")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: c.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, MessageResponseDTO{Message: "Signed out."})
}

func (c *controller) me(w http.ResponseWriter, r *http.Request) {
	current, ok := c.currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": current})
}

func (c *controller) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := c.service.listEvents(r.Context())
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (c *controller) listEventSeats(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || eventID < 1 {
		writeError(w, http.StatusBadRequest, "Invalid event.")
		return
	}
	seats, err := c.service.listEventSeats(r.Context(), eventID)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"seats": seats})
}

func (c *controller) listBookings(w http.ResponseWriter, r *http.Request) {
	current, ok := c.requireUser(w, r)
	if !ok {
		return
	}
	bookings, err := c.service.listBookings(r.Context(), current.ID)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (c *controller) createBooking(w http.ResponseWriter, r *http.Request) {
	current, ok := c.requireUser(w, r)
	if !ok {
		return
	}
	var input CreateBookingRequestDTO
	if !decodeJSON(w, r, &input) {
		return
	}
	created, err := c.service.createBooking(r.Context(), current.ID, input)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"booking": created})
}

func (c *controller) cancelBooking(w http.ResponseWriter, r *http.Request) {
	current, ok := c.requireUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "Invalid booking.")
		return
	}
	if err := c.service.cancelBooking(r.Context(), id, current.ID); err != nil {
		c.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, MessageResponseDTO{Message: "Booking cancelled."})
}

func (c *controller) currentUser(w http.ResponseWriter, r *http.Request) (*UserDTO, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		writeError(w, http.StatusBadRequest, "Invalid session cookie.")
		return nil, false
	}
	token := ""
	if err == nil {
		token = cookie.Value
	}
	current, err := c.service.currentUser(r.Context(), token)
	if err != nil {
		c.writeServiceError(w, err)
		return nil, false
	}
	return current, true
}

func (c *controller) requireUser(w http.ResponseWriter, r *http.Request) (*UserDTO, bool) {
	current, ok := c.currentUser(w, r)
	if !ok {
		return nil, false
	}
	if current == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in to continue.")
		return nil, false
	}
	return current, true
}

func (c *controller) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: c.cookieSecure, SameSite: http.SameSiteLaxMode, Expires: expires,
		MaxAge: int(time.Until(expires).Seconds()),
	})
}

func (c *controller) availabilityStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming is not supported.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	updates, unsubscribe := c.hub.subscribe()
	defer unsubscribe()
	_, _ = fmt.Fprint(w, "event: availability\ndata: {}\n\n")
	flusher.Flush()
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-updates:
			_, _ = fmt.Fprint(w, "event: availability\ndata: {}\n\n")
			flusher.Flush()
		case <-keepalive.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (c *controller) serveFrontend(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	index := filepath.Join(c.webDist, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(c.webDist, filepath.Clean("/"+r.URL.Path))
	if _, err := os.Stat(path); err != nil || r.URL.Path == "/" {
		r.URL.Path = "/"
	}
	http.FileServer(http.Dir(c.webDist)).ServeHTTP(w, r)
}

func (c *controller) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := c.service.health(ctx); err != nil {
		log.Printf("health check database ping: %v", err)
		writeError(w, http.StatusServiceUnavailable, "Database is unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (c *controller) cors(next http.Handler) http.Handler {
	allowedOrigin := strings.TrimRight(envOr("FRONTEND_ORIGIN", "http://localhost:5173"), "/")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(r.Header.Get("Origin"), "/")
		if origin != "" && origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions && strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *controller) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (c *controller) writeServiceError(w http.ResponseWriter, err error) {
	var expected *serviceError
	if errors.As(err, &expected) {
		writeError(w, expected.status, expected.message)
		return
	}
	log.Printf("request failed: %v", err)
	writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Request body must contain a single JSON value.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponseDTO{Error: message})
}
