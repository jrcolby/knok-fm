package worker

import (
	"context"
	"fmt"
	"io"
	"knock-fm/internal/domain"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// JobProcessor handles different types of background jobs
type JobProcessor struct {
	logger           *slog.Logger
	knokRepo         domain.KnokRepository
	serverRepo       domain.ServerRepository
	oembedExtractor  *OEmbedExtractor
}

const (
	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
) *JobProcessor {
	// Initialize oEmbed registry and extractor
	oembedRegistry, err := NewOEmbedRegistry()
	if err != nil {
		logger.Error("Failed to initialize oEmbed registry", "error", err)
		// Continue without oEmbed support
		return &JobProcessor{
			logger:     logger,
			knokRepo:   knokRepo,
			serverRepo: serverRepo,
		}
	}

	logger.Info("oEmbed registry initialized", "provider_count", oembedRegistry.GetProviderCount())

	oembedExtractor := NewOEmbedExtractor(oembedRegistry, logger)

	return &JobProcessor{
		logger:          logger,
		knokRepo:        knokRepo,
		serverRepo:      serverRepo,
		oembedExtractor: oembedExtractor,
	}
}

// ProcessMetadataExtraction processes metadata extraction jobs
func (p *JobProcessor) ProcessMetadataExtraction(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// Extract job parameters
	knokIDStr, ok := payload["knok_id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid knok_id in payload")
	}

	knokID, err := uuid.Parse(knokIDStr)
	if err != nil {
		return fmt.Errorf("invalid knok_id format: %w", err)
	}

	url, ok := payload["url"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid url in payload")
	}

	platform, ok := payload["platform"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid platform in payload")
	}

	logger.Info("Processing metadata extraction job",
		"knok_id", knokID,
		"url", url,
		"platform", platform,
	)

	// Update knok status to processing (if knok repo is available)
	if p.knokRepo != nil {
		if err := p.knokRepo.UpdateExtractionStatus(ctx, knokID, domain.ExtractionStatusProcessing); err != nil {
			logger.Warn("Failed to update knok status to processing", "error", err)
		}
	}

	// Extract metadata using three-tier strategy
	extractedMetadata, extractionMethod, err := p.extractMetadataWithFallbacks(ctx, url)
	if err != nil {
		logger.Error("Failed to extract metadata with fallbacks", "error", err, "url", url)
		// Create minimal fallback metadata
		extractedMetadata = map[string]string{
			"title": "Unknown Title",
		}
		extractionMethod = "error_fallback"
	}

	// Create metadata with extracted data and extraction method info
	metadata := map[string]interface{}{
		"title":             extractedMetadata["title"],
		"description":       extractedMetadata["description"],
		"image":             extractedMetadata["image"],
		"site_name":         extractedMetadata["site_name"],
		"extraction_method": extractionMethod,
		"extracted_at":      time.Now().Unix(),
	}

	p.logger.Info("Metadata extraction completed",
		"extraction_method", extractionMethod,
		"title", metadata["title"],
		"description", metadata["description"],
		"image", metadata["image"],
		"site_name", metadata["site_name"])

	// Update knok with extracted metadata (if knok repo is available)
	if p.knokRepo != nil {
		knok, err := p.knokRepo.GetByID(ctx, knokID)
		if err != nil {
			return fmt.Errorf("failed to get knok for update: %w", err)
		}

		// Update knok title with extracted metadata
		if title, ok := metadata["title"].(string); ok {
			knok.Title = &title
		}

		// Update metadata field
		knok.Metadata = map[string]interface{}{
			"extraction_method": extractionMethod,
			"extraction_time":   time.Now().Unix(),
			"image":             metadata["image"],
			"site_name":         metadata["site_name"],
			"title":             metadata["title"],
			"description":       metadata["description"],
		}

		// Update extraction status
		knok.ExtractionStatus = domain.ExtractionStatusComplete

		// Update knok in database
		if err := p.knokRepo.Update(ctx, knok); err != nil {
			return fmt.Errorf("failed to update knok: %w", err)
		}

		logger.Info("Metadata extraction completed successfully",
			"knok_id", knokID,
			"title", knok.Title,
		)
	} else {
		logger.Info("Metadata extraction completed (no knok repo available)",
			"knok_id", knokID,
		)
	}

	return nil
}

