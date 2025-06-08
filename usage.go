package deepl_go

import (
	"context"
	"fmt"
	"net/http"
)

type Usage struct {
	CharacterCount    uint `json:"character_count"`
	CharacterLimit    uint `json:"character_limit"`
	DocumentCount     uint `json:"document_count"`
	DocumentLimit     uint `json:"document_limit"`
	TeamDocumentCount uint `json:"team_document_count"`
	TeamDocumentLimit uint `json:"team_document_limit"`
}

func (c *Client) GetUsage() (*Usage, error) {
	return c.GetUsageWithContext(context.Background())
}
func (c *Client) GetUsageWithContext(ctx context.Context) (*Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v2/usage", c.baseURL), nil)
	if err != nil {
		return nil, err
	}
	var res = Usage{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
