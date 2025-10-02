package helpers

import (
	"fmt"
	"strings"
	"time"
)

func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Add timestamp to ensure uniqueness
	return fmt.Sprintf("%s-%d", slug, time.Now().Unix())
}
