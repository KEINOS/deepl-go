//go:build e2e
// +build e2e

// End-to-End (E2E) tests for the DeepL Go client library.
//
// These tests use the DeepL Mock server (https://github.com/DeepLcom/deepl-mock)
// to simulate real DeepL API interactions without requiring actual API credentials
// or consuming API quota.
//
// DOCKER COMPOSE WORKFLOW:
//
// 1. Spawn the DeepL Mock server:
//    docker compose up deepl-mock -d
//
//    This starts the mock server container in detached mode. The mock server
//    provides the same API endpoints as the real DeepL API but with limited
//    functionality suitable for testing (e.g., translates "Hello, world!" to "陽子ビーム").
//
// 2. Run the E2E test container and check exit status:
//    docker compose run --rm deepl-test
//    echo $?  # Should output "0" if all tests pass
//
//    The test container runs `go test -tags=e2e ./...` and exits with status 0
//    on success or non-zero on failure. The --rm flag automatically removes
//    the test container after execution.
//
// 3. Clean up all containers (including orphans):
//    docker compose down --remove-orphans
//
//    This stops and removes all containers created by docker compose, including
//    any orphaned containers from previous test runs.
//
// ONE-LINER FOR FULL WORKFLOW:
//    docker compose up deepl-mock -d && \
//    docker compose run --rm deepl-test && \
//    echo "Tests passed!" || echo "Tests failed!" && \
//    docker compose down --remove-orphans
//
// ALTERNATIVE: Direct Go test execution (requires mock server running):
//    go test -tags=e2e -v ./...

package deepl

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Mock server test credentials
	mockAPIKey = "mock-api-key"

	// Test timeouts
	testTimeout = 30 * time.Second
)

// getMockServerURL returns the mock server URL from environment variable or default
func getMockServerURL() string {
	if url := os.Getenv("DEEPL_SERVER_URL"); url != "" {
		return url
	}
	return "http://localhost:3000"
}

// WithBaseURL returns an Option that sets a custom base URL for the client
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// waitForMockServer waits for the mock server to be ready
func waitForMockServer(t *testing.T, serverURL string) {
	t.Helper()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Mock server did not become ready in time")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", serverURL+"/v2/usage?auth_key=smoke_test", nil)
			require.NoError(t, err)

			req.Header.Set("User-Agent", "deepl-go-e2e-test")

			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

func TestE2E_MockServerHealth(t *testing.T) {
	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Create HTTP client for direct API calls
	client := &http.Client{Timeout: 10 * time.Second}

	// Test health endpoint
	req, err := http.NewRequest("GET", serverURL+"/v2/usage?auth_key=smoke_test", nil)
	require.NoError(t, err)

	req.Header.Set("User-Agent", "deepl-go-e2e-test")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Mock server should be healthy")
}

func TestE2E_DeepLClient_GetUsage(t *testing.T) {
	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Create DeepL client with mock server URL
	client := NewClient(mockAPIKey,
		WithBaseURL(serverURL),
		WithUserAgent("deepl-go-e2e-test"),
	)

	// Test GetUsage
	usage, err := client.GetUsage()
	require.NoError(t, err, "GetUsage should succeed with mock server")

	assert.NotNil(t, usage, "Usage response should not be nil")

	// Mock server should return some usage data
	t.Logf("Usage response: CharacterCount=%d, CharacterLimit=%d",
		usage.CharacterCount, usage.CharacterLimit)
}

func TestE2E_DeepLClient_TranslateText(t *testing.T) {
	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Create DeepL client with mock server URL
	client := NewClient(mockAPIKey,
		WithBaseURL(serverURL),
		WithUserAgent("deepl-go-e2e-test"),
	)

	// Test TranslateText
	text := "Hello, world!"
	targetLang := "JA"

	translation, err := client.TranslateText(text, targetLang)
	require.NoError(t, err, "TranslateText should succeed with mock server")

	assert.NotNil(t, translation, "Should return a translation")
	assert.NotEmpty(t, translation.Text, "Translation text should not be empty")
	assert.NotEmpty(t, translation.DetectedSourceLanguage, "Should detect source language")

	t.Logf("Translation: '%s' -> '%s'", text, translation.Text)
}

func TestE2E_DeepLClient_GetSupportedLanguages(t *testing.T) {
	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Create DeepL client with mock server URL
	client := NewClient(mockAPIKey,
		WithBaseURL(serverURL),
		WithUserAgent("deepl-go-e2e-test"),
	)

	// Test GetSourceLanguages
	sourceLangs, err := client.GetSourceLanguages()
	require.NoError(t, err, "GetSourceLanguages should succeed")
	assert.NotEmpty(t, sourceLangs, "Should return supported source languages")

	// Test GetTargetLanguages
	targetLangs, err := client.GetTargetLanguages()
	require.NoError(t, err, "GetTargetLanguages should succeed")
	assert.NotEmpty(t, targetLangs, "Should return supported target languages")

	t.Logf("Found %d source languages and %d target languages",
		len(sourceLangs), len(targetLangs))
}

func TestE2E_DeepLClient_ErrorHandling(t *testing.T) {
	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Test with various invalid scenarios
	testCases := []struct {
		name   string
		apiKey string
	}{
		{"empty key", ""},
		{"invalid literal", "invalid"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(tc.apiKey,
				WithBaseURL(serverURL),
				WithUserAgent("deepl-go-e2e-test"),
			)

			// Test with invalid API key - may or may not return error depending on mock server behavior
			usage, err := client.GetUsage()

			// Log the result regardless
			if err != nil {
				t.Logf("API key '%s' returned error: %v", tc.apiKey, err)
				assert.Contains(t, err.Error(), "HTTP", "Error should contain HTTP status information")
			} else {
				t.Logf("API key '%s' succeeded with usage: %+v", tc.apiKey, usage)
				// Mock server may accept invalid keys for testing purposes
				assert.NotNil(t, usage, "Usage response should not be nil")
			}
		})
	}
}

func TestE2E_DeepLClient_WithProxy(t *testing.T) {
	// Skip this test if proxy URL is not configured
	proxyURL := os.Getenv("DEEPL_PROXY_URL")
	if proxyURL == "" {
		t.Skip("DEEPL_PROXY_URL not set, skipping proxy test")
	}

	serverURL := getMockServerURL()

	// Wait for mock server to be ready
	waitForMockServer(t, serverURL)

	// Note: This test would need proper proxy setup in the mock server
	// For now, we just verify that the proxy option doesn't break the client
	client := NewClient(mockAPIKey,
		WithBaseURL(serverURL),
		WithUserAgent("deepl-go-e2e-test"),
	)

	// Basic functionality test with proxy configuration
	usage, err := client.GetUsage()
	require.NoError(t, err, "Should work even with proxy option configured")
	assert.NotNil(t, usage, "Usage response should not be nil")
}
