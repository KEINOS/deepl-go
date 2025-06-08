package deepl_go

import (
	"encoding/json"
	"fmt"
	"io"
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
	version     = "0.1.0"
)

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

type Client struct {
	apiKey     string
	baseURL    string
	userAgent  string
	httpClient *http.Client
}

type Option func(c *Client)

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

func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

func WithProxy(proxy url.URL) Option {
	return func(c *Client) {
		c.httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(&proxy),
		}
	}
}

func WithTrace() Option {
	return func(c *Client) {
		prev := c.httpClient.Transport
		if prev == nil {
			prev = http.DefaultTransport
		}
		c.httpClient.Transport = &LoggingRoundTripper{
			Proxied: prev,
		}
	}
}

type errorResponse struct {
	Message string `json:"message"`
}

func (c *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Authorization", fmt.Sprintf("DeepL-Auth-Key %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		if found, message := getErrorMessage(res.StatusCode); found {
			return fmt.Errorf("%s, status code: %d", message, res.StatusCode)
		}
		var errRes errorResponse
		if err = json.NewDecoder(res.Body).Decode(&errRes); err == nil {
			return fmt.Errorf("%s, status code: %d", errRes.Message, res.StatusCode)
		}

		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}
	return nil
}

func getBaseURL(apiKey string) string {
	if strings.HasSuffix(apiKey, ":fx") {
		return baseURLFree
	}
	return baseURL
}

func getErrorMessage(status int) (bool, string) {
	if msg, found := httpErrorMessages[status]; found {
		return found, msg
	}
	return false, ""
}

type LoggingRoundTripper struct {
	Proxied http.RoundTripper
}

func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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

func BoolPtr(b bool) *bool {
	return &b
}
