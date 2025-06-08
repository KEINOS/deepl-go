package deepl_go

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTranslateText(t *testing.T) {
	expectedTranslation := &Translation{
		DetectedSourceLanguage: "EN",
		Text:                   "Hallo Welt",
		BilledCharacters:       10,
		ModelTypeUsed:          "quality_optimized",
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost {
			t.Errorf("Expected HTTP method: POST, got: %s", req.Method)
		}

		url := req.URL.String()
		if !strings.Contains(url, "/v2/translate") {
			t.Errorf("Unexpected URL: %s", url)
		}

		body, _ := io.ReadAll(req.Body)
		var requestData TranslateTextOptions
		err := json.Unmarshal(body, &requestData)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(requestData.Text) != 1 || requestData.Text[0] != "Hello World" {
			t.Errorf("Expected text: 'Hello World', got: %v", requestData.Text)
		}

		if requestData.TargetLang != "DE" {
			t.Errorf("Expected target language: 'DE', got: %s", requestData.TargetLang)
		}

		// Return mock response
		response := TranslationsResponse{
			Translations: []*Translation{expectedTranslation},
		}
		return MockResponse(200, response)
	})

	t.Run("TranslateText", func(t *testing.T) {
		translation, err := client.TranslateText("Hello World", "DE")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if translation == nil {
			t.Fatal("Received translation is nil")
		}

		if translation.Text != expectedTranslation.Text {
			t.Errorf("Expected translated text: %s, got: %s",
				expectedTranslation.Text, translation.Text)
		}

		if translation.DetectedSourceLanguage != expectedTranslation.DetectedSourceLanguage {
			t.Errorf("Expected detected source language: %s, got: %s",
				expectedTranslation.DetectedSourceLanguage, translation.DetectedSourceLanguage)
		}

		if translation.BilledCharacters != expectedTranslation.BilledCharacters {
			t.Errorf("Expected billed characters: %d, got: %d",
				expectedTranslation.BilledCharacters, translation.BilledCharacters)
		}
	})

	t.Run("TranslateTextWithContext", func(t *testing.T) {
		ctx := context.Background()
		translation, err := client.TranslateTextWithContext(ctx, "Hello World", "DE")

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if translation == nil {
			t.Fatal("Received translation is nil")
		}

		if translation.Text != expectedTranslation.Text {
			t.Errorf("Expected translated text: %s, got: %s",
				expectedTranslation.Text, translation.Text)
		}
	})
}

func TestTranslateTextWithOptions(t *testing.T) {
	expectedTranslation := &Translation{
		DetectedSourceLanguage: "EN",
		Text:                   "Hallo Welt",
		BilledCharacters:       10,
		ModelTypeUsed:          "deepl-v2.0",
	}

	client := NewTestClient(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		var requestData TranslateTextOptions
		err := json.Unmarshal(body, &requestData)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Check custom options
		if requestData.SourceLang != "EN" {
			t.Errorf("Expected source language: 'EN', got: %s", requestData.SourceLang)
		}

		if requestData.Formality != "more" {
			t.Errorf("Expected formality: 'more', got: %s", requestData.Formality)
		}

		if requestData.PreserveFormatting == nil || *requestData.PreserveFormatting != true {
			t.Errorf("Expected preserve_formatting: true")
		}

		// Return mock response
		response := TranslationsResponse{
			Translations: []*Translation{expectedTranslation},
		}
		return MockResponse(200, response)
	})

	// Test TranslateTextWithOptions
	t.Run("TranslateTextWithOptions", func(t *testing.T) {
		preserve := true
		options := TranslateTextOptions{
			Text:               []string{"Hello World"},
			SourceLang:         "EN",
			TargetLang:         "DE",
			Formality:          "more",
			PreserveFormatting: &preserve,
		}

		translations, err := client.TranslateTextWithOptions(context.Background(), options)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(translations) != 1 {
			t.Fatalf("Expected 1 translation, got: %d", len(translations))
		}

		if translations[0].Text != expectedTranslation.Text {
			t.Errorf("Expected translated text: %s, got: %s",
				expectedTranslation.Text, translations[0].Text)
		}
	})
}

func TestTranslateTextErrors(t *testing.T) {
	t.Run("HTTPError", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 403,
				Body:       nil,
				Header:     make(http.Header),
			}
		})

		_, err := client.TranslateText("Hello World", "DE")
		if err == nil {
			t.Error("Expected error from TranslateText, got: nil")
		}
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			response := TranslationsResponse{
				Translations: []*Translation{},
			}
			return MockResponse(200, response)
		})

		_, err := client.TranslateText("Hello World", "DE")
		if err == nil {
			t.Error("Expected error for empty translations, got: nil")
		}
	})
}
