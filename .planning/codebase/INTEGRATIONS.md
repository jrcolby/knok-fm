# External Integrations

**Analysis Date:** 2026-03-02

## APIs & External Services

**Discord:**
- Discord Bot API - Music sharing bot for Discord servers
  - SDK/Client: `github.com/bwmarrin/discordgo` v0.29.0
  - Auth: `DISCORD_TOKEN` (bot token, required for bot/seeder services)
  - Implementation: `internal/service/bot/` - Handles message listening, command processing, slash commands
  - Integration Points:
    - Message create events for knok detection
    - Slash command handling (`/knok` commands)
    - Guild/server management for allowed guilds/channels
    - Message deletion/editing

**Music Platforms (URL-based detection):**
- Multiple platforms detected via URL pattern matching
- Implementation: `internal/service/platforms/` with platform loader
- Database: Platforms stored in `platforms` table with regex patterns and extraction rules
- No direct SDK integration - URLs are validated and metadata extracted via browser automation
- Currently supported: Spotify, YouTube, SoundCloud, Apple Music, Tidal, etc. (via platform loader)
- Configuration: Dynamic platform loading from database

**Metadata Extraction:**
- Browser Automation: `github.com/go-rod/rod` v0.116.2 - Headless Chromium
  - Used by worker service (`cmd/worker/`) to extract metadata from music platform URLs
  - Environment: Chromium required in container (`/usr/bin/chromium-browser`)
  - Purpose: Extract title, artist, cover art from platform URLs

## Data Storage

**Databases:**

**PostgreSQL (Primary):**
- Type/Provider: PostgreSQL 15+
- Connection: `DATABASE_URL` environment variable
- Client: `database/sql` with `github.com/lib/pq` v1.10.9 driver
- Repositories: `internal/repository/postgres/`
  - `knok.go` - Music link (knok) storage and retrieval
  - `platform.go` - Platform definitions and regex patterns
  - `server.go` - Discord guild/server configuration
  - `migrations.go` - Database schema management
- Key Tables:
  - `knoks` - Music links with titles, platforms, Discord metadata
  - `platforms` - Music platform definitions with extraction rules
  - `servers` - Discord guild settings and per-server configuration
- Migration Strategy: Embedded migrations, run on service startup (`postgres.RunMigrations()`)
- Location: `/Users/disso/Code/knok-fm/migrations/` (empty - migrations embedded in code)

**Redis (Cache & Queues):**
- Type/Provider: Redis 7+
- Connection: `REDIS_URL` environment variable
- Client: `github.com/redis/go-redis/v9` v9.12.0
- Repository: `internal/repository/redis/`
  - `queue.go` - Job queue for asynchronous tasks
  - `client.go` - Redis connection management and health checks
- Use Cases:
  - Background job queue for metadata extraction
  - Potential caching of platform patterns

**File Storage:**
- Local filesystem only - static web assets served from `./web/build` (or `STATIC_DIR`)
- No cloud storage (S3, GCS) integration detected
- Music platform logos/metadata cached in database, not stored separately

**Caching:**
- Redis (see above) - In-memory cache and job queue
- Frontend: React Query (`@tanstack/react-query` v5.85.6)
  - Stale time: 5 minutes for search results, 0 for random knoks
  - Cache invalidation: Manual query refetch

## Authentication & Identity

**Auth Provider:**
- Custom: No third-party identity provider (OAuth, Auth0, etc.)

**Implementation:**
- Discord Bot Token - Required for bot service to authenticate with Discord API
  - Stored in `DISCORD_TOKEN` environment variable
  - Passed to `discordgo.New()` for session creation

**Admin Authentication:**
- Bearer token authentication via `ADMIN_API_KEY` environment variable
  - Implementation: `internal/http/middleware/auth.go`
  - Used for admin endpoints: `/api/v1/admin/knoks/*`
  - Header: `Authorization: Bearer {ADMIN_API_KEY}`
  - Required for: Delete knok, Refresh knok metadata
  - Optional in development (empty ADMIN_API_KEY disables auth)

**Discord Permissions:**
- Guild/Channel Restrictions:
  - `DISCORD_ALLOWED_GUILDS` - Whitelist specific Discord servers (optional)
  - `DISCORD_ALLOWED_CHANNELS` - Whitelist specific channels (optional)
  - Default: Allow all if empty
  - Per-guild overrides possible via `servers` table settings

## Monitoring & Observability

**Error Tracking:**
- None detected - No Sentry, Rollbar, or error tracking service integrated

