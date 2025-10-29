package supervisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/drawohara/ddollar/src/tokens"
)

// Monitor checks rate limits by making periodic API calls
type Monitor struct {
	interval  time.Duration
	threshold float64 // Rotate when usage exceeds this percentage (0.95 = 95%)
}

// RateLimitStatus represents the current rate limit state
type RateLimitStatus struct {
	RequestsLimit     int
	RequestsRemaining int
	TokensLimit       int
	TokensRemaining   int
	ResetTime         time.Time
	Provider          string
}

// NewMonitor creates a monitor that checks limits at the specified interval
func NewMonitor(interval time.Duration, threshold float64) *Monitor {
	return &Monitor{
		interval:  interval,
		threshold: threshold,
	}
}

// Watch continuously monitors rate limits and sends status updates on the channel
func (m *Monitor) Watch(token *tokens.Token, statusChan chan *RateLimitStatus) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	log.Printf("Monitor: Started watching token for %s (checking every %s)", token.Provider.Name, m.interval)

	for range ticker.C {
		status, err := m.checkLimits(token)
		if err != nil {
			log.Printf("Monitor: Error checking limits: %v", err)
			continue
		}

		log.Printf("Monitor: %s - Requests: %d/%d (%.1f%%), Tokens: %d/%d (%.1f%%)",
			token.Provider.Name,
			status.RequestsLimit-status.RequestsRemaining, status.RequestsLimit, status.RequestsPercentUsed(),
			status.TokensLimit-status.TokensRemaining, status.TokensLimit, status.TokensPercentUsed())

		// Send status if rotation needed
		if status.ShouldRotate(m.threshold) {
			statusChan <- status
		}
	}
}

// CheckLimitsPublic is a public wrapper for checkLimits (used by validator)
func (m *Monitor) CheckLimitsPublic(token *tokens.Token) (*RateLimitStatus, error) {
	return m.checkLimits(token)
}

// checkLimits makes a minimal API call to check rate limit headers
func (m *Monitor) checkLimits(token *tokens.Token) (*RateLimitStatus, error) {
	var resp *http.Response
	var err error

	switch token.Provider.Name {
	case "Anthropic":
		resp, err = m.checkAnthropic(token)
	case "OpenAI":
		resp, err = m.checkOpenAI(token)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", token.Provider.Name)
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for HTTP errors (authentication failures, etc)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: authentication failed or invalid token", resp.StatusCode)
	}

	// Parse provider-specific headers
	status := &RateLimitStatus{Provider: token.Provider.Name}

	if token.Provider.Name == "Anthropic" {
		status.parseAnthropicHeaders(resp.Header)
	} else if token.Provider.Name == "OpenAI" {
		status.parseOpenAIHeaders(resp.Header)
	}

	return status, nil
}

// checkAnthropic makes a minimal API call to Anthropic
func (m *Monitor) checkAnthropic(token *tokens.Token) (*http.Response, error) {
	// Minimal request: 1 token response
	reqBody := map[string]interface{}{
		"model":      "claude-3-5-sonnet-20240620",
		"max_tokens": 1,
		"messages": []map[string]string{
			{"role": "user", "content": "."},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", token.Value)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	return http.DefaultClient.Do(req)
}

// checkOpenAI makes a minimal API call to OpenAI
func (m *Monitor) checkOpenAI(token *tokens.Token) (*http.Response, error) {
	// Minimal request: list models (doesn't consume tokens)
	req, _ := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+token.Value)

	return http.DefaultClient.Do(req)
}

// parseAnthropicHeaders extracts rate limit info from Anthropic response headers
func (s *RateLimitStatus) parseAnthropicHeaders(headers http.Header) {
	s.RequestsLimit = parseInt(headers.Get("anthropic-ratelimit-requests-limit"))
	s.RequestsRemaining = parseInt(headers.Get("anthropic-ratelimit-requests-remaining"))
	s.TokensLimit = parseInt(headers.Get("anthropic-ratelimit-tokens-limit"))
	s.TokensRemaining = parseInt(headers.Get("anthropic-ratelimit-tokens-remaining"))

	// Parse reset time
	resetStr := headers.Get("anthropic-ratelimit-requests-reset")
	if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
		s.ResetTime = resetTime
	}
}

// parseOpenAIHeaders extracts rate limit info from OpenAI response headers
func (s *RateLimitStatus) parseOpenAIHeaders(headers http.Header) {
	s.RequestsLimit = parseInt(headers.Get("x-ratelimit-limit-requests"))
	s.RequestsRemaining = parseInt(headers.Get("x-ratelimit-remaining-requests"))
	s.TokensLimit = parseInt(headers.Get("x-ratelimit-limit-tokens"))
	s.TokensRemaining = parseInt(headers.Get("x-ratelimit-remaining-tokens"))

	// Parse reset time (OpenAI uses duration like "1m23s")
	resetStr := headers.Get("x-ratelimit-reset-requests")
	if duration, err := time.ParseDuration(resetStr); err == nil {
		s.ResetTime = time.Now().Add(duration)
	}
}

// ShouldRotate returns true if usage exceeds the threshold
func (s *RateLimitStatus) ShouldRotate(threshold float64) bool {
	return s.RequestsPercentUsed() > threshold*100 || s.TokensPercentUsed() > threshold*100
}

// RequestsPercentUsed returns the percentage of requests used (0-100)
func (s *RateLimitStatus) RequestsPercentUsed() float64 {
	if s.RequestsLimit == 0 {
		return 0
	}
	used := s.RequestsLimit - s.RequestsRemaining
	return float64(used) / float64(s.RequestsLimit) * 100
}

// TokensPercentUsed returns the percentage of tokens used (0-100)
func (s *RateLimitStatus) TokensPercentUsed() float64 {
	if s.TokensLimit == 0 {
		return 0
	}
	used := s.TokensLimit - s.TokensRemaining
	return float64(used) / float64(s.TokensLimit) * 100
}

// PercentUsed returns the higher of requests or tokens percent used
func (s *RateLimitStatus) PercentUsed() int {
	reqPercent := s.RequestsPercentUsed()
	tokPercent := s.TokensPercentUsed()

	if reqPercent > tokPercent {
		return int(reqPercent)
	}
	return int(tokPercent)
}

// TimeUntilReset returns how long until the rate limit resets
func (s *RateLimitStatus) TimeUntilReset() time.Duration {
	return time.Until(s.ResetTime)
}

// parseInt safely parses a string to int, returning 0 on error
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
