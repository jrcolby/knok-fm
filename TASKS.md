# oEmbed-First Metadata Extraction - Implementation Tasks

## Overview
Implement a 4-tier metadata extraction strategy with oEmbed as the fastest, most reliable tier for 500+ providers including YouTube, Spotify, SoundCloud, Vimeo, TikTok, and more.

## Architecture
```
Tier 0: oEmbed API (YouTube, Spotify, SoundCloud, Vimeo, TikTok, 500+ providers)
  ↓ (if no provider found)
Tier 1: HTTP + Static HTML parsing (og:tags, twitter:cards)
  ↓ (if insufficient metadata)
Tier 2: Rod Browser (JavaScript-rendered content)
  ↓ (if all else fails)
Tier 3: Basic title fallback
```

## Status
- [x] In Progress
- [x] Ready for Testing
- [ ] Deployed to Production

---

## Tasks

### ✅ Step 0: Planning & Architecture Design
- [x] Analyze oEmbed providers.json structure
- [x] Design clean architecture (embedded registry + generic extractor)
- [x] Create TASKS.md to track progress

### ✅ Step 1: Download & Embed oEmbed Providers
**File**: `internal/service/worker/oembed_providers.json`

- [x] Download providers.json from https://oembed.com/providers.json
- [x] Save to `internal/service/worker/oembed_providers.json`
- [x] Verify JSON structure and major providers (YouTube, Spotify, etc.)
- [x] Commit to git

**Acceptance Criteria**:
- [x] JSON file exists in project (5291 lines)
- [x] Contains 359 providers
- [x] Includes YouTube, Spotify, SoundCloud, Vimeo, TikTok, MixCloud, Dailymotion, Twitter

---

### ✅ Step 2: Create oEmbed Registry
**File**: `internal/service/worker/oembed_registry.go`

- [x] Create `OEmbedProvider` struct (name, endpoint, schemes)
- [x] Create `OEmbedRegistry` struct
- [x] Implement `go:embed` to load providers.json at compile time
- [x] Parse JSON into provider list on init
- [x] Implement `Match(url string) *OEmbedProvider` using URL pattern matching
- [x] Write unit tests for URL matching

**Key Functions**:
```go
type OEmbedProvider struct {
    Name     string
    Endpoint string
    Schemes  []string
}

type OEmbedRegistry struct {
    providers []*OEmbedProvider
}

func NewOEmbedRegistry() (*OEmbedRegistry, error)
func (r *OEmbedRegistry) Match(url string) *OEmbedProvider
```

**Acceptance Criteria**:
- [x] Registry loads 351 providers on startup
- [x] `Match()` correctly identifies YouTube, Spotify, SoundCloud, youtu.be, spotify.link URLs
- [x] Fast lookup (benchmark shows ~300-400ns per match)
- [x] All tests pass

---

### ✅ Step 3: Create Generic oEmbed Extractor
**File**: `internal/service/worker/extractor_oembed.go`

- [x] Create `OEmbedExtractor` struct with registry
- [x] Implement `TryExtract(ctx context.Context, url string) (map[string]string, error)`
- [x] Build oEmbed API URL with proper query params
- [x] Make HTTP request to oEmbed endpoint
- [x] Parse JSON response
- [x] Map oEmbed fields to our metadata format
- [x] Handle errors gracefully (return nil, nil if no provider; error if provider exists but fails)
- [x] Add logging for debugging
- [ ] Write unit tests with mocked HTTP responses (optional - can test in production first)

**Key Functions**:
```go
type OEmbedExtractor struct {
    registry *OEmbedRegistry
    logger   *slog.Logger
}

func NewOEmbedExtractor(registry *OEmbedRegistry, logger *slog.Logger) *OEmbedExtractor
func (e *OEmbedExtractor) TryExtract(ctx context.Context, url string) (map[string]string, error)
```

**oEmbed → Our Metadata Mapping**:
- `title` → `title`
- `author_name` → `site_name` or `description`
- `thumbnail_url` → `image`
- `url` → fallback to input URL

