# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Running the Application
```bash
# Development server with live reload
air

# Production build and run
go build -o ./tmp/main .
./tmp/main

# Direct run (development)
go run main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific test files
go test ./models/
go test ./tests/campaigns/
go test ./tests/scheduler/

# Run tests with coverage
go test -cover ./...
```

### Database Management
- Database file: `habits.db` (SQLite with WAL mode)
- Schema managed in `models/db.go`
- Migrations run automatically on startup

## Architecture Overview

**Habits** is a Go web application using server-side rendering with minimal JavaScript. It follows clean architecture principles:

### Core Layers
- **API Layer** (`/api/`): REST endpoints for habit operations, user management, admin functions
- **Models Layer** (`/models/`): Database operations, business logic, email system
- **Middleware** (`/middleware/`): Authentication, sessions, rate limiting
- **Web Layer** (`/web/`): HTTP routing, handlers, template rendering
- **UI Layer** (`/ui/`): Go templates with Alpine.js for interactivity

### Key Technologies
- **Backend**: Go 1.23.2, SQLite, session-based auth
- **Frontend**: Go templates, Alpine.js, Tailwind CSS, Chart.js
- **Email**: SMTP with campaign management and cron scheduling
- **PWA**: Service worker, manifest.json for offline capability

### Habit Types
The application supports 4 distinct habit tracking types:
1. **Binary**: Simple yes/no completion (`ui/habits/binary.html`)
2. **Numeric**: Count-based tracking (`ui/habits/numeric.html`) 
3. **Choice**: Multiple option selection (`ui/habits/choice.html`)
4. **Set-Reps**: Exercise tracking with sets/reps (`ui/habits/set-rep.html`)

Each type has specific models in `models/habit.go` and dedicated UI templates.

### Database Schema
- `users`: User accounts and preferences
- `habits`: Habit definitions with type-specific JSON options
- `habit_logs`: Daily completion records
- `goals`: User-defined objectives
- Email campaign tables for automated courses

### Email System
Sophisticated email campaign management in `models/email/`:
- Template-based emails (HTML + text versions in `ui/email/`)
- Automated course delivery via cron jobs
- Campaign management with user segmentation
- SMTP service abstraction for different providers

### Authentication Flow
- Session-based authentication using `alexedwards/scs`
- Rate limiting on auth endpoints (`middleware/ratelimit.go`)
- Password reset functionality with secure tokens
- CSRF protection and bcrypt password hashing

## Development Patterns

### Template System
- Base templates in `ui/components/`
- Page-specific templates in `ui/`
- Email templates have both HTML and text versions
- Go's `html/template` with custom functions for date formatting, etc.

### API Design
- RESTful endpoints in `/api/` directory
- OpenAPI specification in `openapi.yaml`
- JSON responses with consistent error handling
- Rate limiting on sensitive endpoints

### Testing Strategy
- Unit tests for models (`*_test.go` files)
- Integration tests in `/tests/` directory
- Email template testing (`tests/campaigns/templates/`)
- In-memory SQLite for test isolation

### Progressive Web App
- Manifest and service worker in `/static/`
- Offline-first design with local storage fallbacks
- Mobile-responsive grid layouts for habit tracking

## File Organization Notes

- Static assets embedded using Go's `embed` directive
- Blog content stored as Markdown files in `/content/blog/`
- Course content templates in `/ui/courses/` and `/ui/masterclass/`
- Admin functionality separated into dedicated handlers and templates
- GitHub integration for changelog and commit tracking