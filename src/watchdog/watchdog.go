package watchdog

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/drawohara/ddollar/src/hosts"
)

// Start spawns a detached watchdog process that monitors the parent process
// and cleans up /etc/hosts when the parent exits (for any reason, including kill -9)
func Start() error {
	// Get current process PID (this will be the parent PID for the watchdog)
	parentPID := os.Getpid()

	// Get path to current executable
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Spawn watchdog as a detached child process
	// The watchdog will run the same binary with a special flag
	cmd := exec.Command(exe, "__watchdog__", strconv.Itoa(parentPID))

	// Detach from parent process
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start process in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start watchdog: %w", err)
	}

	// Detach - don't wait for watchdog to finish
	go func() {
		cmd.Wait()
	}()

	log.Printf("Watchdog started (PID %d) monitoring parent (PID %d)", cmd.Process.Pid, parentPID)
	return nil
}

// RunWatchdog is the main loop for the watchdog process
// It monitors the parent PID and cleans up when parent exits
func RunWatchdog(parentPID int) {
	// Set up logging to a file so we can debug watchdog issues
	logFile, err := os.OpenFile("/tmp/ddollar-watchdog.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Printf("Watchdog started, monitoring parent PID %d", parentPID)

	// Check parent status every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check if parent process is still alive by checking /proc filesystem
		// This is more reliable than sending signals, especially across privilege boundaries
		procPath := fmt.Sprintf("/proc/%d", parentPID)
		if _, err := os.Stat(procPath); os.IsNotExist(err) {
			// Parent process no longer exists
			log.Printf("Parent process %d no longer exists, cleaning up", parentPID)
			cleanup()
			return
		}
	}
}

// cleanup removes ddollar entries from /etc/hosts
func cleanup() {
	log.Println("Watchdog: Cleaning up /etc/hosts...")

	if err := hosts.Remove(); err != nil {
		log.Printf("Watchdog: Failed to clean up hosts file: %v", err)
		// Don't exit - try to provide manual instructions
		fmt.Fprintf(os.Stderr, "ERROR: Watchdog failed to clean up /etc/hosts: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please manually edit %s and remove ddollar entries\n", hosts.HostsFilePath())
	} else {
		log.Println("Watchdog: Successfully cleaned up /etc/hosts")
	}
}
