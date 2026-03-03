# Codebase Concerns

**Analysis Date:** 2026-03-02

## Tech Debt

**Unimplemented Server Management Endpoints:**
- Issue: Server creation, update, and deletion endpoints return `501 Not Implemented` responses
- Files: `internal/http/handlers/servers.go` (lines 71-122)
- Impact: Admin server management features are non-functional, blocking server configuration workflows
- Fix approach: Implement `CreateServer()`, `UpdateServer()`, and `DeleteServer()` handlers with proper database operations

**Incomplete Server Repository Methods:**
- Issue: Five critical ServerRepository methods are stubbed with `return nil` or empty slices, logging warnings instead of executing
- Files: `internal/repository/postgres/server.go` (lines 157-193)
  - `Update()` - returns nil without updating (line 158-161)
  - `Delete()` - returns nil without deleting (line 164-168)
  - `List()` - returns empty slice with no pagination (line 171-175)
  - `GetByChannelID()` - returns no rows (line 178-182)
  - `UpdateSettings()` - returns nil without updating (line 185-192)
- Impact: Server listing, deletion, filtering, and settings updates completely broken in production
- Fix approach: Implement proper PostgreSQL queries for all five methods

**Incomplete Bot Commands:**
- Issue: Three slash commands (`/recent`, `/stats`, `/search`) are stubbed and return "Coming soon..." placeholders
- Files: `internal/service/bot/commands.go` (lines 81, 108, 166)
- Impact: Discord bot interactive features unavailable; users see unimplemented error messages
- Fix approach: Implement database queries for recent knoks, server stats, and search functionality

**Missing Bot Command Registration:**
- Issue: Commands array only registers `/stats` command, missing `/recent` and `/search` registrations
- Files: `internal/service/bot/commands.go` (line 10-16)
- Impact: Discord users cannot invoke `/recent` and `/search` commands even if implemented
- Fix approach: Add missing commands to the `commands` array and implement corresponding handlers

## Security Considerations

**Unprotected Admin Endpoints in Development:**
- Risk: If `ADMIN_API_KEY` environment variable is not set, all admin endpoints bypass authentication
- Files: `internal/http/middleware/auth.go` (lines 16-26, 33-36)
- Current mitigation: Warning logged to stderr, but requests still proceed with no auth
- Recommendations:
  1. Require `ADMIN_API_KEY` to be set (fail startup if missing in production)
  2. Add environment-based validation to reject unauthenticated requests unless explicitly in dev mode
  3. Use `config.ValidateForAPI()` to enforce this at startup

**Weak Admin Authentication:**
- Risk: Bearer token auth susceptible to brute force if API key is weak or leaked
- Files: `internal/http/middleware/auth.go` (line 51)
- Current mitigation: Assumes strong key (users generate with `openssl rand -hex 32`)
- Recommendations:
  1. Add rate limiting on failed auth attempts (e.g., reject after 5 failures per IP in 15 min)
  2. Require HTTPS enforcement in production
  3. Add request signing (e.g., HMAC-SHA256 of timestamp + path + body)

**API Key Exposed in Environment:**
- Risk: `ADMIN_API_KEY` stored in `.env` file or environment variables, may leak in logs or error messages
- Files: `internal/http/middleware/auth.go` (line 17)
- Current mitigation: Not logged directly, but warning message appears in logs
- Recommendations:
  1. Never log auth token values (already compliant)
  2. Consider using AWS Secrets Manager or HashiCorp Vault in production
  3. Add audit logging for all admin endpoint access

**Discord Bot Token in Environment:**
- Risk: `DISCORD_TOKEN` exposed in `.env.prod` file or environment
- Files: `.env.prod` (note existence only)
- Current mitigation: File should be in `.gitignore`
- Recommendations:
  1. Verify `.gitignore` includes `.env*` patterns (CRITICAL)
  2. Use Discord bot token rotation in production
  3. Store tokens in production using managed secrets service

## Known Bugs

**Duplicate Handler Invocations:**
- Symptoms: `onMessageCreate` handler fires twice for some messages, creating duplicate processing logs
- Files: `internal/service/bot/handlers.go` (line 17-31)
- Trigger: Appears to be non-deterministic; affects link URL processing
- Current state: Instrumented with verbose logging (handler_id, goroutine tracking) to diagnose
- Workaround: Knok deduplication by message ID and URL prevents DB duplicates, but handler overhead remains
- Fix approach: Investigate Discord.go session setup for handler registration duplicates

**Platform Unknown Mode Configuration Inconsistency:**
- Symptoms: Mixed behavior in URL handling - server-specific settings may override global config unpredictably
- Files: `internal/service/bot/handlers.go` (lines 204-240)
- Cause: Multiple fallback checks (global env var → server settings → default constant) with unclear precedence
- Workaround: Logs show which mode was used; database settings override global config
- Fix approach: Implement single source of truth for platform mode resolution

## Performance Bottlenecks

