package deepl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *Client {
	return &Client{
		apiKey:    "test-api-key",
		baseURL:   baseURL,
		userAgent: "deepl-go-test",
		httpClient: &http.Client{
			Transport: fn,
			Timeout:   10 * time.Second,
		},
	}
}

func MockResponse(statusCode int, data any) *http.Response {
	var responseBody string

	if data != nil {
		jsonData, _ := json.Marshal(data)
		responseBody = string(jsonData)
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(responseBody)),
		Header:     make(http.Header),
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("api-key-123")

	if client.apiKey != "api-key-123" {
		t.Errorf("expected apiKey 'api-key-123', got %s", client.apiKey)
	}

	if client.baseURL != baseURL {
		t.Errorf("expected baseURL %s, got %s", baseURL, client.baseURL)
	}

	if client.userAgent != "deepl-go/"+version {
		t.Errorf("expected userAgent 'deepl-go/%s', got %s", version, client.userAgent)
	}

	if client.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
}

func TestNewClientWithFreeApiKey(t *testing.T) {
	client := NewClient("api-key:fx")

	if client.baseURL != baseURLFree {
		t.Errorf("expected baseURL %s, got %s", baseURLFree, client.baseURL)
	}
}

func TestWithUserAgent(t *testing.T) {
	client := NewClient("api-key", WithUserAgent("custom-agent"))

	if client.userAgent != "custom-agent" {
		t.Errorf("expected userAgent 'custom-agent', got %s", client.userAgent)
	}
}

func TestWithProxy(t *testing.T) {
	proxyUrl, _ := url.Parse("http://localhost:8080")
	client := NewClient("api-key", WithProxy(*proxyUrl))

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http.Transport but got %T", client.httpClient.Transport)
	}

	if transport.Proxy == nil {
		t.Error("expected proxy function to be set")
	}
}

func TestSendRequest(t *testing.T) {
	type testResponse struct {
		Value string `json:"value"`
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Header.Get("Authorization") != "DeepL-Auth-Key test-api-key" {
			t.Errorf("expected Authorization header 'DeepL-Auth-Key test-api-key', got %s", req.Header.Get("Authorization"))
		}

		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
		}

		if req.Header.Get("User-Agent") != "deepl-go-test" {
			t.Errorf("expected User-Agent header 'deepl-go-test', got %s", req.Header.Get("User-Agent"))
		}

		return MockResponse(200, testResponse{Value: "test-value"})
	})

	req, _ := http.NewRequest(http.MethodGet, "https://api.deepl.com/some-endpoint", nil)
	var resp testResponse

	err := client.doRequest(context.Background(), req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Value != "test-value" {
		t.Errorf("expected value 'test-value', got %s", resp.Value)
	}
}

func TestSendRequestWithErrorStatus(t *testing.T) {
	testCases := []struct {
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{400, "", "bad request"},
		{403, "", "forbidden"},
		{404, "", "not found"},
		{429, "", "too many requests"},
		{456, "", "character limit has been reached"},
		{500, "", "internal server error"},
		{503, "", "service unavailable"},
		{499, `{"message":"Custom error"}`, "Custom error"},
		{499, "Invalid JSON", "unknown error"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("StatusCode_%d", tc.statusCode), func(t *testing.T) {
			client := NewTestClient(func(req *http.Request) *http.Response {
				var body io.ReadCloser
				if tc.responseBody != "" {
					body = io.NopCloser(strings.NewReader(tc.responseBody))
				} else {
					body = io.NopCloser(strings.NewReader(""))
				}

				return &http.Response{
					StatusCode: tc.statusCode,
					Body:       body,
					Header:     make(http.Header),
				}
			})

			req, _ := http.NewRequest(http.MethodPost, "https://api.deepl.com/some-endpoint", nil)
			var resp any

			err := client.doRequest(context.Background(), req, &resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tc.expectedError) && tc.statusCode != 499 {
				t.Errorf("expected error containing %q, got %q", tc.expectedError, err.Error())
			}
		})
	}
}

