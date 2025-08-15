package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Get migration version from command line
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/generate/main.go <migration_version> [description]")
	}

	version := os.Args[1]
	description := "Auto-generated migration"
	if len(os.Args) > 2 {
		description = strings.Join(os.Args[2:], " ")
	}

	// Use simple migration generator
	generateSimpleMigration(version, description)

	fmt.Printf("✅ Generated simple migration %s successfully!\n", version)
}
