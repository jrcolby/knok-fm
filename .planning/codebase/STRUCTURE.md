# Codebase Structure

**Analysis Date:** 2026-03-02

## Directory Layout

```
knok-fm/
├── cmd/                          # Service entrypoints
│   ├── api/                      # API service
│   │   └── main.go              # Start HTTP API server
│   ├── bot/                      # Discord bot service
│   │   └── main.go              # Start Discord bot
│   ├── worker/                   # Background job processor
│   │   └── main.go              # Start job worker
│   ├── seeder/                   # Database initialization
│   │   └── main.go              # Seed default platforms/data
│   └── dbutil/                   # Database utilities
│       └── main.go              # Run migrations/cleanup
├── internal/                      # Go private packages
│   ├── domain/                   # Domain models and repository interfaces
│   │   ├── knok.go              # Knok struct and validation
│   │   ├── server.go            # Server (Discord guild) struct
│   │   ├── platform.go          # Platform (music service) struct
│   │   ├── repository.go        # Repository interfaces (KnokRepository, etc)
│   │   └── migrations.go        # Migration definitions
│   ├── config/                   # Configuration loading
│   │   └── config.go            # Config struct and env parsing
│   ├── repository/               # Data access layer implementations
│   │   ├── postgres/            # PostgreSQL implementations
│   │   │   ├── knok.go          # KnokRepository implementation
│   │   │   ├── server.go        # ServerRepository implementation
│   │   │   ├── platform.go      # PlatformRepository implementation
│   │   │   └── migrations.go    # Migration runner
│   │   └── redis/               # Redis implementations
│   │       ├── queue.go         # QueueRepository implementation
│   │       └── client.go        # Redis client wrapper
│   ├── service/                  # Business logic services
│   │   ├── api/                 # HTTP API service
│   │   │   └── server.go        # APIService setup and lifecycle
│   │   ├── bot/                 # Discord bot service
│   │   │   ├── service.go       # BotService setup and lifecycle
│   │   │   ├── handlers.go      # Discord event handlers
│   │   │   └── commands.go      # Bot command implementations
│   │   ├── worker/              # Background job processing
│   │   │   ├── service.go       # WorkerService setup and lifecycle
│   │   │   ├── processor.go     # Job dispatch logic
│   │   │   ├── extractor_oembed.go  # oEmbed metadata extraction
│   │   │   ├── oembed_registry.go   # oEmbed provider registry
│   │   │   ├── oembed_providers.json # oEmbed provider definitions
│   │   │   └── *_test.go        # Extraction tests
│   │   └── platforms/           # Platform cache and loader
│   │       └── loader.go        # PlatformLoader implementation
│   ├── http/                     # HTTP request handling
│   │   ├── router.go            # Route registration
│   │   ├── handlers/            # Request handlers
│   │   │   ├── knoks.go         # Knok CRUD and search handlers
│   │   │   ├── servers.go       # Server CRUD handlers
│   │   │   ├── stats.go         # Statistics handler
│   │   │   ├── health.go        # Health check handler
│   │   │   └── admin_platforms.go # Admin platform management
│   │   └── middleware/          # HTTP middleware
│   │       ├── auth.go          # API key authentication
│   │       └── cors.go          # CORS header injection
│   └── pkg/                      # Utility packages
│       ├── logger/              # Logging utilities
│       │   └── logger.go        # slog-based logger wrapper
│       └── urldetector/         # URL platform detection
│           ├── detector.go      # URL parsing and platform matching
│           ├── normalize.go     # URL normalization
│           └── detector_test.go # Detection tests
├── web/                          # React frontend
│   ├── src/
│   │   ├── main.tsx             # React app entry point
│   │   ├── App.tsx              # Root router and layout
│   │   ├── index.css            # Global styles
│   │   ├── api/                 # API client and types
│   │   │   ├── client.ts        # API client implementation
│   │   │   └── types.ts         # TypeScript types for API responses
│   │   ├── pages/               # Page components (routes)
│   │   │   ├── KnokTimeline.tsx # Main knok timeline view
│   │   │   ├── JanitorLogin.tsx # Admin login page
│   │   │   ├── SearchBar.tsx    # Search interface
│   │   │   └── SingleKnok.tsx   # Individual knok view
│   │   ├── components/          # Reusable UI components
│   │   │   ├── Header.tsx       # Navigation header
│   │   │   ├── KnokCard.tsx     # Knok display card
│   │   │   ├── SearchComponent.tsx # Search UI
│   │   │   ├── DeleteKnokModal.tsx # Delete confirmation modal
│   │   │   ├── RefreshKnokModal.tsx # Refresh metadata modal
│   │   │   ├── Modal.tsx        # Base modal component
│   │   │   └── icons/           # SVG icon components
│   │   │       ├── KnokFmLogo.tsx
│   │   │       ├── KnokSpiral.tsx
│   │   │       ├── KnokStar.tsx
│   │   │       └── index.ts     # Icon exports
│   │   ├── contexts/            # React context providers
│   │   │   └── AdminContext.tsx # Admin auth state management
│   │   ├── hooks/               # Custom React hooks
│   │   │   ├── useKnokData.ts   # Knok data fetching
│   │   │   ├── useInfiniteKnoks.ts # Infinite scroll logic
│   │   │   └── useIntersectionObserver.ts # Scroll trigger detection
│   │   ├── lib/                 # Utility functions and config
│   │   ├── utils/               # Helper utilities
│   │   │   └── logoFallback.ts  # Logo generation fallback
│   │   └── vite-env.d.ts        # Vite type definitions
│   ├── package.json             # Frontend dependencies
│   ├── tsconfig.json            # TypeScript configuration
│   ├── vite.config.ts           # Vite build config
│   ├── eslint.config.js         # ESLint rules
│   ├── tailwind.config.ts       # Tailwind CSS config
│   ├── public/                  # Static assets
│   ├── dist/                    # Built output (gitignored)
│   └── node_modules/            # Dependencies (gitignored)
├── migrations/                   # Database schema (empty - migrations in code)
├── bin/                          # Utility scripts
├── scripts/                      # Maintenance scripts
├── go.mod                        # Go module definition
├── go.sum                        # Go dependency lock
├── Caddyfile                     # Reverse proxy configuration
├── docker-compose.dev.yml        # Local development setup
├── docker-compose.prod.yml       # Production container definitions
├── DEPLOYMENT.md                 # Deployment documentation
├── .env.example                  # Example environment variables
├── .env.prod.example             # Example production environment
├── .gitignore                    # Git ignore rules
├── .planning/codebase/           # GSD documentation (generated)
└── .claude/                      # Claude Code preferences
```

