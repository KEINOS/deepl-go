package deepl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	version     = "0.2.0"
)

type retryPolicy struct {
	MaxRetries int
	MaxDelay   time.Duration
}

var defaultRetryPolicy = retryPolicy{
	MaxRetries: 5,
	MaxDelay:   10 * time.Second,
}

// httpErrorMessages maps HTTP status codes to human-readable error messages.
var httpErrorMessages = map[int]string{
	400: "Bad request. Please check error message and your parameters.",
	403: "Authorization failed. Please supply a valid auth_key parameter.",
	404: "The requested resource could not be found.",
	413: "The request size exceeds the limit.",
	414: "The request URL is too long. You can avoid this error by using a POST request instead of a GET request, and sending the parameters in the HTTP body.",
	429: "Too many requests. Please wait and resend your request.",
	456: "Quota exceeded. The character limit has been reached.",
	500: "Internal server error.",
	503: "Resource currently unavailable. Try again later.",
	529: "Too many requests. Please wait and resend your request.",
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
func NewClient(apiKey string, opts ...func(c *Client)) *Client {
	client := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
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

// errorResponse represents the error message returned by the DeepL API in JSON format.
type errorResponse struct {
	Message string `json:"message"` // Human-readable error message
}

// sendRequestWithRetry wraps sendRequest adding retry logic for 429 and 503 errors.
func (c *Client) sendRequestWithRetry(ctx context.Context, req *http.Request, v interface{}) error {
	var lastErr error
	backoff := 500 * time.Millisecond // initial backoff

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		clonedReq, err := cloneRequest(req)
		if err != nil {
			return fmt.Errorf("failed to clone request: %w", err)
		}

		err = c.sendRequest(clonedReq, v)
		if err == nil {
			return nil
		}

		if !c.shouldRetry(err) {
			return err
		}

		lastErr = err

		// If last attempt, break
		if attempt == c.retryPolicy.MaxRetries {
			break
		}

		// Calculate delay with binary exponential backoff + jitter
		delay := c.calculateRetryDelay(attempt, backoff)

		select {
		case <-time.After(delay):
			backoff = delay // update backoff to last used delay before next retry
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}
	}

	return fmt.Errorf("after %d retries: %w", c.retryPolicy.MaxRetries, lastErr)
}

// shouldRetry examines the error message and returns true if it's retryable
func (c *Client) shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "HTTP 429") || strings.Contains(s, "HTTP 503")
}

// calculateRetryDelay uses binary exponential backoff with jitter
func (c *Client) calculateRetryDelay(attempt int, prevBackoff time.Duration) time.Duration {
	// Calculate exponential delay: 2^attempt * base, capped at max delay
	exp := time.Duration(math.Pow(2, float64(attempt))) * prevBackoff
	if exp > c.retryPolicy.MaxDelay {
		exp = c.retryPolicy.MaxDelay
	}

	// Jitter: random delay between 0 and exp
	return time.Duration(rand.Int63n(int64(exp) + 1))
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
	// Reset the original body for potential reuse
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return cloned, nil
}

// sendRequest sends an HTTP request to the DeepL API and decodes the response into the provided value v.
// It handles setting authorization headers and error handling for non-200 HTTP responses.
func (c *Client) sendRequest(req *http.Request, v interface{}) (err error) {
	req.Header.Set("Authorization", fmt.Sprintf("DeepL-Auth-Key %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		errorMsg := "unknown error"
		if found, message := getErrorMessage(res.StatusCode); found {
			errorMsg = message
		}
		var errRes errorResponse
		errDecode := json.NewDecoder(res.Body).Decode(&errRes)
		if errDecode == nil && errRes.Message != "" {
			errorMsg = fmt.Sprintf(
				"HTTP %d: %s --> %s",
				res.StatusCode,
				errorMsg,
				errRes.Message,
			)
		}
		return fmt.Errorf(errorMsg)
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}

// getBaseURL returns the appropriate API base URL based on the API key type.
// Free API keys (ending with ":fx") use the free API endpoint.
func getBaseURL(apiKey string) string {
	if strings.HasSuffix(apiKey, ":fx") {
		return baseURLFree
	}
	return baseURL
}

// getErrorMessage retrieves a predefined error message for a given HTTP status code, if available.
func getErrorMessage(status int) (bool, string) {
	if msg, found := httpErrorMessages[status]; found {
		return found, msg
	}
	return false, ""
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
