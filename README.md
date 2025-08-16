# Knock FM - Discord Music Bot

A Discord music bot built with Go, featuring a modular architecture with separate services for bot functionality, background workers, and HTTP API.

## Project Structure

```
knock-fm/
├── cmd/                    # Application entry points
│   ├── bot/               # Discord bot main application
│   ├── worker/            # Background worker processes
│   └── api/               # HTTP API server
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models and business logic
│   ├── repository/        # Data access layer
│   │   ├── postgres/      # PostgreSQL repository implementations
│   │   └── redis/         # Redis repository implementations
│   ├── service/           # Business logic services
│   │   ├── bot/           # Bot-specific services
│   │   ├── worker/        # Worker-specific services
│   │   └── api/           # API-specific services
│   ├── http/              # HTTP handlers and middleware
│   └── pkg/               # Internal packages
│       └── logger/        # Logging utilities
├── scripts/                # Build and deployment scripts
├── migrations/             # Database migration files
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Getting Started

1. Ensure you have Go 1.21+ installed
2. Clone the repository
3. Run `go mod tidy` to download dependencies
4. Build and run the desired component:
   - Bot: `go run cmd/bot/main.go`
   - Worker: `go run cmd/worker/main.go`
   - API: `go run cmd/api/main.go`

## Development

This project follows Go project layout conventions and uses a clean architecture approach with clear separation of concerns between different layers.

## License

[Add your license here]
