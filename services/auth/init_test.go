package auth_test

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Load .env.test file before running tests
	if err := godotenv.Load("../../.env.test"); err != nil {
		log.Printf("Warning: Could not load .env.test file: %v", err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