// ProcessKnok processes knok processing jobs
func (p *JobProcessor) ProcessKnok(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// This would handle additional knok processing beyond metadata extraction
	// For now, just log that we received the job
	logger.Info("Processing knok job", "payload", payload)
	return nil
}

// ProcessNotification processes notification jobs
func (p *JobProcessor) ProcessNotification(ctx context.Context, payload map[string]interface{}, logger *slog.Logger) error {
	// This would handle sending notifications about completed jobs
	// For now, just log that we received the job
	logger.Info("Processing notification job", "payload", payload)
	return nil
}

// extractOgMetadata fetches the HTML page and extracts the opengraph metadata tag values
func (p *JobProcessor) extractOgMetadata(ctx context.Context, url string) (map[string]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Follow redirects automatically
		CheckRedirect: nil,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a real browser and avoid bot detection
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Limit response body size to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 1024*1024) // 1MB limit

	// Parse HTML once and extract all Open Graph metadata
	return p.extractOgMetadataFromHTML(limitedReader)
}

// extractOgMetadataFromHTML parses HTML and extracts all Open Graph metadata tags
func (p *JobProcessor) extractOgMetadataFromHTML(r io.Reader) (map[string]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	ogData := make(map[string]string)
	p.findOgMetaInNode(doc, ogData)

	// Clean up all extracted values (trim whitespace, normalize spaces)
	for key, value := range ogData {
		value = strings.TrimSpace(value)
		value = regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")
		ogData[key] = value
	}

	return ogData, nil
}

// findOgMetaInNode recursively searches for Open Graph and Twitter Card meta tags and collects their content
func (p *JobProcessor) findOgMetaInNode(n *html.Node, ogData map[string]string) {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var property, content string

		// Parse attributes to find property/name and content
		for _, attr := range n.Attr {
			if attr.Key == "content" {
				content = attr.Val
			} else if attr.Key == "property" && strings.HasPrefix(attr.Val, "og:") {
				// OpenGraph tags: <meta property="og:title" content="...">
				property = strings.TrimPrefix(attr.Val, "og:")
			} else if attr.Key == "name" && strings.HasPrefix(attr.Val, "twitter:") {
				// Twitter Card tags: <meta name="twitter:title" content="...">
				twitterProperty := strings.TrimPrefix(attr.Val, "twitter:")
				// Map Twitter Card properties to OpenGraph equivalents
				switch twitterProperty {
				case "title":
					property = "title"
				case "description":
					property = "description"
				case "image":
					property = "image"
				case "site":
					property = "site_name"
				default:
					property = twitterProperty // Keep other Twitter properties as-is
				}
			}
		}

		// Only store if we have both property and content, and don't overwrite existing OpenGraph data
		if property != "" && content != "" {
			// Prioritize OpenGraph over Twitter Cards - only set if not already present
			if _, exists := ogData[property]; !exists {
				ogData[property] = content
			}
		}
	}

	// Recursively search child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p.findOgMetaInNode(c, ogData)
	}
}

// extractTitleFromURL fetches the HTML page and extracts the title
func (p *JobProcessor) extractTitleFromURL(ctx context.Context, url string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Follow redirects automatically
		CheckRedirect: nil,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a real browser and avoid bot detection
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Limit response body size to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 1024*1024) // 1MB limit

	// Parse HTML and extract title
	title, err := p.extractTitleFromHTML(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to extract title from HTML: %w", err)
	}

	return title, nil
}

// extractTitleFromHTML parses HTML and extracts the title tag content
func (p *JobProcessor) extractTitleFromHTML(r io.Reader) (string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	title := p.findTitleInNode(doc)
	if title == "" {
		return "", fmt.Errorf("no title tag found")
	}

	// Clean up the title (trim whitespace, normalize spaces)
	title = strings.TrimSpace(title)
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	return title, nil
}