func TestSendRequestWithJSONDecodeError(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("invalid json")),
			Header:     make(http.Header),
		}
	})

	req, _ := http.NewRequest(http.MethodGet, "https://api.deepl.com/some-endpoint", nil)
	var resp any

	err := client.doRequest(context.Background(), req, &resp)
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
}

func TestGetBaseURL(t *testing.T) {
	testCases := []struct {
		apiKey      string
		expectedURL string
	}{
		{"normal-key", baseURL},
		{"free-key:fx", baseURLFree},
	}

	for _, tc := range testCases {
		u := getBaseURL(tc.apiKey)
		if u != tc.expectedURL {
			t.Errorf("getBaseURL(%q) = %q, expected %q", tc.apiKey, u, tc.expectedURL)
		}
	}
}

func TestSendRequestWithRetry_RetryOn429ThenSuccess(t *testing.T) {
	attempt := 0
	client := NewTestClient(func(req *http.Request) *http.Response {
		attempt++
		if attempt == 1 {
			return MockResponse(429, map[string]string{"message": "too many requests"})
		}
		return MockResponse(200, map[string]string{"message": "ok"})
	})
	client.retryPolicy = retryPolicy{MaxRetries: 3, MaxDelay: 500 * time.Millisecond}

	req, _ := http.NewRequest(http.MethodGet, "https://api.deepl.com/some-endpoint", nil)

	var er errorResponse
	start := time.Now()
	err := client.doRequest(context.Background(), req, &er)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after retry, got error %v", err)
	}
	if attempt != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempt)
	}
	if elapsed < 0 {
		t.Fatalf("expected some retry delay, got %v", elapsed)
	}
}

func TestSendRequestWithRetry_ExceedsMaxRetries(t *testing.T) {
	attempt := 0
	client := NewTestClient(func(req *http.Request) *http.Response {
		attempt++
		return MockResponse(503, map[string]string{"message": "service unavailable"})
	})
	client.retryPolicy = retryPolicy{MaxRetries: 2, MaxDelay: 10 * time.Millisecond}

	req, _ := http.NewRequest(http.MethodPost, "https://api.deepl.com/some-endpoint", nil)
	var er errorResponse

	err := client.doRequest(context.Background(), req, &er)

	if err == nil {
		t.Fatalf("expected error after retries exceeded, got nil")
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempt)
	}
}

func TestSendRequestWithRetry_DoNotRetryOnOtherError(t *testing.T) {
	attempt := 0
	client := NewTestClient(func(req *http.Request) *http.Response {
		attempt++
		return MockResponse(400, map[string]string{"message": "bad request"})
	})
	client.retryPolicy = retryPolicy{MaxRetries: 3, MaxDelay: time.Second}

	req, _ := http.NewRequest(http.MethodGet, "https://api.deepl.com/some-endpoint", nil)
	var er errorResponse

	err := client.doRequest(context.Background(), req, &er)
	if err == nil {
		t.Fatalf("expected error on 400 response")
	}
	if attempt != 1 {
		t.Errorf("expected no retries on 400, got %d attempts", attempt)
	}
}

func TestSendRequestWithRetry_ContextCancel(t *testing.T) {
	attempt := 0
	client := NewTestClient(func(req *http.Request) *http.Response {
		attempt++
		time.Sleep(50 * time.Millisecond)
		return MockResponse(503, map[string]string{"message": "service unavailable"})
	})
	client.retryPolicy = retryPolicy{MaxRetries: 3, MaxDelay: time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.deepl.com/some-endpoint", nil)
	var er errorResponse

	err := client.doRequest(ctx, req, &er)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error due to cancellation, got: %v", err)
	}
	if attempt < 1 {
		t.Fatalf("expected at least one attempt before cancellation, got %d", attempt)
	}
}
