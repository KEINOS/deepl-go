package deepl_go

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Language struct {
	Language          string `json:"language"`
	Name              string `json:"name"`
	SupportsFormality bool   `json:"supports_formality"`
}

func (c *Client) GetTargetLanguages() ([]*Language, error) {
	return c.GetTargetLanguagesWithContext(context.Background())
}

func (c *Client) GetSourceLanguages() ([]*Language, error) {
	return c.GetSourceLanguagesWithContext(context.Background())
}

func (c *Client) GetTargetLanguagesWithContext(ctx context.Context) ([]*Language, error) {
	return c.getLanguages(ctx, url.Values{"type": {"target"}})
}

func (c *Client) GetSourceLanguagesWithContext(ctx context.Context) ([]*Language, error) {
	return c.getLanguages(ctx, url.Values{"type": {"source"}})
}

func (c *Client) getLanguages(ctx context.Context, v url.Values) ([]*Language, error) {
	u := fmt.Sprintf("%s/v2/languages?", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var languages []*Language
	if err := c.sendRequest(req, &languages); err != nil {
		return nil, err
	}
	return languages, nil
}
