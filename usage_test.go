package deepl

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGetUsage(t *testing.T) {
	now := time.Now()
	expectedUsage := &Usage{
		CharacterCount: 2150000,
		CharacterLimit: 20000000,
		Products: []ProductUsage{
			{
				ProductType:          "write",
				APIKeyCharacterCount: 1000000,
				CharacterCount:       1250000,
			},
		},
		APIKeyCharacterCount: func() *int64 { val := int64(1880000); return &val }(),
		APIKeyCharacterLimit: func() *int64 { val := int64(0); return &val }(),
		StartTime:            &now,
		EndTime:              &now,
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost {
			t.Errorf("Expected HTTP method: POST, got: %s", req.Method)
		}

		url := req.URL.String()
		if !strings.Contains(url, "/v2/usage") {
			t.Errorf("Unexpected URL: %s", url)
		}

		return MockResponse(200, expectedUsage)
	})

	t.Run("GetUsage", func(t *testing.T) {
		usage, err := client.GetUsage()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if usage == nil {
			t.Fatal("Received usage data is nil")
		}

		if usage.CharacterCount != expectedUsage.CharacterCount {
			t.Errorf("Expected CharacterCount: %d, got: %d",
				expectedUsage.CharacterCount, usage.CharacterCount)
		}

		if usage.CharacterLimit != expectedUsage.CharacterLimit {
			t.Errorf("Expected CharacterLimit: %d, got: %d",
				expectedUsage.CharacterLimit, usage.CharacterLimit)
		}

		if len(usage.Products) != len(expectedUsage.Products) {
			t.Errorf("Expected number of products: %d, got: %d",
				len(expectedUsage.Products), len(usage.Products))
		}
	})

	t.Run("GetUsageWithContext", func(t *testing.T) {
		ctx := context.Background()
		usage, err := client.GetUsageWithContext(ctx)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if usage == nil {
			t.Fatal("Received usage data is nil")
		}

		if usage.CharacterCount != expectedUsage.CharacterCount {
			t.Errorf("Expected CharacterCount: %d, got: %d",
				expectedUsage.CharacterCount, usage.CharacterCount)
		}
	})
}

func TestGetUsageError(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 403,
			Body:       nil,
			Header:     make(http.Header),
		}
	})

	_, err := client.GetUsage()
	if err == nil {
		t.Error("Expected error from GetUsage, got: nil")
	}

	_, err = client.GetUsageWithContext(context.Background())
	if err == nil {
		t.Error("Expected error from GetUsageWithContext, got: nil")
	}
}
