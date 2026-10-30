package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errEmailExists      = errors.New("email already exists")
	errEventNotFound    = errors.New("event not found")
	errEventStarted     = errors.New("event already started")
	errSeatsUnavailable = errors.New("selected seats unavailable")
	errBookingNotFound  = errors.New("active booking not found")
)

type userRecord struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
}

type eventRecord struct {
	ID             int64
	Slug           string
	Title          string
	Category       string
	Venue          string
	City           string
	StartsAt       time.Time
	Description    string
	TotalCapacity  int
	BookedQuantity int
	PriceCents     int
	Accent         string
}

type bookingRecord struct {
	ID         int64
	EventID    int64
	Quantity   int
	Status     string
	CreatedAt  time.Time
	Title      string
	Category   string
	Venue      string
	City       string
	StartsAt   time.Time
	PriceCents int
	SeatLabels []string
}

type repository struct {
	db *pgxpool.Pool
}

func newRepository(db *pgxpool.Pool) *repository {
	return &repository{db: db}
}

func (repo *repository) createUser(ctx context.Context, name, email, passwordHash string) (userRecord, error) {
	var record userRecord
	err := repo.db.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, name, email`,
		name, email, passwordHash,
	).Scan(&record.ID, &record.Name, &record.Email)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return userRecord{}, errEmailExists
		}
		return userRecord{}, fmt.Errorf("create user: %w", err)
	}
	return record, nil
}

func (repo *repository) findUserByEmail(ctx context.Context, email string) (userRecord, error) {
	var record userRecord
	err := repo.db.QueryRow(ctx,
		`SELECT id, name, email, password_hash FROM users WHERE email = $1`, email,
	).Scan(&record.ID, &record.Name, &record.Email, &record.PasswordHash)
	if err != nil {
		return userRecord{}, err
	}
	return record, nil
}

func (repo *repository) findUserBySession(ctx context.Context, tokenHash string) (userRecord, error) {
	var record userRecord
	err := repo.db.QueryRow(ctx, `
		SELECT u.id, u.name, u.email
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > NOW()`, tokenHash,
	).Scan(&record.ID, &record.Name, &record.Email)
	return record, err
}

func (repo *repository) createSession(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time) error {
	if _, err := repo.db.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt,
	); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (repo *repository) deleteSession(ctx context.Context, tokenHash string) error {
	if _, err := repo.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (repo *repository) listEvents(ctx context.Context) ([]eventRecord, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT e.id, e.slug, e.title, e.category, e.venue, e.city, e.starts_at, e.description,
			e.total_capacity, COALESCE(SUM(b.quantity) FILTER (WHERE b.status = 'confirmed'), 0)::int,
			e.price_cents, e.accent
		FROM events e LEFT JOIN bookings b ON b.event_id = e.id
		WHERE e.starts_at > NOW()
		GROUP BY e.id ORDER BY e.starts_at`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	records := make([]eventRecord, 0)
	for rows.Next() {
		var record eventRecord
		if err := rows.Scan(&record.ID, &record.Slug, &record.Title, &record.Category, &record.Venue,
			&record.City, &record.StartsAt, &record.Description, &record.TotalCapacity,
			&record.BookedQuantity, &record.PriceCents, &record.Accent); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return records, nil
}

func (repo *repository) listEventSeats(ctx context.Context, eventID int64) ([]SeatDTO, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT s.seat_label, s.row_label, s.seat_number,
			NOT EXISTS (
				SELECT 1 FROM booking_seats bs
				WHERE bs.event_id = s.event_id
					AND bs.seat_label = s.seat_label
					AND bs.released_at IS NULL
			) AS available
		FROM event_seats s
		WHERE s.event_id = $1
		ORDER BY s.row_label, s.seat_number`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seats for event %d: %w", eventID, err)
	}
	defer rows.Close()
	seats := make([]SeatDTO, 0)
	for rows.Next() {
		var seat SeatDTO
		if err := rows.Scan(&seat.Label, &seat.Row, &seat.Number, &seat.Available); err != nil {
			return nil, fmt.Errorf("scan event seat: %w", err)
		}
		seats = append(seats, seat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event seats: %w", err)
	}
	return seats, nil
}

func (repo *repository) listBookings(ctx context.Context, userID int64) ([]bookingRecord, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT b.id, b.event_id, b.quantity, b.status, b.created_at, e.title, e.category,
			e.venue, e.city, e.starts_at, e.price_cents,
			COALESCE(array_agg(bs.seat_label ORDER BY seat.row_label, seat.seat_number)
				FILTER (WHERE bs.seat_label IS NOT NULL), ARRAY[]::text[])
		FROM bookings b
		JOIN events e ON e.id = b.event_id
		LEFT JOIN booking_seats bs ON bs.booking_id = b.id
		LEFT JOIN event_seats seat ON seat.event_id = bs.event_id AND seat.seat_label = bs.seat_label
		WHERE b.user_id = $1
		GROUP BY b.id, e.id
		ORDER BY e.starts_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()
	records := make([]bookingRecord, 0)
	for rows.Next() {
		var record bookingRecord
		if err := rows.Scan(&record.ID, &record.EventID, &record.Quantity, &record.Status,
			&record.CreatedAt, &record.Title, &record.Category, &record.Venue, &record.City,
			&record.StartsAt, &record.PriceCents, &record.SeatLabels); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bookings: %w", err)
	}
	return records, nil
}

func (repo *repository) createBooking(ctx context.Context, userID, eventID int64, seatLabels []string) (bookingRecord, error) {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return bookingRecord{}, fmt.Errorf("begin booking transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var startsAt time.Time
	err = tx.QueryRow(ctx, `SELECT starts_at FROM events WHERE id = $1 FOR UPDATE`, eventID).Scan(&startsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return bookingRecord{}, errEventNotFound
	}
	if err != nil {
		return bookingRecord{}, fmt.Errorf("lock event for booking: %w", err)
	}
	if !startsAt.After(time.Now()) {
		return bookingRecord{}, errEventStarted
	}
	var validSeats int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM event_seats seat
		WHERE seat.event_id = $1 AND seat.seat_label = ANY($2::text[])
			AND NOT EXISTS (
				SELECT 1 FROM booking_seats bs
				WHERE bs.event_id = seat.event_id
					AND bs.seat_label = seat.seat_label
					AND bs.released_at IS NULL
			)`, eventID, seatLabels).Scan(&validSeats); err != nil {
		return bookingRecord{}, fmt.Errorf("check selected seats: %w", err)
	}
	if validSeats != len(seatLabels) {
		return bookingRecord{}, errSeatsUnavailable
	}

	var record bookingRecord
	err = tx.QueryRow(ctx, `
		INSERT INTO bookings (user_id, event_id, quantity) VALUES ($1, $2, $3)
		RETURNING id, event_id, quantity, status, created_at`,
		userID, eventID, len(seatLabels),
	).Scan(&record.ID, &record.EventID, &record.Quantity, &record.Status, &record.CreatedAt)
	if err != nil {
		return bookingRecord{}, fmt.Errorf("insert booking: %w", err)
	}
	for _, seatLabel := range seatLabels {
		if _, err := tx.Exec(ctx,
			`INSERT INTO booking_seats (booking_id, event_id, seat_label) VALUES ($1, $2, $3)`,
			record.ID, eventID, seatLabel,
		); err != nil {
			var postgresError *pgconn.PgError
			if errors.As(err, &postgresError) && postgresError.Code == "23505" {
				return bookingRecord{}, errSeatsUnavailable
			}
			return bookingRecord{}, fmt.Errorf("assign seat %s: %w", seatLabel, err)
		}
	}
	record.SeatLabels = append([]string(nil), seatLabels...)
	if err := tx.Commit(ctx); err != nil {
		return bookingRecord{}, fmt.Errorf("commit booking: %w", err)
	}
	return record, nil
}

func (repo *repository) cancelBooking(ctx context.Context, bookingID, userID int64) error {
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin cancellation transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	commandTag, err := tx.Exec(ctx, `
		UPDATE bookings b SET status = 'cancelled', cancelled_at = NOW()
		FROM events e
		WHERE b.id = $1 AND b.user_id = $2 AND b.status = 'confirmed'
			AND e.id = b.event_id AND e.starts_at > NOW()`, bookingID, userID)
	if err != nil {
		return fmt.Errorf("cancel booking: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return errBookingNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE booking_seats SET released_at = NOW() WHERE booking_id = $1 AND released_at IS NULL`, bookingID); err != nil {
		return fmt.Errorf("release seats for booking: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit booking cancellation: %w", err)
	}
	return nil
}

func (repo *repository) ping(ctx context.Context) error {
	if err := repo.db.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return nil
}
