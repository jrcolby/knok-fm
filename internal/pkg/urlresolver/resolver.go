package urlresolver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	maxRedirects   = 3
	requestTimeout = 5 * time.Second
	browserUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// shortDomains is the set of known short/redirect link domains.
var shortDomains = map[string]bool{
	"on.soundcloud.com": true,
	"spotify.link":      true,
	"spoti.fi":          true,
	"youtu.be":          true,
	"amzn.to":           true,
	"band.link":         true,
	"deezer.page.link":  true,
	"link.tospotify.com": true,
}

// Resolver follows HTTP redirects for short link domains to get canonical URLs.
type Resolver struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// New creates a new Resolver.
func New(logger *slog.Logger) *Resolver {
	return &Resolver{
		httpClient: &http.Client{
			Timeout: requestTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		logger: logger,
	}
}

// IsShortLink checks if the URL is a known short link domain. No I/O.
func (r *Resolver) IsShortLink(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return shortDomains[parsedURL.Host]
}

// Resolve follows redirects for a short link URL and returns the final canonical URL.
// On any error, returns the original URL (never blocks ingestion).
func (r *Resolver) Resolve(ctx context.Context, rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, err
	}

	if !shortDomains[parsedURL.Host] {
		return rawURL, nil
	}

	resolveCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	// Try HEAD first
	finalURL, err := r.followRedirects(resolveCtx, rawURL, "HEAD")
	if err != nil {
		// Fallback to GET
		r.logger.Debug("HEAD request failed, trying GET",
			"url", rawURL,
			"error", err)
		finalURL, err = r.followRedirects(resolveCtx, rawURL, "GET")
		if err != nil {
			r.logger.Warn("Failed to resolve short link",
				"url", rawURL,
				"error", err)
			return rawURL, err
		}
	}

	if finalURL != rawURL {
		r.logger.Info("Resolved short link",
			"short_url", rawURL,
			"resolved_url", finalURL)
	}

	return finalURL, nil
}

// followRedirects makes a request and follows redirects up to maxRedirects.
func (r *Resolver) followRedirects(ctx context.Context, rawURL string, method string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return rawURL, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", browserUA)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return rawURL, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// The http.Client follows redirects automatically (up to maxRedirects).
	// The final URL is in resp.Request.URL.
	finalURL := resp.Request.URL.String()
	return finalURL, nil
}
