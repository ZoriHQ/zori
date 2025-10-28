package classifier

import (
	"testing"
)

func TestClassifier_ClassifyButton(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("button", ".btn-primary", "Sign Up Now")

	if result.ElementType != ElementTypeButton {
		t.Errorf("Expected ElementTypeButton, got %s", result.ElementType)
	}

	if !result.IsCTAClick {
		t.Error("Expected IsCTAClick to be true for 'Sign Up Now' button")
	}

	if result.ElementCategory != ElementCategoryCTA {
		t.Errorf("Expected ElementCategoryCTA, got %s", result.ElementCategory)
	}
}

func TestClassifier_ClassifyLink(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("a", "a[href='/about']", "About Us")

	if result.ElementType != ElementTypeLink {
		t.Errorf("Expected ElementTypeLink, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryNavigation {
		t.Errorf("Expected ElementCategoryNavigation, got %s", result.ElementCategory)
	}

	if result.LinkDestination == nil {
		t.Error("Expected LinkDestination to be set")
	} else if *result.LinkDestination != "/about" {
		t.Errorf("Expected LinkDestination '/about', got %s", *result.LinkDestination)
	}

	if result.IsExternalLink {
		t.Error("Expected IsExternalLink to be false for internal link")
	}
}

func TestClassifier_ClassifyExternalLink(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("a", "a[href='https://external.com/page']", "External Link")

	if result.ElementType != ElementTypeLink {
		t.Errorf("Expected ElementTypeLink, got %s", result.ElementType)
	}

	if !result.IsExternalLink {
		t.Error("Expected IsExternalLink to be true for external domain")
	}

	if result.LinkDestination == nil {
		t.Error("Expected LinkDestination to be set")
	}
}

func TestClassifier_ClassifyDownloadLink(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("a", "a[href='/docs/guide.pdf']", "Download PDF")

	if result.ElementType != ElementTypeLink {
		t.Errorf("Expected ElementTypeLink, got %s", result.ElementType)
	}

	if !result.IsDownloadLink {
		t.Error("Expected IsDownloadLink to be true for PDF link")
	}
}

func TestClassifier_ClassifyCTAPatterns(t *testing.T) {
	c := NewClassifier("example.com")

	ctaTexts := []string{
		"Sign Up",
		"Buy Now",
		"Get Started",
		"Download",
		"Subscribe",
		"Try Free",
		"Contact Us",
		"Learn More",
	}

	for _, text := range ctaTexts {
		result := c.Classify("button", ".btn", text)

		if !result.IsCTAClick {
			t.Errorf("Expected IsCTAClick to be true for '%s'", text)
		}

		if result.ElementCategory != ElementCategoryCTA {
			t.Errorf("Expected ElementCategoryCTA for '%s', got %s", text, result.ElementCategory)
		}
	}
}

func TestClassifier_ClassifyNavigation(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("div", ".navbar .menu-item", "Products")

	if result.ElementType != ElementTypeNavigation {
		t.Errorf("Expected ElementTypeNavigation, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryNavigation {
		t.Errorf("Expected ElementCategoryNavigation, got %s", result.ElementCategory)
	}
}

func TestClassifier_ClassifyInput(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("input", "input[type='email']", "")

	if result.ElementType != ElementTypeInput {
		t.Errorf("Expected ElementTypeInput, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryForm {
		t.Errorf("Expected ElementCategoryForm, got %s", result.ElementCategory)
	}
}

func TestClassifier_ClassifyImage(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("img", "img[alt='Product Photo']", "")

	if result.ElementType != ElementTypeImage {
		t.Errorf("Expected ElementTypeImage, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryMedia {
		t.Errorf("Expected ElementCategoryMedia, got %s", result.ElementCategory)
	}

	if result.ImageAlt == nil {
		t.Error("Expected ImageAlt to be set")
	} else if *result.ImageAlt != "Product Photo" {
		t.Errorf("Expected ImageAlt 'Product Photo', got %s", *result.ImageAlt)
	}
}

func TestClassifier_ClassifyVideo(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("video", "video[src='/videos/demo.mp4']", "")

	if result.ElementType != ElementTypeVideo {
		t.Errorf("Expected ElementTypeVideo, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryMedia {
		t.Errorf("Expected ElementCategoryMedia, got %s", result.ElementCategory)
	}
}

func TestClassifier_ClassifyTextElement(t *testing.T) {
	c := NewClassifier("example.com")

	result := c.Classify("div", ".content", "Some text content")

	if result.ElementType != ElementTypeTextElement {
		t.Errorf("Expected ElementTypeTextElement, got %s", result.ElementType)
	}

	if result.ElementCategory != ElementCategoryContent {
		t.Errorf("Expected ElementCategoryContent, got %s", result.ElementCategory)
	}
}

func TestClassifier_CTASelectorPatterns(t *testing.T) {
	c := NewClassifier("example.com")

	// Test CTA detection by selector class names
	result := c.Classify("button", ".btn-cta", "Click")

	if !result.IsCTAClick {
		t.Error("Expected IsCTAClick to be true for .btn-cta selector")
	}
}

func TestClassifier_NoClickElement(t *testing.T) {
	c := NewClassifier("example.com")

	// Test with empty values
	result := c.Classify("", "", "")

	if result.ElementType != ElementTypeOther {
		t.Errorf("Expected ElementTypeOther for empty tag, got %s", result.ElementType)
	}
}
