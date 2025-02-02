## System Setup

1. Install Go 1.20 or later
2. Clone the repository
3. Copy `.env.example` to `.env` and configure your environment variables
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Start the development server:
   ```bash
   go run main.go
   ```

The API server will be available at `http://localhost:8080`


## Project Structure

```
├── api/               - API handlers and routes
│   ├── github.go     - GitHub synchronization logic
├── models/           - Database models and ORM layer
│   ├── user.go       - User model and DB operations
│   ├── habit.go      - Habit tracking logic
│   └── goal.go       - Goal management
├── middleware/       - Authentication and session management
│   ├── auth.go       - Authorization middleware
│   └── session.go    - Session configuration
├── ui/               - User interface components
│   ├── components/   - Reusable template components
│   ├── layouts/      - Base page layouts
│   └── static/       - Static assets (CSS, JS, images)
├── main.go           - Application entry point
└── .env.example      - Environment configuration template
```

## Key Components:
- `api/`: Handles external integrations and REST API endpoints
- `models/`: Database schema definitions and CRUD operations
- `middleware/`: Authentication flow and session management
- `ui/`: HTML templates and frontend assets using Go's template engine


## License

This project is licensed under the [GNU Affero General Public License v3](LICENSE.txt).  
Copyright © 2024 Emanuele Faja. See [LICENSE.txt](LICENSE.txt) for full terms.