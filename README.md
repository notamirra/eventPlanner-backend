# EventPlanner Backend

Go backend server for the EventPlanner application using Gin framework and Postgres. This document covers the Phase 1 implementation including database migrations, API endpoints, and setup instructions.

## Project Structure

```
internal/
  database/       # DB connection (pgx pool)
  handlers/       # HTTP handlers (Gin)
  models/         # Domain models and request DTOs
  repositories/   # Data access layer
  router/         # Router wiring and middleware
  services/       # Business logic
migrations/
  001_init.sql    # Initial schema with users, events, participants, and tasks
  002_add_search_indexes.sql  # Search optimization indexes
main.go           # Entry point wiring layers together
```

## Prerequisites

- Go 1.21+
- Postgres 13+

## Database Setup (Postgres)

### Phase 1 Database Schema

The Phase 1 implementation includes the following tables:
- `users`: User accounts and authentication
- `events`: Event information
- `event_participants`: Many-to-many relationship between users and events
- `tasks`: Tasks associated with events
- `task_assignments`: Many-to-many relationship between tasks and users

### Setup Instructions

1) Create a database and user (example):

```sql
CREATE DATABASE eventplanner;
CREATE USER eventplanner_user WITH PASSWORD 'strongpassword';
GRANT ALL PRIVILEGES ON DATABASE eventplanner TO eventplanner_user;
```

2) Apply migrations in order:

```bash
# Apply initial schema
psql "postgresql://eventplanner_user:strongpassword@localhost:5432/eventplanner?sslmode=disable" -f migrations/001_init.sql

# Apply search optimization indexes
psql "postgresql://eventplanner_user:strongpassword@localhost:5432/eventplanner?sslmode=disable" -f migrations/002_add_search_indexes.sql


psql "postgresql://eventplanner_user:strongpassword@localhost:5432/eventplanner?sslmode=disable" -f migrations/003_add_collaborator_role.sql
```


3) Set environment variable before running the server:

```bash
# Windows PowerShell
$env:DATABASE_URL = "postgresql://eventplanner_user:strongpassword@localhost:5432/eventplanner?sslmode=disable"

# Linux/macOS
export DATABASE_URL="postgresql://eventplanner_user:strongpassword@localhost:5432/eventplanner?sslmode=disable"
```

4) To run migrations using the environment variable (after setting it):

```bash
# Apply initial schema
psql $env:DATABASE_URL -f migrations/001_init.sql

# Apply search optimization indexes
psql $env:DATABASE_URL -f migrations/002_add_search_indexes.sql

# Apply collaborator role
psql $env:DATABASE_URL -f migrations/003_add_collaborator_role.sql
```

## Getting Started

```bash
# Install dependencies
go mod tidy

# Run the server
go run main.go
```

The server will start on `http://localhost:8080`.

## API Endpoints

### Authentication
- `POST /signup` - Register a new user
  - body: `{ "name": string, "email": string, "password": string }`
- `POST /login` - Login with credentials
  - body: `{ "email": string, "password": string }`
- `GET /health` - Health check

### Events
- `POST /events` - Create a new event (organizer only)
  - headers: `X-User-ID: <userId>`
  - body: 
    ```json
    {
      "title": "Event Title",
      "description": "Event description",
      "location": "Event location",
      "startTime": "2025-11-20T14:00:00+02:00"
    }
    ```

- `GET /events/organized` - List events where current user is organizer
  - headers: `X-User-ID: <userId>`

- `GET /events/invited` - List events where current user is attendee
  - headers: `X-User-ID: <userId>`

- `POST /events/:eventId/invite` - Invite a user to an event (organizer only)
  - headers: `X-User-ID: <organizerId>`
  - body: 
    ```json
    {
      "userId": 123,
      "role": "attendee"
    }
    ```
  - roles: `"organizer" | "attendee" | "collaborator"`

- `GET /events/:eventId/attendees` - List event attendees
  - headers: `X-User-ID: <userId>`

- `PUT /events/:eventId/attendance` - Update attendance status
  - headers: `X-User-ID: <userId>`
  - body: 
    ```json
    {
      "userId": 123,
      "status": "going"
    }
    ```
  - status: `"going" | "maybe" | "not_going"`

- `DELETE /events/:eventId` - Delete an event (organizer only)
  - headers: `X-User-ID: <organizerId>`

- `POST /events/:eventId/tasks` - Create a new task (organizer/collaborator)
  - headers: `X-User-ID: <userId>`
  - body:
    ```json
    {
      "title": "Task title",
      "description": "Task description",
      "dueDate": "2025-11-15T14:00:00+02:00",
      "assigneeId": 123
    }
    ```

### Search
- `GET /search` - Search across events and tasks
  - headers: `X-User-ID: <userId>`
  - query params:
    - `q`: Search term (required)
    - `from`: Start date (YYYY-MM-DD)
    - `to`: End date (YYYY-MM-DD)
    - `role`: Filter by role (e.g., "organizer")

### Phase 1 Implementation Details

#### Database Migrations

**001_init.sql**
- Creates all necessary tables with appropriate constraints and relationships
- Sets up indexes for frequently queried fields
- Includes sample data for testing

**002_add_search_indexes.sql**
- Adds GIN indexes for full-text search on event titles and descriptions
- Optimizes search performance for the /search endpoint

#### Authentication
- Password hashing using bcrypt
- Session management with refresh tokens



## Development Workflow

1. **Database Migrations**
   - All schema changes must be made through migrations
   - Name migration files with sequential numbers and descriptive names
   - Test migrations in development before committing

2. **Environment Variables**
   ```
   DATABASE_URL=postgresql://user:password@localhost:5432/eventplanner
   JWT_SECRET=your-secret-key
   PORT=8080
   ENV=development|production
   ```

3. **Testing**
   ```bash
   # Run unit tests
   go test ./...
   
   # Run integration tests
   go test -tags=integration ./...
   ```

## API Rate Limiting
- 1000 requests per hour per IP address
- 100 requests per minute per authenticated user
- Headers included in rate-limited responses:
  - `X-RateLimit-Limit`: Request limit
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: Time when the limit resets (UTC timestamp)

## Migrations

Apply all migrations in order:

```bash
# Windows PowerShell
psql $env:DATABASE_URL -f migrations/001_init.sql
psql $env:DATABASE_URL -f migrations/002_phase1.sql
psql $env:DATABASE_URL -f migrations/003_add_collaborator_role.sql

# Linux/macOS
psql "$DATABASE_URL" -f migrations/001_init.sql
psql "$DATABASE_URL" -f migrations/002_phase1.sql
```

## Dependencies

- Gin web framework
- pgx (Postgres driver)
- bcrypt for password hashing
- CORS middleware for cross-origin requests
