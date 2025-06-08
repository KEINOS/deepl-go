package deepl_go

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Usage struct {
	CharacterCount       int64          `json:"character_count"`
	CharacterLimit       int64          `json:"character_limit"`
	Products             []ProductUsage `json:"products,omitempty"`
	APIKeyCharacterCount *int64         `json:"api_key_character_count,omitempty"`
	APIKeyCharacterLimit *int64         `json:"api_key_character_limit,omitempty"`
	StartTime            *time.Time     `json:"start_time,omitempty"`
	EndTime              *time.Time     `json:"end_time,omitempty"`
}

type ProductUsage struct {
	ProductType          string `json:"product_type"`
	APIKeyCharacterCount int64  `json:"api_key_character_count"`
	CharacterCount       int64  `json:"character_count"`
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
