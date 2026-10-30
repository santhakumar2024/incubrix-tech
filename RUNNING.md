# Running Seatflow locally

This guide covers the quickest way to run the full application and the separate frontend/backend development workflow.

## Prerequisites

Choose one setup:

- **Docker Compose:** Docker Desktop with Docker Compose.
- **Development mode:** Node.js 22 or later, Go 1.22 or later, and PostgreSQL 16.

On Windows PowerShell, use `npm.cmd` if your execution policy prevents running `npm.ps1`.

## Option 1: Run the whole application with Docker

From the project root:

```powershell
docker compose up --build
```

Wait for the API container to start, then open [http://localhost:8080](http://localhost:8080). Compose starts PostgreSQL, builds the frontend and API, and serves both from the API container. On first startup, the API creates the tables and sample events.

To stop the application, press `Ctrl+C`. You can restart it with the same command; your database data is retained in the Compose volume.

To stop the application and remove its local database data:

```powershell
docker compose down -v
```

> The `-v` option permanently removes the local Seatflow database volume.

## Option 2: Run the frontend and API separately

### 1. Start PostgreSQL

Start a local PostgreSQL 16 server and create a database and user. For example, using `psql` as a PostgreSQL administrator:

```sql
CREATE USER seatflow WITH PASSWORD 'seatflow';
CREATE DATABASE seatflow OWNER seatflow;
```

The credentials above are for local development only. Do not use them in a public or production environment.

### 2. Start the Go API

Open a PowerShell terminal at the project root:

```powershell
Set-Location server
$env:DATABASE_URL = "postgres://seatflow:seatflow@localhost:5432/seatflow?sslmode=disable"
$env:PORT = "8080"
$env:FRONTEND_ORIGIN = "http://localhost:5173"
$env:COOKIE_SECURE = "false"
go run .
```

The API initializes the database schema and sample event data when it starts. Keep this terminal open. Check that the API and database are ready by visiting [http://localhost:8080/api/healthz](http://localhost:8080/api/healthz). The response should be:

```json
{"status":"ok"}
```

### 3. Start the React frontend

Open a second PowerShell terminal at the project root:

```powershell
npm.cmd install
npm.cmd run dev
```

Open [http://localhost:5173](http://localhost:5173). Vite forwards `/api` requests to the Go API at `http://localhost:8080`.

If the API is running on another address, set the Vite proxy before starting the frontend:

```powershell
$env:VITE_API_PROXY = "http://localhost:8081"
npm.cmd run dev
```

Make sure `FRONTEND_ORIGIN` in the API terminal matches the origin used by your browser.

## Environment variables

| Variable | Used by | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Go API | PostgreSQL connection URL. Required. |
| `PORT` | Go API | API listen port; defaults to `8080`. |
| `FRONTEND_ORIGIN` | Go API | Allowed browser origin for credentialed API requests; defaults to `http://localhost:5173`. |
| `COOKIE_SECURE` | Go API | Use `false` for local HTTP. Set to `true` when serving the application over HTTPS. |
| `WEB_DIST` | Go API | Built frontend directory for production serving; defaults to `../dist`. |
| `VITE_API_PROXY` | Vite | Go API address for the development proxy; defaults to `http://localhost:8080`. |

An example configuration is in [.env.example](./.env.example). The Go API does not automatically load `.env` files; set its variables in the shell or your deployment environment.

## Run checks

From the project root:

```powershell
npm.cmd run typecheck
npm.cmd run build
```

Run the Go tests and static checks:

```powershell
Set-Location server
go test ./...
go vet ./...
```

## Test a booking

1. Open the application and choose **Join us** to create an account. Use a valid email address and a password of at least eight characters.
2. Browse an event and select **Find your spot**.
3. Select available seats on the map (up to 10 per booking) and confirm.
4. Open **My bookings** to see your reservation and assigned seats. Cancel an upcoming booking to return those seats to availability.
5. To observe live availability, open the app in another browser session and make or cancel a booking.
