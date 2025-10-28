package classifier

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	// CTA text patterns (case-insensitive)
	ctaPatterns = []string{
		"sign up", "signup", "register", "get started", "start free",
		"buy now", "purchase", "add to cart", "checkout", "order",
		"subscribe", "join", "download", "try free", "free trial",
		"contact", "request", "schedule", "book", "reserve",
		"learn more", "get", "claim", "unlock", "upgrade",
		"submit", "send", "apply", "enroll",
	}

	// Navigation patterns in selectors
	navPatterns = []string{
		"nav", "menu", "header", "sidebar", "footer",
		"breadcrumb", "navigation", "navbar",
	}

	// Form control selectors
	formControlPatterns = []string{
		"input", "select", "textarea", "checkbox", "radio",
	}

	// Download link patterns
	downloadPatterns = []string{
		".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx",
		".ppt", ".pptx", ".csv", ".txt", ".xml", ".json",
	}
)

// Classifier classifies click elements based on their properties
type Classifier struct {
	currentHost string
}

// NewClassifier creates a new element classifier
func NewClassifier(currentHost string) *Classifier {
	return &Classifier{
		currentHost: currentHost,
	}
}

// Classify analyzes a click element and returns its classification
func (c *Classifier) Classify(tag, selector, text string) *Classification {
	classification := &Classification{
		ElementType:     c.classifyElementType(tag, selector),
		IsCTAClick:      c.isCTA(text, selector),
		IsExternalLink:  false,
		IsDownloadLink:  false,
	}

	// Extract additional metadata based on element type
	switch classification.ElementType {
	case ElementTypeLink:
		c.classifyLink(selector, text, classification)
	case ElementTypeImage:
		c.classifyImage(selector, classification)
	case ElementTypeVideo:
		c.classifyVideo(selector, classification)
	}

	// Determine category
	classification.ElementCategory = c.determineCategory(classification)

	return classification
}

// classifyElementType determines the specific element type
func (c *Classifier) classifyElementType(tag, selector string) ElementType {
	tag = strings.ToLower(tag)
	selector = strings.ToLower(selector)

	switch tag {
	case "button":
		return ElementTypeButton
	case "a":
		return ElementTypeLink
	case "input", "select", "textarea":
		return ElementTypeInput
	case "img":
		return ElementTypeImage
	case "video":
		return ElementTypeVideo
	case "p", "span", "div", "h1", "h2", "h3", "h4", "h5", "h6":
		// Check if it's in navigation
		if c.containsAnyPattern(selector, navPatterns) {
			return ElementTypeNavigation
		}
		return ElementTypeTextElement
	}

	// Check selector patterns
	if c.containsAnyPattern(selector, formControlPatterns) {
		return ElementTypeFormControl
	}

	if c.containsAnyPattern(selector, navPatterns) {
		return ElementTypeNavigation
	}

	return ElementTypeOther
}

// isCTA checks if the element is likely a call-to-action
func (c *Classifier) isCTA(text, selector string) bool {
	text = strings.ToLower(text)
	selector = strings.ToLower(selector)

	// Check text content
	for _, pattern := range ctaPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	// Check selector for CTA indicators
	ctaSelectorPatterns := []string{"cta", "call-to-action", "btn-primary", "btn-cta"}
	for _, pattern := range ctaSelectorPatterns {
		if strings.Contains(selector, pattern) {
			return true
		}
	}

	return false
}

// classifyLink extracts link-specific information
func (c *Classifier) classifyLink(selector, text string, classification *Classification) {
	// Try to extract href from selector
	href := c.extractHref(selector)
	if href != "" {
		classification.LinkDestination = &href

		// Check if external
		if c.isExternalURL(href) {
			classification.IsExternalLink = true
		}

		// Check if download
		if c.isDownloadLink(href) {
			classification.IsDownloadLink = true
		}
	}

	// Extract aria-label if present
	ariaLabel := c.extractAriaLabel(selector)
	if ariaLabel != "" {
		classification.AriaLabel = &ariaLabel
	}
}

// classifyImage extracts image-specific information
func (c *Classifier) classifyImage(selector string, classification *Classification) {
	// Try to extract alt text
	altText := c.extractAltText(selector)
	if altText != "" {
		classification.ImageAlt = &altText
	}
}

// classifyVideo extracts video-specific information
func (c *Classifier) classifyVideo(selector string, classification *Classification) {
	// Try to extract video source
	src := c.extractSrc(selector)
	if src != "" {
		classification.VideoSource = &src
	}
}

// determineCategory determines the high-level category
func (c *Classifier) determineCategory(classification *Classification) ElementCategory {
	// CTA takes precedence
	if classification.IsCTAClick {
		return ElementCategoryCTA
	}

	switch classification.ElementType {
	case ElementTypeButton:
		// Buttons are often CTAs, but if not marked as CTA, could be form or other
		return ElementCategoryOther
	case ElementTypeLink:
		if classification.IsExternalLink {
			return ElementCategoryNavigation
		}
		return ElementCategoryNavigation
	case ElementTypeNavigation:
		return ElementCategoryNavigation
	case ElementTypeInput, ElementTypeFormControl:
		return ElementCategoryForm
	case ElementTypeImage, ElementTypeVideo:
		return ElementCategoryMedia
	case ElementTypeTextElement:
		return ElementCategoryContent
	default:
		return ElementCategoryOther
	}
}

// Helper methods

func (c *Classifier) containsAnyPattern(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func (c *Classifier) extractHref(selector string) string {
	// Simple regex to extract href from selector
	// Format might be like: a[href="/path"]
	re := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := re.FindStringSubmatch(selector)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (c *Classifier) extractAriaLabel(selector string) string {
	re := regexp.MustCompile(`aria-label=["']([^"']+)["']`)
	matches := re.FindStringSubmatch(selector)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (c *Classifier) extractAltText(selector string) string {
	re := regexp.MustCompile(`alt=["']([^"']+)["']`)
	matches := re.FindStringSubmatch(selector)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (c *Classifier) extractSrc(selector string) string {
	re := regexp.MustCompile(`src=["']([^"']+)["']`)
	matches := re.FindStringSubmatch(selector)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (c *Classifier) isExternalURL(href string) bool {
	// If it's a relative URL, it's internal
	if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
		return false
	}

	// Parse URL and compare host
	parsedURL, err := url.Parse(href)
	if err != nil {
		return false
	}

	return parsedURL.Host != c.currentHost && parsedURL.Host != ""
}

func (c *Classifier) isDownloadLink(href string) bool {
	href = strings.ToLower(href)
	for _, ext := range downloadPatterns {
		if strings.HasSuffix(href, ext) {
			return true
		}
	}

	// Check for download parameter
	if strings.Contains(href, "download=") {
		return true
	}

	return false
}
