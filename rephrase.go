package deepl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// WritingStyle represents the desired style in which the text should be rephrased.
// The `prefer_` prefix allows falling back to the default style if the language
// does not yet support specific styles.
type WritingStyle int8

const (
	WritingStyleUnset WritingStyle = iota
	WritingStyleAcademic
	WritingStyleBusiness
	WritingStyleCasual
	WritingStyleDefault
	WritingStyleSimple
	WritingStylePreferAcademic
	WritingStylePreferBusiness
	WritingStylePreferCasual
	WritingStylePreferSimple
)

// String returns the string representation of the WritingStyle enum.
func (ws WritingStyle) String() string {
	styles := [...]string{
		"academic", "business", "casual", "default", "simple",
		"prefer_academic", "prefer_business", "prefer_casual", "prefer_simple",
	}
	return styles[ws]
}

// MarshalJSON implements the json.Marshaler interface for WritingStyle.
// It serializes the WritingStyle value as its string representation.
func (ws WritingStyle) MarshalJSON() ([]byte, error) {
	return json.Marshal(ws.String())
}

// WritingTone specifies the desired tone for the text.
// The `prefer_` prefix allows falling back to the default tone if the language
// does not yet support specific tones.
type WritingTone int8

const (
	WritingToneUnset WritingTone = iota
	WritingToneConfident
	WritingToneDefault
	WritingToneDiplomatic
	WritingToneEnthusiastic
	WritingToneFriendly
	WritingTonePreferConfident
	WritingTonePreferDiplomatic
	WritingTonePreferEnthusiastic
	WritingTonePreferFriendly
)

// String returns the string representation of the WritingTone enum.
func (wt WritingTone) String() string {
	tones := [...]string{
		"confident", "default", "diplomatic", "enthusiastic", "friendly",
		"prefer_confident", "prefer_diplomatic", "prefer_enthusiastic",
		"prefer_friendly",
	}
	return tones[wt]
}

// MarshalJSON implements the json.Marshaler interface for WritingTone.
// It serializes the WritingTone value as its string representation.
func (wt WritingTone) MarshalJSON() ([]byte, error) {
	return json.Marshal(wt.String())
}

// RephraseOptions represents the payload for the rephrase API call.
// Text contains one or more strings to rephrase.
// TargetLang is the target language code (optional).
// WritingStyle specifies the desired style to adapt the text to audience and goals (optional).
// WritingTone specifies the desired tone for the output text (optional).
// Only one of WritingStyle or WritingTone can be set.
type RephraseOptions struct {
	Text         []string     `json:"text"`
	TargetLang   string       `json:"target_lang,omitempty"`
	WritingStyle WritingStyle `json:"writing_style,omitempty"`
	WritingTone  WritingTone  `json:"tone,omitempty"`
}

// Improvement contains a single rephrased result along with detected language info.
type Improvement struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

// RephraseResponse models the response from the rephrase endpoint.
type RephraseResponse struct {
	Improvements []*Improvement `json:"improvements"`
}

// Rephrase is a convenience method to rephrase a single string using background context.
func (c *Client) Rephrase(text string) (*Improvement, error) {
	return c.RephraseWithContext(context.Background(), text)
}

// RephraseWithContext rephrases text with the provided context for timeout or cancellation.
func (c *Client) RephraseWithContext(ctx context.Context, text string) (*Improvement, error) {
	options := RephraseOptions{
		Text: []string{text},
	}
	translations, err := c.RephraseWithOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	if len(translations) == 0 {
		return nil, errors.New("no improvements returned")
	}
	return translations[0], nil
}

// RephraseWithOptions performs the rephrase request with complete options and returns improvements.
func (c *Client) RephraseWithOptions(ctx context.Context, opts RephraseOptions) ([]*Improvement, error) {
	if opts.WritingStyle != WritingStyle(0) && opts.WritingTone != WritingTone(0) {
		return nil, errors.New("only one of WritingStyle or WritingTone can be set")
	}
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v2/write/rephrase", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	var response RephraseResponse
	if err := c.doRequest(ctx, req, &response); err != nil {
		return nil, err
	}
	return response.Improvements, nil
}
