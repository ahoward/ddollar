package validator

import (
	"fmt"
	"time"

	"github.com/drawohara/ddollar/src/supervisor"
	"github.com/drawohara/ddollar/src/tokens"
)

// Validate tests all tokens by making a minimal API call to each
func Validate(pool *tokens.Pool) error {
	fmt.Println("\n🔍 Validating tokens...\n")

	totalTokens := pool.TotalTokenCount()
	validTokens := 0
	invalidTokens := 0

	// Create a monitor for making API calls
	monitor := supervisor.NewMonitor(60*time.Second, 0.95)

	// Test each token
	for i := 0; i < totalTokens; i++ {
		token := pool.CurrentToken()
		if token == nil {
			break
		}

		fmt.Printf("[%d/%d] Testing %s token...\n", i+1, totalTokens, token.Provider.Name)

		// Make a test API call
		status, err := testToken(monitor, token)

		if err != nil {
			fmt.Printf("  ✗ FAILED: %v\n\n", err)
			invalidTokens++
		} else {
			fmt.Printf("  ✓ Valid\n")

			// Only show rate limit details if we got data
			if status.RequestsLimit > 0 || status.TokensLimit > 0 {
				if status.RequestsLimit > 0 {
					fmt.Printf("    Requests: %d/%d remaining (%.1f%% used)\n",
						status.RequestsRemaining, status.RequestsLimit, status.RequestsPercentUsed())
				}
				if status.TokensLimit > 0 {
					fmt.Printf("    Tokens:   %d/%d remaining (%.1f%% used)\n",
						status.TokensRemaining, status.TokensLimit, status.TokensPercentUsed())
				}
				if !status.ResetTime.IsZero() {
					fmt.Printf("    Reset:    %s\n", formatDuration(status.TimeUntilReset()))
				}
			} else {
				fmt.Printf("    Rate limits: Will be monitored during supervision\n")
			}

			fmt.Println()
			validTokens++
		}

		// Move to next token
		pool.Next()
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Summary: %d valid, %d invalid, %d total\n", validTokens, invalidTokens, totalTokens)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if invalidTokens > 0 {
		return fmt.Errorf("\n⚠️  %d token(s) failed validation", invalidTokens)
	}

	fmt.Println("\n✓ All tokens validated successfully!")
	return nil
}

// testToken makes a minimal API call to verify the token works
func testToken(monitor *supervisor.Monitor, token *tokens.Token) (*supervisor.RateLimitStatus, error) {
	// Use the existing checkLimits method from monitor
	// This makes a minimal API call and parses rate limit headers
	status, err := monitor.CheckLimitsPublic(token)
	if err != nil {
		return nil, err
	}

	// Check for authentication errors or invalid tokens
	// Note: Some providers may not return rate limit headers on all endpoints
	// We consider the token valid if we got a successful response

	return status, nil
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
