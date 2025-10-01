package tests

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// init automatically loads .env file for tests if it exists
// This runs before any test functions
func init() {
	// Try to find .env file in the parent directory (audit-data-adapter-go root)
	envPath := "../.env"

	// Check if .env exists
	if _, err := os.Stat(envPath); err == nil {
		// Load .env file
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: Could not load .env file: %v", err)
		} else {
			log.Printf("Loaded .env file for tests from: %s", envPath)
		}
	} else {
		// Also try current directory (in case tests are run from tests/)
		currentDirEnv := ".env"
		if _, err := os.Stat(currentDirEnv); err == nil {
			if err := godotenv.Load(currentDirEnv); err != nil {
				log.Printf("Warning: Could not load .env file: %v", err)
			} else {
				log.Printf("Loaded .env file for tests from: %s", currentDirEnv)
			}
		} else {
			// Try to find .env in project root
			if projectRoot := findProjectRoot(); projectRoot != "" {
				rootEnvPath := filepath.Join(projectRoot, ".env")
				if _, err := os.Stat(rootEnvPath); err == nil {
					if err := godotenv.Load(rootEnvPath); err != nil {
						log.Printf("Warning: Could not load .env file: %v", err)
					} else {
						log.Printf("Loaded .env file for tests from: %s", rootEnvPath)
					}
				}
			}
		}
	}
}

// findProjectRoot tries to find the project root by looking for go.mod
func findProjectRoot() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up directories looking for go.mod
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parent
	}

	return ""
}