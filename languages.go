package deepl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Language represents a language supported by the DeepL API, including its code, display name, and formality support.
type Language struct {
	Language          string `json:"language"`           // Language code, e.g. "EN", "DE"
	Name              string `json:"name"`               // Full language name, e.g. "English"
	SupportsFormality bool   `json:"supports_formality"` // Indicates if the language supports formality settings
}

// GetTargetLanguages retrieves the list of target languages supported by DeepL.
func (c *Client) GetTargetLanguages() ([]*Language, error) {
	return c.GetTargetLanguagesWithContext(context.Background())
}

// GetSourceLanguages retrieves the list of source languages supported by DeepL.
func (c *Client) GetSourceLanguages() ([]*Language, error) {
	return c.GetSourceLanguagesWithContext(context.Background())
}

// GetTargetLanguagesWithContext retrieves the list of target languages supported by DeepL,
// respecting the provided context for cancellation and timeouts.
func (c *Client) GetTargetLanguagesWithContext(ctx context.Context) ([]*Language, error) {
	return c.getLanguages(ctx, url.Values{"type": {"target"}})
}

// GetSourceLanguagesWithContext retrieves the list of source languages supported by DeepL,
// respecting the provided context for cancellation and timeouts.
func (c *Client) GetSourceLanguagesWithContext(ctx context.Context) ([]*Language, error) {
	return c.getLanguages(ctx, url.Values{"type": {"source"}})
}

// getLanguages is an internal method that fetches either source or target languages from the DeepL API.
func (c *Client) getLanguages(ctx context.Context, v url.Values) ([]*Language, error) {
	u := fmt.Sprintf("%s/v2/languages?", c.baseURL)

	// Construct a POST request with the query parameters appended to the URL.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var languages []*Language

	// Send the request and decode the response JSON into languages slice.
	if err := c.doRequest(ctx, req, &languages); err != nil {
		return nil, err
	}
	return languages, nil
}
