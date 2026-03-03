# Coding Conventions

**Analysis Date:** 2026-03-02

## Naming Patterns

**Files:**
- Components: PascalCase, e.g., `KnokCard.tsx`, `Modal.tsx`, `Header.tsx`
- Hooks: camelCase with `use` prefix, e.g., `useKnokData.ts`, `useInfiniteKnoks.ts`, `useIntersectionObserver.ts`
- Utilities: camelCase, e.g., `logoFallback.ts`, `utils.ts`
- Types: Placed in dedicated `types.ts` files (e.g., `src/api/types.ts`)
- Constants: UPPERCASE, e.g., `LOGO_TYPES`, `PLATFORMS`, `EXTRACTION_STATUS` (in `src/api/types.ts`)

**Functions:**
- Event handlers: `handleX` pattern, e.g., `handleDelete`, `handleImageError`, `handleSearchChange`, `handleLogout`
- Custom hooks: `useX` pattern returning object with properties
- Factory functions: `NewX` in Go (e.g., `NewKnokRepository`), plain names in TypeScript
- Utility functions: camelCase, e.g., `getRandomLogoType`, `needsFallbackLogo`, `sanitizeSearchQuery`

**Variables:**
- State setters: `setX` (React setState convention)
- Boolean flags: prefix with `is`, `has`, or `should`, e.g., `isSearchMode`, `isAdmin`, `isLoading`, `shouldUseFallback`, `hasNextPage`
- Loading states: `isLoading`, `isFetchingNextPage`, `isTyping`
- Query variables: Follow tanstack/react-query pattern, e.g., `timelineData`, `searchData`, `randomData`

**Types:**
- Interfaces: PascalCase with suffix, e.g., `KnokCardProps`, `ModalProps`, `AdminContextType`, `UseInfiniteKnoksResult`
- Data Transfer Objects: `Dto` suffix in TypeScript, e.g., `KnokDto`, `ServerDto`
- Response types: `Response` suffix, e.g., `KnoksResponse`, `DeleteKnokResponse`
- Request types: `Request` suffix, e.g., `RefreshKnokRequest`

## Code Style

**Formatting:**
- No explicit Prettier configuration found; uses ESLint defaults
- Two-space indentation (observed in all files)
- Semicolons required (TypeScript strict mode enforced)
- Import statements at top, organized by type

**Linting:**
- ESLint 9.33.0 with TypeScript support
- Config: `web/eslint.config.js` (flat config format)
- Extends: `@eslint/js`, `typescript-eslint`, `eslint-plugin-react-hooks`, `eslint-plugin-react-refresh`
- Strict TypeScript checking enforced:
  - `strict: true`
  - `noUnusedLocals: true`
  - `noUnusedParameters: true`
  - `noFallthroughCasesInSwitch: true`
  - `noUncheckedSideEffectImports: true`

## Import Organization

**Order:**
1. React and React Router imports
2. Third-party libraries (tanstack/react-query, lucide-react, clsx, etc.)
3. Internal API/types imports
4. Hooks imports
5. Component/context imports
6. Utility imports

**Path Aliases:**
- `@/*` maps to `src/*` (configured in `tsconfig.json`)
- Used in imports like: `import { useAdmin } from '@/contexts/AdminContext'`
- Applied consistently across components, hooks, and utilities

**Example pattern from `src/components/KnokCard.tsx`:**
```typescript
import type { KnokDto } from "../api/types";
import { useState, memo } from "react";
import { KnokSpiral, KnokStar } from "./icons";
import { getRandomLogoType, needsFallbackLogo, LOGO_TYPES } from "../utils/logoFallback";
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { useAdmin } from '../contexts/AdminContext';
import { DeleteKnokModal } from './DeleteKnokModal';
import { RefreshKnokModal } from './RefreshKnokModal';
import { apiClient } from '../api/client';
import { Trash2, RefreshCw } from 'lucide-react';
```

## Error Handling

**Custom Error Classes:**
- `ApiError` in `src/api/client.ts`: custom class extending `Error` with `status` property
- Distinguishes between API errors and network errors
- HTTP status-specific error messages (401 Unauthorized, 404 Not Found)

**Patterns:**
```typescript
// Try-catch with instanceof checks
try {
  await apiClient.deleteKnok(knok.id, apiKey);
} catch (error) {
  if (error instanceof Error && error.message.includes('Unauthorized')) {
    logout();
    navigate('/janitor');
  }
  throw error; // Re-throw for caller handling
}
```

**Error propagation:**
- Errors thrown upward when appropriate (component doesn't need to handle)
- Modal components display errors to user
- API errors logged with context via console.error()

## Logging

**Framework:** console.error() for error logging (Go backend uses `log/slog`)

**Patterns:**
- Minimal logging in frontend (only on failures)
- Go backend uses structured logging with `slog.Logger`
- Go logger configured with key-value pairs: `log.Error("message", "error", err)`
- Frontend: `console.error('API key validation failed:', error);`

## Comments

**When to Comment:**
- Functional complexity requiring explanation
- Non-obvious logic or workarounds
- Important side effects or dependencies

**Example from `src/hooks/useInfiniteKnoks.ts`:**
```typescript
// Only depend on the input values, not the entire query object.
// Including the full `searchQuery_infinite` result in deps can
// change identity across renders and cause the debounce effect
// to re-schedule repeatedly which leads to repeated refetches.
```

**JSDoc/TSDoc:**
- Used selectively for public APIs
- Function documentation in utility functions (e.g., `src/api/client.ts` methods)
- Parameter documentation: `/** @param url Optional new URL to refresh from */`
- No strict requirement; used when behavior needs clarification

## Function Design

**Size:** Prefer small, focused functions
- Hooks typically 50-100 lines
- Component functions under 200 lines when possible
- Extract complex logic into separate utilities

**Parameters:**
- Use object destructuring for multiple parameters in React components
- Single parameter functions preferred for utilities
- Props interfaces always defined above components

**Return Values:**
- Hooks return objects with named properties for consistency
- Components render JSX directly
- Utilities return typed values with clear semantics

**Example from `src/hooks/useInfiniteKnoks.ts`:**
```typescript
export function useInfiniteKnoks({
  searchQuery,
  enabled = true,
}: UseInfiniteKnoksOptions): UseInfiniteKnoksResult {
  // Returns well-typed object
  return {
    knoks,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    error,
    fetchNextPage,
    refetch,
  };
}
```

## Module Design

**Exports:**
- Named exports for functions and components (preferred)
- Default exports for main entry points (e.g., `export default App`)
- Component memoization with `memo()` for performance: `export const KnokCard = memo(KnokCardComponent);`

**Barrel Files:**
- Used for icon exports: `src/components/icons/index.ts` re-exports icon components
- Simplifies imports: `import { KnokSpiral, KnokStar } from "./icons"`

## Go Backend Conventions

**Package Structure:**
- `cmd/` - Application entry points
- `internal/` - Private packages
- `internal/repository/` - Data access layer
- `internal/service/` - Business logic layer
- `internal/domain/` - Domain models
- `internal/pkg/` - Shared utilities

**Naming:**
- Functions: PascalCase (exported), camelCase (unexported)
- Variables: camelCase
- Constants: UPPER_CASE
- Error variables: `Err*` prefix
- Receiver names: Short (1-2 letters)

**Error Handling:**
```go
if err := db.Ping(); err != nil {
  log.Error("Failed to ping database", "error", err)
  os.Exit(1)
}
```

**Logging:**
- Structured logging with `slog.Logger`
- Key-value pairs for context
- Log levels: Info, Error (observe test file uses LevelError for tests)

---

*Convention analysis: 2026-03-02*
