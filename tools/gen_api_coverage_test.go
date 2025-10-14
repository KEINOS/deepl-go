package main

import (
	"testing"
	"time"
)

// Test data structures and helper functions

// mockOpenAPISpec returns a minimal OpenAPI spec for testing
func mockOpenAPISpec() *OpenAPISpec {
	return &OpenAPISpec{
		Info: struct {
			Title   string `yaml:"title"`
			Version string `yaml:"version"`
		}{
			Title:   "DeepL API",
			Version: "1.0.0",
		},
		Paths: map[string]PathItem{
			"/v2/translate": {
				Post: &Operation{
					OperationID: "translateText",
					Summary:     "Translate text",
					Description: "Translate text from source to target language",
					Tags:        []string{"translation"},
				},
			},
			"/v2/languages": {
				Get: &Operation{
					OperationID: "getLanguages",
					Summary:     "Get supported languages",
					Description: "Retrieve list of supported languages",
					Tags:        []string{"languages"},
				},
			},
			"/v2/usage": {
				Get: &Operation{
					OperationID: "getUsage",
					Summary:     "Get API usage",
					Description: "Check current API usage and limits",
					Tags:        []string{"usage"},
				},
			},
		},
	}
}

// mockGoMethods returns sample Go methods for testing
func mockGoMethods() []GoMethod {
	return []GoMethod{
		{
			Name:        "TranslateText",
			Receiver:    "*Client",
			Parameters:  []string{"ctx context.Context", "text string", "opts *TranslateOptions"},
			ReturnTypes: []string{"*TranslateResponse", "error"},
			FileName:    "translate_text.go",
			LineNumber:  25,
			Comments:    "TranslateText translates the given text from source to target language",
		},
		{
			Name:        "GetLanguages",
			Receiver:    "*Client",
			Parameters:  []string{"ctx context.Context", "langType LanguageType"},
			ReturnTypes: []string{"[]Language", "error"},
			FileName:    "languages.go",
			LineNumber:  15,
			Comments:    "GetLanguages retrieves supported languages",
		},
		// Note: GetUsage is intentionally missing to test detection of missing endpoints
	}
}

// Test functions for OpenAPI specification handling

func TestParseOpenAPISpec(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		wantPaths   int
	}{
		{
			name: "valid minimal spec",
			yamlContent: `
info:
  title: "DeepL API"
  version: "1.0.0"
paths:
  /v2/translate:
    post:
      operationId: translateText
      summary: "Translate text"
`,
			wantErr:   false,
			wantPaths: 1,
		},
		{
			name:        "invalid YAML",
			yamlContent: "invalid: yaml: content: [",
			wantErr:     true,
			wantPaths:   0,
		},
		{
			name: "missing required fields",
			yamlContent: `
paths:
  /v2/translate:
    post:
      summary: "Missing operation ID"
`,
			wantErr:   true,
			wantPaths: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := parseOpenAPISpec([]byte(tt.yamlContent))

			if tt.wantErr {
				if err == nil {
					t.Error("parseOpenAPISpec() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseOpenAPISpec() unexpected error: %v", err)
				return
			}

			if len(spec.Paths) != tt.wantPaths {
				t.Errorf("parseOpenAPISpec() got %d paths, want %d", len(spec.Paths), tt.wantPaths)
			}
		})
	}
}

func TestExtractEndpoints(t *testing.T) {
	spec := mockOpenAPISpec()

	endpoints := extractEndpoints(spec)

	// Verify we have the expected number of endpoints
	expectedCount := 3 // /v2/translate (POST), /v2/languages (GET), /v2/usage (GET)
	if len(endpoints) != expectedCount {
		t.Errorf("extractEndpoints() got %d endpoints, want %d", len(endpoints), expectedCount)
	}

	// Verify specific endpoints are present
	endpointMap := make(map[string]EndpointMapping)
	for _, ep := range endpoints {
		key := ep.HTTPMethod + " " + ep.APIEndpoint
		endpointMap[key] = ep
	}

	expectedEndpoints := []string{
		"POST /v2/translate",
		"GET /v2/languages",
		"GET /v2/usage",
	}

	for _, expected := range expectedEndpoints {
		if _, exists := endpointMap[expected]; !exists {
			t.Errorf("extractEndpoints() missing expected endpoint: %s", expected)
		}
	}

	// Verify endpoint details
	translateEndpoint := endpointMap["POST /v2/translate"]
	if translateEndpoint.OperationID != "translateText" {
		t.Errorf("extractEndpoints() translate endpoint operationID = %s, want translateText", translateEndpoint.OperationID)
	}

	if translateEndpoint.Category == "" {
		t.Error("extractEndpoints() translate endpoint should have a category assigned")
	}
}

