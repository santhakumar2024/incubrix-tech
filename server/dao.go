package main

import (
	"context"
	"time"
)

type dao struct {
	repository *repository
}

func newDAO(repository *repository) *dao {
	return &dao{repository: repository}
}

func (data *dao) createUser(ctx context.Context, name, email, passwordHash string) (UserDTO, error) {
	record, err := data.repository.createUser(ctx, name, email, passwordHash)
	if err != nil {
		return UserDTO{}, err
	}
	return UserDTO{ID: record.ID, Name: record.Name, Email: record.Email}, nil
}

func (data *dao) findUserByEmail(ctx context.Context, email string) (UserDTO, string, error) {
	record, err := data.repository.findUserByEmail(ctx, email)
	if err != nil {
		return UserDTO{}, "", err
	}
	return UserDTO{ID: record.ID, Name: record.Name, Email: record.Email}, record.PasswordHash, nil
}

func (data *dao) findUserBySession(ctx context.Context, tokenHash string) (UserDTO, error) {
	record, err := data.repository.findUserBySession(ctx, tokenHash)
	if err != nil {
		return UserDTO{}, err
	}
	return UserDTO{ID: record.ID, Name: record.Name, Email: record.Email}, nil
}

func (data *dao) createSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	return data.repository.createSession(ctx, tokenHash, userID, expiresAt)
}

func (data *dao) deleteSession(ctx context.Context, tokenHash string) error {
	return data.repository.deleteSession(ctx, tokenHash)
}

func (data *dao) listEvents(ctx context.Context) ([]EventDTO, error) {
	records, err := data.repository.listEvents(ctx)
	if err != nil {
		return nil, err
	}
	events := make([]EventDTO, 0, len(records))
	for _, record := range records {
		events = append(events, EventDTO{
			ID: record.ID, Slug: record.Slug, Title: record.Title, Category: record.Category,
			Venue: record.Venue, City: record.City, StartsAt: record.StartsAt,
			Description: record.Description, TotalCapacity: record.TotalCapacity,
			BookedQuantity: record.BookedQuantity, Available: record.TotalCapacity - record.BookedQuantity,
			PriceCents: record.PriceCents, Accent: record.Accent,
		})
	}
	return events, nil
}

func (data *dao) listEventSeats(ctx context.Context, eventID int64) ([]SeatDTO, error) {
	return data.repository.listEventSeats(ctx, eventID)
}

func (data *dao) listBookings(ctx context.Context, userID int64) ([]BookingDTO, error) {
	records, err := data.repository.listBookings(ctx, userID)
	if err != nil {
		return nil, err
	}
	bookings := make([]BookingDTO, 0, len(records))
	for _, record := range records {
		bookings = append(bookings, toBookingDTO(record))
	}
	return bookings, nil
}

func (data *dao) createBooking(ctx context.Context, userID, eventID int64, seatLabels []string) (BookingDTO, error) {
	record, err := data.repository.createBooking(ctx, userID, eventID, seatLabels)
	if err != nil {
		return BookingDTO{}, err
	}
	return toBookingDTO(record), nil
}

func (data *dao) cancelBooking(ctx context.Context, bookingID, userID int64) error {
	return data.repository.cancelBooking(ctx, bookingID, userID)
}

func (data *dao) ping(ctx context.Context) error {
	return data.repository.ping(ctx)
}

func toBookingDTO(record bookingRecord) BookingDTO {
	seatLabels := record.SeatLabels
	if seatLabels == nil {
		seatLabels = []string{}
	}
	return BookingDTO{
		ID: record.ID, EventID: record.EventID, Quantity: record.Quantity,
		SeatLabels: seatLabels, Status: record.Status, CreatedAt: record.CreatedAt, Title: record.Title,
		Category: record.Category, Venue: record.Venue, City: record.City,
		StartsAt: record.StartsAt, PriceCents: record.PriceCents,
	}
}
