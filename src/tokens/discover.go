package tokens

import (
	"bufio"
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
		tokens := discoverProviderTokens(&provider)

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

// discoverProviderTokens finds all tokens for a specific provider
func discoverProviderTokens(provider *Provider) []string {
	var tokens []string
	seen := make(map[string]bool)

	// Helper to add token if not already seen
	addToken := func(token string) {
		token = strings.TrimSpace(token)
		if token != "" && !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}

	// 1. Check primary env var (e.g., ANTHROPIC_API_KEY)
	if len(provider.EnvVars) > 0 {
		primaryVar := provider.EnvVars[0]
		if value := os.Getenv(primaryVar); value != "" {
			addToken(value)
		}

		// 2. Check for comma-separated list (e.g., ANTHROPIC_API_KEYS)
		pluralVar := primaryVar + "S"
		if value := os.Getenv(pluralVar); value != "" {
			for _, token := range strings.Split(value, ",") {
				addToken(token)
			}
		}

		// 3. Check for file with tokens (e.g., ANTHROPIC_API_KEYS_FILE)
		fileVar := primaryVar + "S_FILE"
		if filePath := os.Getenv(fileVar); filePath != "" {
			if fileTokens := readTokensFromFile(filePath); len(fileTokens) > 0 {
				for _, token := range fileTokens {
					addToken(token)
				}
			}
		}
	}

	// 4. Check all other env var aliases (for backwards compatibility)
	for _, envVar := range provider.EnvVars {
		if value := os.Getenv(envVar); value != "" {
			addToken(value)
		}
	}

	return tokens
}

// readTokensFromFile reads tokens from a file, one per line
func readTokensFromFile(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			tokens = append(tokens, line)
		}
	}

	return tokens
}

// DiscoverForProvider finds tokens for a specific provider by name
func DiscoverForProvider(providerName string) []string {
	for _, provider := range SupportedProviders {
		if strings.EqualFold(provider.Name, providerName) {
			return discoverProviderTokens(&provider)
		}
	}
	return nil
}
