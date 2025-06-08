package deepl_go

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestGetSourceLanguages(t *testing.T) {
	expectedLanguages := []*Language{
		{Language: "EN", Name: "English", SupportsFormality: false},
		{Language: "DE", Name: "German", SupportsFormality: true},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", req.Method)
		}

		url := req.URL.String()
		if !strings.Contains(url, "languages") || !strings.Contains(url, "type=source") {
			t.Errorf("unexpected URL: %s", url)
		}

		return MockResponse(200, expectedLanguages)
	})

	languages, err := client.GetSourceLanguages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(languages) != len(expectedLanguages) {
		t.Fatalf("expected %d languages, got %d", len(expectedLanguages), len(languages))
	}

	for i, lang := range languages {
		if lang.Language != expectedLanguages[i].Language ||
			lang.Name != expectedLanguages[i].Name ||
			lang.SupportsFormality != expectedLanguages[i].SupportsFormality {
			t.Errorf("language at index %d differs from expected", i)
		}
	}
}

func TestGetTargetLanguages(t *testing.T) {
	expectedLanguages := []*Language{
		{Language: "EN", Name: "English", SupportsFormality: false},
		{Language: "DE", Name: "German", SupportsFormality: true},
		{Language: "FR", Name: "French", SupportsFormality: true},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", req.Method)
		}

		url := req.URL.String()
		if !strings.Contains(url, "languages") || !strings.Contains(url, "type=target") {
			t.Errorf("unexpected URL: %s", url)
		}

		return MockResponse(200, expectedLanguages)
	})

	languages, err := client.GetTargetLanguages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(languages) != len(expectedLanguages) {
		t.Fatalf("expected %d languages, got %d", len(expectedLanguages), len(languages))
	}

	for i, lang := range languages {
		if lang.Language != expectedLanguages[i].Language ||
			lang.Name != expectedLanguages[i].Name ||
			lang.SupportsFormality != expectedLanguages[i].SupportsFormality {
			t.Errorf("language at index %d differs from expected", i)
		}
	}
}

func TestGetLanguagesWithContext(t *testing.T) {
	expectedLanguages := []*Language{
		{Language: "EN", Name: "English", SupportsFormality: false},
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Context() == nil {
			t.Error("expected non-nil context")
		}

		return MockResponse(200, expectedLanguages)
	})

	ctx := context.Background()

	_, err := client.GetSourceLanguagesWithContext(ctx)
	if err != nil {
		t.Fatalf("GetSourceLanguagesWithContext failed: %v", err)
	}

	// Teste GetTargetLanguagesWithContext
	_, err = client.GetTargetLanguagesWithContext(ctx)
	if err != nil {
		t.Fatalf("GetTargetLanguagesWithContext failed: %v", err)
	}
}

func TestGetLanguagesError(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 403,
			Body:       nil,
			Header:     make(http.Header),
		}
	})

	_, err := client.GetSourceLanguages()
	if err == nil {
		t.Error("expected error from GetSourceLanguages, got nil")
	}

	_, err = client.GetTargetLanguages()
	if err == nil {
		t.Error("expected error from GetTargetLanguages, got nil")
	}
}