**Logs:**
- Custom logger: `internal/pkg/logger/` using Go's `log/slog` package
- Implementation: Structured logging with level control
- Levels: debug, info, warn, error (configurable via `LOG_LEVEL`)
- Output: stdout (captured by Docker logging drivers)
- Docker Logging: json-file driver with max-size 10m, max-file 3 (rotating)
- Frontend: console.log (development) - no centralized logging

**Health Checks:**
- PostgreSQL: `db.Ping()` on startup
- Redis: `redis.HealthCheck()` custom function
- API: `/health` endpoint returns `{ "status": "ok" }`
- Docker Compose: Health checks on postgres and redis services (10s interval, 5s timeout, 5 retries)

## CI/CD & Deployment

**Hosting:**
- Docker-based containerized deployment
- Orchestration: Docker Compose
- Reverse Proxy: Caddy 2.x (auto-SSL via Let's Encrypt)

**Deployment Target:**
- Containerized on Linux server
- Compose services: postgres, redis, api, bot, worker, web, seeder, caddy

**CI Pipeline:**
- None detected - No GitHub Actions, GitLab CI, or Jenkins integration

**Build Process:**
- Makefile: `internal/Makefile` - Development commands only
  - `make dev-services` - Start Docker Compose dev stack
  - `make dev-api/bot/worker` - Run services locally with env vars
  - `make seed-channel` - Database seeding utility
- Docker multi-stage builds for all services
- No automated tests or linting in build pipeline

## Environment Configuration

**Required Environment Variables:**
- `DATABASE_URL` - PostgreSQL connection (e.g., `postgresql://user:pass@host:5432/knockfm?sslmode=disable`)
- `REDIS_URL` - Redis connection (e.g., `redis://localhost:6379`)
- `DISCORD_TOKEN` - Discord bot token (required for bot/seeder)

**Optional Environment Variables:**
- `PORT` - API server port (default: 8080)
- `LOG_LEVEL` - Logging verbosity (default: info)
- `STATIC_DIR` - Static assets path (default: ./web/build)
- `ADMIN_API_KEY` - Admin API authentication key
- `DISCORD_ALLOWED_GUILDS` - CSV list of guild IDs
- `DISCORD_ALLOWED_CHANNELS` - CSV list of channel IDs
- `UNKNOWN_PLATFORM_MODE` - "permissive" or "strict" (default: permissive)

**Secrets Location:**
- `.env` files (development, git-ignored)
- `.env.prod.local` (production local override, git-ignored)
- Docker Compose `env_file: .env.prod` reference in `docker-compose.prod.yml`
- Environment variables passed at container runtime

**Configuration Files:**
- `Caddyfile` - Reverse proxy and HTTPS configuration
- `docker-compose.prod.yml` - Production service orchestration
- Application configuration: Loaded via `internal/config/config.go`

## Webhooks & Callbacks

**Incoming:**
- No webhook receivers detected
- Bot responds to Discord events (message create, slash commands) but not via webhooks

**Outgoing:**
- No outgoing webhooks detected
- Integration is one-directional (read from Discord, write to PostgreSQL)

## Data Flow

**Message Processing Flow:**
1. Discord user posts/edits message in guild
2. Discord bot (via discordgo) receives message create event
3. Bot handlers (`internal/service/bot/handlers.go`) parse message for URLs
4. URL detector (`internal/service/platforms/`) identifies music platform
5. Platform validation: Checks against platform patterns in database
6. Job queued to Redis for async processing
7. Worker service (`cmd/worker/`) fetches job from queue
8. Worker uses Rod to extract metadata from music platform URL
9. Metadata stored in PostgreSQL `knoks` table
10. Frontend queries `/api/v1/knoks` to display in timeline

**Search Flow:**
1. Frontend user types search query
2. React Query triggers API call to `/api/v1/knoks/search?q={query}`
3. API searches `knoks` table using PostgreSQL full-text search
4. Results returned with cursor pagination
5. Frontend caches results for 5 minutes

**Admin Operations:**
1. Admin provides API key and knok ID
2. Frontend calls `/api/v1/admin/knoks/{id}` with Bearer token
3. API middleware validates `ADMIN_API_KEY`
4. Operation performed (delete, refresh metadata)
5. Redis queue updated if refresh triggers worker job

## External API Rate Limits & Quotas

**Discord API:**
- Standard Discord bot rate limits apply
- No explicit rate limiting in code

**Music Platforms (via Rod):**
- Browser automation may trigger platform rate limiting
- No proxy/delay mechanism detected
- Risk: Platform IP bans if scraping too aggressively

---

*Integration audit: 2026-03-02*
