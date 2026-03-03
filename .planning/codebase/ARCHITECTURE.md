# Architecture

**Analysis Date:** 2026-03-02

## Pattern Overview

**Overall:** Multi-service microservices architecture with clean hexagonal layering. The system decomposes into independent services (API, Bot, Worker) that share domain models and repositories but operate independently.

**Key Characteristics:**
- Clear separation between domain logic, repositories, services, and HTTP handlers
- Dependency injection at startup (no IoC containers)
- Interfaces-based abstraction for repositories and external dependencies
- Graceful shutdown patterns with context cancellation across all services
- Shared infrastructure: PostgreSQL for persistence, Redis for job queuing

## Layers

**Domain Layer:**
- Purpose: Core business models and repository interfaces (contracts)
- Location: `internal/domain/`
- Contains: Domain structs (`Knok`, `Server`, `Platform`), repository interfaces, constants
- Depends on: External libraries only (uuid, time)
- Used by: All other layers (services, repositories, handlers)

**Repository Layer:**
- Purpose: Data access abstraction for PostgreSQL and Redis
- Location: `internal/repository/postgres/`, `internal/repository/redis/`
- Contains: Concrete implementations of domain repository interfaces
- Depends on: Domain layer, standard database/sql, redis client
- Used by: Services and handlers

**Service Layer:**
- Purpose: Business orchestration and external integrations
- Location: `internal/service/api/`, `internal/service/bot/`, `internal/service/worker/`, `internal/service/platforms/`
- Contains: Three main services (API, Bot, Worker) plus platform loader
- Depends on: Domain layer, repository layer, config, logging, external SDKs (Discord)
- Used by: Command-line entrypoints

**HTTP Handler Layer:**
- Purpose: Request/response handling and routing
- Location: `internal/http/handlers/`, `internal/http/middleware/`
- Contains: HTTP handlers for knoks, servers, stats, admin platforms; middleware for auth and CORS
- Depends on: Domain layer, repository layer
- Used by: HTTP router

**HTTP Router:**
- Purpose: Endpoint registration and request dispatch
- Location: `internal/http/router.go`
- Contains: All route definitions with middleware chains
- Depends on: Handlers, middleware
- Used by: API service

**Configuration:**
- Purpose: Environment-based configuration loading
- Location: `internal/config/config.go`
- Contains: Config struct with validation methods per service type
- Depends on: Standard library (os, flag)
- Used by: All service entrypoints

**Utilities/Packages:**
- Purpose: Cross-cutting concerns and utilities
- Location: `internal/pkg/logger/`, `internal/pkg/urldetector/`
- Contains: Structured logging (slog wrapper), URL platform detection
- Depends on: Standard library
- Used by: All layers

## Data Flow

**New Knok (Discord Bot to Database):**

1. User posts a music link in Discord channel
2. Bot service (`internal/service/bot/service.go`) registers `onMessageCreate` handler
3. Handler detects URLs using `urldetector.Detector` (checks against loaded platforms)
4. Valid knok enqueued to Redis queue via `QueueRepository.Enqueue()` (job type: `JobTypeExtractMetadata`)
5. Worker service (`internal/service/worker/service.go`) polls Redis queue via `QueueRepository.Dequeue()`
6. Worker extracts metadata (title, artwork, etc.) using oEmbed extractors
7. Worker creates knok record via `KnokRepository.Create()` with `ExtractionStatusComplete`
8. Worker enqueues completion notification job
9. Bot notifies user in Discord channel about successful processing

**Knok Retrieval (Web UI to API):**

1. Frontend (`web/src/`) calls API endpoint via `ApiClient` in `web/src/api/client.ts`
2. API service (`internal/service/api/server.go`) HTTP router dispatches to `KnoksHandler`
3. Handler calls `KnokRepository.GetRecent()` with cursor pagination
4. PostgreSQL returns paginated results (25 per page)
5. Handler transforms domain knoks to DTOs and returns JSON
6. Frontend renders knok timeline with infinite scroll

**Platform Management (Admin):**