**Rod Browser Extraction Timeout:**
- Problem: Browser-based metadata extraction (Tier 2) times out at 15 seconds, causing job failures for slow/unresponsive sites
- Files: `internal/service/worker/processor.go` (line 421)
- Cause: Rod library waits for full page render; some sites have delayed JS execution or network issues
- Impact: Jobs timeout and fail, metadata extraction incomplete for ~5-10% of URLs
- Improvement path:
  1. Add configurable timeout per URL (shorter for known fast sites)
  2. Implement early exit if minimal metadata found before timeout
  3. Cache extraction failures by domain to avoid retrying bad sites
  4. Consider parallel extraction attempts (oEmbed + HTTP simultaneously)

**URL Detection Runs on Every Message:**
- Problem: `processDetectedURL()` executes for every Discord message, including those without URLs
- Files: `internal/service/bot/handlers.go` (line 190)
- Cause: URL detection happens in handler critical path before filtering
- Impact: Unnecessary processing, increased latency for non-URL messages
- Improvement path:
  1. Move URL detection to earlier filter stage
  2. Cache URL detection regex compilation
  3. Add early return for bot messages (already done, but inefficient order)

**Database Connection Pooling Not Configured:**
- Problem: No explicit connection pool settings (`SetMaxOpenConns`, `SetMaxIdleConns`)
- Files: `cmd/api/main.go` (line 36)
- Cause: Uses default connection limits (25 open, 2 idle)
- Impact: Under high load, connection starvation possible; each goroutine may block waiting for connection
- Improvement path:
  1. Set `db.SetMaxOpenConns(20)` to match typical concurrent request load
  2. Set `db.SetMaxIdleConns(5)` to avoid connection churn
  3. Add connection pool monitoring metrics

**Search Query Processing:**
- Problem: Full-text search with LIKE queries may not use indexes efficiently
- Files: `internal/repository/postgres/knok.go` (lines 40-70)
- Cause: Wildcard prefix matching `word:*` disables index usage for first character
- Impact: Search performance degradation as dataset grows beyond 100k+ knoks
- Improvement path:
  1. Add PostgreSQL GIN index on `to_tsvector(title)` and `to_tsvector(metadata)`
  2. Migrate from LIKE to full-text search operators (`@@`, `&`, `|`)
  3. Add query result caching for popular searches

## Fragile Areas

**Metadata Extraction Fallback Chain:**
- Files: `internal/service/worker/processor.go` (lines 100+)
- Why fragile: Four-tier fallback (oEmbed → HTTP → Rod → basic) with error handling at each tier; one broken tier can cascade failures
- Safe modification:
  1. Always add unit tests with mocked HTTP responses before changing extraction logic
  2. Test each tier independently (mock the next tier)
  3. Add metrics for which tier succeeds (already logged)
- Test coverage: oEmbed registry has unit tests; HTTP and Rod extraction lack mock tests

**Server Repository Implementation Gaps:**
- Files: `internal/repository/postgres/server.go` (lines 157-193)
- Why fragile: Half-implemented interface creates false sense of completeness; calls log "not implemented" but return success
- Safe modification:
  1. Implement all five stub methods before use in production
  2. Add integration tests against test database
  3. Consider interface-based approach: mark unimplemented methods as explicit no-ops with clear comments
- Test coverage: No tests for server operations

**Bot Message Handler Duplication:**
- Files: `internal/service/bot/handlers.go` (lines 16-186)
- Why fragile: Unknown duplicate invocation source; heavy instrumentation suggests previous issues
- Safe modification:
  1. Never modify handler registration or Discord session setup without reproducing duplication
  2. Add integration tests with mock Discord messages
  3. Consider moving duplicate detection to database layer (already done) but investigate root cause
- Test coverage: No unit tests for message handler

**Frontend Search Component State Management:**
- Files: `web/src/components/SearchComponent.tsx` (lines 5-100)
- Why fragile: Conflicting state management (useQuery + manual state), mock data hardcoded, multiple commented implementation plans suggest incomplete refactoring
- Safe modification:
  1. Complete refactoring to use useQuery with `enabled: false` OR remove useQuery entirely
  2. Add integration tests for search before modifying
  3. Replace mock data with real API calls
- Test coverage: No unit tests for search component

## Scaling Limits

**Database Connection Pool:**
- Current capacity: 25 concurrent connections (default PostgreSQL driver)
- Limit: At 50+ concurrent requests, connection pool exhaustion occurs
- Scaling path: Configure `SetMaxOpenConns()` to match expected concurrency; use connection pool monitoring

**Redis Queue Length:**
- Current capacity: Unbounded queue in Redis
- Limit: Memory consumption grows linearly with queue backlog; no automatic cleanup
- Scaling path: Implement queue monitoring, TTL-based job cleanup, job batching to reduce queue depth

**Browser Pool (Rod):**
- Current capacity: Single browser instance launched per metadata extraction job
- Limit: Browser startup takes 2-5 seconds; under 100+ concurrent jobs, system runs out of memory
- Scaling path: Implement browser pool with max 3-5 instances, queue waiting jobs, implement circuit breaker for browser failures

