package main

import "time"

type UserDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type EventDTO struct {
	ID             int64     `json:"id"`
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	Venue          string    `json:"venue"`
	City           string    `json:"city"`
	StartsAt       time.Time `json:"startsAt"`
	Description    string    `json:"description"`
	TotalCapacity  int       `json:"totalCapacity"`
	BookedQuantity int       `json:"bookedQuantity"`
	Available      int       `json:"available"`
	PriceCents     int       `json:"priceCents"`
	Accent         string    `json:"accent"`
}

type BookingDTO struct {
	ID         int64     `json:"id"`
	EventID    int64     `json:"eventId"`
	Quantity   int       `json:"quantity"`
	SeatLabels []string  `json:"seatLabels"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	Title      string    `json:"title,omitempty"`
	Category   string    `json:"category,omitempty"`
	Venue      string    `json:"venue,omitempty"`
	City       string    `json:"city,omitempty"`
	StartsAt   time.Time `json:"startsAt,omitempty"`
	PriceCents int       `json:"priceCents,omitempty"`
}

type SeatDTO struct {
	Label     string `json:"label"`
	Row       string `json:"row"`
	Number    int    `json:"number"`
	Available bool   `json:"available"`
}

type RegisterRequestDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequestDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateBookingRequestDTO struct {
	EventID    int64    `json:"eventId"`
	SeatLabels []string `json:"seatLabels"`
}

type ErrorResponseDTO struct {
	Error string `json:"error"`
}

type MessageResponseDTO struct {
	Message string `json:"message"`
}
