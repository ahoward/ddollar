package unit

import (
	"testing"

	"github.com/drawohara/ddollar/src/tokens"
)

func TestTokenPoolRoundRobin(t *testing.T) {
	pool := tokens.NewPool()

	provider := &tokens.Provider{
		Name:       "TestProvider",
		Domain:     "api.test.com",
		AuthHeader: "Authorization",
		AuthPrefix: "Bearer ",
	}

	testTokens := []string{"token1", "token2", "token3"}

	err := pool.AddProvider(provider, testTokens)
	if err != nil {
		t.Fatalf("Failed to add provider: %v", err)
	}

	// Test round-robin rotation
	for i := 0; i < 6; i++ {
		token, _, err := pool.GetToken("api.test.com")
		if err != nil {
			t.Fatalf("Failed to get token: %v", err)
		}

		expectedToken := testTokens[i%3]
		if token != expectedToken {
			t.Errorf("Expected token %s, got %s (iteration %d)", expectedToken, token, i)
		}
	}
}

func TestTokenPoolMultipleProviders(t *testing.T) {
	pool := tokens.NewPool()

	provider1 := &tokens.Provider{
		Name:       "Provider1",
		Domain:     "api.provider1.com",
		AuthHeader: "Authorization",
		AuthPrefix: "Bearer ",
	}

	provider2 := &tokens.Provider{
		Name:       "Provider2",
		Domain:     "api.provider2.com",
		AuthHeader: "x-api-key",
		AuthPrefix: "",
	}

	pool.AddProvider(provider1, []string{"token1a", "token1b"})
	pool.AddProvider(provider2, []string{"token2a", "token2b", "token2c"})

	if pool.ProviderCount() != 2 {
		t.Errorf("Expected 2 providers, got %d", pool.ProviderCount())
	}

	if pool.TokenCount() != 5 {
		t.Errorf("Expected 5 total tokens, got %d", pool.TokenCount())
	}

	// Test provider1 tokens
	token1, _, _ := pool.GetToken("api.provider1.com")
	token2, _, _ := pool.GetToken("api.provider1.com")

	if token1 != "token1a" || token2 != "token1b" {
		t.Errorf("Provider1 rotation failed")
	}

	// Test provider2 tokens
	token3, _, _ := pool.GetToken("api.provider2.com")
	if token3 != "token2a" {
		t.Errorf("Provider2 first token should be token2a, got %s", token3)
	}
}

func TestTokenPoolNoTokensError(t *testing.T) {
	pool := tokens.NewPool()

	_, _, err := pool.GetToken("api.unknown.com")
	if err == nil {
		t.Error("Expected error for unknown domain, got nil")
	}
}

func TestTokenPoolEmptyTokensError(t *testing.T) {
	pool := tokens.NewPool()

	provider := &tokens.Provider{
		Name:   "TestProvider",
		Domain: "api.test.com",
	}

	err := pool.AddProvider(provider, []string{})
	if err == nil {
		t.Error("Expected error when adding provider with no tokens")
	}
}
