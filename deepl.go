package deepl

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	apiKey     string       // API authentication key
	baseURL    string       // Base URL for API endpoints (depends on API key type)
	userAgent  string       // User-Agent header value sent with requests
	httpClient *http.Client // Underlying HTTP client used for requests
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
		baseURL:   getBaseURL(apiKey),
		userAgent: "deepl-go/" + version,
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
