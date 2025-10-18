package hosts

import "runtime"

// HostsFilePath returns the platform-specific path to the hosts file
func HostsFilePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts`
	}
	// macOS and Linux use the same path
	return "/etc/hosts"
}

// BackupPath returns the path for the backup hosts file
func BackupPath() string {
	if runtime.GOOS == "windows" {
		return `C:\Windows\System32\drivers\etc\hosts.ddollar.backup`
	}
	return "/etc/hosts.ddollar.backup"
}
