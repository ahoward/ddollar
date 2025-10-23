package supervisor

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/drawohara/ddollar/src/tokens"
)

// Supervisor manages a long-running subprocess with automatic token rotation
type Supervisor struct {
	pool        *tokens.Pool
	monitor     *Monitor
	command     []string
	interactive bool
	subprocess  *exec.Cmd
	statusChan  chan *RateLimitStatus
}

// New creates a new supervisor for the given command
func New(pool *tokens.Pool, command []string, interactive bool) *Supervisor {
	return &Supervisor{
		pool:        pool,
		command:     command,
		interactive: interactive,
		monitor:     NewMonitor(60*time.Second, 0.95), // Check every 60s, rotate at 95%
		statusChan:  make(chan *RateLimitStatus),
	}
}

// Run starts the supervisor and manages the subprocess lifecycle
func (s *Supervisor) Run() error {
	log.SetFlags(log.Ltime)

	fmt.Println("Starting supervision mode...")
	fmt.Printf("✓ Loaded %d token(s) across %d provider(s)\n", s.pool.TotalTokenCount(), s.pool.ProviderCount())
	fmt.Println("✓ Monitor started (checking limits every 60s)")

	// Start subprocess with first token
	if err := s.startSubprocess(); err != nil {
		return err
	}

	// Get current token and start monitoring
	currentToken := s.pool.CurrentToken()
	if currentToken == nil {
		return fmt.Errorf("no token available")
	}

	// Start monitor in background
	go s.monitor.Watch(currentToken, s.statusChan)

	// Wait for limit events and subprocess completion
	subprocessDone := make(chan error)
	go func() {
		subprocessDone <- s.subprocess.Wait()
	}()

	for {
		select {
		case status := <-s.statusChan:
			// Rate limit approaching - handle rotation
			s.handleRotation(status)

			// Restart monitoring with new token
			currentToken = s.pool.CurrentToken()
			if currentToken != nil {
				go s.monitor.Watch(currentToken, s.statusChan)
			}

		case err := <-subprocessDone:
			// Subprocess finished
			if err != nil {
				fmt.Printf("\n✗ Process exited with error: %v\n", err)
				return err
			}
			fmt.Println("\n✓ Process completed successfully")
			return nil
		}
	}
}

// startSubprocess launches the command with the current token in ENV
func (s *Supervisor) startSubprocess() error {
	currentToken := s.pool.CurrentToken()
	if currentToken == nil {
		return fmt.Errorf("no token available")
	}

	fmt.Printf("▶  Launching: %s\n\n", strings.Join(s.command, " "))

	s.subprocess = exec.Command(s.command[0], s.command[1:]...)

	// Set environment with current token
	env := os.Environ()
	tokenEnvVar := currentToken.Provider.EnvVars[0] // Use first env var name
	env = append(env, fmt.Sprintf("%s=%s", tokenEnvVar, currentToken.Value))
	s.subprocess.Env = env

	// Connect stdio
	s.subprocess.Stdin = os.Stdin
	s.subprocess.Stdout = os.Stdout
	s.subprocess.Stderr = os.Stderr

	return s.subprocess.Start()
}

// handleRotation manages the token rotation process
func (s *Supervisor) handleRotation(status *RateLimitStatus) {
	fmt.Printf("\n⚠️  Token limit approaching (%d%% used)\n", status.PercentUsed())

	if s.interactive {
		s.promptUser(status)
	} else {
		s.autoRotate()
	}
}

// autoRotate automatically rotates to the next token
func (s *Supervisor) autoRotate() {
	// Check if we have another token available
	nextToken := s.pool.Peek()
	if nextToken == nil {
		s.handleAllTokensExhausted()
		return
	}

	fmt.Println("▶  Auto-rotating to next token...")

	// Gracefully stop subprocess
	if err := s.subprocess.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Error sending SIGTERM: %v", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error)
	go func() {
		done <- s.subprocess.Wait()
	}()

	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(10 * time.Second):
		// Timeout - force kill
		log.Println("Subprocess didn't exit cleanly, forcing kill...")
		s.subprocess.Process.Kill()
		<-done
	}

	// Rotate token
	s.pool.Next()
	currentIndex := s.pool.CurrentIndex()
	totalTokens := s.pool.TotalTokenCount()
	fmt.Printf("▶  Switched to token %d/%d\n", currentIndex+1, totalTokens)

	// Restart subprocess with new token
	if err := s.startSubprocess(); err != nil {
		fmt.Printf("ERROR: Failed to restart subprocess: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Session resumed\n")
}

// handleAllTokensExhausted handles the case when all tokens hit their limits
func (s *Supervisor) handleAllTokensExhausted() {
	fmt.Println("\n⚠️  All tokens exhausted!")

	// For now, estimate reset time (typically 1 minute for rate limits)
	// TODO: Track actual reset times from rate limit headers
	shortestReset := 1 * time.Minute

	if s.interactive {
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("  1) Wait for limits to reset")
		fmt.Println("  2) Exit and save state")

		choice := s.readChoice(1)

		switch choice {
		case 1:
			fmt.Printf("▶  Pausing for limits to reset (approximately %s)...\n", shortestReset)
			time.Sleep(shortestReset)
			s.autoRotate()
		case 2:
			s.gracefulExit()
		}
	} else {
		// Headless mode - wait and retry
		fmt.Printf("▶  Waiting for limits to reset (approximately %s)...\n", shortestReset)
		time.Sleep(shortestReset)
		s.autoRotate()
	}
}

// promptUser presents interactive options when limit is hit
func (s *Supervisor) promptUser(status *RateLimitStatus) {
	fmt.Println("\nWhat would you like to do?")
	fmt.Println("  1) Rotate to next token and continue")
	fmt.Printf("  2) Wait for limit to reset (%s)\n", formatDuration(status.TimeUntilReset()))
	fmt.Println("  3) Exit and save state")
	fmt.Println("  4) Keep going (may hit 429 errors)")

	choice := s.readChoice(1)

	switch choice {
	case 1:
		s.autoRotate()
	case 2:
		s.waitForReset(status)
	case 3:
		s.gracefulExit()
	case 4:
		fmt.Println("▶  Continuing with current token...\n")
	}
}

// waitForReset pauses the subprocess until the rate limit resets
func (s *Supervisor) waitForReset(status *RateLimitStatus) {
	duration := status.TimeUntilReset()
	fmt.Printf("▶  Pausing subprocess for %s...\n", formatDuration(duration))

	// Send SIGTSTP to pause (like Ctrl+Z)
	if err := s.subprocess.Process.Signal(syscall.SIGTSTP); err != nil {
		log.Printf("Error pausing process: %v", err)
		return
	}

	// Wait
	time.Sleep(duration)

	// Resume with SIGCONT
	fmt.Println("▶  Resuming subprocess...")
	if err := s.subprocess.Process.Signal(syscall.SIGCONT); err != nil {
		log.Printf("Error resuming process: %v", err)
	}
}

// gracefulExit stops the subprocess and exits
func (s *Supervisor) gracefulExit() {
	fmt.Println("▶  Stopping subprocess gracefully...")

	if err := s.subprocess.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Error sending SIGTERM: %v", err)
		s.subprocess.Process.Kill()
	} else {
		s.subprocess.Wait()
	}

	fmt.Println("✓ Session saved. Run with --continue to resume.")
	os.Exit(0)
}

// readChoice prompts for user input and returns the choice
func (s *Supervisor) readChoice(defaultChoice int) int {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\nChoice [%d]: ", defaultChoice)

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultChoice
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultChoice
	}

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil {
		return defaultChoice
	}

	return choice
}

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
