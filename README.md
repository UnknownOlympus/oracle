# Oracle - Telegram Task Management Bot

A feature-rich Telegram bot for managing field service tasks, built with Go. Oracle provides authentication, task tracking, geolocation-based task assignment, reporting, and administrative capabilities with full internationalization support (English/Ukrainian).

## Features

- **User Authentication**: Secure email-based authentication with Telegram ID linking
- **Task Management**:
  - View active tasks assigned to you
  - Find tasks near your location (geolocation-based)
  - Add comments to tasks
  - View detailed task information with map links
- **Reporting**: Generate Excel reports for completed tasks (daily, monthly, yearly)
- **Statistics**: Track your task completion metrics over different time periods
- **Admin Panel**:
  - Broadcast messages to all users
  - Admin-specific controls and monitoring
- **Internationalization**: Full support for English and Ukrainian languages
- **Metrics & Monitoring**: Prometheus metrics integration for observability

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12+ (for user and task data)
- Redis (for caching and state management)
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))
- Access to Hermes gRPC service (for external integrations)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd oracle
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o oracle ./cmd/oracle
```

Or use the provided Makefile:
```bash
make build
```

## Configuration

Oracle is configured via environment variables. Create a `.env` file or export these variables:

### Required Configuration

```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=oracle
DB_PASSWORD=your_db_password
DB_NAME=oracle_db
DB_SSLMODE=disable

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# gRPC Configuration
GRPC_HERMES_ADDR=localhost:50051

# Logging
LOG_LEVEL=info  # debug, info, warn, error

# Metrics
METRICS_PORT=9090
```

### Optional Configuration

```bash
# Alert Manager (for notifications)
ALERTMANAGER_URL=http://localhost:9093

# Geolocation
DEFAULT_SEARCH_RADIUS_KM=15  # Default radius for nearby task search
```

## Database Schema

Oracle requires the following database tables:

### Users Table
- `id` - User ID from external system
- `telegram_id` - Telegram user ID (nullable, unique)
- `full_name` - Full employee name
- `short_name` - Short/display name
- `email` - Email address (unique)
- `phone` - Phone number
- `position` - Job position
- `is_admin` - Admin privileges flag
- `language` - Preferred language (en/uk)

### Tasks Table
- `id` - Task ID
- `type` - Task type/category
- `created_at` - Creation timestamp
- `client_name` - Customer name
- `address` - Task address
- `description` - Task description
- `latitude`, `longitude` - Geolocation coordinates
- `assigned_to` - Array of assigned user IDs
- `comments` - Task comments
- `status` - Task status

### Customers Table
- `id` - Customer ID
- `name` - Customer name
- Additional customer details

## Usage

### Running the Bot

```bash
./oracle
```

Or with Docker:
```bash
docker build -t oracle .
docker run --env-file .env oracle
```

### User Commands

- `/start` - Initialize the bot and show main menu
- `/language` - Change interface language

### Menu Options

**For All Users:**
- 🔐 Login - Authenticate with your email
- 🙍‍♂️ About me - View your profile information
- ✅ Active tasks - See tasks assigned to you
- 🗺️ Tasks near you - Find tasks based on your location
- 📈 My statistic - View your completion statistics
- 📊 Create report - Generate Excel report
- 🌐 Change Language - Switch between English/Ukrainian
- 🔓 Logout - Disconnect your account

**For Admins:**
- 👑 Admin Panel - Access administrative features
- 📣 Broadcast - Send messages to all users

## Architecture

### Project Structure

```
oracle/
├── cmd/oracle/          # Application entry point
│   └── main.go
├── internal/
│   ├── bot/             # Telegram bot logic
│   │   ├── bot.go       # Core bot setup
│   │   ├── handlers.go  # Message handlers
│   │   ├── auth_handlers.go
│   │   ├── admin_handlers.go
│   │   ├── language_handlers.go
│   │   ├── stat_handlers.go
│   │   ├── buttons.go   # Menu builders
│   │   └── state.go     # User state management
│   ├── i18n/            # Internationalization
│   │   ├── locales/
│   │   │   ├── en.json
│   │   │   └── uk.json
│   │   └── localizer.go
│   ├── repository/      # Database layer
│   │   ├── user_repo.go
│   │   └── task_repo.go
│   ├── models/          # Data models
│   ├── config/          # Configuration
│   └── metrics/         # Prometheus metrics
├── Dockerfile
├── Makefile
└── go.mod
```

### Key Components

- **Bot Layer** ([internal/bot](internal/bot)): Handles all Telegram interactions, routing, and user interface
- **Repository Layer** ([internal/repository](internal/repository)): Database operations with pgx driver
- **Localization** ([internal/i18n](internal/i18n)): Translation system with embedded JSON locale files
- **State Management**: Redis-backed user state for multi-step interactions
- **Metrics**: Prometheus instrumentation for monitoring bot performance

## Development

### Running Tests

```bash
go test ./...
```

### Code Quality

Run linters:
```bash
make lint
```

### Adding New Translations

1. Add translation keys to [internal/i18n/locales/en.json](internal/i18n/locales/en.json)
2. Add corresponding Ukrainian translations to [internal/i18n/locales/uk.json](internal/i18n/locales/uk.json)
3. Use `b.t(ctx, telegramCtx, "translation.key")` in handlers

Example:
```go
func (b *Bot) myHandler(ctx telebot.Context) error {
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    return ctx.Send(b.t(timeoutCtx, ctx, "my.translation.key"))
}
```

### Adding New Handlers

1. Create handler function in appropriate file (e.g., [auth_handlers.go](internal/bot/auth_handlers.go))
2. Add translation keys for messages
3. Register route in `registerRoutes()` method
4. Add button to menu in [buttons.go](internal/bot/buttons.go) if needed
5. Update `routeTextHandler` if adding menu button

## Deployment

### Docker Deployment

```bash
# Build image
docker build -t oracle:latest .

