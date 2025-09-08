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

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// JobProcessor handles different types of background jobs
type JobProcessor struct {
	logger     *slog.Logger
	knokRepo   domain.KnokRepository
	serverRepo domain.ServerRepository
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	logger *slog.Logger,
	knokRepo domain.KnokRepository,
	serverRepo domain.ServerRepository,
) *JobProcessor {
	return &JobProcessor{
		logger:     logger,
		knokRepo:   knokRepo,
		serverRepo: serverRepo,
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

	// Extract metadata from the URL
	title, err := p.extractTitleFromURL(ctx, url)
	if err != nil {
		logger.Warn("Failed to extract title from URL", "error", err, "url", url)
		title = "Unknown Title" // Fallback if extraction fails
	}

	// Extract all open graph metadata from HTML
	opengraphMetadata, err := p.extractOgMetadata(ctx, url)
	if err != nil {
		logger.Warn("Failed to extract opengraph metadata from URL", "error", err, "url", url)
		opengraphMetadata = make(map[string]string) // Ensure it's not nil
	}

	// Create metadata with extracted title and flattened Open Graph data
	metadata := map[string]interface{}{
		"title":        title,
		"description":  opengraphMetadata["description"],
		"image":        opengraphMetadata["image"],
		"site_name":    opengraphMetadata["site_name"],
		"extracted_at": time.Now().Unix(),
	}

	p.logger.Info("map has", "title", title, "description", metadata["description"], "image", metadata["image"], "site_name", metadata["site_name"])

	// Update knok with extracted metadata (if knok repo is available)
	if p.knokRepo != nil {
		knok, err := p.knokRepo.GetByID(ctx, knokID)
		if err != nil {
			return fmt.Errorf("failed to get knok for update: %w", err)
		}

		// Update knok fields with extracted metadata
		if title, ok := metadata["title"].(string); ok {
			knok.Title = &title
		}

		// Update metadata field
		knok.Metadata = map[string]interface{}{
			"extraction_method": "worker_processor",
			"extraction_time":   time.Now().Unix(),
			"raw_metadata":      metadata,
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
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; KnokFM/1.0)")

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

// findOgMetaInNode recursively searches for Open Graph meta tags and collects their content
func (p *JobProcessor) findOgMetaInNode(n *html.Node, ogData map[string]string) {
	if n.Type == html.ElementNode && n.Data == "meta" {
		// Look for <meta property="og:*" content="...">
		var property, content string
		for _, attr := range n.Attr {
			if attr.Key == "property" && strings.HasPrefix(attr.Val, "og:") {
				property = strings.TrimPrefix(attr.Val, "og:")
			} else if attr.Key == "content" {
				content = attr.Val
			}
		}
		if property != "" && content != "" {
			ogData[property] = content
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
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; KnokFM/1.0)")

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
