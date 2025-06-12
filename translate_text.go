package deepl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TranslateTextOptions holds the parameters for a text translation request.
type TranslateTextOptions struct {
	Text                 []string `json:"text"`                             // Text(s) to translate
	SourceLang           string   `json:"source_lang,omitempty"`            // Source language code
	TargetLang           string   `json:"target_lang"`                      // Target language code
	Context              string   `json:"context,omitempty"`                // Additional context for translation
	ShowBilledCharacters *bool    `json:"show_billed_characters,omitempty"` // Include billed character count in response
	SplitSentences       string   `json:"split_sentences,omitempty"`        // Sentence splitting mode: "0", "1", or "nonewlines"
	PreserveFormatting   *bool    `json:"preserve_formatting,omitempty"`    // Preserve original formatting
	Formality            string   `json:"formality,omitempty"`              // Formality preference
	ModelType            string   `json:"model_type,omitempty"`             // Translation model type
	GlossaryID           string   `json:"glossary_id,omitempty"`            // Glossary ID to apply
	TagHandling          string   `json:"tag_handling,omitempty"`           // Tag handling mode: "xml" or "html"
	OutlineDetection     *bool    `json:"outline_detection,omitempty"`      // Enable XML outline detection (default true)
	NonSplittingTags     []string `json:"non_splitting_tags,omitempty"`     // XML tags never splitting sentences
	SplittingTags        []string `json:"splitting_tags,omitempty"`         // XML tags that split sentences
	IgnoreTags           []string `json:"ignore_tags,omitempty"`            // XML tags marking untranslatable text
}

// Translation contains a single translation result corresponding to one input text.
type Translation struct {
	DetectedSourceLanguage string `json:"detected_source_language"` // Detected source language code
	Text                   string `json:"text"`                     // Translated text
	BilledCharacters       int    `json:"billed_characters"`        // Characters billed for translation
	ModelTypeUsed          string `json:"model_type_used"`          // Model used for translation
}

// TranslationsResponse wraps a list of one or more Translation objects returned from the API.
type TranslationsResponse struct {
	Translations []*Translation `json:"translations"` // Translations in same order as requested texts
}

// TranslateText translates a single text string into the target language using default options.
// It uses a background context.
func (c *Client) TranslateText(text, targetLanguage string) (*Translation, error) {
	return c.TranslateTextWithContext(context.Background(), text, targetLanguage)
}

// TranslateTextWithContext translates a single text string into the target language, supporting context for cancellation.
func (c *Client) TranslateTextWithContext(ctx context.Context, text, targetLanguage string) (*Translation, error) {
	options := TranslateTextOptions{
		Text:       []string{text},
		TargetLang: targetLanguage,
	}
	translations, err := c.TranslateTextWithOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	if len(translations) == 0 {
		return nil, fmt.Errorf("no translation returned")
	}
	return translations[0], nil
}

// TranslateTextWithOptions translates one or more texts with full control via TranslateTextOptions.
// Supports context for cancellation and timeout.
func (c *Client) TranslateTextWithOptions(ctx context.Context, opts TranslateTextOptions) ([]*Translation, error) {
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v2/translate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	var response TranslationsResponse
	if err := c.doRequest(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.Translations, nil
}
