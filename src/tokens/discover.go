package tokens

import (
	"os"
	"strings"
)

// ProviderTokens holds discovered tokens for a provider
type ProviderTokens struct {
	Provider *Provider
	Tokens   []string
}

// Discover scans environment variables for API tokens
func Discover() []ProviderTokens {
	var results []ProviderTokens

	for _, provider := range SupportedProviders {
		var tokens []string

		// Check each environment variable
		for _, envVar := range provider.EnvVars {
			if value := os.Getenv(envVar); value != "" {
				// Check if token is already in the list (dedup)
				if !contains(tokens, value) {
					tokens = append(tokens, value)
				}
			}
		}

		// Only add provider if tokens were found
		if len(tokens) > 0 {
			results = append(results, ProviderTokens{
				Provider: &provider,
				Tokens:   tokens,
			})
		}
	}

	return results
}

// DiscoverForProvider finds tokens for a specific provider by name
func DiscoverForProvider(providerName string) []string {
	for _, provider := range SupportedProviders {
		if strings.EqualFold(provider.Name, providerName) {
			var tokens []string
			for _, envVar := range provider.EnvVars {
				if value := os.Getenv(envVar); value != "" {
					if !contains(tokens, value) {
						tokens = append(tokens, value)
					}
				}
			}
			return tokens
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