// findTitleInNode recursively searches for title tag content
func (p *JobProcessor) findTitleInNode(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		// Found title tag, extract text content
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				return c.Data
			}
		}
	}

	// Recursively search child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := p.findTitleInNode(c); title != "" {
			return title
		}
	}

	return ""
}

// extractMetadataWithRod uses a headless browser to extract metadata from JavaScript-rendered pages
func (p *JobProcessor) extractMetadataWithRod(ctx context.Context, url string) (map[string]string, error) {
	p.logger.Info("Starting Rod-based metadata extraction", "url", url)

	// Create context with timeout for Rod operations
	rodCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Launch headless browser with proper flags for JavaScript execution
	l := launcher.New().
		Bin("/usr/bin/chromium-browser"). // Use system Chromium in Alpine
		Headless(true).
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		Set("disable-extensions").
		Set("disable-plugins")

	p.logger.Info("Using Chromium browser", "path", "/usr/bin/chromium-browser")

	defer l.Cleanup()

	// Add timeout for browser launch
	launchCtx, launchCancel := context.WithTimeout(rodCtx, 15*time.Second)
	defer launchCancel()

	p.logger.Info("Launching browser with Rod", "url", url)
	controlURL, err := l.Context(launchCtx).Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser (timeout after 15s): %w", err)
	}
	p.logger.Info("Browser launched successfully", "url", url, "control_url", controlURL)

	// Connect to browser with timeout
	browser := rod.New().ControlURL(controlURL)
	connectCtx, connectCancel := context.WithTimeout(rodCtx, 10*time.Second)
	defer connectCancel()

	if err := browser.Context(connectCtx).Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser (timeout after 10s): %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			p.logger.Warn("Failed to close browser", "error", err)
		}
	}()
	p.logger.Info("Connected to browser successfully", "url", url)

	// Create page and navigate
	p.logger.Info("Creating browser page", "url", url)
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer func() {
		if err := page.Close(); err != nil {
			p.logger.Warn("Failed to close page", "error", err)
		}
	}()

	// Set user agent to mimic a real browser and avoid bot detection
	p.logger.Info("Setting user agent", "url", url)
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: browserUserAgent,
	}); err != nil {
		p.logger.Warn("Failed to set user agent", "error", err)
	}

	// Navigate to URL with proper error handling
	p.logger.Info("Navigating to URL", "url", url)

	// Use a more basic navigation approach with timeout
	navCtx, navCancel := context.WithTimeout(rodCtx, 15*time.Second)
	defer navCancel()

	if err := page.Context(navCtx).Navigate(url); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}
	p.logger.Info("Page navigation completed", "url", url)

	// Simple wait for load
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}
	p.logger.Info("Page load completed", "url", url)

	// Wait a fixed amount of time for JavaScript to execute
	time.Sleep(3 * time.Second)
	p.logger.Info("JavaScript wait completed", "url", url)

	// Get the rendered HTML
	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get page HTML: %w", err)
	}

	// Debug: Log first 500 chars of HTML to see what Rod extracted
	htmlPreview := html
	if len(htmlPreview) > 500 {
		htmlPreview = htmlPreview[:500] + "..."
	}
	p.logger.Debug("Rod HTML preview", "url", url, "html_start", htmlPreview)

	// Parse the rendered HTML using existing HTML parsing logic
	metadata, err := p.extractOgMetadataFromHTML(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata from rendered HTML: %w", err)
	}

	// Debug: Log what metadata was found
	p.logger.Debug("Rod metadata parsed",
		"url", url,
		"metadata_count", len(metadata),
		"has_title", metadata["title"] != "",
		"has_description", metadata["description"] != "",
		"has_image", metadata["image"] != "")

	p.logger.Info("Rod extraction completed",
		"url", url,
		"title", metadata["title"],
		"description", metadata["description"],
		"image", metadata["image"])

	return metadata, nil
}

