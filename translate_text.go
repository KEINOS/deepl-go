package deepl_go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type TranslateTextOptions struct {
	Text                 []string `json:"text"`
	SourceLang           string   `json:"source_lang,omitempty"`
	TargetLang           string   `json:"target_lang"`
	Context              string   `json:"context,omitempty"`
	ShowBilledCharacters *bool    `json:"show_billed_characters,omitempty"`
	SplitSentences       string   `json:"split_sentences,omitempty"`
	PreserveFormatting   *bool    `json:"preserve_formatting,omitempty"`
	Formality            string   `json:"formality,omitempty"`
	ModelType            string   `json:"model_type,omitempty"`
	GlossaryID           string   `json:"glossary_id,omitempty"`
	TagHandling          string   `json:"tag_handling,omitempty"`
	OutlineDetection     *bool    `json:"outline_detection,omitempty"`
	NonSplittingTags     []string `json:"non_splitting_tags,omitempty"`
	SplittingTags        []string `json:"splitting_tags,omitempty"`
	IgnoreTags           []string `json:"ignore_tags,omitempty"`
}

type Translation struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
	BilledCharacters       int    `json:"billed_characters"`
	ModelTypeUsed          string `json:"model_type_used"`
}

type TranslationsResponse struct {
	Translations []*Translation `json:"translations"`
}

func (c *Client) TranslateText(text, targetLanguage string) (*Translation, error) {
	return c.TranslateTextWithContext(context.Background(), text, targetLanguage)
}

func (c *Client) TranslateTextWithContext(ctx context.Context, text, targetLanguage string) (*Translation, error) {
	o := TranslateTextOptions{
		Text:       []string{text},
		TargetLang: targetLanguage,
	}
	t, err := c.TranslateTextWithOptions(ctx, o)
	if err != nil {
		return nil, err
	}
	if len(t) == 0 {
		return nil, fmt.Errorf("no translation returned")
	}
	return t[0], nil
}

func (c *Client) TranslateTextWithOptions(ctx context.Context, o TranslateTextOptions) ([]*Translation, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/v2/translate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	var response TranslationsResponse
	if err := c.sendRequest(req, &response); err != nil {
		return nil, err
	}
	return response.Translations, nil
}
