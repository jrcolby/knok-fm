# Technology Stack

**Analysis Date:** 2026-03-02

## Languages

**Primary:**
- Go 1.23.0 (toolchain 1.24.6) - Backend services (API, bot, worker, seeder, dbutil)
- TypeScript 5.8.3 - Frontend web application
- JavaScript/JSX - React components and build tooling

**Secondary:**
- SQL - PostgreSQL database queries
- Shell/Bash - Makefile commands and deployment scripts

## Runtime

**Environment:**
- Go 1.23.0+ for backend services
- Node.js 20 (Alpine) for frontend build
- Alpine Linux 3.x for production Docker images

**Package Manager:**
- Go modules (go.mod/go.sum) - Backend dependencies
- pnpm - Frontend package management
  - Lockfile: `pnpm-lock.yaml` (committed)

## Frameworks

**Backend:**
- No web framework used - raw net/http package with custom routing
- Discord Bot: `github.com/bwmarrin/discordgo` v0.29.0 - Discord bot protocol
- Browser Automation: `github.com/go-rod/rod` v0.116.2 - Headless browser control (worker service)

**Frontend:**
- React 19.1.1 - UI framework
- React Router 7.8.2 - Client-side routing
- Vite 7.1.2 - Build tool and dev server
- TailwindCSS 4.1.12 - Utility-first CSS framework
- @tailwindcss/vite 4.1.12 - Vite integration for Tailwind

**Data Fetching:**
- @tanstack/react-query 5.85.6 - Server state management and caching
- Fetch API (native) - HTTP requests from browser

**Testing:**
- No detected test framework (no test runner config found)

**Build/Dev:**
- TypeScript - Type checking
- ESLint 9.33.0 - Linting with TypeScript support
- @vitejs/plugin-react 5.0.0 - React support in Vite

## Key Dependencies

**Backend - Critical:**
- `github.com/lib/pq` v1.10.9 - PostgreSQL driver for Go
- `github.com/redis/go-redis/v9` v9.12.0 - Redis client
- `github.com/bwmarrin/discordgo` v0.29.0 - Discord API integration
- `github.com/google/uuid` v1.6.0 - UUID generation
- `github.com/go-rod/rod` v0.116.2 - Browser automation for metadata extraction

**Backend - Indirect:**
- `github.com/gorilla/mux` v1.8.1 - HTTP router (used by discordgo)
- `github.com/gorilla/websocket` v1.4.2 - WebSocket support (Discord bot)
- `golang.org/x/net` v0.43.0 - Low-level networking
- `golang.org/x/crypto` v0.41.0 - Cryptographic primitives
- `golang.org/x/sys` v0.35.0 - System call wrappers

**Frontend - UI Components:**
- `class-variance-authority` 0.7.1 - CSS class composition utility
- `clsx` 2.1.1 - Conditional className builder
- `lucide-react` 0.542.0 - Icon library
- `tailwind-merge` 3.3.1 - TailwindCSS class utilities

## Configuration

**Environment:**
- `.env` - Development configuration (git-ignored)
- `.env.example` - Documentation of required variables
- `.env.prod.example` - Production configuration template
- `.env.prod.local` - Local production-like config (git-ignored)

**Required Environment Variables:**
- `DATABASE_URL` - PostgreSQL connection string (required for all services)
- `REDIS_URL` - Redis connection string (required for all services)
- `DISCORD_TOKEN` - Discord bot token (required for bot/seeder services)
- `PORT` - Server port (default: 8080, API only)
- `LOG_LEVEL` - Logging verbosity (default: info; options: debug, info, warn, error)
- `STATIC_DIR` - Static files directory (default: ./web/build, API only)
- `ADMIN_API_KEY` - API key for admin endpoints (optional, empty disables auth in dev)
- `DISCORD_ALLOWED_GUILDS` - Comma-separated guild IDs to restrict bot (optional)
- `DISCORD_ALLOWED_CHANNELS` - Comma-separated channel IDs to restrict bot (optional)
- `UNKNOWN_PLATFORM_MODE` - "permissive" or "strict" for unknown platform handling (default: permissive)

**Build:**
- `vite.config.ts` - Vite build configuration (`/Users/disso/Code/knok-fm/web/vite.config.ts`)
- `tsconfig.json`, `tsconfig.app.json`, `tsconfig.node.json` - TypeScript configuration
- `eslint.config.js` - ESLint configuration
- `components.json` - Component library metadata
- `.prettierrc` - Code formatting (not found, uses ESLint defaults)

**Go Module:**
- `go.mod` - Backend module definition
- `go.sum` - Dependency lock file

## Platform Requirements

**Development:**
- macOS/Linux/Windows with Go 1.23+
- Node.js 20+ with pnpm
- PostgreSQL 15+ (via Docker: postgres:15-alpine)
- Redis 7+ (via Docker: redis:7-alpine)
- Docker & Docker Compose for service orchestration

**Production:**
- Alpine Linux containers
- PostgreSQL 15+
- Redis 7+
- Caddy 2.x reverse proxy (auto-SSL)
- 4 services: api, bot, worker, web
- Static assets served by Nginx (web container)

**Special Requirements:**
- Chromium/Chrome required in worker container for browser automation (via `go-rod`)
- Discord API access (requires bot token)
- Music platform APIs (Spotify, YouTube, SoundCloud, etc.) - accessed via URL detection, not direct SDK

## Docker Images

**Build Pipeline:**
- All services use multi-stage Alpine builds
- Builder stage: golang:1.23-alpine or node:20-alpine
- Runtime stage: alpine:latest (except web: nginx:alpine)
- All services run as non-root user (appuser, UID 1000)

**Services:**
- `Dockerfile.api` - REST API server, runs migrations
- `Dockerfile.bot` - Discord bot service
- `Dockerfile.worker` - Background task processor (includes Chromium)
- `Dockerfile.web` - React frontend served by Nginx
- `Dockerfile.seeder` - One-off database population tool

---

*Stack analysis: 2026-03-02*