// extractMetadataWithRodSimple uses the simplest possible Rod approach with proper error handling
func (p *JobProcessor) extractMetadataWithRodSimple(ctx context.Context, url string) (map[string]string, error) {
	p.logger.Info("Starting simple Rod metadata extraction", "url", url)

	// Create timeout context
	rodCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Launch headless browser using system Chromium
	l := launcher.New().
		Bin("/usr/bin/chromium-browser"). // Use system Chromium in Alpine
		Headless(true).
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-features", "VizDisplayCompositor").
		Set("disable-extensions").
		Set("disable-plugins")

	p.logger.Info("Using Chromium browser", "path", "/usr/bin/chromium-browser")

	defer l.Cleanup()

	// Launch browser
	controlURL, err := l.Context(rodCtx).Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	p.logger.Info("Browser launched successfully", "control_url", controlURL)

	// Connect to browser
	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			p.logger.Warn("Failed to close browser", "error", err)
		}
	}()

	p.logger.Info("Rod browser connected", "url", url)

	// Create page
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer func() {
		if err := page.Close(); err != nil {
			p.logger.Warn("Failed to close page", "error", err)
		}
	}()

	// Navigate and wait for load with 15-second timeout using Rod's Timeout helper
	p.logger.Info("Rod starting navigation with timeout", "url", url, "timeout", "15s")
	err = rod.Try(func() {
		page.Timeout(15 * time.Second).MustNavigate(url).MustWaitLoad()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate/load (timeout after 15s): %w", err)
	}

	p.logger.Info("Rod page loaded successfully", "url", url)

	// Wait for JavaScript (fixed time)
	time.Sleep(3 * time.Second)
	p.logger.Info("Rod wait completed", "url", url)

	// Get HTML with error handling
	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to extract HTML: %w", err)
	}

	// Log first 100 lines of HTML to debug what Rod is seeing
	htmlLines := strings.Split(html, "\n")
	previewLines := htmlLines
	if len(htmlLines) > 100 {
		previewLines = htmlLines[:100]
	}
	p.logger.Info("Rod HTML extracted",
		"url", url,
		"length", len(html),
		"total_lines", len(htmlLines),
		"preview_lines", len(previewLines),
		"html_preview", strings.Join(previewLines, "\n"))

	// Parse metadata
	metadata, err := p.extractOgMetadataFromHTML(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	p.logger.Info("Rod metadata parsed",
		"url", url,
		"title", metadata["title"],
		"description", metadata["description"],
		"image", metadata["image"])

	return metadata, nil
}

