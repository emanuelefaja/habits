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
│   ├── campaign.go   - Email campaign management
│   ├── github.go     - GitHub synchronization
│   ├── goal.go       - Goal management
│   ├── habit.go      - Habit operations
│   ├── password_reset.go - Password reset functionality
│   ├── roadmap.go    - Product roadmap
│   ├── stats.go      - Statistics endpoints
│   └── user.go       - User profile API
├── content/           - Content files
│   ├── blog/         - Blog posts (.md files)
│   └── media/        - Media files (images, videos, etc.)
├── models/            - Database models and ORM
│   ├── admin.go      - Admin models
│   ├── blog.go       - Blog models
│   ├── commit.go     - GitHub commit tracking
│   ├── db.go         - Database connection and schema
│   ├── email/        - Email functionality
│   │   ├── campaign.go  - Email campaign management
│   │   ├── email.go     - Core email types
│   │   ├── smtp.go      - SMTP implementation
│   │   └── templates.go - Template rendering
│   ├── goal.go       - Goal models
│   ├── habit.go      - Habit tracking logic
│   ├── habit_test.go - Habit tests
│   ├── quotes.go     - Motivational quotes functionality
│   ├── quotes_test.go - Quotes tests
│   ├── scheduler.go  - Email notification scheduler
│   ├── stats.go      - Statistics models
│   ├── user.go       - User models
│   └── user_test.go  - User tests
├── middleware/        - Request processing
│   ├── auth.go       - Authentication
│   ├── ratelimit.go  - Rate limiting (auth & password reset)
│   └── sessions.go   - Session management
├── specs/             - Feature specifications
│   ├── email-campaigns.md - Email campaign design
│   ├── main-refactor.md   - Core refactoring plans
│   └── notifications.md   - Notification system design
├── static/            - Static assets
│   ├── favicon.png    - Site favicon
│   ├── github-mark.svg - GitHub logos
│   ├── icons/        - Application icons
│   ├── images/       - Screenshots/illustrations
│   ├── js/           - JavaScript files
│   ├── manifest.json - PWA manifest
│   ├── quotes.json   - Motivational quotes collection
│   ├── sitemap.xml   - Site map for SEO
│   ├── sounds/       - Notification sounds
│   ├── sw.js         - Service worker
│   └── videos/       - Changelog and feature videos
├── tests/             - Integration tests
│   ├── campaigns/    - Campaign tests
│   ├── email/        - Email system tests
│   └── scheduler/    - Scheduler tests
├── ui/                - User interface
│   ├── blog/         - Blog templates
│   ├── components/   - Reusable templates
│   │   ├── demo-grid.html    - Demo grid for homepage
│   │   ├── footer.html       - Site footer
│   │   ├── goal.html         - Goal cards
│   │   ├── habit-modal.html  - Habit creation modal
│   │   ├── monthly-grid.html - Monthly habit grid
│   │   ├── subscription-form.html - Email subscription
│   │   ├── sum-line-graph.html - Statistics visualization
│   │   └── yearly-grid.html  - Yearly habit view
│   ├── courses/      - Course content pages
│   ├── email/        - Email templates
│   │   ├── base.html           - Base email template
│   │   ├── *.html              - Html versions
│   │   └── *.txt               - Plain text versions
│   ├── habits/       - Habit-type views
│   │   ├── binary.html  - Binary habit view
│   │   ├── choice.html  - Option-select view
│   │   ├── numeric.html - Numeric habit view
│   │   └── set-rep.html - Set-rep tracking view
│   ├── about.html    - About page
│   ├── admin.html    - Admin dashboard
│   ├── changelog.html - Version history
│   ├── forgot.html   - Forgot password page
│   ├── goals.html    - Goals dashboard
│   ├── guest-home.html - Homepage for guests
│   ├── home.html     - Main dashboard
│   ├── login.html    - Login page
│   ├── privacy.html  - Privacy policy
│   ├── register.html - Registration page
│   ├── reset.html    - Password reset page
│   ├── roadmap.html  - Product roadmap
│   ├── settings.html - User settings
│   ├── terms.html    - Terms of service
│   └── unsubscribe.html - Email unsubscribe page
├── bugs.md           - Known issues tracking
├── ideas.md          - Future feature ideas
├── main.go           - Application entry point
├── openapi.yaml      - API documentation
└── LICENSE.txt       - AGPL-3.0 license
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
- **robfig/cron**: Scheduled email notifications

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
