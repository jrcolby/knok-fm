package platforms

import (
	"context"
	"fmt"
	"knock-fm/internal/domain"
	"log/slog"
	"sync"
)

// PlatformRepository defines the interface for platform data access
type PlatformRepository interface {
	GetAllPlatforms(ctx context.Context) ([]*domain.Platform, error)
}

// Loader manages platform configuration with in-memory caching
type Loader struct {
	repo   PlatformRepository
	logger *slog.Logger

	mu        sync.RWMutex
	platforms map[string]*domain.Platform // cached platforms by ID
	loaded    bool
}

// NewLoader creates a new platform loader
func NewLoader(repo PlatformRepository, logger *slog.Logger) *Loader {
	return &Loader{
		repo:      repo,
		logger:    logger,
		platforms: make(map[string]*domain.Platform),
		loaded:    false,
	}
}

// Load fetches platforms from the database and caches them in memory.
// Falls back to hardcoded defaults if database is unavailable.
func (l *Loader) Load(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Info("Loading platforms from database...")

	platforms, err := l.repo.GetAllPlatforms(ctx)
	if err != nil {
		l.logger.Warn("Failed to load platforms from database, falling back to defaults",
			"error", err,
		)
		// Fall back to hardcoded defaults
		return l.loadDefaults()
	}

	// Cache platforms by ID
	l.platforms = make(map[string]*domain.Platform, len(platforms))
	enabledCount := 0
	for _, platform := range platforms {
		if platform.Enabled {
			l.platforms[platform.ID] = platform
			enabledCount++
		}
	}

	l.loaded = true
	l.logger.Info("Platforms loaded successfully from database",
		"total", len(platforms),
		"enabled", enabledCount,
	)

	return nil
}

// loadDefaults loads hardcoded default platforms as fallback
func (l *Loader) loadDefaults() error {
	l.logger.Info("Loading hardcoded default platforms...")

	config := domain.GetDefaultPlatformConfig()
	l.platforms = make(map[string]*domain.Platform, len(config.Platforms))

	for id, platform := range config.Platforms {
		// Convert to pointer and set default values
		p := platform
		p.Priority = 0
		p.Enabled = true
		l.platforms[id] = &p
	}

	l.loaded = true
	l.logger.Info("Loaded hardcoded default platforms",
		"count", len(l.platforms),
	)

	return nil
}

// Refresh reloads platforms from the database
func (l *Loader) Refresh(ctx context.Context) error {
	l.logger.Info("Refreshing platform configuration...")
	return l.Load(ctx)
}

// Get retrieves a platform by ID from the cache
func (l *Loader) Get(id string) (*domain.Platform, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.loaded {
		return nil, fmt.Errorf("platforms not loaded yet")
	}

	platform, exists := l.platforms[id]
	if !exists {
		return nil, fmt.Errorf("platform not found: %s", id)
	}

	return platform, nil
}

// GetAll returns all enabled platforms from the cache
func (l *Loader) GetAll() ([]*domain.Platform, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.loaded {
		return nil, fmt.Errorf("platforms not loaded yet")
	}

	platforms := make([]*domain.Platform, 0, len(l.platforms))
	for _, platform := range l.platforms {
		if platform.Enabled {
			platforms = append(platforms, platform)
		}
	}

	return platforms, nil
}

// GetAllByPriority returns all enabled platforms sorted by priority (highest first)
func (l *Loader) GetAllByPriority() ([]*domain.Platform, error) {
	platforms, err := l.GetAll()
	if err != nil {
		return nil, err
	}

	// Sort by priority descending (higher priority first)
	// Using simple insertion sort since platform count is small
	for i := 1; i < len(platforms); i++ {
		key := platforms[i]
		j := i - 1
		for j >= 0 && platforms[j].Priority < key.Priority {
			platforms[j+1] = platforms[j]
			j--
		}
		platforms[j+1] = key
	}

	return platforms, nil
}

// IsLoaded returns true if platforms have been loaded
func (l *Loader) IsLoaded() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded
}

// Count returns the number of cached platforms
func (l *Loader) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.platforms)
}
