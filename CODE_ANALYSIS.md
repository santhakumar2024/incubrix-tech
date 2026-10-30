# Seatflow code analysis

This document explains the frontend and backend logic implemented in Seatflow, what each part is responsible for, and why the project uses this structure.

## 1. What the application does

Seatflow is an event discovery and seat reservation application. A visitor can browse future events, search and filter the list, open a live seat map, select specific seats, and reserve them. Account holders can review bookings, see their seat labels, and cancel upcoming bookings. The application updates seat availability after bookings and cancellations.

The project uses:

- **React and TypeScript** for the browser application.
- **Go and `net/http`** for the JSON API and production static-file server.
- **PostgreSQL** for users, sessions, events, seats, and bookings.
- **Server-Sent Events (SSE)** plus periodic refresh for seat-availability updates.

## 2. Application shape

```text
Browser
  React pages and reusable components
       |
       | same-origin JSON requests + session cookie
       v
  Go HTTP controller
       |
  Go service (validation and business rules)
       |
  Go DAO (database-record / DTO mapping)
       |
  Go repository (SQL and transactions)
       |
       v
  PostgreSQL
```

In production, the Go process also serves the built React files from the same origin. During development, Vite serves the frontend and proxies `/api` requests to the Go server.

## 3. Frontend organization and logic

### Application state and page selection

[`src/App.tsx`](./src/App.tsx) is the frontend coordinator. It owns state shared between pages:

- Events, the current user, and the user’s bookings.
- The selected page: Discover, My bookings, or Profile.
- Search text and category selection.
- Authentication and booking dialogs.
- Loading, error, and temporary success-notice state.

Keeping this state above the pages lets a booking update availability and booking history without duplicating data-loading and account logic on each page. Pages receive the data and actions they need through typed props.

On startup, the app requests the current session (`GET /api/me`) and event list in parallel. If a valid user session exists, it also loads that user’s bookings. It cleans up the startup flow if the component unmounts while requests are in progress.

### Discover page

[`src/pages/DiscoverPage.tsx`](./src/pages/DiscoverPage.tsx) displays the introduction, event search, category filters, result count, event cards, loading state, and errors.

- Search matches the event title, category, venue, or city.
- Category choices are derived from the event data, so a new category does not need a separate hard-coded filter.
- Search and category filtering happen in the browser because the current sample event list is small.
- Retry actions call the shared event loader.

[`src/components/EventCard.tsx`](./src/components/EventCard.tsx) shows each event and its remaining capacity. It disables booking when the event is sold out and delegates selection to the application coordinator.

### Seat selection and booking

[`src/components/BookingDialog.tsx`](./src/components/BookingDialog.tsx) requests the event’s seat map from `GET /api/events/{id}/seats`. It groups returned seats by row and renders each seat as an accessible button.

- Available, selected, and booked seats have distinct visual states.
- A user can select up to ten seats, remove selected seats, and see the seat labels and price total before confirming.
- The dialog listens to the availability SSE stream and refreshes the map every 15 seconds as a fallback.
- If another person books a selected seat, the map removes that seat from the selection and explains what changed.
- Confirmation sends the event ID and exact seat labels to `POST /api/bookings`.
- The server checks availability again when confirming; a seat shown as available in the browser is not treated as a guarantee.

Seats are not temporarily held while a guest is looking at the map. This avoids creating abandoned seat holds; instead, the backend makes the final reservation atomic and reports a conflict if another booking wins first.

### Authentication dialog

[`src/components/AuthDialog.tsx`](./src/components/AuthDialog.tsx) supports registration and sign-in using the same dialog, with the mode switching between the two forms.

When a guest tries to book an event, `App.tsx` remembers that event, opens the sign-in dialog, and opens that event’s seat map after successful authentication. This preserves the user's intended action rather than sending them back to discover the event again.

### Bookings page

[`src/pages/BookingsPage.tsx`](./src/pages/BookingsPage.tsx) displays all bookings, including their status, date, location, price, quantity, and seat labels. It offers cancellation only for confirmed bookings whose event has not started. An empty state links back to event discovery.

### Profile page

[`src/pages/ProfilePage.tsx`](./src/pages/ProfilePage.tsx) has two states:

- Logged-out visitors see sign-in and account-creation actions.
- Signed-in users see the name, email, account number, booking counts, and sign-out action.

