package unit

import (
	"os"
	"testing"

	"github.com/drawohara/ddollar/src/tokens"
)

func TestTokenDiscovery(t *testing.T) {
	// Set test environment variables
	os.Setenv("OPENAI_API_KEY", "sk-test-openai")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	discovered := tokens.Discover()

	if len(discovered) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(discovered))
	}

	// Check OpenAI tokens
	foundOpenAI := false
	for _, pt := range discovered {
		if pt.Provider.Name == "OpenAI" {
			foundOpenAI = true
			if len(pt.Tokens) != 1 {
				t.Errorf("Expected 1 OpenAI token, got %d", len(pt.Tokens))
			}
			if pt.Tokens[0] != "sk-test-openai" {
				t.Errorf("Expected token 'sk-test-openai', got '%s'", pt.Tokens[0])
			}
		}
	}

	if !foundOpenAI {
		t.Error("OpenAI provider not discovered")
	}
}

func TestTokenDiscoveryNone(t *testing.T) {
	// Ensure no tokens are set
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("COHERE_API_KEY")
	os.Unsetenv("GOOGLE_API_KEY")

	discovered := tokens.Discover()

	if len(discovered) != 0 {
		t.Errorf("Expected 0 providers when no env vars set, got %d", len(discovered))
	}
}

func TestDiscoverForProvider(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "sk-test-123")
	defer os.Unsetenv("OPENAI_API_KEY")

	tokensFound := tokens.DiscoverForProvider("OpenAI")

	if len(tokensFound) != 1 {
		t.Errorf("Expected 1 token for OpenAI, got %d", len(tokensFound))
	}

	if tokensFound[0] != "sk-test-123" {
		t.Errorf("Expected 'sk-test-123', got '%s'", tokensFound[0])
	}
}

func TestGetProviderByDomain(t *testing.T) {
	provider := tokens.GetProviderByDomain("api.openai.com")

	if provider == nil {
		t.Fatal("Expected provider for api.openai.com, got nil")
	}

	if provider.Name != "OpenAI" {
		t.Errorf("Expected provider name 'OpenAI', got '%s'", provider.Name)
	}

	if provider.AuthHeader != "Authorization" {
		t.Errorf("Expected auth header 'Authorization', got '%s'", provider.AuthHeader)
	}

	unknownProvider := tokens.GetProviderByDomain("api.unknown.com")
	if unknownProvider != nil {
		t.Error("Expected nil for unknown domain")
	}
}
