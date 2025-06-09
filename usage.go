package deepl

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Usage represents the API usage statistics returned by DeepL.
type Usage struct {
	CharacterCount       int64          `json:"character_count"`                   // Total number of characters translated
	CharacterLimit       int64          `json:"character_limit"`                   // Total character limit for the account/plan
	Products             []ProductUsage `json:"products,omitempty"`                // Usage details per product (optional)
	APIKeyCharacterCount *int64         `json:"api_key_character_count,omitempty"` // Character count specific to the API key (optional)
	APIKeyCharacterLimit *int64         `json:"api_key_character_limit,omitempty"` // Character limit specific to the API key (optional)
	StartTime            *time.Time     `json:"start_time,omitempty"`              // Start time of current usage period (optional)
	EndTime              *time.Time     `json:"end_time,omitempty"`                // End time of current usage period (optional)
}

// ProductUsage provides detailed usage information related to a specific DeepL product.
type ProductUsage struct {
	ProductType          string `json:"product_type"`            // The type/name of the product
	APIKeyCharacterCount int64  `json:"api_key_character_count"` // Characters translated using this product with the API key
	CharacterCount       int64  `json:"character_count"`         // Total characters translated using this product
}

// GetUsage retrieves the current account API usage.
func (c *Client) GetUsage() (*Usage, error) {
	return c.GetUsageWithContext(context.Background())
}

// GetUsageWithContext retrieves the current account API usage respecting the provided context for cancellation or timeout.
func (c *Client) GetUsageWithContext(ctx context.Context) (*Usage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v2/usage", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	var res Usage

	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
