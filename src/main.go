package main

import (
	"fmt"
	"os"

	"github.com/drawohara/ddollar/src/supervisor"
	"github.com/drawohara/ddollar/src/tokens"
	"github.com/drawohara/ddollar/src/validator"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("ddollar %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	case "validate", "--validate":
		validateTokens()
	default:
		// Everything else is a command to supervise
		superviseCommand(os.Args[1:])
	}
}

func printUsage() {
	fmt.Println(`ddollar - Never hit token limits again

Usage:
  ddollar [--interactive] <command> [args...]
  ddollar --validate                     # Test all tokens

Examples:
  ddollar claude --continue              # All-night AI sessions
  ddollar python train_model.py          # Long-running scripts
  ddollar --interactive node agent.js    # Prompt on limit hit
  ddollar --validate                     # Validate token config

Flags:
  --interactive, -i    Prompt user when limit hit (default: auto-rotate)
  --validate           Test all tokens and show rate limit status
  --help, -h           Show this help
  --version, -v        Show version

How it works:
  1. Monitors rate limits every 60s
  2. When >95% used → SIGTERM → rotate token → restart
  3. Your command's --continue flag picks up where it left off

Supports: Anthropic · OpenAI · Cohere · Google AI`)
}

func superviseCommand(args []string) {
	interactive := false

	// Check for --interactive flag
	if len(args) > 0 && (args[0] == "--interactive" || args[0] == "-i") {
		interactive = true
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("ERROR: No command specified")
		fmt.Println("\nExamples:")
		fmt.Println("  ddollar claude --continue")
		fmt.Println("  ddollar python script.py")
		os.Exit(1)
	}

	// Discover tokens
	fmt.Println("Discovering API tokens...")
	discovered := tokens.Discover()

	if len(discovered) == 0 {
		fmt.Println("ERROR: No API tokens found in environment.")
		fmt.Println("\nSet one or more:")
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
		fmt.Println("ERROR: No providers configured")
		os.Exit(1)
	}

	// Run supervisor
	sup := supervisor.New(pool, args, interactive)
	if err := sup.Run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func validateTokens() {
	// Discover tokens
	fmt.Println("Discovering API tokens...")
	discovered := tokens.Discover()

	if len(discovered) == 0 {
		fmt.Println("ERROR: No API tokens found in environment.")
		fmt.Println("\nSet one or more:")
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
		fmt.Println("ERROR: No providers configured")
		os.Exit(1)
	}

	// Run validation
	if err := validator.Validate(pool); err != nil {
		fmt.Printf("\n%v\n", err)
		os.Exit(1)
	}
}
