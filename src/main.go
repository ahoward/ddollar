package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/drawohara/ddollar/src/hosts"
	"github.com/drawohara/ddollar/src/proxy"
	"github.com/drawohara/ddollar/src/tokens"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "start":
		startCommand()
	case "stop":
		stopCommand()
	case "status":
		statusCommand()
	case "version", "--version", "-v":
		versionCommand()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ddollar - DDoS for tokens

Usage:
  ddollar start    Start the proxy and configure DNS
  ddollar stop     Stop the proxy and restore DNS
  ddollar status   Show proxy and token status
  ddollar version  Show version information
  ddollar help     Show this help message

Examples:
  ddollar start       # Start with auto-discovered tokens
  ddollar status      # Check if proxy is running
  ddollar stop        # Stop and cleanup

Note: Requires sudo/administrator privileges for DNS modification.`)
}

func versionCommand() {
	fmt.Printf("ddollar %s\n", version)
}

func statusCommand() {
	// Check if hosts file is modified
	isActive := hosts.IsActive()

	fmt.Println("ddollar status:")
	fmt.Printf("  Hosts file modified: %v\n", isActive)

	// Discover tokens
	discovered := tokens.Discover()
	if len(discovered) == 0 {
		fmt.Println("  Tokens discovered: 0")
		fmt.Println("\nNo API tokens found in environment variables.")
		fmt.Println("Set one or more of the following:")
		for _, p := range tokens.SupportedProviders {
			for _, envVar := range p.EnvVars {
				fmt.Printf("  - %s\n", envVar)
			}
		}
		return
	}

	totalTokens := 0
	for _, pt := range discovered {
		totalTokens += len(pt.Tokens)
	}

	fmt.Printf("  Tokens discovered: %d\n", totalTokens)
	fmt.Printf("  Providers configured: %d\n", len(discovered))

	fmt.Println("\nConfigured providers:")
	for _, pt := range discovered {
		fmt.Printf("  - %s: %d token(s)\n", pt.Provider.Name, len(pt.Tokens))
	}

	if !isActive {
		fmt.Println("\nProxy is not running. Use 'ddollar start' to begin.")
	}
}

func startCommand() {
	log.SetFlags(log.Ltime)

	fmt.Println("Starting ddollar...")

	// Discover tokens
	fmt.Println("Discovering API tokens...")
	discovered := tokens.Discover()

	if len(discovered) == 0 {
		fmt.Println("ERROR: No API tokens found in environment variables.")
		fmt.Println("\nPlease set one or more of the following environment variables:")
		for _, p := range tokens.SupportedProviders {
			for _, envVar := range p.EnvVars {
				fmt.Printf("  export %s=your-token-here\n", envVar)
			}
		}
		os.Exit(1)
	}

	// Create token pool
	pool := tokens.NewPool()
	for _, pt := range discovered {
		if err := pool.AddProvider(pt.Provider, pt.Tokens); err != nil {
			fmt.Printf("Warning: Failed to add provider %s: %v\n", pt.Provider.Name, err)
			continue
		}
		fmt.Printf("  ✓ Loaded %d token(s) for %s\n", len(pt.Tokens), pt.Provider.Name)
	}

	if pool.ProviderCount() == 0 {
		fmt.Println("ERROR: No providers configured successfully.")
		os.Exit(1)
	}

	// Generate certificate if needed
	if !proxy.HasCert() {
		fmt.Println("\nGenerating self-signed certificate...")
		certPath, keyPath, err := proxy.GenerateCert()
		if err != nil {
			fmt.Printf("ERROR: Failed to generate certificate: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  ✓ Certificate created: %s\n", certPath)
		fmt.Printf("  ✓ Private key created: %s\n", keyPath)
		fmt.Println("\n⚠️  IMPORTANT: You need to trust the certificate for HTTPS to work.")
		fmt.Println("See README.md for platform-specific instructions.")
	}

	// Modify hosts file
	fmt.Println("\nModifying hosts file (requires sudo)...")
	if hosts.IsActive() {
		fmt.Println("  Hosts file already modified (ddollar may already be running)")
	} else {
		if err := hosts.Add(); err != nil {
			fmt.Printf("ERROR: Failed to modify hosts file: %v\n", err)
			fmt.Println("\nMake sure you run ddollar with sudo:")
			fmt.Println("  sudo ddollar start")
			os.Exit(1)
		}
		fmt.Println("  ✓ Hosts file modified")
	}

	// Start proxy server
	fmt.Println("\nStarting HTTPS proxy on port 443...")
	server := proxy.NewServer(pool, 443)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived shutdown signal...")
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}

		// Restore hosts file
		fmt.Println("Restoring hosts file...")
		if err := hosts.Remove(); err != nil {
			log.Printf("Error restoring hosts file: %v", err)
		}
		fmt.Println("ddollar stopped.")
		os.Exit(0)
	}()

	// Start server (blocks until stopped)
	if err := server.Start(); err != nil {
		fmt.Printf("ERROR: Failed to start server: %v\n", err)
		fmt.Println("\nMake sure:")
		fmt.Println("  1. You run ddollar with sudo (port 443 requires privileges)")
		fmt.Println("  2. No other service is using port 443")
		fmt.Println("  3. You've trusted the certificate (see README.md)")

		// Cleanup hosts file on error
		hosts.Remove()
		os.Exit(1)
	}
}

func stopCommand() {
	fmt.Println("Stopping ddollar...")

	// Check if active
	if !hosts.IsActive() {
		fmt.Println("ddollar is not running (hosts file not modified).")
		return
	}

	// Restore hosts file
	fmt.Println("Restoring hosts file...")
	if err := hosts.Remove(); err != nil {
		fmt.Printf("ERROR: Failed to restore hosts file: %v\n", err)
		fmt.Println("\nYou may need to manually edit your hosts file:")
		fmt.Printf("  %s\n", hosts.HostsFilePath())
		os.Exit(1)
	}

	fmt.Println("✓ Hosts file restored")
	fmt.Println("\nNote: If the proxy is still running, stop it with Ctrl+C")
	fmt.Println("or find and kill the process:")
	fmt.Println("  ps aux | grep ddollar")
}