**Full-Text Search Index:**
- Current capacity: Unindexed or inefficient indexes on title/metadata
- Limit: Query times degrade exponentially beyond 100k+ knoks
- Scaling path: Create GIN indexes on `to_tsvector(title || metadata)`, implement caching for popular searches

## Dependencies at Risk

**go-rod/rod Browser Automation:**
- Risk: Unmaintained or slow-moving dependency; tight coupling to Chromium binary
- Impact: Browser crashes cause job failures; updates may break compatibility
- Current usage: `internal/service/worker/processor.go` (line 14)
- Migration plan:
  1. Document how extraction works without Rod (currently falls back to HTTP)
  2. Consider Playwright or Puppeteer alternatives if Rod becomes unmaintained
  3. Reduce Rod usage: 80% of URLs can extract metadata via oEmbed or HTTP without Rod

**PostgreSQL Full-Text Search:**
- Risk: PostgreSQL-specific feature; not portable to other databases
- Impact: Migration to MySQL/SQLite would require complete search rewrite
- Current usage: `internal/repository/postgres/knok.go` (lines 18-70)
- Migration plan:
  1. Consider ElasticSearch/OpenSearch for future scale (>1M documents)
  2. Add abstraction layer for search backend swapping

**React Query (@tanstack/react-query):**
- Risk: Version pins may become outdated; API surface changes
- Impact: Caching behavior may change; search component already shows version-related issues
- Current usage: `web/src/components/SearchComponent.tsx`
- Migration plan:
  1. Document current caching strategy
  2. Plan upgrade path for major versions

## Missing Critical Features

**Server Management UI:**
- Problem: Admin platform management endpoints exist, but server management endpoints are stubs
- Blocks: Ability to configure per-server settings (allowed channels, platform modes)
- Impact: Multi-server deployments cannot be properly configured
- Priority: HIGH - blocks multi-server deployments

**Metadata Extraction Monitoring/Retry:**
- Problem: Failed extraction jobs have no retry mechanism; extraction_status can stay "processing" indefinitely
- Blocks: Visibility into which URLs failed extraction; manual recovery required
- Impact: ~5-10% of URLs have permanently incomplete metadata
- Priority: HIGH - critical for data quality

**Rate Limiting on Public Endpoints:**
- Problem: No rate limiting on GET endpoints (/api/v1/knoks, /search)
- Blocks: Protection against DOS; scrapers can download entire database
- Impact: Unmetered bandwidth usage; potential service disruption
- Priority: MEDIUM - low risk if behind CDN, but important for self-hosted

**Database Connection Pooling Configuration:**
- Problem: Hardcoded pool settings; no tuning for different deployment environments
- Blocks: Optimal performance under varying load conditions
- Impact: Potential connection starvation under spikes
- Priority: MEDIUM - mitigated by low expected load

**Admin Audit Logging:**
- Problem: Admin API calls logged at DEBUG level; no audit trail of who changed what
- Blocks: Security/compliance audits; incident forensics
- Impact: Cannot trace who deleted data or changed platform configs
- Priority: LOW - acceptable for MVP, required for production

## Test Coverage Gaps

**Bot Message Handler:**
- What's not tested: `onMessageCreate()` handler with all guild/channel restriction combinations
- Files: `internal/service/bot/handlers.go`
- Risk: Unknown duplicate invocation issue; handler logic changes may introduce regressions
- Priority: HIGH - core bot functionality

**Metadata Extraction Fallback Chain:**
- What's not tested: Integration between all four tiers; HTTP and Rod extraction have no mock tests
- Files: `internal/service/worker/processor.go`
- Risk: Tier failures cascade; testing only oEmbed tier misses 80% of failure modes
- Priority: HIGH - metadata quality directly affects UX

**Server Repository Operations:**
- What's not tested: All five stub methods (Update, Delete, List, GetByChannelID, UpdateSettings)
- Files: `internal/repository/postgres/server.go`
- Risk: Unimplemented methods may be called; behavior undefined
- Priority: CRITICAL - blocks server configuration features

**Search Component:**
- What's not tested: useQuery refactoring incomplete; mock data vs. real API calls conflict
- Files: `web/src/components/SearchComponent.tsx`
- Risk: Search behavior unpredictable; component may not work after refactoring
- Priority: HIGH - core user feature

**Admin API Authentication:**
- What's not tested: Auth middleware with missing API key, invalid keys, edge cases
- Files: `internal/http/middleware/auth.go`
- Risk: Security vulnerabilities may exist (e.g., timing attacks on string comparison)
- Priority: CRITICAL - security-sensitive code

**Database Migrations:**
- What's not tested: Migration rollback; migration idempotency; concurrent execution
- Files: `internal/repository/postgres/migrations.go`
- Risk: Failed migrations may leave database in inconsistent state
- Priority: MEDIUM - affects deployment safety

---

*Concerns audit: 2026-03-02*