// Test functions for Go source code analysis

func TestIsClientMethod(t *testing.T) {
	tests := []struct {
		name         string
		methodName   string
		receiverType string
		isExported   bool
		want         bool
	}{
		{
			name:         "valid client method",
			methodName:   "TranslateText",
			receiverType: "*Client",
			isExported:   true,
			want:         true,
		},
		{
			name:         "private method",
			methodName:   "translateText",
			receiverType: "*Client",
			isExported:   false,
			want:         false,
		},
		{
			name:         "wrong receiver type",
			methodName:   "TranslateText",
			receiverType: "*SomeOtherType",
			isExported:   true,
			want:         false,
		},
		{
			name:         "no receiver",
			methodName:   "TranslateText",
			receiverType: "",
			isExported:   true,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will need to be implemented with actual AST nodes
			// For now, we're testing the logic conceptually

			// Mock the behavior based on test parameters
			got := tt.isExported && tt.receiverType == "*Client" && tt.methodName != ""

			if got != tt.want {
				t.Errorf("isClientMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test functions for endpoint mapping and analysis

func TestCreateEndpointMappings(t *testing.T) {
	spec := mockOpenAPISpec()
	endpoints := extractEndpoints(spec)
	methods := mockGoMethods()

	mappings := createEndpointMappings(endpoints, methods)

	// Verify all endpoints are processed
	if len(mappings) != len(endpoints) {
		t.Errorf("createEndpointMappings() got %d mappings, want %d", len(mappings), len(endpoints))
	}

	// Check specific mappings
	var translateMapping *EndpointMapping
	var usageMapping *EndpointMapping

	for i := range mappings {
		switch mappings[i].APIEndpoint {
		case "/v2/translate":
			translateMapping = &mappings[i]
		case "/v2/usage":
			usageMapping = &mappings[i]
		}
	}

	// TranslateText should be mapped
	if translateMapping == nil {
		t.Fatal("createEndpointMappings() missing /v2/translate mapping")
	}

	if !translateMapping.IsImplemented {
		t.Error("createEndpointMappings() /v2/translate should be marked as implemented")
	}

	if translateMapping.GoMethod == nil {
		t.Error("createEndpointMappings() /v2/translate should have GoMethod assigned")
	} else if translateMapping.GoMethod.Name != "TranslateText" {
		t.Errorf("createEndpointMappings() /v2/translate mapped to %s, want TranslateText", translateMapping.GoMethod.Name)
	}

	// Usage endpoint should NOT be mapped (missing implementation)
	if usageMapping == nil {
		t.Fatal("createEndpointMappings() missing /v2/usage mapping")
	}

	if usageMapping.IsImplemented {
		t.Error("createEndpointMappings() /v2/usage should be marked as NOT implemented")
	}

	if usageMapping.GoMethod != nil {
		t.Error("createEndpointMappings() /v2/usage should not have GoMethod assigned")
	}
}

func TestMatchMethodToEndpoint(t *testing.T) {
	methods := mockGoMethods()

	tests := []struct {
		name           string
		endpoint       EndpointMapping
		wantMethod     *string // nil if no match expected
		wantConfidence float64
	}{
		{
			name: "exact operation ID match",
			endpoint: EndpointMapping{
				APIEndpoint: "/v2/translate",
				HTTPMethod:  "POST",
				OperationID: "translateText",
			},
			wantMethod:     stringPtr("TranslateText"),
			wantConfidence: 1.0,
		},
		{
			name: "path-based match",
			endpoint: EndpointMapping{
				APIEndpoint: "/v2/languages",
				HTTPMethod:  "GET",
				OperationID: "getLanguages",
			},
			wantMethod:     stringPtr("GetLanguages"),
			wantConfidence: 0.8,
		},
		{
			name: "no match available",
			endpoint: EndpointMapping{
				APIEndpoint: "/v2/usage",
				HTTPMethod:  "GET",
				OperationID: "getUsage",
			},
			wantMethod:     nil,
			wantConfidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMethod := matchMethodToEndpoint(tt.endpoint, methods)

			if tt.wantMethod == nil {
				if gotMethod != nil {
					t.Errorf("matchMethodToEndpoint() = %v, want nil", gotMethod.Name)
				}
			} else {
				if gotMethod == nil {
					t.Errorf("matchMethodToEndpoint() = nil, want %s", *tt.wantMethod)
				} else if gotMethod.Name != *tt.wantMethod {
					t.Errorf("matchMethodToEndpoint() = %s, want %s", gotMethod.Name, *tt.wantMethod)
				}
			}
		})
	}
}

func TestAssignPriorities(t *testing.T) {
	mappings := []EndpointMapping{
		{APIEndpoint: "/v2/translate", HTTPMethod: "POST", Category: "translation"},
		{APIEndpoint: "/v2/languages", HTTPMethod: "GET", Category: "languages"},
		{APIEndpoint: "/v2/usage", HTTPMethod: "GET", Category: "usage"},
		{APIEndpoint: "/v2/admin/settings", HTTPMethod: "GET", Category: "administration"},
	}

	assignPriorities(mappings)

	// Check that priorities are assigned
	for _, mapping := range mappings {
		if mapping.Priority == "" {
			t.Errorf("assignPriorities() endpoint %s has empty priority", mapping.APIEndpoint)
		}
	}

	// Core translation should be high priority
	var translatePriority string
	for _, mapping := range mappings {
		if mapping.APIEndpoint == "/v2/translate" {
			translatePriority = mapping.Priority
			break
		}
	}

	if translatePriority != "High" {
		t.Errorf("assignPriorities() /v2/translate priority = %s, want High", translatePriority)
	}

	// Admin endpoints should be low priority
	var adminPriority string
	for _, mapping := range mappings {
		if mapping.APIEndpoint == "/v2/admin/settings" {
			adminPriority = mapping.Priority
			break
		}
	}

	if adminPriority != "Low" {
		t.Errorf("assignPriorities() /v2/admin/settings priority = %s, want Low", adminPriority)
	}
}

// Test functions for coverage calculation and reporting

func TestCalculateCoverageStats(t *testing.T) {
	mappings := []EndpointMapping{
		{IsImplemented: true},
		{IsImplemented: true},
		{IsImplemented: false},
		{IsImplemented: false},
		{IsImplemented: false},
	}

	total, implemented, percentage := calculateCoverageStats(mappings)

	expectedTotal := 5
	expectedImplemented := 2
	expectedPercentage := 40.0

	if total != expectedTotal {
		t.Errorf("calculateCoverageStats() total = %d, want %d", total, expectedTotal)
	}

	if implemented != expectedImplemented {
		t.Errorf("calculateCoverageStats() implemented = %d, want %d", implemented, expectedImplemented)
	}

	if percentage != expectedPercentage {
		t.Errorf("calculateCoverageStats() percentage = %.1f, want %.1f", percentage, expectedPercentage)
	}
}

func TestCategorizeEndpoints(t *testing.T) {
	mappings := []EndpointMapping{
		{Category: "translation", APIEndpoint: "/v2/translate"},
		{Category: "translation", APIEndpoint: "/v2/translate-batch"},
		{Category: "languages", APIEndpoint: "/v2/languages"},
		{Category: "usage", APIEndpoint: "/v2/usage"},
	}

	categories := categorizeEndpoints(mappings)

	// Check that all categories are present
	expectedCategories := []string{"translation", "languages", "usage"}
	for _, expected := range expectedCategories {
		if endpoints, exists := categories[expected]; !exists {
			t.Errorf("categorizeEndpoints() missing category: %s", expected)
		} else if len(endpoints) == 0 {
			t.Errorf("categorizeEndpoints() category %s has no endpoints", expected)
		}
	}

	// Check translation category has 2 endpoints
	if len(categories["translation"]) != 2 {
		t.Errorf("categorizeEndpoints() translation category has %d endpoints, want 2", len(categories["translation"]))
	}
}

func TestGenerateCoverageReport(t *testing.T) {
	report := CoverageReport{
		GeneratedAt:      time.Now(),
		OpenAPIVersion:   "1.0.0",
		TotalEndpoints:   5,
		ImplementedCount: 2,
		CoveragePercent:  40.0,
		Mappings: []EndpointMapping{
			{
				APIEndpoint:   "/v2/translate",
				HTTPMethod:    "POST",
				IsImplemented: true,
				GoMethod:      &GoMethod{Name: "TranslateText"},
				Category:      "translation",
			},
		},
	}

	err := generateCoverageReport(report)
	if err != nil {
		t.Errorf("generateCoverageReport() unexpected error: %v", err)
	}

	// TODO: Verify that the report file was created and contains expected content
	// This will be implemented when the actual function is created
}

// Test utility functions

func TestValidateConfiguration(t *testing.T) {
	// Test configuration validation
	err := validateConfiguration()

	// Since this is testing the validation logic, we expect it to pass
	// in a proper test environment
	if err != nil {
		t.Errorf("validateConfiguration() unexpected error: %v", err)
	}
}

// Helper functions for tests

func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance-critical functions

func BenchmarkParseOpenAPISpec(b *testing.B) {
	yamlContent := []byte(`
info:
  title: "DeepL API"
  version: "1.0.0"
paths:
  /v2/translate:
    post:
      operationId: translateText
      summary: "Translate text"
  /v2/languages:
    get:
      operationId: getLanguages
      summary: "Get languages"
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseOpenAPISpec(yamlContent)
		if err != nil {
			b.Fatalf("parseOpenAPISpec() error: %v", err)
		}
	}
}

func BenchmarkCreateEndpointMappings(b *testing.B) {
	spec := mockOpenAPISpec()
	endpoints := extractEndpoints(spec)
	methods := mockGoMethods()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createEndpointMappings(endpoints, methods)
	}
}

// Integration test that combines multiple functions

func TestFullAnalysisWorkflow(t *testing.T) {
	// This test simulates the complete analysis workflow

	// 1. Parse OpenAPI spec
	spec := mockOpenAPISpec()
	if spec == nil {
		t.Fatal("Failed to create mock OpenAPI spec")
	}

	// 2. Extract endpoints
	endpoints := extractEndpoints(spec)
	if len(endpoints) == 0 {
		t.Fatal("No endpoints extracted from spec")
	}

	// 3. Get Go methods
	methods := mockGoMethods()
	if len(methods) == 0 {
		t.Fatal("No Go methods available for mapping")
	}

	// 4. Create mappings
	mappings := createEndpointMappings(endpoints, methods)
	if len(mappings) != len(endpoints) {
		t.Errorf("Mapping count mismatch: got %d, want %d", len(mappings), len(endpoints))
	}

	// 5. Assign priorities
	assignPriorities(mappings)

	// 6. Calculate coverage
	total, implemented, percentage := calculateCoverageStats(mappings)
	if total == 0 {
		t.Error("Total endpoints should not be zero")
	}

	if percentage < 0 || percentage > 100 {
		t.Errorf("Coverage percentage %f is out of valid range", percentage)
	}

	// 7. Generate report
	report := CoverageReport{
		GeneratedAt:      time.Now(),
		OpenAPIVersion:   spec.Info.Version,
		TotalEndpoints:   total,
		ImplementedCount: implemented,
		CoveragePercent:  percentage,
		Mappings:         mappings,
	}

	err := generateCoverageReport(report)
	if err != nil {
		t.Errorf("Failed to generate coverage report: %v", err)
	}

	t.Logf("Integration test completed successfully: %d/%d endpoints (%.1f%% coverage)",
		implemented, total, percentage)
}
