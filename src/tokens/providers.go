package tokens

// Provider represents an AI provider configuration
type Provider struct {
	Name       string
	Domain     string
	EnvVars    []string // Environment variables to check for tokens
	AuthHeader string   // HTTP header name for authentication
	AuthPrefix string   // Prefix for the auth value (e.g., "Bearer ")
}

// SupportedProviders is the list of supported AI providers
var SupportedProviders = []Provider{
	{
		Name:       "OpenAI",
		Domain:     "api.openai.com",
		EnvVars:    []string{"OPENAI_API_KEY"},
		AuthHeader: "Authorization",
		AuthPrefix: "Bearer ",
	},
	{
		Name:       "Anthropic",
		Domain:     "api.anthropic.com",
		EnvVars:    []string{"ANTHROPIC_API_KEY"},
		AuthHeader: "x-api-key",
		AuthPrefix: "",
	},
	{
		Name:       "Cohere",
		Domain:     "api.cohere.ai",
		EnvVars:    []string{"COHERE_API_KEY", "CO_API_KEY"},
		AuthHeader: "Authorization",
		AuthPrefix: "Bearer ",
	},
	{
		Name:       "Google AI",
		Domain:     "generativelanguage.googleapis.com",
		EnvVars:    []string{"GOOGLE_AI_API_KEY", "GOOGLE_API_KEY"},
		AuthHeader: "x-goog-api-key",
		AuthPrefix: "",
	},
}

// GetProviderByDomain returns the provider for a given domain
func GetProviderByDomain(domain string) *Provider {
	for _, p := range SupportedProviders {
		if p.Domain == domain {
			return &p
		}
	}
	return nil
}
