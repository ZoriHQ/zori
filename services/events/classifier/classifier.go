package classifier

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	ctaPatterns = []string{
		"sign up", "signup", "register", "get started", "start free",
		"buy now", "purchase", "add to cart", "checkout", "order",
		"subscribe", "join", "download", "try free", "free trial",
		"contact", "request", "schedule", "book", "reserve",
		"learn more", "get", "claim", "unlock", "upgrade",
		"submit", "send", "apply", "enroll",
	}

	navPatterns = []string{
		"nav", "menu", "header", "sidebar", "footer",
		"breadcrumb", "navigation", "navbar",
	}

	formControlPatterns = []string{
		"input", "select", "textarea", "checkbox", "radio",
	}

	downloadPatterns = []string{
		".pdf", ".zip", ".doc", ".docx", ".xls", ".xlsx",
		".ppt", ".pptx", ".csv", ".txt", ".xml", ".json",
	}
)

type Classifier struct {
	currentHost string
}

func NewClassifier(currentHost string) *Classifier {
	return &Classifier{
		currentHost: currentHost,
	}
}

func (c *Classifier) Classify(tag, selector, text string) *Classification {
	classification := &Classification{
		ElementType:    c.classifyElementType(tag, selector),
		IsCTAClick:     c.isCTA(text, selector),
		IsExternalLink: false,
		IsDownloadLink: false,
	}

	switch classification.ElementType {
	case ElementTypeLink:
		c.classifyLink(selector, text, classification)
	case ElementTypeImage:
		c.classifyImage(selector, classification)
	case ElementTypeVideo:
		c.classifyVideo(selector, classification)
	}

	classification.ElementCategory = c.determineCategory(classification)

	return classification
}

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
		if c.containsAnyPattern(selector, navPatterns) {
			return ElementTypeNavigation
		}
		return ElementTypeTextElement
	}

	if c.containsAnyPattern(selector, formControlPatterns) {
		return ElementTypeFormControl
	}

	if c.containsAnyPattern(selector, navPatterns) {
		return ElementTypeNavigation
	}

	return ElementTypeOther
}

func (c *Classifier) isCTA(text, selector string) bool {
	text = strings.ToLower(text)
	selector = strings.ToLower(selector)

	for _, pattern := range ctaPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	ctaSelectorPatterns := []string{"cta", "call-to-action", "btn-primary", "btn-cta"}
	for _, pattern := range ctaSelectorPatterns {
		if strings.Contains(selector, pattern) {
			return true
		}
	}

	return false
}

func (c *Classifier) classifyLink(selector, text string, classification *Classification) {
	href := c.extractHref(selector)
	if href != "" {
		classification.LinkDestination = &href

		if c.isExternalURL(href) {
			classification.IsExternalLink = true
		}

		if c.isDownloadLink(href) {
			classification.IsDownloadLink = true
		}
	}

	ariaLabel := c.extractAriaLabel(selector)
	if ariaLabel != "" {
		classification.AriaLabel = &ariaLabel
	}
}

func (c *Classifier) classifyImage(selector string, classification *Classification) {
	altText := c.extractAltText(selector)
	if altText != "" {
		classification.ImageAlt = &altText
	}
}

func (c *Classifier) classifyVideo(selector string, classification *Classification) {
	src := c.extractSrc(selector)
	if src != "" {
		classification.VideoSource = &src
	}
}

func (c *Classifier) determineCategory(classification *Classification) ElementCategory {
	if classification.IsCTAClick {
		return ElementCategoryCTA
	}

	switch classification.ElementType {
	case ElementTypeButton:
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

func (c *Classifier) containsAnyPattern(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func (c *Classifier) extractHref(selector string) string {
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
	if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
		return false
	}

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

	if strings.Contains(href, "download=") {
		return true
	}

	return false
}