# Run container
docker run -d \
  --name oracle \
  --env-file .env \
  -p 9090:9090 \
  oracle:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  oracle:
    build: .
    env_file: .env
    ports:
      - "9090:9090"
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: oracle_db
      POSTGRES_USER: oracle
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

## Monitoring

Oracle exposes Prometheus metrics on the configured metrics port (default: 9090):

```
http://localhost:9090/metrics
```

### Key Metrics

- `oracle_commands_received_total` - Total commands received by type
- `oracle_messages_sent_total` - Total messages sent by type
- `oracle_db_query_duration_seconds` - Database query performance
- `oracle_new_users_total` - New user registrations
- `oracle_active_users` - Currently active users

## Security Considerations

- Telegram Bot Token should be kept secret and never committed to version control
- Database credentials should be managed securely (use secrets management in production)
- Admin privileges are controlled via the `is_admin` database field
- User authentication requires email verification against existing employee records

## Troubleshooting

### Bot not responding
- Verify `TELEGRAM_BOT_TOKEN` is correct
- Check bot is not running in multiple instances
- Verify network connectivity to Telegram API

### Database connection errors
- Confirm PostgreSQL is running and accessible
- Verify database credentials and connection string
- Check database exists and schema is initialized

### Translation not working
- Ensure locale files are embedded correctly (check `go:embed` directives)
- Verify user's language preference is set in database
- Check translation key exists in both en.json and uk.json

## Dependencies

- [telebot.v4](https://gopkg.in/telebot.v4) - Telegram Bot API framework
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [redis](https://github.com/redis/go-redis) - Redis client
- [excelize](https://github.com/xuri/excelize) - Excel file generation
- [prometheus](https://github.com/prometheus/client_golang) - Metrics collection
- [grpc](https://google.golang.org/grpc) - gRPC client for Hermes service

## Support

For issues and questions, please [open an issue](link-to-issues) on the repository.