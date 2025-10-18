package hosts

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	ddollarMarkerStart = "# ddollar START - DO NOT EDIT"
	ddollarMarkerEnd   = "# ddollar END"
)

// Providers is the list of AI provider domains to redirect
var Providers = []string{
	"api.openai.com",
	"api.anthropic.com",
	"api.cohere.ai",
	"generativelanguage.googleapis.com",
}

// Add appends ddollar entries to the hosts file
func Add() error {
	hostsPath := HostsFilePath()

	// Create backup first
	if err := Backup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Read current hosts file
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	content := string(data)

	// Check if ddollar entries already exist
	if strings.Contains(content, ddollarMarkerStart) {
		return fmt.Errorf("ddollar entries already exist in hosts file")
	}

	// Build ddollar entries
	var entries strings.Builder
	entries.WriteString("\n")
	entries.WriteString(ddollarMarkerStart)
	entries.WriteString("\n")
	for _, provider := range Providers {
		entries.WriteString(fmt.Sprintf("127.0.0.1 %s\n", provider))
	}
	entries.WriteString(ddollarMarkerEnd)
	entries.WriteString("\n")

	// Append to hosts file
	newContent := content + entries.String()

	// Write atomically (write to temp, then rename)
	tmpPath := hostsPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, hostsPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to update hosts file: %w", err)
	}

	return nil
}

// Remove removes ddollar entries from the hosts file
func Remove() error {
	hostsPath := HostsFilePath()

	// Read current hosts file
	file, err := os.Open(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to open hosts file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	inDdollarBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, ddollarMarkerStart) {
			inDdollarBlock = true
			continue
		}

		if strings.Contains(line, ddollarMarkerEnd) {
			inDdollarBlock = false
			continue
		}

		if !inDdollarBlock {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan hosts file: %w", err)
	}

	// Write cleaned content
	newContent := strings.Join(lines, "\n")
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}

	// Write atomically
	tmpPath := hostsPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, hostsPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to update hosts file: %w", err)
	}

	return nil
}

// IsActive checks if ddollar entries are present in hosts file
func IsActive() bool {
	hostsPath := HostsFilePath()
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), ddollarMarkerStart)
}