## Directory Purposes

**cmd/:**
- Purpose: Service entry points, one binary per process type
- Contains: `main()` functions that orchestrate initialization and service startup
- Key files: `cmd/*/main.go` - Each validates config, connects to DB/Redis, initializes repositories, starts service

**internal/domain/:**
- Purpose: Domain-driven design layer with core business models and contracts
- Contains: Knok, Server, Platform structs; repository interfaces; job type constants
- Key files:
  - `knok.go`: Core music track model (~46 lines)
  - `repository.go`: All repository interfaces (~123 lines)
  - `platform.go`: Music service configuration model

**internal/repository/**
- Purpose: Data persistence implementations (PostgreSQL and Redis)
- Key patterns:
  - Postgres: Raw SQL queries with manual scanning (no ORM)
  - Redis: JSON serialization for queue jobs
  - All implement domain repository interfaces

**internal/service/**
- Purpose: Service-level business logic and orchestration
- Key services:
  - **api/**: HTTP server setup and routing orchestration
  - **bot/**: Discord event handling and message processing
  - **worker/**: Async job processing (metadata extraction)
  - **platforms/**: Platform configuration caching and refresh

**internal/http/**
- Purpose: HTTP request/response handling
- Pattern: Router dispatches to handlers; handlers use repositories directly
- Handlers convert domain models to DTOs for API responses
- Middleware applied at router level (CORS on all, Auth on admin endpoints)

**internal/pkg/**
- Purpose: Cross-cutting utilities (logging, URL detection)
- Pattern: Small, focused packages with no dependencies on other internal packages
- Logger: Wraps slog for structured logging across all services
- URL Detector: Matches URLs to platforms using priority-ordered list

**web/src/**
- Purpose: React frontend for viewing knoks and admin management
- Key patterns:
  - Pages: Route components (KnokTimeline, JanitorLogin)
  - Components: Reusable UI pieces (cards, modals, header)
  - Hooks: React Query for API calls + custom scroll/intersection hooks
  - Context: AdminContext for auth state across app
  - API Client: Fetch-based HTTP client with error handling

## Key File Locations

**Entry Points:**
- `cmd/api/main.go`: Start API server (port 8080 by default)
- `cmd/bot/main.go`: Start Discord bot
- `cmd/worker/main.go`: Start background job processor
- `web/src/main.tsx`: React app initialization

**Configuration:**
- `internal/config/config.go`: All config loading from environment
- `.env`: Local environment variables (not committed)
- `go.mod`: Go dependencies and version

**Core Logic:**
- `internal/http/router.go`: All API endpoint definitions
- `internal/service/bot/handlers.go`: Discord message and interaction handlers
- `internal/service/worker/processor.go`: Job dispatch and error handling
- `internal/repository/postgres/knok.go`: Knok query implementations (search, pagination, etc)

**Testing:**
- `internal/pkg/urldetector/detector_test.go`: URL detection unit tests
- `internal/service/worker/extractor_oembed_test.go`: oEmbed extraction tests
- `internal/service/worker/oembed_registry_test.go`: Provider registry tests
- `web/src/api/client.ts`: Fetch-based client (no test file - integration tested in pages)

## Naming Conventions

**Files:**
- Go files: `snake_case.go` (e.g., `admin_platforms.go`, `url_detector.go`)
- TypeScript files: `PascalCase.tsx` for components, `camelCase.ts` for utilities
- Test files: `*_test.go` (Go) or `.test.ts` (TypeScript)

**Directories:**
- Lowercase plural for packages: `repositories/`, `handlers/`, `services/`, `components/`
- Feature-grouped: `internal/service/bot/`, `internal/repository/postgres/`

**Functions/Methods:**
- Go: `PascalCase` for exported, `camelCase` for unexported (e.g., `GetRecent`, `sanitizeSearchQuery`)
- TypeScript: `camelCase` for all (e.g., `getKnoks`, `loginAdmin`)

**Types/Structs:**
- Go: `PascalCase` with -er suffix for interfaces (e.g., `KnokRepository`, `PlatformLoader`)
- TypeScript: `PascalCase` for types (e.g., `KnokDto`, `AdminContextType`)

**Constants:**
- Go: `PascalCase` or `SCREAMING_SNAKE_CASE` for package-level (e.g., `JobTypeExtractMetadata`, `DefaultPaginationLimit`)
- TypeScript: `camelCase` for string/config constants

## Where to Add New Code

**New Feature (e.g., user ratings):**
- Primary code: `internal/domain/rating.go` (model) + `internal/service/*/handlers` (if API)
- Tests: `internal/repository/postgres/rating_test.go`
- Database: Add struct fields and run migrations via `internal/repository/postgres/migrations.go`

**New Handler/Endpoint:**
- Implementation: `internal/http/handlers/{entity}.go` (follow pattern from `knoks.go`)
- Registration: Add route in `internal/http/router.go` SetupRoutes()
- Tests: Create handler method on struct following error handling pattern

**New Repository Method:**
- Interface: Add method signature to `KnokRepository` interface in `internal/domain/repository.go`
- Implementation: Add to `internal/repository/postgres/knok.go` following SQL patterns (use parameterized queries)
- Usage: Inject through service constructors

**New Discord Bot Command:**
- Handler: Add to `internal/service/bot/handlers.go` following discordgo event pattern
- Registration: Register with `session.AddHandler()` in `service.go`
- Testing: Add test case in worker extraction tests

**New React Component:**
- Implementation: `web/src/components/{Name}.tsx` following existing card/modal patterns
- Styling: Use Tailwind classes (configured in `tailwind.config.ts`)
- Integration: Import in page component or export from index

**Shared Utility/Helper:**
- Location: `internal/pkg/{feature}/` for Go, `web/src/utils/` for TypeScript
- Pattern: No circular dependencies; utils should not depend on domain or services

## Special Directories

**migrations/:**
- Purpose: Directory exists but migrations are implemented in-code
- Generated: No (migrations are explicit SQL/Go code in `postgres/migrations.go`)
- Committed: Yes (migrations are version controlled)
- Note: To add migration, edit migration version in `internal/repository/postgres/migrations.go`

**bin/:**
- Purpose: Compiled binaries and build output
- Generated: Yes (built from cmd/ sources)
- Committed: No (binaries gitignored)

**dist/, node_modules/:**
- Purpose: Build artifacts and dependencies
- Generated: Yes
- Committed: No (both in .gitignore)

**.planning/codebase/:**
- Purpose: GSD codebase documentation (ARCHITECTURE.md, STRUCTURE.md, etc)
- Generated: Yes (created by /gsd:map-codebase)
- Committed: Yes (reference documentation)

---

*Structure analysis: 2026-03-02*
