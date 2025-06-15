package deepl

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRephrase(t *testing.T) {
	mockImprovement := &Improvement{
		DetectedSourceLanguage: "EN",
		Text:                   "Rephrased text",
	}
	mockResp := RephraseResponse{
		Improvements: []*Improvement{mockImprovement},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(http.StatusOK, mockResp)
	})

	got, err := client.Rephrase("original text")
	if err != nil {
		t.Fatalf("Rephrase() error = %v", err)
	}
	if got == nil || got.Text != mockImprovement.Text || got.DetectedSourceLanguage != mockImprovement.DetectedSourceLanguage {
		t.Errorf("Rephrase() = %v, want %v", got, mockImprovement)
	}
}

func TestRephraseWithContext(t *testing.T) {
	mockImprovement := &Improvement{
		DetectedSourceLanguage: "EN",
		Text:                   "Rephrased text with context",
	}
	mockResp := RephraseResponse{
		Improvements: []*Improvement{mockImprovement},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(http.StatusOK, mockResp)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := client.RephraseWithContext(ctx, "original text")
	if err != nil {
		t.Fatalf("RephraseWithContext() error = %v", err)
	}
	if got == nil || got.Text != mockImprovement.Text || got.DetectedSourceLanguage != mockImprovement.DetectedSourceLanguage {
		t.Errorf("RephraseWithContext() = %v, want %v", got, mockImprovement)
	}
}

func TestRephraseWithOptions(t *testing.T) {
	mockImprovements := []*Improvement{
		{DetectedSourceLanguage: "EN", Text: "Rephrased 1"},
		{DetectedSourceLanguage: "EN", Text: "Rephrased 2"},
	}
	mockResp := RephraseResponse{
		Improvements: mockImprovements,
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		// Optionally, verify request method and URL here
		if req.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", req.Method)
		}
		if req.URL.Path != "/v2/write/rephrase" {
			t.Errorf("Unexpected URL path: %s", req.URL.Path)
		}
		return MockResponse(http.StatusOK, mockResp)
	})

	opts := RephraseOptions{
		Text:         []string{"text1", "text2"},
		TargetLang:   "DE",
		WritingStyle: WritingStyleBusiness,
		Tone:         WritingToneFriendly,
	}

	got, err := client.RephraseWithOptions(context.Background(), opts)
	if err != nil {
		t.Fatalf("RephraseWithOptions() error = %v", err)
	}
	if len(got) != len(mockImprovements) {
		t.Fatalf("RephraseWithOptions() returned %d improvements, want %d", len(got), len(mockImprovements))
	}
	for i, imp := range got {
		if imp.Text != mockImprovements[i].Text || imp.DetectedSourceLanguage != mockImprovements[i].DetectedSourceLanguage {
			t.Errorf("Improvement[%d] = %v, want %v", i, imp, mockImprovements[i])
		}
	}
}

func TestRephraseWithContext_NoImprovements(t *testing.T) {
	mockResp := RephraseResponse{
		Improvements: []*Improvement{},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(http.StatusOK, mockResp)
	})

	_, err := client.RephraseWithContext(context.Background(), "text")
	if err == nil || err.Error() != "no improvements returned" {
		t.Errorf("Expected 'no improvements returned' error, got %v", err)
	}
}

func TestRephraseWithOptions_ErrorResponse(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(http.StatusBadRequest, map[string]string{"error": "bad request"})
	})

	opts := RephraseOptions{
		Text: []string{"text"},
	}

	_, err := client.RephraseWithOptions(context.Background(), opts)
	if err == nil {
		t.Errorf("Expected error on bad response, got nil")
	}
}