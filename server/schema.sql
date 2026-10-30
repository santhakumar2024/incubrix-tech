CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash CHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    category TEXT NOT NULL,
    venue TEXT NOT NULL,
    city TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    description TEXT NOT NULL,
    total_capacity INTEGER NOT NULL CHECK (total_capacity > 0),
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    accent TEXT NOT NULL DEFAULT 'violet'
);

CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 10),
    status TEXT NOT NULL DEFAULT 'confirmed' CHECK (status IN ('confirmed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS bookings_user_created_idx ON bookings (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS bookings_event_status_idx ON bookings (event_id, status);

CREATE TABLE IF NOT EXISTS event_seats (
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    seat_label TEXT NOT NULL,
    row_label TEXT NOT NULL,
    seat_number INTEGER NOT NULL CHECK (seat_number > 0),
    PRIMARY KEY (event_id, seat_label),
    UNIQUE (event_id, row_label, seat_number)
);

CREATE TABLE IF NOT EXISTS booking_seats (
    booking_id BIGINT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    seat_label TEXT NOT NULL,
    released_at TIMESTAMPTZ,
    PRIMARY KEY (booking_id, seat_label),
    FOREIGN KEY (event_id, seat_label) REFERENCES event_seats(event_id, seat_label)
);

CREATE UNIQUE INDEX IF NOT EXISTS booking_seats_active_unique_idx
    ON booking_seats (event_id, seat_label) WHERE released_at IS NULL;

INSERT INTO events (slug, title, category, venue, city, starts_at, description, total_capacity, price_cents, accent)
VALUES
    ('moonlight-cinema', 'Cinema under the stars', 'Film', 'The Garden Screen', 'Brooklyn, NY', NOW() + INTERVAL '3 days', 'A cult-classic double feature, cozy blankets, and a sky full of stars.', 84, 1800, 'citrus'),
    ('blue-note-session', 'The Blue Note session', 'Live music', 'Room 27', 'Brooklyn, NY', NOW() + INTERVAL '5 days', 'An intimate night of soul, improvisation, and very good company.', 48, 3200, 'lilac'),
    ('sunday-table', 'The Sunday supper club', 'Food & drink', 'Maison Rue', 'Manhattan, NY', NOW() + INTERVAL '7 days', 'Six courses, one long table, and a menu inspired by the market.', 36, 6800, 'coral'),
    ('after-hours-gallery', 'After hours at the gallery', 'Art & culture', 'Northside Arts', 'Queens, NY', NOW() + INTERVAL '9 days', 'Explore the new contemporary collection with a curator-led tour.', 60, 2400, 'sky'),
    ('comedy-basement', 'Laugh a little louder', 'Comedy', 'The Basement Stage', 'Manhattan, NY', NOW() + INTERVAL '12 days', 'A surprise lineup of sharp new voices and downtown favorites.', 72, 2200, 'rose'),
    ('sunrise-flow', 'Slow morning, good energy', 'Wellness', 'Good Day Studio', 'Brooklyn, NY', NOW() + INTERVAL '14 days', 'A feel-good flow, a fresh juice, and a softer start to your weekend.', 30, 2600, 'mint')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO event_seats (event_id, seat_label, row_label, seat_number)
SELECT e.id,
       chr(65 + ((seat.seat_index - 1) / 10)) || (((seat.seat_index - 1) % 10) + 1)::text,
       chr(65 + ((seat.seat_index - 1) / 10)),
       ((seat.seat_index - 1) % 10) + 1
FROM events e
CROSS JOIN LATERAL generate_series(1, e.total_capacity) AS seat(seat_index)
ON CONFLICT (event_id, seat_label) DO NOTHING;

DO $$
DECLARE
    legacy_booking RECORD;
    assigned_count INTEGER;
BEGIN
    FOR legacy_booking IN
        SELECT b.id, b.event_id, b.quantity, b.status
        FROM bookings b
        WHERE NOT EXISTS (
            SELECT 1 FROM booking_seats bs WHERE bs.booking_id = b.id
        )
        ORDER BY b.created_at, b.id
    LOOP
        INSERT INTO booking_seats (booking_id, event_id, seat_label, released_at)
        SELECT legacy_booking.id, legacy_booking.event_id, available.seat_label,
               CASE WHEN legacy_booking.status = 'cancelled' THEN NOW() ELSE NULL END
        FROM (
            SELECT s.seat_label
            FROM event_seats s
            WHERE s.event_id = legacy_booking.event_id
              AND NOT EXISTS (
                  SELECT 1 FROM booking_seats active
                  WHERE active.event_id = s.event_id
                    AND active.seat_label = s.seat_label
                    AND active.released_at IS NULL
              )
            ORDER BY s.row_label, s.seat_number
            LIMIT legacy_booking.quantity
        ) available;

        GET DIAGNOSTICS assigned_count = ROW_COUNT;
        IF assigned_count < legacy_booking.quantity AND legacy_booking.status = 'confirmed' THEN
            RAISE EXCEPTION 'Cannot migrate booking %: only % seats remain, but % are required',
                legacy_booking.id, assigned_count, legacy_booking.quantity;
        END IF;
    END LOOP;
END $$;
