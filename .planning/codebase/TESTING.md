# Testing Patterns

**Analysis Date:** 2026-03-02

## Test Framework

**Runner:**
- Frontend: No test runner detected (no Jest or Vitest config found)
- Backend: Go's built-in testing package (`testing`)
- Config: None for frontend; Go tests use standard naming convention

**Assertion Library:**
- Frontend: Not applicable (no tests detected)
- Backend: None; uses standard Go testing with manual assertions

**Run Commands:**
```bash
# Backend tests
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test ./internal/service/...   # Test specific package
```

## Test File Organization

**Location:**
- Backend: Co-located with source - `*_test.go` files in same directory
- Frontend: No test files found

**Naming:**
- Go tests: `*_test.go` (e.g., `extractor_oembed_test.go`, `detector_test.go`)
- Go test functions: `Test*` prefix (e.g., `TestBuildOEmbedURL`, `TestDoubleEncodedURLTransformation`)

**Structure:**
```
internal/
├── service/
│   └── worker/
│       ├── extractor_oembed.go
│       └── extractor_oembed_test.go
├── pkg/
│   └── urldetector/
│       ├── detector.go
│       └── detector_test.go
```

## Test Structure

**Suite Organization:**
Go uses table-driven tests as primary pattern:

```go
func TestBuildOEmbedURL(t *testing.T) {
	// Create a minimal logger for testing
	logger := createTestLogger()

	// Create registry (we need this for the extractor)
	registry, err := NewOEmbedRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Create extractor
	extractor := NewOEmbedExtractor(registry, logger)

	tests := []struct {
		name        string
		endpoint    string
		resourceURL string
		wantURL     string
	}{
		{
			name:        "YouTube URL with si parameter",
			endpoint:    "https://www.youtube.com/oembed",
			resourceURL: "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
			wantURL:     "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A%26si%3D5c2KrgyqSfo_0jSE",
		},
		{
			name:        "YouTube URL without si parameter",
			endpoint:    "https://www.youtube.com/oembed",
			resourceURL: "https://youtube.com/watch?v=D1avYj7q42A",
			wantURL:     "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3DD1avYj7q42A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := extractor.buildOEmbedURL(tt.endpoint, tt.resourceURL)
			if err != nil {
				t.Fatalf("buildOEmbedURL() error = %v", err)
			}

			if gotURL != tt.wantURL {
				t.Errorf("buildOEmbedURL() = %v, want %v", gotURL, tt.wantURL)
			}

			t.Logf("✓ Built oEmbed URL correctly:\n  Resource: %s\n  OEmbed:   %s", tt.resourceURL, gotURL)
		})
	}
}
```

**Patterns:**
- Table-driven test structure: define test cases in slice of structs
- Test names describe behavior: "YouTube URL with si parameter"
- Use `t.Run()` for sub-tests
- Assertions via manual comparison: `if gotURL != tt.wantURL { t.Errorf(...) }`
- Fatal errors for setup failures: `t.Fatalf()`
- Non-fatal errors for assertion failures: `t.Errorf()`

## Mocking

**Framework:** Manual mocking; no external mocking library detected

**Patterns:**
- Dependency injection via constructor arguments
- Helper functions for test setup: `createTestLogger()`
- Logger configured with reduced verbosity for tests:
  ```go
  func createTestLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
      Level: slog.LevelError, // Only show errors during tests
    }))
  }
  ```

**What to Mock:**
- External dependencies: registries, loggers, HTTP clients
- Time-dependent behavior if needed
- File I/O operations

**What NOT to Mock:**
- Domain logic (test the actual behavior)
- Data structures and calculations
- String manipulation and parsing (like URL building)

## Fixtures and Factories

**Test Data:**
- Inline test data in table-driven tests
- Data defined directly in test case structs
- Example from `internal/service/worker/extractor_oembed_test.go`:
  ```go
  tests := []struct {
    name        string
    endpoint    string
    resourceURL string
    wantURL     string
  }{
    {
      name:        "YouTube URL with si parameter",
      endpoint:    "https://www.youtube.com/oembed",
      resourceURL: "https://youtube.com/watch?v=D1avYj7q42A&si=5c2KrgyqSfo_0jSE",
      wantURL:     "https://www.youtube.com/oembed?format=json&url=...",
    },
  }
  ```

**Location:**
- Test fixtures defined in same test function or helper functions
- No separate fixtures directory (not observed in codebase)
- Factories created as helper functions in test file (e.g., `createTestLogger()`)

## Coverage

**Requirements:** Not enforced (no coverage tool configuration detected)

**View Coverage:**
```bash
go test -cover ./...            # Show coverage percentage
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out   # Generate HTML report
```

## Test Types

**Unit Tests:**
- Scope: Single function behavior (e.g., URL building, query sanitization)
- Approach: Table-driven tests with multiple scenarios
- Example: `TestBuildOEmbedURL` tests URL building with various input formats
- Typically test pure functions and isolated business logic

**Integration Tests:**
- Not explicitly identified (would test repository/database interactions)
- Likely exist but separate structure not observed in sample files

**E2E Tests:**
- Not detected in codebase
- No Cypress, Playwright, or similar configuration found

## Frontend Testing Status

**Current State:**
- No testing infrastructure configured
- No test files found in `web/` directory
- No test runner (Jest, Vitest) configured

**Recommendations for Adding Tests:**
- Install testing framework: `npm install --save-dev vitest @testing-library/react @testing-library/user-event`
- Create `vitest.config.ts` at `web/` root
- Place tests co-located with components: `Component.test.tsx`
- Use pattern: query user interactions, assert visible output

## Common Patterns

**Async Testing:**
- Backend: `context.Context` and `context.WithTimeout()` for async operation control
- Frontend: Not applicable (no async testing infrastructure)

**Error Testing:**
- Verify error messages: `if err.Message != expected { t.Errorf(...) }`
- Test status codes in API errors
- Test fallback behavior when errors occur
- Example from `src/api/client.ts`:
  ```typescript
  if (response.status === 401) {
    throw new ApiError(401, 'Unauthorized - invalid admin credentials');
  }
  if (response.status === 404) {
    throw new ApiError(404, 'Knok not found - may have been already deleted');
  }
  ```

**Setup and Teardown:**
- Go: Use helper functions that return initialized objects
- Frontend N/A (no test infrastructure)

---

*Testing analysis: 2026-03-02*