The profile displays data already provided by the existing user and booking API; it does not introduce a separate profile database table.

### Shared components, API client, and styling

- [`src/components/SiteChrome.tsx`](./src/components/SiteChrome.tsx) contains the header/navigation, footer, and success toast.
- [`src/components/Icons.tsx`](./src/components/Icons.tsx) contains shared icons and date, time, and currency formatting helpers.
- [`src/api.ts`](./src/api.ts) centralizes fetch requests, includes browser credentials, converts API error responses into readable exceptions, and defines typed operations for auth, events, seats, and bookings.
- [`src/types.ts`](./src/types.ts) describes the API objects shared between pages and components.
- [`src/styles.css`](./src/styles.css) contains the responsive design and the visual states for filters, event cards, dialogs, seat maps, booking history, and profiles.

Separating page components from reusable components keeps page-specific layout separate from the event, seat, account, and navigation elements used in multiple places.

## 4. Backend organization and request flow

The Go files in `server/` share one Go package. The application starts at `main()` in [`server/main.go`](./server/main.go); run it from `server/` with `go run .` so Go compiles all files in that package.

| File | Responsibility |
| --- | --- |
| [`main.go`](./server/main.go) | Reads configuration, initializes dependencies, wires the layers together, registers routes through the controller, and starts the HTTP server. |
| [`controller.go`](./server/controller.go) | Defines HTTP routes, decodes requests, reads the session cookie, calls services, sets response cookies, and writes JSON/errors. |
| [`service.go`](./server/service.go) | Validates input and applies application rules for accounts, sessions, events, seats, booking, and cancellation. |
| [`dao.go`](./server/dao.go) | Maps repository records into response DTOs and delegates data operations to the repository. |
| [`dto.go`](./server/dto.go) | Defines JSON request and response data shapes, including event, booking, user, and seat details. |
| [`repository.go`](./server/repository.go) | Contains PostgreSQL queries, record types, and transactional booking/cancellation operations. |
| [`db.go`](./server/db.go) | Parses the PostgreSQL settings, retries temporary connection errors, initializes the schema, and returns the connection pool. |
| [`hub.go`](./server/hub.go) | Sends availability-change notifications to connected SSE clients in this Go process. |
| [`schema.sql`](./server/schema.sql) | Creates and seeds the database schema, seat maps, and sample events. |

The normal data path is **main → controller → service → DAO → repository → PostgreSQL**. Response data returns through the DAO mapping as DTOs, then the controller serializes JSON. DTOs are shared data definitions, rather than a runtime step that queries the database.

### HTTP API

| Route | Purpose | Authentication |
| --- | --- | --- |
| `POST /api/auth/register` | Create an account and start a session. | Public |
| `POST /api/auth/login` | Verify credentials and start a session. | Public |
| `POST /api/auth/logout` | Delete the current session and expire its cookie. | Session cookie |
| `GET /api/me` | Return the current user, or `null` when signed out. | Optional |
| `GET /api/events` | List future events and current availability. | Public |
| `GET /api/events/{id}/seats` | List a specific event’s seats and availability. | Public |
| `GET /api/bookings` | List the current user's bookings. | Required |
| `POST /api/bookings` | Book specific seats for an event. | Required |
| `DELETE /api/bookings/{id}` | Cancel an upcoming booking owned by the current user. | Required |
| `GET /api/availability/stream` | Stream availability-change notifications. | Public |
| `GET /api/healthz` | Verify that the API can reach PostgreSQL. | Public |

The controller rejects malformed JSON, unknown JSON fields, multiple JSON values, and invalid path IDs. It returns appropriate client errors for validation/authentication/conflict cases, and logs unexpected server/database failures instead of presenting them as success.

### Authentication and sessions

The service hashes passwords with bcrypt. After successful login or registration, it generates a cryptographically random session token. Only a SHA-256 hash of that token is saved in PostgreSQL; the token itself is sent to the browser in an HttpOnly, SameSite=Lax cookie with a 30-day expiration. In HTTPS deployment, `COOKIE_SECURE=true` makes the browser send the cookie only over secure connections.

For booking requests, the controller resolves the cookie to a user, and the service/repository scope booking reads and cancellations to that user ID. Passwords and session tokens are never returned by the user DTO.

### Seat reservation and cancellation

The service validates that an event was supplied and that the request has between one and ten unique, non-empty seat labels. The repository then:

