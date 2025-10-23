package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/drawohara/ddollar/src/hosts"
	"github.com/drawohara/ddollar/src/proxy"
	"github.com/drawohara/ddollar/src/supervisor"
	"github.com/drawohara/ddollar/src/tokens"
	"github.com/drawohara/ddollar/src/watchdog"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle watchdog mode (internal use only)
	if command == "__watchdog__" {
		if len(os.Args) < 3 {
			fmt.Println("ERROR: watchdog requires parent PID")
			os.Exit(1)
		}
		parentPID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Printf("ERROR: invalid parent PID: %v\n", err)
			os.Exit(1)
		}
		watchdog.RunWatchdog(parentPID)
		return
	}

	switch command {
	case "start":
		startCommand()
	case "stop":
		stopCommand()
	case "status":
		statusCommand()
	case "supervise":
		superviseCommand()
	case "trust":
		trustCommand()
	case "untrust":
		untrustCommand()
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
  ddollar start                        Start the proxy and configure DNS
  ddollar stop                         Stop the proxy and restore DNS
  ddollar status                       Show proxy, certificate, and token status
  ddollar supervise [--interactive] -- <command>
                                       Run command with automatic token rotation
  ddollar trust                        Manually install SSL certificate trust
  ddollar untrust                      Remove SSL certificate trust
  ddollar version                      Show version information
  ddollar help                         Show this help message

Examples:
  # Proxy mode (intercepts all apps):
  sudo ddollar start                   # Start proxy with auto-trust
  ddollar status                       # Check if proxy is running
  ddollar stop                         # Stop and cleanup

  # Supervisor mode (long-running sessions):
  ddollar supervise -- claude --continue
                                       # Run Claude with auto token rotation
  ddollar supervise --interactive -- claude --continue
                                       # Prompt on limit hit

Note: Proxy mode requires sudo for DNS modification and SSL trust.`)
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
	} else {
		totalTokens := 0
		for _, pt := range discovered {
			totalTokens += len(pt.Tokens)
		}
		fmt.Printf("  Tokens discovered: %d\n", totalTokens)
		fmt.Printf("  Providers configured: %d\n", len(discovered))
	}

	// Check certificate status
	fmt.Println("\nCertificates:")
	certPath, _, err := proxy.CertPaths()
	if err != nil {
		fmt.Printf("  Error getting cert paths: %v\n", err)
	} else {
		// Check if CA exists
		ca, caErr := proxy.EnsureCA()
		if caErr != nil {
			fmt.Println("  CA certificate: Not created")
			fmt.Println("  Run 'sudo ddollar start' to initialize")
		} else {
			fmt.Printf("  CA certificate: %s\n", ca.RootCAPath)

			// Check if trusted
			trustErr := proxy.VerifyTrust(ca)
			if trustErr == nil {
				fmt.Println("  CA trusted: ✓ Yes")
			} else {
				fmt.Println("  CA trusted: ✗ No")
				fmt.Println("  Action required: Run 'sudo ddollar trust' to install")
			}
		}

		// Check leaf certificate
		info, certErr := proxy.GetCertInfo(certPath)
		if certErr != nil {
			fmt.Println("  Leaf certificate: Not generated")
			fmt.Println("  Run 'ddollar start' to create")
		} else {
			fmt.Printf("  Leaf certificate: %s\n", certPath)
			fmt.Printf("  Valid until: %s (%d days remaining)\n",
				info.ValidUntil.Format("2006-01-02"), info.DaysRemaining)
			fmt.Printf("  Domains: %s\n", proxy.FormatDomains(info.Domains))
		}
	}

	// Show providers if tokens exist
	if len(discovered) > 0 {
		fmt.Println("\nConfigured providers:")
		for _, pt := range discovered {
			fmt.Printf("  - %s: %d token(s)\n", pt.Provider.Name, len(pt.Tokens))
		}
	} else {
		fmt.Println("\nNo API tokens found in environment variables.")
		fmt.Println("Set one or more of the following:")
		for _, p := range tokens.SupportedProviders {
			for _, envVar := range p.EnvVars {
				fmt.Printf("  - %s\n", envVar)
			}
		}
	}

	if !isActive {
		fmt.Println("\nProxy is not running. Use 'sudo ddollar start' to begin.")
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

	// Setup SSL certificates
	fmt.Println("\nSetting up SSL certificates...")

	// Create or load CA
	ca, err := proxy.EnsureCA()
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize CA: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Certificate authority ready")

	// Attempt automatic trust installation
	if err := proxy.InstallTrust(ca); err != nil {
		fmt.Printf("  ⚠️  Automatic certificate trust failed: %v\n", err)
		fmt.Println("\nProxy will start anyway, but API calls will fail until certificates are trusted.")
		fmt.Println("To manually trust certificates, run: sudo ddollar trust")
		proxy.PrintManualInstructions()
	} else {
		// Verify trust was successful
		if err := proxy.VerifyTrust(ca); err != nil {
			fmt.Println("  ⚠️  Certificate trust verification failed")
			fmt.Println("  Run 'sudo ddollar trust' to manually install trust")
		} else {
			fmt.Println("  ✓ CA installed to system trust store")
			fmt.Println("  ✓ Certificate authority trusted")
		}
	}

	// Generate leaf certificate
	_, _, err = proxy.GenerateCert()
	if err != nil {
		fmt.Printf("ERROR: Failed to generate certificate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ Generated certificates")

	// Start watchdog process BEFORE modifying hosts file
	// This ensures cleanup happens even if ddollar is killed with kill -9
	fmt.Println("\nStarting cleanup watchdog...")
	if err := watchdog.Start(); err != nil {
		fmt.Printf("  ⚠️  Failed to start watchdog: %v\n", err)
		fmt.Println("  Warning: /etc/hosts may not be cleaned up if ddollar is killed")
	} else {
		fmt.Println("  ✓ Watchdog started (will cleanup /etc/hosts on exit)")
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

func trustCommand() {
	fmt.Println("Installing ddollar CA certificate...")

	// Ensure CA exists
	ca, err := proxy.EnsureCA()
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize CA: %v\n", err)
		os.Exit(1)
	}

	// Check if already trusted
	if err := proxy.VerifyTrust(ca); err == nil {
		fmt.Println("✓ Certificate authority is already trusted")
		fmt.Println("No action needed.")
		return
	}

	// Install trust
	if err := proxy.InstallTrust(ca); err != nil {
		fmt.Printf("❌ Failed to install CA certificate\n\n")
		fmt.Println("ddollar needs administrator privileges to install certificates.")
		fmt.Println("Please run with sudo:")
		fmt.Println("\n  sudo ddollar trust\n")
		fmt.Println("Or manually trust the certificate:")
		proxy.PrintManualInstructions()
		os.Exit(1)
	}

	// Verify installation
	if err := proxy.VerifyTrust(ca); err != nil {
		fmt.Println("⚠️  Trust installation completed but verification failed")
		fmt.Println("The certificate may still work. Check with: ddollar status")
	} else {
		fmt.Println("✓ CA installed to system trust store")
		fmt.Println("\nYour system now trusts ddollar certificates.")
	}
}

func untrustCommand() {
	fmt.Println("Removing ddollar CA certificate...")

	// Ensure CA exists
	ca, err := proxy.EnsureCA()
	if err != nil {
		fmt.Printf("ERROR: Failed to load CA: %v\n", err)
		fmt.Println("CA may not exist. Nothing to remove.")
		return
	}

	// Check if trusted
	if err := proxy.VerifyTrust(ca); err != nil {
		fmt.Println("✓ CA certificate is not installed")
		fmt.Println("No action needed.")
		return
	}

	// Uninstall trust
	if err := proxy.UninstallTrust(ca); err != nil {
		fmt.Printf("❌ Failed to remove CA certificate\n\n")
		fmt.Println("ddollar needs administrator privileges to modify trust stores.")
		fmt.Println("Please run with sudo:")
		fmt.Println("\n  sudo ddollar untrust\n")
		os.Exit(1)
	}

	fmt.Println("✓ CA removed from system trust store")
	fmt.Println("\nddollar certificates are no longer trusted by your system.")
	fmt.Println("\nNote: CA files still exist at ~/.ddollar/ca/")
	fmt.Println("To completely remove ddollar:")
	fmt.Println("  1. Run: rm -rf ~/.ddollar")
	fmt.Println("  2. Uninstall ddollar binary")
}

func superviseCommand() {
	// Parse flags and command
	interactive := false
	args := os.Args[2:]

	// Check for --interactive or -i flag
	if len(args) > 0 && (args[0] == "--interactive" || args[0] == "-i") {
		interactive = true
		args = args[1:]
	}

	// Skip "--" separator if present
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	// Validate command provided
	if len(args) == 0 {
		fmt.Println("ERROR: No command specified")
		fmt.Println("\nUsage: ddollar supervise [--interactive] -- <command>")
		fmt.Println("\nExamples:")
		fmt.Println("  ddollar supervise -- claude --continue")
		fmt.Println("  ddollar supervise --interactive -- python long_running_script.py")
		os.Exit(1)
	}

	// Discover tokens from environment
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
	}

	if pool.ProviderCount() == 0 {
		fmt.Println("ERROR: No providers configured successfully.")
		os.Exit(1)
	}

	// Create and run supervisor
	sup := supervisor.New(pool, args, interactive)
	if err := sup.Run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}