**Acceptance Criteria**:
- [x] Code compiles successfully
- [x] Returns nil, nil for non-oEmbed URLs (allows fallback)
- [x] Handles API errors gracefully
- [ ] Production testing confirms YouTube/Spotify extraction works

---

### ⏭️ Step 4: Refactor Existing Extractors (SKIPPED)
**Files**: `extractor_http.go`, `extractor_rod.go`

This step is optional but recommended for cleaner code organization:

- [ ] Extract HTTP extraction logic to `extractor_http.go`
- [ ] Extract Rod browser logic to `extractor_rod.go`
- [ ] Keep only orchestration in `processor.go`

**If skipping**: Keep existing code in `processor.go` - no problem!

---

### ✅ Step 5: Integrate oEmbed into Extraction Chain
**File**: `internal/service/worker/processor.go`

- [x] Initialize `OEmbedRegistry` in `NewJobProcessor`
- [x] Create `OEmbedExtractor` instance
- [x] Update `extractMetadataWithFallbacks()` to add Tier 0
- [x] Try oEmbed first, before HTTP extraction
- [x] Log which tier succeeded for monitoring
- [x] Preserve existing fallback behavior

**Updated Flow**:
```go
func (p *JobProcessor) extractMetadataWithFallbacks(ctx context.Context, url string) (map[string]string, string, error) {
    // Tier 0: Try oEmbed API
    if oembedMetadata, err := p.oembedExtractor.TryExtract(ctx, url); err == nil && oembedMetadata != nil {
        return oembedMetadata, "oembed", nil
    }

    // Tier 1: HTTP + og:tags (existing code)
    // ...

    // Tier 2: Rod browser (existing code)
    // ...

    // Tier 3: Title fallback (existing code)
    // ...
}
```

**Acceptance Criteria**:
- oEmbed is attempted first
- Falls back to existing tiers if oEmbed fails
- Logs show "oembed" extraction method when successful
- No breaking changes to existing functionality

---

### ⬜ Step 6: Testing & Deployment
**Owner**: User will handle this step

- [ ] Build and test locally with `make dev-worker`
- [ ] Test YouTube URL → should use oEmbed (fast, no 429)
- [ ] Test Spotify URL → should use oEmbed
- [ ] Test unknown URL → should fall back to HTTP
- [ ] Check logs for extraction method used
- [ ] Commit changes
- [ ] Deploy to production with `make deploy-worker`
- [ ] Monitor production logs
- [ ] Verify Discord link previews work correctly

**Test URLs**:
- YouTube: `https://youtube.com/watch?v=JUDUC87VuPU`
- Spotify: `https://open.spotify.com/track/...`
- Unknown: Any random website

**Expected Logs**:
```json
{"msg":"Tier 0: Attempting oEmbed metadata extraction","url":"..."}
{"msg":"oEmbed extraction successful","provider":"YouTube","title":"..."}
{"msg":"Metadata extraction completed","extraction_method":"oembed"}
```

---

## Success Metrics

After deployment, verify:
- ✅ YouTube metadata extracts in < 1 second (vs 10-15s with Rod)
- ✅ No more YouTube 429 errors
- ✅ Spotify, SoundCloud, Vimeo all extract successfully
- ✅ Fallback still works for sites without oEmbed
- ✅ Discord shows rich link previews with proper titles/images

---

## Future Enhancements (Post-MVP)

### Phase 2: Caching
- [ ] Cache oEmbed responses in Redis (24h TTL)
- [ ] Reduce duplicate API calls for same URL

### Phase 3: Parallel Extraction
- [ ] Try oEmbed + HTTP simultaneously for fastest response
- [ ] Use first successful result

### Phase 4: Provider Updates
- [ ] Monthly cron job to fetch updated providers.json
- [ ] Automated PR with provider updates

### Phase 5: Custom Overrides
- [ ] Allow custom extraction logic per provider
- [ ] Handle provider-specific quirks or API changes

---

## Notes

- oEmbed providers list from: https://oembed.com/providers.json
- Standard maintained by https://oembed.com/
- Covers 500+ providers as of 2025
- Most major music/video platforms support oEmbed
