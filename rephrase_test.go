package deepl

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRephrase_Success(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(200, RephraseResponse{
			Improvements: []*Improvement{
				{DetectedSourceLanguage: "EN", Text: "Rephrased text"},
			},
		})
	})

	impr, err := client.Rephrase("Original text")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if impr == nil || impr.Text != "Rephrased text" {
		t.Errorf("expected improvement with text 'Rephrased text', got %+v", impr)
	}
}

func TestRephraseWithContext_Success(t *testing.T) {
	ctx := context.Background()
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(200, RephraseResponse{
			Improvements: []*Improvement{
				{DetectedSourceLanguage: "DE", Text: "Umformulierter Text"},
			},
		})
	})

	impr, err := client.RephraseWithContext(ctx, "Ursprünglicher Text")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if impr == nil || impr.Text != "Umformulierter Text" {
		t.Errorf("expected improvement with text 'Umformulierter Text', got %+v", impr)
	}
}

func TestRephraseWithOptions_MultipleTexts(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(200, RephraseResponse{
			Improvements: []*Improvement{
				{DetectedSourceLanguage: "EN", Text: "First rephrased"},
				{DetectedSourceLanguage: "EN", Text: "Second rephrased"},
			},
		})
	})

	opts := RephraseOptions{
		Text:         []string{"First", "Second"},
		WritingStyle: WritingStyleBusiness,
	}
	improvements, err := client.RephraseWithOptions(context.Background(), opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(improvements) != 2 {
		t.Fatalf("expected 2 improvements, got %d", len(improvements))
	}
	if improvements[0].Text != "First rephrased" || improvements[1].Text != "Second rephrased" {
		t.Errorf("unexpected improvement texts: %+v", improvements)
	}
}

func TestRephraseWithOptions_ErrorIfBothStyleAndToneSet(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		t.Fatal("should not send request when both style and tone are set")
		return nil
	})

	opts := RephraseOptions{
		Text:         []string{"Some text"},
		WritingStyle: WritingStyleAcademic,
		WritingTone:  WritingToneConfident,
	}
	_, err := client.RephraseWithOptions(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "only one of WritingStyle or WritingTone can be set") {
		t.Errorf("expected error about mutually exclusive options, got %v", err)
	}
}

func TestRephraseWithOptions_ApiError(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(400, map[string]string{"message": "bad request"})
	})

	opts := RephraseOptions{
		Text:         []string{"Some text"},
		WritingStyle: WritingStyleSimple,
	}
	_, err := client.RephraseWithOptions(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected API error, got %v", err)
	}
}

func TestRephraseWithContext_NoImprovements(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return MockResponse(200, RephraseResponse{
			Improvements: []*Improvement{},
		})
	})

	_, err := client.RephraseWithContext(context.Background(), "No result text")
	if err == nil || !strings.Contains(err.Error(), "no improvements returned") {
		t.Errorf("expected error about no improvements, got %v", err)
	}
}

func TestRephraseWithOptions_ContextCancel(t *testing.T) {
	requestStarted := make(chan struct{})

	client := NewTestClient(func(req *http.Request) *http.Response {
		close(requestStarted)
		select {
		case <-req.Context().Done():
			return nil
		case <-time.After(50 * time.Millisecond):
			return MockResponse(200, RephraseResponse{
				Improvements: []*Improvement{{DetectedSourceLanguage: "EN", Text: "Should not reach"}},
			})
		}

	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-requestStarted
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	opts := RephraseOptions{Text: []string{"Some text"}}
	_, err := client.RephraseWithOptions(ctx, opts)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}
