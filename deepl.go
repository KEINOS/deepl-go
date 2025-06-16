package deepl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL     = "https://api.deepl.com"
	baseURLFree = "https://api-free.deepl.com"
	version     = "0.3.0"
)

type retryPolicy struct {
	MaxRetries  int
	MaxDelay    time.Duration
	BackoffBase time.Duration
}

var defaultRetryPolicy = retryPolicy{
	MaxRetries:  5,
	MaxDelay:    10 * time.Second,
	BackoffBase: 500 * time.Millisecond,
}

// Client represents a DeepL API client.
type Client struct {
	apiKey      string       // API authentication key
	baseURL     string       // Base URL for API endpoints (depends on API key type)
	userAgent   string       // User-Agent header value sent with requests
	httpClient  *http.Client // Underlying HTTP client used for requests
	retryPolicy retryPolicy  // retryPolicy represents the retry logic configuration including maximum retries and maximum delay duration.
}

// Option defines a functional option for configuring the DeepL Client.
type Option func(c *Client)

// NewClient creates and returns a new DeepL API client with the given API key and optional configurations.
// If options are provided, they will be applied to the client.
func NewClient(apiKey string, opts ...Option) *Client {
	client := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL:     getBaseURL(apiKey),
		userAgent:   "deepl-go/" + version,
		retryPolicy: defaultRetryPolicy,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// WithUserAgent returns an Option that sets the User-Agent header for HTTP requests.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithProxy returns an Option that configures the client to use the specified proxy URL.
func WithProxy(proxy url.URL) Option {
	return func(c *Client) {
		c.httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(&proxy),
		}
	}
}

// WithRetryPolicy returns an Option that sets the maximum retry attempts and maximum delay for retrying failed requests.
func WithRetryPolicy(maxRetryAttempts, maxDelaySeconds int) Option {
	return func(c *Client) {
		c.retryPolicy = retryPolicy{
			MaxRetries: maxRetryAttempts,
			MaxDelay:   time.Duration(maxDelaySeconds) * time.Second,
		}
	}
}

// WithTrace returns an Option that enables HTTP request and response logging for debugging.
func WithTrace() Option {
	return func(c *Client) {
		prev := c.httpClient.Transport
		if prev == nil {
			prev = http.DefaultTransport
		}
		c.httpClient.Transport = &loggingRoundTripper{
			Proxied: prev,
		}
	}
}

// doRequest sends an HTTP request using the client's configuration, applies authentication and content headers,
// performs the request with retry logic, and decodes the JSON response body into the provided interface.
// It returns any error encountered during the request or decoding process.
func (c *Client) doRequest(ctx context.Context, req *http.Request, v any) error {
	req.Header.Set("Authorization", fmt.Sprintf("DeepL-Auth-Key %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, respErr := c.performRetryableRequest(ctx, req)

	if respErr != nil {
		return respErr
	}

	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

// performRetryableRequest executes an HTTP request with retry logic based on the client's retry policy.
func (c *Client) performRetryableRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var respErr error

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		cloneReq, err := cloneRequest(req)
		if err != nil {
			return nil, fmt.Errorf("failed to clone request: %w", err)
		}

		cloneReq = cloneReq.WithContext(ctx)
		resp, respErr = c.httpClient.Do(cloneReq)
		shouldRetry, delay := c.shouldRetry(resp, respErr, attempt)
		if !shouldRetry {
			break
		}

		select {
		case <-time.After(delay):
			continue // continue to next attempt
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}
	}

	if respErr != nil {
		return nil, respErr
	}

	if resp.StatusCode != http.StatusOK {
		return nil, createErrorFromResponse(resp)
	}

	return resp, nil
}

// errorResponse represents the error message returned by the DeepL API in JSON format.
type errorResponse struct {
	Message string `json:"message"` // Human-readable error message
}

// createErrorFromResponse generates an error describing the HTTP response including status and message if available.
func createErrorFromResponse(resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()
	statusText := "unknown error"
	if resp.StatusCode == 456 {
		statusText = "character limit has been reached"
	} else if http.StatusText(resp.StatusCode) != "" {
		statusText = strings.ToLower(http.StatusText(resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d %s; error reading the body: %w", resp.StatusCode, statusText, err)
	}

	var errResp errorResponse
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&errResp)
	if err == nil && errResp.Message != "" {
		return fmt.Errorf("HTTP %d %s: %s", resp.StatusCode, statusText, errResp.Message)
	}

	return fmt.Errorf("HTTP %d %s", resp.StatusCode, statusText)
}

// shouldRetry examines the error message and returns true if it's retryable
func (c *Client) shouldRetry(resp *http.Response, err error, attempt int) (shouldRetry bool, delay time.Duration) {
	if err != nil || resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return true, calculateRetryDelay(attempt, c.retryPolicy)
	}
	return false, 0
}

// calculateRetryDelay returns a randomized backoff duration with exponential growth capped at maxDelay.
func calculateRetryDelay(attempt int, policy retryPolicy) time.Duration {
	expDelay := time.Duration(math.Pow(2, float64(attempt))) * policy.BackoffBase
	if expDelay > policy.MaxDelay {
		expDelay = policy.MaxDelay
	}
	// jitter between 0 and expDelay
	return time.Duration(rand.Int63n(int64(expDelay) + 1))
}

// cloneRequest creates a deep copy of the *http.Request including the body.
func cloneRequest(req *http.Request) (*http.Request, error) {
	cloned := req.Clone(req.Context())

	if req.Body == nil || req.Body == http.NoBody {
		return cloned, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	// Reset the original body for potential reuse
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return cloned, nil
}

// getBaseURL returns the appropriate API base URL based on the API key type.
// Free API keys (ending with ":fx") use the free API endpoint.
func getBaseURL(apiKey string) string {
	if strings.HasSuffix(apiKey, ":fx") {
		return baseURLFree
	}
	return baseURL
}

// loggingRoundTripper is an http.RoundTripper that logs HTTP requests and responses.
type loggingRoundTripper struct {
	Proxied http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
// It logs the outgoing HTTP request and the incoming HTTP response for debugging.
func (lrt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Printf("error dumping request: %v", err)
	} else {
		log.Printf("HTTP Request:\n%s", string(reqDump))
	}

	res, err := lrt.Proxied.RoundTrip(req)
	if err != nil {
		log.Printf("error during round trip: %v", err)
		return nil, err
	}

	resDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		log.Printf("error dumping response: %v", err)
	} else {
		log.Printf("HTTP Response:\n%s", string(resDump))
	}

	return res, nil
}

// BoolPtr is a helper function that returns a pointer to a bool value.
func BoolPtr(b bool) *bool {
	return &b
}
