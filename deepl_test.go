package deepl

import (
	"encoding/json"
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

func MockResponse(statusCode int, data interface{}) *http.Response {
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

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	var resp testResponse

	err := client.sendRequest(req, &resp)
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
		{400, "", "Bad request"},
		{403, "", "Authorization failed"},
		{404, "", "resource could not be found"},
		{429, "", "Too many requests"},
		{456, "", "Quota exceeded"},
		{500, "", "Internal server error"},
		{503, "", "Resource currently unavailable"},
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

			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			var resp interface{}

			err := client.sendRequest(req, &resp)
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

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	var resp interface{}

	err := client.sendRequest(req, &resp)
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

func TestGetErrorMessage(t *testing.T) {
	testCases := []struct {
		statusCode     int
		expectedFound  bool
		expectedPrefix string
	}{
		{400, true, "Bad request"},
		{403, true, "Authorization failed"},
		{404, true, "The requested resource"},
		{999, false, ""},
	}

	for _, tc := range testCases {
		found, message := getErrorMessage(tc.statusCode)

		if found != tc.expectedFound {
			t.Errorf("getErrorMessage(%d) found = %v, expected %v", tc.statusCode, found, tc.expectedFound)
		}

		if tc.expectedFound && !strings.HasPrefix(message, tc.expectedPrefix) {
			t.Errorf("getErrorMessage(%d) message = %q, expected prefix %q", tc.statusCode, message, tc.expectedPrefix)
		}
	}
}