// extractMetadataWithFallbacks implements the four-tier metadata extraction strategy
func (p *JobProcessor) extractMetadataWithFallbacks(ctx context.Context, url string) (map[string]string, string, error) {
	p.logger.Info("Starting four-tier metadata extraction", "url", url)

	// Tier 0: oEmbed API (fastest, most reliable for supported providers)
	if p.oembedExtractor != nil {
		p.logger.Info("Tier 0: Attempting oEmbed metadata extraction", "url", url)
		oembedMetadata, err := p.oembedExtractor.TryExtract(ctx, url)
		if err != nil {
			// oEmbed failed, but continue to fallback tiers
			p.logger.Warn("oEmbed extraction failed", "error", err, "url", url)
		} else if oembedMetadata != nil {
			// oEmbed succeeded!
			p.logger.Info("oEmbed extraction successful",
				"url", url,
				"title", oembedMetadata["title"],
				"has_image", oembedMetadata["image"] != "")

			// Ensure description fallback
			if oembedMetadata["description"] == "" {
				oembedMetadata["description"] = url
			}

			return oembedMetadata, "oembed", nil
		}
		// oembedMetadata == nil means no provider found, continue to next tier
		p.logger.Debug("No oEmbed provider for URL, trying fallback tiers", "url", url)
	}

	// Tier 1: HTTP + Static HTML Parsing
	p.logger.Info("Tier 1: Attempting HTTP-based metadata extraction", "url", url)
	httpMetadata, err := p.extractOgMetadata(ctx, url)
	if err != nil {
		p.logger.Warn("HTTP metadata extraction failed", "error", err, "url", url)
		httpMetadata = make(map[string]string)
	}

	// Debug: Log what HTTP extraction found
	p.logger.Info("HTTP metadata extraction results",
		"url", url,
		"title", httpMetadata["title"],
		"description", httpMetadata["description"],
		"image", httpMetadata["image"],
		"site_name", httpMetadata["site_name"],
		"total_fields", len(httpMetadata))

	// Get basic title as fallback
	title, titleErr := p.extractTitleFromURL(ctx, url)
	if titleErr != nil {
		p.logger.Warn("Title extraction failed", "error", titleErr, "url", url)
		title = "Unknown Title"
	}

	// Check if we have sufficient metadata from HTTP parsing
	hasTitle := httpMetadata["title"] != ""
	hasDescription := httpMetadata["description"] != ""
	hasImage := httpMetadata["image"] != ""

	// Use URL as description if missing
	if hasTitle && !hasDescription {
		httpMetadata["description"] = url
		hasDescription = true
		p.logger.Debug("Using URL as description fallback", "url", url)
	}

	// If we have all key metadata, use HTTP results
	if hasTitle && (hasDescription || hasImage) {
		p.logger.Info("HTTP extraction successful, using tier 1 results",
			"url", url,
			"title", httpMetadata["title"],
			"has_description", hasDescription,
			"has_image", hasImage)
		return httpMetadata, "http_static", nil
	}

	// Tier 2: Rod Headless Browser (for JavaScript-rendered content)
	p.logger.Info("Tier 2: Attempting Rod-based metadata extraction", "url", url)
	rodMetadata, rodErr := p.extractMetadataWithRodSimple(ctx, url)

	if rodErr != nil {
		p.logger.Warn("Rod metadata extraction skipped/failed", "error", rodErr, "url", url)
	} else {
		// Merge Rod results with HTTP results, prioritizing Rod for missing fields
		mergedMetadata := make(map[string]string)

		// Copy HTTP results first
		for k, v := range httpMetadata {
			mergedMetadata[k] = v
		}

		// Fill in missing fields with Rod results
		for k, v := range rodMetadata {
			if v != "" && mergedMetadata[k] == "" {
				mergedMetadata[k] = v
			}
		}

		// Check if Rod improved our metadata
		rodHasTitle := mergedMetadata["title"] != ""
		rodHasDescription := mergedMetadata["description"] != ""
		rodHasImage := mergedMetadata["image"] != ""

		// Use URL as description if missing
		if rodHasTitle && !rodHasDescription {
			mergedMetadata["description"] = url
			rodHasDescription = true
			p.logger.Debug("Using URL as description fallback for Rod results", "url", url)
		}

		if rodHasTitle && (rodHasDescription || rodHasImage) {
			p.logger.Info("Rod extraction successful, using tier 2 results",
				"url", url,
				"title", mergedMetadata["title"],
				"has_description", rodHasDescription,
				"has_image", rodHasImage)
			return mergedMetadata, "rod_browser", nil
		}
	}

	// Tier 3: Final Fallback (basic title only)
	p.logger.Info("Tier 3: Using basic title fallback", "url", url, "title", title)
	fallbackMetadata := map[string]string{
		"title": title,
	}

	// Include any partial metadata we managed to extract
	if httpMetadata["description"] != "" {
		fallbackMetadata["description"] = httpMetadata["description"]
	} else {
		// Always use URL as description if no description was found
		fallbackMetadata["description"] = url
		p.logger.Debug("Using URL as description fallback in tier 3", "url", url)
	}

	if httpMetadata["image"] != "" {
		fallbackMetadata["image"] = httpMetadata["image"]
	}
	if httpMetadata["site_name"] != "" {
		fallbackMetadata["site_name"] = httpMetadata["site_name"]
	}

	return fallbackMetadata, "title_fallback", nil
}
