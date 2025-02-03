Welcome to habits — a free, open-source, minimalist habit tracker.

![Habits Grid](/static/images/habit-grid.png)

With habits, you can track various different types of habits, including:

- Binary: Choose whether you did or didn't do the habit (i.e. did you meditate, did you exercise, etc.)
- Options: Choose from a set of options (i.e. your mood, your rating, etc.)
- Number: Track a number (i.e. how many times you did the habit)
- Set-Reps: Track a set of reps (i.e. how many sets and reps you did)

For each habit type, you also have the ability to drill down into the habit to see your progress over time and some interesting statistics:

![Habits Stats](/static/images/habit-statistics.png)

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

This project is open source and licensed under the AGPL-3.0 license. See the [LICENSE.txt](LICENSE.txt) file for more details. 

Feel free to contribute to the project by opening a PR or by reporting an issue.

# Self Host Locally

- git clone https://github.com/emanuelefaja/habits
- go build -o habits
- ./habits

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

**Start your journey** ➡️ [habits.co](https://habits.co)
