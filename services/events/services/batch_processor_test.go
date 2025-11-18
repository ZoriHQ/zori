package services

import (
	"testing"

	trieclassifier "github.com/ZoriHQ/trie-url-classifier"
)

func TestURLClassification(t *testing.T) {
	urls := []string{
		"/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics",
		"/projects/a1b2c3d4-e5f6-7890-abcd-ef1234567890/analytics",
		"/projects/12345678-1234-1234-1234-123456789012/settings",
	}

	classifier := trieclassifier.NewClassifier()
	classifier.Learn(urls)

	pattern1 := classifier.Classify("/projects/d381b052-99eb-40f2-9ede-9bce790faae1/analytics")
	if pattern1 != "/projects/{uuid}/analytics" {
		t.Errorf("Expected pattern '/projects/{uuid}/analytics', got '%s'", pattern1)
	}

	pattern2 := classifier.Classify("/projects/ffffffff-ffff-ffff-ffff-ffffffffffff/settings")
	if pattern2 != "/projects/{uuid}/settings" {
		t.Errorf("Expected pattern '/projects/{uuid}/settings', got '%s'", pattern2)
	}

	pattern3 := classifier.Classify("/projects/99999999-9999-9999-9999-999999999999/dashboard")
	if pattern3 != "/projects/{uuid}/dashboard" {
		t.Errorf("Expected pattern '/projects/{uuid}/dashboard', got '%s'", pattern3)
	}
}

func TestBatchProcessorConstants(t *testing.T) {
	if defaultBatchSize != 10 {
		t.Errorf("Expected default batch size to be 10, got %d", defaultBatchSize)
	}
}
