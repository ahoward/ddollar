package hosts

import (
	"fmt"
	"os"
)

// Backup creates a backup of the hosts file
func Backup() error {
	hostsPath := HostsFilePath()
	backupPath := BackupPath()

	// Read original hosts file
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	// Write backup atomically
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// Restore restores the hosts file from backup
func Restore() error {
	hostsPath := HostsFilePath()
	backupPath := BackupPath()

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist")
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Restore hosts file atomically
	if err := os.WriteFile(hostsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore hosts file: %w", err)
	}

	// Remove backup file
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to remove backup: %w", err)
	}

	return nil
}

// HasBackup checks if a backup exists
func HasBackup() bool {
	_, err := os.Stat(BackupPath())
	return err == nil
}