1. Admin logs in via `JanitorLogin` page with API key
2. Frontend sends key to `AdminContext.login()` which validates against `/api/v1/admin/platforms`
3. Auth middleware (`internal/http/middleware/auth.go`) validates Bearer token
4. Protected endpoints allow CRUD operations on platforms
5. Platform changes trigger cache refresh via `PlatformLoader.Refresh()`
6. Bot service reloads platform list for next URL detections

**State Management:**

- **Database State**: PostgreSQL holds knoks, servers, platforms (authoritative)
- **Cache State**: Redis holds job queue only (temporary, can be rebuilt)
- **In-Memory State**: Platform loader keeps platforms in memory with priority ordering for bot detection
- **API Key State**: Stored in frontend localStorage after validation

## Key Abstractions

**Domain Knok:**
- Purpose: Represents a music track/mix shared in Discord
- Examples: `internal/domain/knok.go`
- Pattern: Value object with computed validation (`IsValidPlatform()`)

**Repository Interfaces:**
- Purpose: Abstract data access across implementations
- Examples: `KnokRepository`, `ServerRepository`, `QueueRepository` in `internal/domain/repository.go`
- Pattern: Interface segregation (each repo has single responsibility)

**PlatformLoader:**
- Purpose: In-memory cache of platform configurations with refresh capability
- Examples: `internal/service/platforms/loader.go`
- Pattern: Singleton cache loaded at startup, refreshable without restart

**Services (API, Bot, Worker):**
- Purpose: Top-level orchestrators that own start/stop lifecycle
- Examples: `internal/service/api/server.go`, `internal/service/bot/service.go`, `internal/service/worker/service.go`
- Pattern: Each service is independent and gracefully stoppable

**URL Detector:**
- Purpose: Maps URLs to platforms and validates platform support
- Examples: `internal/pkg/urldetector/detector.go`
- Pattern: Uses platform loader to match URLs against configured platforms

## Entry Points

**API Service:**
- Location: `cmd/api/main.go`
- Triggers: Docker container startup via `api` binary
- Responsibilities: Load config, connect to DB/Redis, initialize repositories, start HTTP server

**Bot Service:**
- Location: `cmd/bot/main.go`
- Triggers: Docker container startup via `bot` binary
- Responsibilities: Load config, validate Discord token, initialize repositories, connect to Discord, register handlers

**Worker Service:**
- Location: `cmd/worker/main.go`
- Triggers: Docker container startup via `worker` binary
- Responsibilities: Load config, connect to DB/Redis, poll job queue, process extraction jobs

**Seeder:**
- Location: `cmd/seeder/main.go`
- Triggers: Manual invocation for database initialization
- Responsibilities: Create default platforms, seed test data

**Database Utility:**
- Location: `cmd/dbutil/main.go`
- Triggers: Manual invocation for migrations/cleanup
- Responsibilities: Run migrations, manage schema

**Web Frontend:**
- Location: `web/src/main.tsx`
- Triggers: Browser page load
- Responsibilities: Initialize React app, set up query client, render root component

## Error Handling

**Strategy:** Explicit error propagation with context-aware logging

**Patterns:**
- Go services use explicit `error` returns; caller decides if it's fatal
- HTTP handlers convert domain errors to appropriate HTTP status codes (400, 401, 404, 500)
- Frontend API client throws `ApiError` with status code for handler decision
- All services log errors at initialization and shut down gracefully on critical failures
- Context cancellation (`context.WithCancel`) used to signal shutdown across goroutines

## Cross-Cutting Concerns

**Logging:**
- Implementation: `internal/pkg/logger/logger.go` wraps `log/slog` standard library
- Usage: All layers log at appropriate levels (Debug for details, Info for progress, Error for failures)
- Configuration: `LOG_LEVEL` env var controls verbosity per service

**Validation:**
- Domain models validate themselves (e.g., `Knok.IsValidPlatform()`)
- Handler layer validates input (e.g., cursor parsing, limit boundaries)
- Repository layer sanitizes queries (e.g., `KnokRepository.sanitizeSearchQuery()`)

**Authentication:**
- API key-based for admin endpoints
- Middleware checks `Authorization: Bearer <key>` header
- Implementation: `internal/http/middleware/auth.go`
- Frontend stores key in localStorage after validation

---

*Architecture analysis: 2026-03-02*
