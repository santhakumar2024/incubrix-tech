# Seatflow

Seatflow is a real-time event booking application. Browse local experiences, select specific seats from an interactive availability map, review your bookings, and cancel upcoming plans. Availability changes are broadcast over Server-Sent Events and refreshed periodically as a fallback.

## Stack

- **Frontend:** React 18, TypeScript, Vite
- **API:** Go 1.22+, `net/http`, PostgreSQL driver (`pgx`)
- **Database:** PostgreSQL 16
- **Authentication:** bcrypt password hashes and opaque, expiring, HttpOnly session cookies
- **Deployment:** Multi-stage Docker image; suitable for Amazon ECS on AWS Fargate with an Application Load Balancer and Amazon RDS for PostgreSQL

## Run locally

For a step-by-step Windows/PowerShell guide, see [RUNNING.md](./RUNNING.md).

### Docker Compose (quick start)

Prerequisites: Docker Desktop with Compose.

```powershell
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080). The compose file starts PostgreSQL and a single application container that serves both the built frontend and API. The database is initialized and sample events are inserted at startup.

Stop the services with `Ctrl+C`; run `docker compose down` to stop them. Add `-v` only if you also want to remove the local database volume and its data.

### Frontend and API in development mode

Prerequisites: Node.js 22+, Go 1.22+, and PostgreSQL 16. Start a local PostgreSQL instance and create a database/user matching the example `DATABASE_URL` below. The API applies `server/schema.sql` automatically.

In PowerShell, start the API:

```powershell
cd server
$env:DATABASE_URL = "postgres://seatflow:seatflow@localhost:5432/seatflow?sslmode=disable"
$env:PORT = "8080"
$env:FRONTEND_ORIGIN = "http://localhost:5173"
$env:COOKIE_SECURE = "false"
go run .
```

In a second terminal, start the Vite frontend from the project root:

```powershell
npm.cmd install
npm.cmd run dev
```

The `.cmd` suffix avoids PowerShell's script execution policy blocking `npm.ps1`. In Command Prompt or another shell, `npm install` and `npm run dev` also work.

Open [http://localhost:5173](http://localhost:5173). Vite proxies `/api` requests to `http://localhost:8080`; set `VITE_API_PROXY` if your API runs elsewhere.

Run frontend checks with `npm.cmd run typecheck` and `npm.cmd run build` in PowerShell. Run backend tests from `server` with `go test ./...`.

## Environment variables

See [.env.example](./.env.example) for a copyable, non-secret example. The Go API reads environment variables directly; it does not load a `.env` file.

| Variable | Required | Description |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL connection URL. Use `sslmode=require` for a TLS-enabled hosted database. |
| `PORT` | No | HTTP listen port; defaults to `8080`. |
| `FRONTEND_ORIGIN` | No | Allowed browser origin for credentialed API requests; defaults to `http://localhost:5173`. |
| `COOKIE_SECURE` | No | Set to `true` behind HTTPS so session cookies are sent only over secure connections. |
| `WEB_DIST` | No | Built frontend directory served by Go; defaults to `../dist`. |
| `VITE_API_PROXY` | No | Vite development proxy target; defaults to `http://localhost:8080`. |

Never put production credentials in `.env.example`, source control, or a frontend `VITE_*` variable. Use AWS Secrets Manager and ECS task-definition secret references for production database credentials.

## Architecture and decisions

The browser talks to a Go JSON API. PostgreSQL stores users, opaque session-token hashes, events, and bookings. Passwords are bcrypt-hashed; the browser receives only an expiring HttpOnly, SameSite=Lax session cookie. Session tokens are random and stored as SHA-256 digests.

The React entry point is `src/App.tsx`, which owns shared session and navigation state. Page UI lives in `src/pages/DiscoverPage.tsx` and `src/pages/BookingsPage.tsx`; reusable cards, dialogs, icons, and site navigation live in `src/components/`.

The Go request path is `main.go` → `controller.go` → `service.go` → `dao.go` → `repository.go`. `dto.go` defines the request/response shapes passed between the controller and service and returned from DAO mapping; `db.go` initializes PostgreSQL and applies the schema. Controllers handle HTTP and cookies, services enforce business rules, DAOs translate database records into DTOs, and repositories own SQL and transactions.

Each event has a persistent seat map. Reservation creation locks its event row in a database transaction, checks each selected seat, and records the exact seats on the booking. A database uniqueness constraint also prevents two active bookings from owning the same seat. Cancellation is scoped to the signed-in user and upcoming events, releases the selected seats, and retains them in booking history. Availability updates are announced through an SSE stream, with a 15-second browser refresh as a recovery path.

The SSE broadcaster is process-local. Run one API instance for immediate cross-user push updates; with multiple replicas, use a shared PostgreSQL `LISTEN/NOTIFY` or another pub/sub service to fan out SSE events across instances. Database reads remain authoritative, so the periodic refresh still converges.

The API serves the built Vite app in the production container, keeping the browser and API on one origin. For local development, Vite provides the same-origin `/api` proxy. The schema is idempotently applied at API startup and provisions seat rows for events, mapping existing bookings onto seats when upgrading a database; use versioned migrations for production schema evolution.

For a detailed walkthrough of the frontend and backend logic and the reasoning behind these choices, see [CODE_ANALYSIS.md](./CODE_ANALYSIS.md).

## AWS deployment

New to AWS? Start with the [AWS account setup guide](./AWS_ACCOUNT_SETUP.md) to create and secure your account and add a billing alert. Then follow the [AWS deployment guide](./AWS_DEPLOYMENT.md) for ECR, private RDS for PostgreSQL, Secrets Manager, ECS Fargate networking, an Application Load Balancer, and deployment verification.

Production AWS resources and a public URL have **not** been provisioned from this workspace: AWS account access, deployment credentials, and a Git remote were not provided. Add the deployed application URL and Git repository URL here when you publish the project. Do not commit credentials.

## Test the application

1. Open the app and choose **Join us** to create an account (password must be at least 8 characters).
2. Browse and filter events; select **Find your spot**, choose specific seats from the seat map, and confirm.
3. Open **My bookings** to verify the reservation, then cancel an upcoming booking.
4. Open the app in another browser session to see availability update after a reservation or cancellation.
5. The API health check is available at `/api/healthz`.
