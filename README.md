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
│   ├── admin.go      - Admin endpoints
│   ├── github.go     - GitHub synchronization
│   ├── goal.go       - Goal management
│   ├── habit.go      - Habit operations
│   ├── password_reset.go - Password reset functionality
│   ├── roadmap.go    - Product roadmap
│   ├── stats.go      - Statistics endpoints
│   └── user.go       - User profile API
├── cmd/              - Command-line tools
│   └── send_email/   - Email testing tool
├── content/           - Content files
│   ├── blog/         - Blog posts (.md files)
│   └── media/        - Media files (images, videos, etc.)
├── models/            - Database models and ORM
│   ├── admin.go      - Admin models
│   ├── blog.go       - Blog models
│   ├── db.go         - Database connection
│   ├── email/        - Email functionality
│   │   ├── email.go      - Core email types
│   │   ├── templates.go  - Template rendering
│   │   ├── smtp.go       - SMTP implementation
│   │   └── email_test.go - Email tests
│   ├── goal.go       - Goal models
│   ├── habit.go      - Habit tracking logic
│   ├── stats.go      - Statistics models
│   └── user.go       - User models
├── middleware/        - Request processing
│   ├── auth.go       - Authentication
│   ├── ratelimit.go  - Rate limiting (auth & password reset)
│   └── sessions.go   - Session management
├── static/            - Static assets
│   ├── icons/        - Application icons
│   ├── images/       - Screenshots/illustrations
│   ├── sounds/       - Notification sounds
│   ├── videos/       - Changelog videos
│   ├── manifest.json - PWA manifest
│   └── sw.js         - Service worker
├── ui/                - User interface
│   ├── components/   - Reusable templates
│   ├── email/        - Email templates
│   │   ├── reset-password.html    - Password reset email
│   │   ├── reset-password.txt     - Plain text version
│   │   ├── reset-success.html     - Reset confirmation
│   │   ├── reset-success.txt      - Plain text version
│   │   ├── welcome.html          - Welcome email
│   │   └── welcome.txt           - Plain text version
│   ├── habits/       - Habit-type views
│   ├── blog/         - Blog templates
│   ├── forgot.html   - Forgot password page
│   ├── reset.html    - Password reset page
│   └── *.html        - Core application pages
├── main.go           - Application entry
├── openapi.yaml      - API documentation
└── .env.example      - Environment template
```

## Key Components:
- `api/`: Handles external integrations and REST API endpoints
- `models/`: Database schema definitions and CRUD operations
- `middleware/`: Authentication flow and session management
- `ui/`: HTML templates and frontend assets using Go's template engine

## Tech Stack

### Backend
- **Go**: Fast, scalable and reliable backend
- **alexedwards/scs**: Session management
- **mattn/go-sqlite3**: SQLite driver
- **golang.org/x/crypto**: Secure password hashing
- **joho/godotenv**: Environment variables management
- **yuin/goldmark**: Markdown processing
- **Air**: Live reload development

### Frontend
- **AlpineJS**: Lightweight interactivity
- **Tailwind CSS**: Modern, responsive styling
- **Chart.js**: Data visualization
- **SortableJS**: Drag-and-drop functionality
- **Emoji Mart**: Emoji picker and search
- **Google Fonts**: Modern typography (Inter)
- **Canvas Confetti**: Celebration animations
- **PinesUI**: Alpine & Tailwind UI component library

### Database
- **SQLite**: Reliable, zero-dependency data storage
- **Litestream**: Continuous SQLite replication

### Infrastructure
- **Cloudflare**: DNS and DDoS protection
- **Render**: Cloud & database hosting
- **Github**: Version control

This project is open source and licensed under the AGPL-3.0 license. See the [LICENSE.txt](LICENSE.txt) file for more details. 

Feel free to contribute to the project by opening a PR or by reporting an issue.


## Self Host Locally

You can self host habits on your own server or computer.

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