1. Starts a PostgreSQL transaction and locks the event row with `SELECT ... FOR UPDATE`.
2. Rejects missing or already-started events.
3. Checks that every requested label belongs to that event and is not actively booked.
4. Inserts a confirmed booking and a `booking_seats` record for each selected seat.
5. Commits the transaction before notifying SSE listeners.

The event-row lock serializes booking attempts for the same event. A partial unique index on active event/seat assignments provides an additional database-level safeguard against assigning a seat to two active bookings. If any validation or database write fails, the transaction is rolled back.

Cancellation runs in a transaction. It changes the booking status to cancelled and timestamps the associated seat assignments as released. The historical booking and its seat labels remain visible to the original user, while the seat map makes those seats available again.

### Database schema and startup

[`server/schema.sql`](./server/schema.sql) defines:

- `users`: account name, unique email, password hash, and creation time.
- `sessions`: hashed session token, user reference, expiry, and creation time.
- `events`: event details, capacity, price, and presentation accent.
- `bookings`: user/event references, seat quantity, status, and timestamps.
- `event_seats`: the canonical seats belonging to each event.
- `booking_seats`: seat assignments and release timestamps.

Foreign keys, check constraints, indexes, and uniqueness constraints keep invalid or duplicate records out of the database. At API startup, `db.go` applies this SQL. It creates missing seat maps for seeded events and assigns seats to existing bookings during the transition from quantity-only reservations.

Applying schema and data migration work at application startup is convenient for this small demo. For production evolution, use versioned migrations and a controlled migration step rather than running arbitrary schema changes from every application instance.

### Real-time availability

After a committed booking or cancellation, the service publishes a lightweight notification through `eventHub`. The controller streams those notifications over Server-Sent Events. Browsers use the event to request fresh seat/event data; the stream carries an update signal, not the database state itself. A periodic 15-second refresh is a fallback if a stream disconnects.

The current hub is in-process. It is suitable for the documented single-instance deployment, but does not broadcast to other replicas. If running multiple API instances, add shared notification fan-out (for example PostgreSQL `LISTEN/NOTIFY`) so each instance's connected clients receive updates. The database remains authoritative regardless.

## 5. Why this approach

- **React + TypeScript:** React supports interactive page and seat-map state; TypeScript catches mismatched event, seat, booking, and API data during development.
- **Page/component separation:** Page layouts stay independent while cards, dialogs, icons, and site navigation are reusable. Shared session and data state stays in the app coordinator.
- **Go `net/http`:** The standard HTTP server is sufficient for the small JSON API and avoids adding a web framework before it is needed.
- **Explicit backend layers:** Controllers handle HTTP details, services keep business rules testable, DAOs map records, and repositories own SQL. This makes database details less likely to leak into HTTP handlers.
- **PostgreSQL transactions and constraints:** Event-row locking and unique active seat assignments protect the core booking invariant even when requests arrive concurrently.
- **Cookie sessions:** HttpOnly cookies keep session tokens out of JavaScript-accessible browser storage; storing token hashes means a database leak does not directly expose reusable session tokens.
- **Single-origin Docker deployment:** One container runs the built frontend and Go API, simplifying initial hosting and avoiding production CORS configuration.
- **SSE with refresh fallback:** SSE gives fast availability updates with a simple browser-native connection; polling provides eventual refresh if the stream cannot stay connected.
- **No payment integration:** The current booking flow records reservations only. Payments, refunds, and checkout are intentionally outside the current implementation.

## 6. Important limitations

- The supplied event catalog and seat layouts are seeded demo data. Event seats are generated as labeled rows of up to ten seats, not imported venue-specific maps.
- Selecting a seat in the browser does not place a temporary hold. Availability is rechecked when the booking is submitted, and a conflict asks the user to select again.
- The SSE hub is process-local; use shared pub/sub before scaling beyond one API instance.
- Schema initialization happens at application startup; use versioned production migrations when the schema needs to evolve.
- The application has no payment provider, email verification, password reset, admin event editor, or production-ready account recovery.
- AWS resources and a public deployment URL must be created in an AWS account; the guides describe the process but do not provision resources automatically.

## 7. Build and tests

From the project root:

```powershell
npm.cmd run typecheck
npm.cmd run build
```

From `server/`:

```powershell
go test ./...
go vet ./...
```

