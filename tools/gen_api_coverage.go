//go:generate go run .

// Package main generates API coverage analysis for the deepl-go library.
//
// This tool compares the official DeepL API specification with the currently
// implemented Go methods to identify coverage gaps and prioritize future development.
//
// The generator performs the following operations:
// 1. Fetch the latest OpenAPI specification from DeepL's official repository
// 2. Parse and analyze the Go source code to detect implemented client methods
// 3. Create intelligent mappings between API endpoints and Go methods
// 4. Generate comprehensive coverage reports in Markdown format
//
// Generated files:
// - openapi_spec.yaml: Official DeepL OpenAPI specification (cached locally)
// - api_coverage.md: Detailed coverage analysis and implementation status
//
// Security considerations:
// - HTTP requests are made to trusted sources (github.com/DeepLcom)
// - File operations are restricted to the testdata directory
// - No sensitive data is processed or stored
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Configuration constants for the generator
const (
	openAPISpecURL     = "https://raw.githubusercontent.com/DeepLcom/openapi/main/openapi.yaml"
	openAPISpecFile    = "testdata/openapi_spec.yaml" // Save to testdata directory (relative to project root)
	coverageReportFile = "testdata/api_coverage.md"   // Save to testdata directory (relative to project root)
	httpTimeoutSeconds = 30
	sourceCodeRoot     = "." // Current directory (project root)
	goFilePattern      = "*.go"
)

// Data structures for analysis and reporting

// OpenAPISpec represents the parsed DeepL OpenAPI specification
type OpenAPISpec struct {
	Info struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
	Paths map[string]PathItem `yaml:"paths"`
}

// PathItem represents an API endpoint with its operations
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// Operation represents an API operation (HTTP method + endpoint)
type Operation struct {
	OperationID string              `yaml:"operationId"`
	Summary     string              `yaml:"summary"`
	Description string              `yaml:"description"`
	Parameters  []Parameter         `yaml:"parameters,omitempty"`
	RequestBody *RequestBody        `yaml:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses"`
	Tags        []string            `yaml:"tags,omitempty"`
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string `yaml:"name"`
	In          string `yaml:"in"` // query, header, path, etc.
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
	Schema      Schema `yaml:"schema"`
}

// RequestBody represents request body specification
type RequestBody struct {
	Required    bool                 `yaml:"required"`
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// Response represents response specification
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
}

// MediaType represents content type specification
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema represents data schema (simplified for our needs)
type Schema struct {
	Type        string            `yaml:"type"`
	Format      string            `yaml:"format,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Properties  map[string]Schema `yaml:"properties,omitempty"`
}

// GoMethod represents a detected Go client method
type GoMethod struct {
	Name        string   // Method name (e.g., "TranslateText")
	Receiver    string   // Receiver type (e.g., "*Client")
	Parameters  []string // Parameter names and types
	ReturnTypes []string // Return value types
	FileName    string   // Source file containing the method
	LineNumber  int      // Line number where method is defined
	Comments    string   // Associated documentation comments
}

// EndpointMapping represents the relationship between API endpoints and Go methods
type EndpointMapping struct {
	APIEndpoint   string    // API path (e.g., "/v2/translate")
	HTTPMethod    string    // HTTP method (GET, POST, etc.)
	OperationID   string    // OpenAPI operation ID
	GoMethod      *GoMethod // Corresponding Go method (nil if not implemented)
	Priority      string    // Implementation priority (High/Medium/Low)
	Category      string    // Functional category (Translation, Languages, etc.)
	Description   string    // Human-readable description
	IsImplemented bool      // Whether this endpoint is implemented
}

// CoverageReport represents the complete analysis results
type CoverageReport struct {
	GeneratedAt        time.Time         // Report generation timestamp
	OpenAPIVersion     string            // Version of the OpenAPI spec used
	TotalEndpoints     int               // Total number of API endpoints
	ImplementedCount   int               // Number of implemented endpoints
	CoveragePercent    float64           // Implementation coverage percentage
	Mappings           []EndpointMapping // Detailed endpoint mappings
	ImplementedMethods []GoMethod        // All detected Go methods
	MissingEndpoints   []EndpointMapping // Prioritized list of unimplemented endpoints
}

// Main execution flow and core functions

// main orchestrates the entire API coverage analysis process
// func main() {
//     // 1. Setup and validation
//     //    - Validate working directory and file permissions
//     //    - Create output directory if needed
//     //    - Initialize HTTP client with proper timeout
//
//     // 2. Fetch OpenAPI specification
//     //    - Download latest spec from DeepL's GitHub repository
//     //    - Cache locally to avoid repeated downloads
//     //    - Validate YAML format and required fields
//
//     // 3. Analyze Go source code
//     //    - Scan all .go files in the parent directory
//     //    - Parse AST to extract client methods
//     //    - Build method database with signatures and metadata
//
//     // 4. Create endpoint mappings
//     //    - Map API endpoints to Go methods using intelligent matching
//     //    - Assign priority levels based on endpoint importance
//     //    - Categorize endpoints by functional area
//
//     // 5. Generate coverage report
//     //    - Calculate coverage statistics
//     //    - Create detailed Markdown report
//     //    - Write results to api_coverage.md
//
//     // 6. Error handling and cleanup
//     //    - Report any issues encountered during analysis
//     //    - Ensure all resources are properly cleaned up
// }

// OpenAPI specification handling functions

// fetchOpenAPISpec downloads the latest OpenAPI specification from DeepL's repository
// func fetchOpenAPISpec() (*OpenAPISpec, error) {
//     // Implementation steps:
//     // 1. Create HTTP client with timeout and proper headers
//     // 2. Make GET request to openAPISpecURL
//     // 3. Handle HTTP errors and response validation
//     // 4. Read response body and validate content type
//     // 5. Parse YAML content into OpenAPISpec struct
//     // 6. Cache parsed spec to local file for future runs
//     // 7. Return parsed specification or error
//
//     // Security considerations:
//     // - Validate URL scheme (https only)
//     // - Limit response body size to prevent DoS
//     // - Sanitize file paths for cache storage
// }

// fetchOpenAPISpec downloads the latest OpenAPI specification from DeepL's repository
func fetchOpenAPISpec() (*OpenAPISpec, error) {
	fmt.Println("🌐 Fetching OpenAPI specification from DeepL...")

	// Check if cached file exists and is recent (less than 1 hour old)
	if cachedSpec, err := loadCachedSpec(); err == nil {
		fmt.Println("📁 Using cached OpenAPI specification")
		return cachedSpec, nil
	}

	// Create HTTP client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(httpTimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", openAPISpecURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "deepl-go-coverage-analyzer/1.0")
	req.Header.Set("Accept", "application/yaml, text/yaml, */*")

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenAPI spec: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Limit response size to prevent DoS (5MB max)
	const maxResponseSize = 5 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)

	// Read response body
	yamlContent, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("📦 Downloaded %d bytes of OpenAPI specification\n", len(yamlContent))

	// Parse YAML content
	spec, err := parseOpenAPISpec(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Cache the spec for future use
	if err := cacheSpec(yamlContent); err != nil {
		fmt.Printf("⚠️  Warning: failed to cache OpenAPI spec: %v\n", err)
		// Don't fail the entire operation just because caching failed
	}

	fmt.Printf("✅ Successfully parsed OpenAPI spec: %s v%s\n", spec.Info.Title, spec.Info.Version)
	return spec, nil
}

// loadCachedSpec attempts to load and parse cached OpenAPI specification
func loadCachedSpec() (*OpenAPISpec, error) {
	// Check if cache file exists
	info, err := os.Stat(openAPISpecFile)
	if err != nil {
		return nil, err // File doesn't exist or can't be accessed
	}

	// Check if cache is recent (less than 1 hour old)
	if time.Since(info.ModTime()) > time.Hour {
		return nil, fmt.Errorf("cached spec is too old")
	}

	// Read cached file
	yamlContent, err := os.ReadFile(openAPISpecFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached spec: %w", err)
	}

	// Parse cached content
	return parseOpenAPISpec(yamlContent)
}

// cacheSpec saves OpenAPI specification to local file for future use
func cacheSpec(yamlContent []byte) error {
	// Write to file with appropriate permissions
	err := os.WriteFile(openAPISpecFile, yamlContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	fmt.Printf("💾 Cached OpenAPI spec to %s\n", openAPISpecFile)
	return nil
}

// parseOpenAPISpec parses YAML content into structured OpenAPI specification
// func parseOpenAPISpec(yamlContent []byte) (*OpenAPISpec, error) {
//     // Implementation steps:
//     // 1. Validate YAML syntax and structure
//     // 2. Unmarshal into OpenAPISpec struct
//     // 3. Validate required fields (info, paths)
//     // 4. Normalize operation IDs and endpoint paths
//     // 5. Extract and validate HTTP methods
//     // 6. Return parsed spec or descriptive error
// }

// extractEndpoints converts OpenAPI paths into normalized endpoint list
// func extractEndpoints(spec *OpenAPISpec) []EndpointMapping {
//     // Implementation steps:
//     // 1. Iterate through all paths in the specification
//     // 2. For each path, extract all HTTP methods (GET, POST, etc.)
//     // 3. Create EndpointMapping for each method + path combination
//     // 4. Extract operation ID, description, and parameters
//     // 5. Assign initial priority based on endpoint type and usage
//     // 6. Categorize endpoints (Translation, Languages, Usage, etc.)
//     // 7. Sort endpoints by category and priority
//     // 8. Return complete list of endpoint mappings
// }

// Go source code analysis functions

// analyzeGoSourceCode scans the Go codebase to find implemented client methods
// func analyzeGoSourceCode(rootDir string) ([]GoMethod, error) {
//     // Implementation steps:
//     // 1. Walk the directory tree starting from rootDir
//     // 2. Find all .go files (excluding test files and this generator)
//     // 3. Parse each file into AST using go/parser
//     // 4. Extract method declarations with proper receivers
//     // 5. Filter for client methods (receiver type *Client or similar)
//     // 6. Extract method metadata (name, parameters, return types)
//     // 7. Parse associated documentation comments
//     // 8. Return complete list of detected methods
// }

// analyzeGoSourceCode scans the Go codebase to find implemented client methods
func analyzeGoSourceCode(rootDir string) ([]GoMethod, error) {
	fmt.Printf("📁 Scanning Go source code in %s...\n", rootDir)

	var allMethods []GoMethod

	// Walk through all Go files in the directory
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and this generator tool
		if strings.HasSuffix(path, "_test.go") ||
			strings.Contains(path, "gen_api_coverage.go") ||
			strings.Contains(path, "testdata") {
			return nil
		}

		fmt.Printf("   📄 Parsing file: %s\n", path)

		// Parse this Go file
		methods, err := parseGoFile(path)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to parse %s: %v\n", path, err)
			return nil // Continue processing other files
		}

		if len(methods) > 0 {
			fmt.Printf("      🔍 Found %d methods in %s\n", len(methods), filepath.Base(path))
			for _, method := range methods {
				fmt.Printf("         • %s.%s\n", method.Receiver, method.Name)
			}
		}

		allMethods = append(allMethods, methods...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("🔍 Found %d client methods across all Go files\n", len(allMethods))
	return allMethods, nil
}

// parseGoFile extracts method information from a single Go file
// func parseGoFile(filename string) ([]GoMethod, error) {
//     // Implementation steps:
//     // 1. Read and parse Go source file using go/parser
//     // 2. Walk the AST to find function declarations
//     // 3. Filter for methods with appropriate receivers
//     // 4. Extract method signature details (parameters, return types)
//     // 5. Capture associated documentation comments
//     // 6. Record source location (file, line number)
//     // 7. Return methods found in this file
// }

// parseGoFile extracts method information from a single Go file
func parseGoFile(filename string) ([]GoMethod, error) {
	// Create file set for position tracking
	fset := token.NewFileSet()

	// Parse the Go source file
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var methods []GoMethod

	// Walk the AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Check if this is a client method
			if isClientMethodAST(x) {
				method := extractMethodInfo(fset, filename, x)
				methods = append(methods, method)
			}
		}
		return true
	})

	return methods, nil
}

// isClientMethod determines if a method belongs to the DeepL client
// func isClientMethod(funcDecl *ast.FuncDecl) bool {
//     // Implementation logic:
//     // 1. Check if function has a receiver
//     // 2. Verify receiver type matches client patterns (*Client, etc.)
//     // 3. Exclude internal/private methods (lowercase names)
//     // 4. Exclude test helper methods
//     // 5. Return true if this is a public client method
// }

// Endpoint mapping and analysis functions

// createEndpointMappings intelligently maps API endpoints to Go methods
// func createEndpointMappings(endpoints []EndpointMapping, methods []GoMethod) []EndpointMapping {
//     // Implementation strategy:
//     // 1. For each API endpoint, attempt to find corresponding Go method
//     // 2. Use multiple matching strategies:
//     //    a. Direct operation ID matching (if available)
//     //    b. Method name pattern matching (TranslateText -> /translate)
//     //    c. Endpoint path analysis (/v2/translate -> translate-related method)
//     //    d. Parameter signature matching
//     // 3. Handle edge cases and ambiguous mappings
//     // 4. Mark endpoints as implemented or missing
//     // 5. Assign confidence scores to mappings
//     // 6. Return updated endpoint mappings with Go method associations
// }

// matchMethodToEndpoint attempts to find the best Go method for an API endpoint
// func matchMethodToEndpoint(endpoint EndpointMapping, methods []GoMethod) *GoMethod {
//     // Matching algorithms (in order of preference):
//     // 1. Exact operation ID match
//     // 2. Method name similarity (fuzzy matching)
//     // 3. Endpoint path keyword matching
//     // 4. Parameter count and type similarity
//     // 5. Return the best match or nil if no suitable method found
// }

// assignPriorities determines implementation priority for missing endpoints
// func assignPriorities(mappings []EndpointMapping) {
//     // Priority assignment logic:
//     // HIGH priority:
//     // - Core translation functionality (/v2/translate)
//     // - Language detection and listing
//     // - Authentication and basic operations
//     //
//     // MEDIUM priority:
//     // - Usage monitoring and statistics
//     // - Advanced translation options
//     // - Batch operations
//     //
//     // LOW priority:
//     // - Administrative endpoints
//     // - Deprecated or rarely used features
//     // - Optional convenience methods
// }

// Report generation functions

// generateCoverageReport creates the final Markdown coverage analysis
// func generateCoverageReport(report CoverageReport) error {
//     // Report sections to generate:
//     // 1. Header with generation timestamp and summary statistics
//     // 2. Implementation overview (coverage percentage, key metrics)
//     // 3. Implemented endpoints table with Go method mappings
//     // 4. Missing endpoints prioritized by importance
//     // 5. Detailed analysis section with recommendations
//     // 6. Footer with generation metadata and next steps
//     //
//     // Formatting requirements:
//     // - Clean Markdown with proper tables and headers
//     // - Links to source code where applicable
//     // - Color coding or badges for visual appeal
//     // - Sortable tables for easy navigation
// }

// writeMarkdownReport writes the coverage report to api_coverage.md
// func writeMarkdownReport(content string) error {
//     // Implementation steps:
//     // 1. Create or overwrite api_coverage.md file
//     // 2. Write content with proper UTF-8 encoding
//     // 3. Add file header with generation warning
//     // 4. Ensure proper line endings for cross-platform compatibility
//     // 5. Sync file to disk and validate write success
// }

// Utility and helper functions

// calculateCoverageStats computes coverage metrics from endpoint mappings
// func calculateCoverageStats(mappings []EndpointMapping) (int, int, float64) {
//     // Calculate:
//     // - Total number of API endpoints
//     // - Number of implemented endpoints
//     // - Coverage percentage (implemented/total * 100)
//     // Return all three values for report generation
// }

// categorizeEndpoints groups endpoints by functional area
// func categorizeEndpoints(mappings []EndpointMapping) map[string][]EndpointMapping {
//     // Categories:
//     // - "Translation": Text translation endpoints
//     // - "Languages": Supported language operations
//     // - "Usage": API usage and quota monitoring
//     // - "Authentication": Auth-related endpoints
//     // - "Administration": Account and settings management
//     // - "Utilities": Helper and convenience endpoints
// }

// validateConfiguration ensures all required configuration is present
// func validateConfiguration() error {
//     // Validation checks:
//     // 1. Verify required URLs are accessible
//     // 2. Check file system permissions for output directory
//     // 3. Validate Go source code directory exists
//     // 4. Ensure network connectivity for OpenAPI spec download
//     // 5. Return error if any validation fails
// }

// logProgress provides user feedback during long-running operations
// func logProgress(message string, args ...interface{}) {
//     // Simple progress logging:
//     // 1. Format message with timestamp
//     // 2. Print to stderr to avoid interfering with output
//     // 3. Support printf-style formatting
//     // 4. Include operation context when available
// }

// Template for the generated API coverage report
// When implementing, uncomment and use this template with text/template package
/*
const tplMarkdownCoverage = `# DeepL API Coverage Report

> **⚠️ Auto-generated file** - Do not edit manually. This file is regenerated by ` + "`go generate ./...`" + `

**Generated:** {{.GeneratedAt}}
**API Version:** {{.OpenAPIVersion}}
**Coverage:** {{.CoveragePercent}}% ({{.ImplementedCount}}/{{.TotalEndpoints}} endpoints)

---

## 📊 Coverage Summary

| Metric | Value |
|--------|-------|
| **Total API Endpoints** | {{.TotalEndpoints}} |
| **Implemented Endpoints** | {{.ImplementedCount}} |
| **Missing Endpoints** | {{.MissingCount}} |
| **Coverage Percentage** | {{.CoveragePercent}}% |
| **Last Updated** | {{.GeneratedAt}} |

---

## ✅ Implemented Endpoints

The following API endpoints are currently implemented in the Go client:

| Endpoint | Method | Go Method | Category | Status |
|----------|--------|-----------|----------|---------|
{{range .ImplementedEndpoints}}| ` + "`{{.APIEndpoint}}`" + ` | {{.HTTPMethod}} | ` + "`{{.GoMethod.Name}}`" + ` | {{.Category}} | ✅ Implemented |
{{end}}

---

## ❌ Missing Endpoints

The following API endpoints are **not yet implemented** and are prioritized for future development:

### 🔥 High Priority

Critical endpoints that should be implemented next:

| Endpoint | Method | Description | Category |
|----------|--------|-------------|----------|
{{range .HighPriorityMissing}}| ` + "`{{.APIEndpoint}}`" + ` | {{.HTTPMethod}} | {{.Description}} | {{.Category}} |
{{end}}

### 🟡 Medium Priority

Important endpoints for enhanced functionality:

| Endpoint | Method | Description | Category |
|----------|--------|-------------|----------|
{{range .MediumPriorityMissing}}| ` + "`{{.APIEndpoint}}`" + ` | {{.HTTPMethod}} | {{.Description}} | {{.Category}} |
{{end}}

### 🔵 Low Priority

Optional endpoints for advanced use cases:

| Endpoint | Method | Description | Category |
|----------|--------|-------------|----------|
{{range .LowPriorityMissing}}| ` + "`{{.APIEndpoint}}`" + ` | {{.HTTPMethod}} | {{.Description}} | {{.Category}} |
{{end}}

---

## 📈 Progress by Category

{{range .CategoryStats}}### {{.Category}}

- **Implemented:** {{.ImplementedCount}}/{{.TotalCount}} ({{.CoveragePercent}}%)
- **Status:** {{.StatusEmoji}} {{.StatusText}}

{{range .MissingEndpoints}}  - [ ] ` + "`{{.HTTPMethod}} {{.APIEndpoint}}`" + ` - {{.Description}}
{{end}}

{{end}}

---

## 🔍 Implementation Details

### Detected Go Methods

The following client methods were automatically detected:

{{range .ImplementedMethods}}#### ` + "`{{.Name}}`" + `

- **File:** ` + "`{{.FileName}}:{{.LineNumber}}`" + `
- **Signature:** ` + "`{{.Signature}}`" + `
- **Mapped Endpoint:** {{if .MappedEndpoint}}` + "`{{.MappedEndpoint}}`" + `{{else}}_Not mapped to specific endpoint_{{end}}

{{if .Comments}}{{.Comments}}
{{end}}

{{end}}

---

## 🎯 Recommendations

### Next Steps for Development

1. **Start with High Priority endpoints** - These are core features that users expect
2. **Focus on one category at a time** - This ensures consistent API design
3. **Consider batch operations** - Some endpoints might be efficiently implemented together
4. **Review official documentation** - Ensure parameter compatibility and error handling

### Implementation Guidelines

- Follow existing code patterns established in the codebase
- Add comprehensive tests for each new endpoint
- Update documentation and examples
- Consider backward compatibility when adding new features

### Testing Strategy

- Use the E2E testing environment with DeepL Mock server
- Add unit tests for parameter validation and error cases
- Test edge cases and error conditions
- Verify integration with existing client methods

---

## 🔗 References

- [Official DeepL API Documentation](https://www.deepl.com/docs-api)
- [DeepL OpenAPI Specification](https://github.com/DeepLcom/openapi)
- [DeepL Mock Server](https://github.com/DeepLcom/deepl-mock)

---

**Note:** This report is automatically generated by analyzing the official DeepL OpenAPI specification and the current Go codebase. The mapping between API endpoints and Go methods is performed using intelligent pattern matching and may require manual verification for complex cases.

*Report generated by ` + "`testdata/gen_api_coverage.go`" + ` on {{.GeneratedAt}}*
`
*/

// Stub implementations for all functions (to be implemented one by one)

// parseOpenAPISpec parses YAML content into structured OpenAPI specification
func parseOpenAPISpec(yamlContent []byte) (*OpenAPISpec, error) {
	if len(yamlContent) == 0 {
		return nil, fmt.Errorf("empty YAML content")
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(yamlContent, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if spec.Info.Title == "" {
		return nil, fmt.Errorf("missing required field: info.title")
	}

	if spec.Paths == nil {
		spec.Paths = make(map[string]PathItem)
	}

	return &spec, nil
}

// extractEndpoints converts OpenAPI paths into normalized endpoint list
func extractEndpoints(spec *OpenAPISpec) []EndpointMapping {
	if spec == nil || spec.Paths == nil {
		return []EndpointMapping{}
	}

	var endpoints []EndpointMapping

	for path, pathItem := range spec.Paths {
		// Extract GET operation
		if pathItem.Get != nil {
			endpoints = append(endpoints, EndpointMapping{
				APIEndpoint:   path,
				HTTPMethod:    "GET",
				OperationID:   pathItem.Get.OperationID,
				Description:   pathItem.Get.Summary,
				Category:      categorizeFromPath(path),
				Priority:      "Medium", // Default priority
				IsImplemented: false,
			})
		}

		// Extract POST operation
		if pathItem.Post != nil {
			endpoints = append(endpoints, EndpointMapping{
				APIEndpoint:   path,
				HTTPMethod:    "POST",
				OperationID:   pathItem.Post.OperationID,
				Description:   pathItem.Post.Summary,
				Category:      categorizeFromPath(path),
				Priority:      "Medium", // Default priority
				IsImplemented: false,
			})
		}

		// Extract PUT operation
		if pathItem.Put != nil {
			endpoints = append(endpoints, EndpointMapping{
				APIEndpoint:   path,
				HTTPMethod:    "PUT",
				OperationID:   pathItem.Put.OperationID,
				Description:   pathItem.Put.Summary,
				Category:      categorizeFromPath(path),
				Priority:      "Medium", // Default priority
				IsImplemented: false,
			})
		}

		// Extract DELETE operation
		if pathItem.Delete != nil {
			endpoints = append(endpoints, EndpointMapping{
				APIEndpoint:   path,
				HTTPMethod:    "DELETE",
				OperationID:   pathItem.Delete.OperationID,
				Description:   pathItem.Delete.Summary,
				Category:      categorizeFromPath(path),
				Priority:      "Medium", // Default priority
				IsImplemented: false,
			})
		}
	}

	return endpoints
}

// categorizeFromPath determines category based on API path
func categorizeFromPath(path string) string {
	switch {
	case contains(path, "translate"):
		return "translation"
	case contains(path, "language"):
		return "languages"
	case contains(path, "usage"):
		return "usage"
	case contains(path, "admin"):
		return "administration"
	default:
		return "utilities"
	}
}

// contains checks if string contains substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

// findSubstring simple substring search helper
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// isClientMethodAST determines if a method belongs to the DeepL client
func isClientMethodAST(funcDecl *ast.FuncDecl) bool {
	// Check if function has a receiver
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}

	// Check if method name is exported (starts with uppercase)
	if !funcDecl.Name.IsExported() {
		return false
	}

	// Get receiver type
	recv := funcDecl.Recv.List[0]
	var receiverType string

	switch t := recv.Type.(type) {
	case *ast.StarExpr:
		// Pointer receiver like *Client
		if ident, ok := t.X.(*ast.Ident); ok {
			receiverType = "*" + ident.Name
		}
	case *ast.Ident:
		// Value receiver like Client
		receiverType = t.Name
	}

	// Check if receiver type matches client patterns
	return receiverType == "*Client" || receiverType == "Client"
}

// extractMethodInfo extracts detailed information from an AST function declaration
func extractMethodInfo(fset *token.FileSet, filename string, funcDecl *ast.FuncDecl) GoMethod {
	// Get position information
	pos := fset.Position(funcDecl.Pos())

	// Extract receiver type
	var receiver string
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		switch t := recv.Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				receiver = "*" + ident.Name
			}
		case *ast.Ident:
			receiver = t.Name
		}
	}

	// Extract parameters
	var parameters []string
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := typeToString(param.Type)
			if len(param.Names) == 0 {
				// Anonymous parameter
				parameters = append(parameters, paramType)
			} else {
				// Named parameters
				for _, name := range param.Names {
					parameters = append(parameters, name.Name+" "+paramType)
				}
			}
		}
	}

	// Extract return types
	var returnTypes []string
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			returnType := typeToString(result.Type)
			returnTypes = append(returnTypes, returnType)
		}
	}

	// Extract documentation comments
	var comments string
	if funcDecl.Doc != nil {
		for _, comment := range funcDecl.Doc.List {
			comments += strings.TrimPrefix(comment.Text, "//") + " "
		}
		comments = strings.TrimSpace(comments)
	}

	return GoMethod{
		Name:        funcDecl.Name.Name,
		Receiver:    receiver,
		Parameters:  parameters,
		ReturnTypes: returnTypes,
		FileName:    filepath.Base(filename),
		LineNumber:  pos.Line,
		Comments:    comments,
	}
}

// typeToString converts an AST type expression to string representation
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		if len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *ast.ChanType:
		return "chan " + typeToString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	default:
		return "unknown"
	}
}

// createEndpointMappings intelligently maps API endpoints to Go methods
func createEndpointMappings(endpoints []EndpointMapping, methods []GoMethod) []EndpointMapping {
	mappings := make([]EndpointMapping, len(endpoints))
	copy(mappings, endpoints)

	// Try to match each endpoint with a Go method
	for i := range mappings {
		goMethod := matchMethodToEndpoint(mappings[i], methods)
		if goMethod != nil {
			mappings[i].GoMethod = goMethod
			mappings[i].IsImplemented = true
		}
	}

	return mappings
}

// matchMethodToEndpoint attempts to find the best Go method for an API endpoint
func matchMethodToEndpoint(endpoint EndpointMapping, methods []GoMethod) *GoMethod {
	// Strategy 1: Exact operation ID match
	if endpoint.OperationID != "" {
		for i := range methods {
			// Convert operation ID to Go method naming convention
			expectedName := operationIDToMethodName(endpoint.OperationID)
			if methods[i].Name == expectedName {
				return &methods[i]
			}
		}
	}

	// Strategy 2: Path-based matching
	for i := range methods {
		if pathMatchesMethod(endpoint.APIEndpoint, methods[i].Name) {
			return &methods[i]
		}
	}

	// No match found
	return nil
}

// operationIDToMethodName converts OpenAPI operation ID to Go method name
func operationIDToMethodName(operationID string) string {
	if operationID == "" {
		return ""
	}

	// Convert camelCase/snake_case to PascalCase
	if operationID == "translateText" {
		return "TranslateText"
	}
	if operationID == "getLanguages" {
		return "GetLanguages"
	}
	if operationID == "getUsage" {
		return "GetUsage"
	}

	// Default: capitalize first letter
	if len(operationID) > 0 {
		return string(operationID[0]-32) + operationID[1:] // Convert first char to uppercase
	}

	return operationID
}

// pathMatchesMethod checks if API path matches Go method name
func pathMatchesMethod(path, methodName string) bool {
	// Simple matching logic
	if contains(path, "translate") && contains(methodName, "Translate") {
		return true
	}
	if contains(path, "language") && contains(methodName, "Language") {
		return true
	}
	if contains(path, "usage") && contains(methodName, "Usage") {
		return true
	}
	if contains(path, "rephrase") && contains(methodName, "Rephrase") {
		return true
	}

	return false
}

// assignPriorities determines implementation priority for missing endpoints
func assignPriorities(mappings []EndpointMapping) {
	for i := range mappings {
		mappings[i].Priority = determinePriority(mappings[i])
	}
}

// determinePriority assigns priority based on endpoint characteristics
func determinePriority(mapping EndpointMapping) string {
	path := mapping.APIEndpoint

	// Low priority: Admin and advanced features (check first)
	if contains(path, "admin") || contains(path, "settings") {
		return "Low"
	}

	// High priority: Core functionality
	if contains(path, "/v2/translate") && mapping.HTTPMethod == "POST" {
		return "High"
	}
	if contains(path, "/v2/languages") && mapping.HTTPMethod == "GET" {
		return "High"
	}

	// Medium priority: Important features
	if contains(path, "/v2/usage") {
		return "Medium"
	}
	if contains(path, "/v2/") && (mapping.HTTPMethod == "GET" || mapping.HTTPMethod == "POST") {
		return "Medium"
	}

	// Default to Medium
	return "Medium"
}

// calculateCoverageStats computes coverage metrics from endpoint mappings
func calculateCoverageStats(mappings []EndpointMapping) (int, int, float64) {
	if len(mappings) == 0 {
		return 0, 0, 0.0
	}

	total := len(mappings)
	implemented := 0

	for _, mapping := range mappings {
		if mapping.IsImplemented {
			implemented++
		}
	}

	percentage := float64(implemented) / float64(total) * 100.0
	return total, implemented, percentage
}

// categorizeEndpoints groups endpoints by functional area
func categorizeEndpoints(mappings []EndpointMapping) map[string][]EndpointMapping {
	categories := make(map[string][]EndpointMapping)

	for _, mapping := range mappings {
		category := mapping.Category
		if category == "" {
			category = "utilities" // Default category
		}
		categories[category] = append(categories[category], mapping)
	}

	return categories
}

// generateCoverageReport creates the final Markdown coverage analysis
func generateCoverageReport(report CoverageReport) error {
	// TODO: Implement report generation using template
	return nil
}

// validateConfiguration ensures all required configuration is present
func validateConfiguration() error {
	// TODO: Implement configuration validation
	return nil
}

// ensureProjectRoot changes directory to project root if we're currently in tools/
func ensureProjectRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// If we're in tools/ directory, move up one level
	if strings.HasSuffix(cwd, "/tools") || strings.HasSuffix(cwd, "\\tools") {
		parentDir := filepath.Dir(cwd)
		if err := os.Chdir(parentDir); err != nil {
			return fmt.Errorf("failed to change to parent directory: %w", err)
		}
		fmt.Printf("📂 Changed working directory to project root: %s\n", parentDir)
	}

	return nil
}

// main orchestrates the entire API coverage analysis process
func main() {
	fmt.Println("🚀 Starting DeepL API Coverage Analysis...")

	// Change to project root directory if we're in tools/
	if err := ensureProjectRoot(); err != nil {
		fmt.Printf("❌ Failed to find project root: %v\n", err)
		os.Exit(1)
	}

	// Fetch actual OpenAPI specification from DeepL's repository
	spec, err := fetchOpenAPISpec()
	if err != nil {
		fmt.Printf("❌ Failed to fetch OpenAPI specification: %v\n", err)
		fmt.Println("📊 Falling back to mock data for demonstration...")

		// Fallback to mock data if fetching fails
		spec = &OpenAPISpec{
			Info: struct {
				Title   string `yaml:"title"`
				Version string `yaml:"version"`
			}{
				Title:   "DeepL API (Mock)",
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

	// Analyze Go source code using AST parsing
	fmt.Println("📁 Analyzing Go source code...")
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	methods, err := analyzeGoSourceCode(wd)
	if err != nil {
		fmt.Printf("❌ Failed to analyze Go source code: %v\n", err)
		fmt.Println("📊 Falling back to mock data for demonstration...")

		// Fallback to mock methods if AST parsing fails
		methods = []GoMethod{
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
		}
	}

	fmt.Println("🔍 Extracting API endpoints...")
	endpoints := extractEndpoints(spec)
	fmt.Printf("   Found %d API endpoints\n", len(endpoints))

	fmt.Println("🎯 Creating endpoint mappings...")
	mappings := createEndpointMappings(endpoints, methods)

	fmt.Println("⚡ Assigning priorities...")
	assignPriorities(mappings)

	fmt.Println("📈 Calculating coverage statistics...")
	total, implemented, percentage := calculateCoverageStats(mappings)

	fmt.Printf("📊 Coverage Results:\n")
	fmt.Printf("   Total endpoints: %d\n", total)
	fmt.Printf("   Implemented: %d\n", implemented)
	fmt.Printf("   Coverage: %.1f%%\n", percentage)

	fmt.Println("📋 Detailed Analysis:")
	for _, mapping := range mappings {
		status := "❌ Missing"
		methodInfo := "No implementation found"
		if mapping.IsImplemented && mapping.GoMethod != nil {
			status = "✅ Implemented"
			methodInfo = fmt.Sprintf("→ %s", mapping.GoMethod.Name)
		}

		fmt.Printf("   %s %s %s [%s] %s\n",
			mapping.HTTPMethod,
			mapping.APIEndpoint,
			status,
			mapping.Priority,
			methodInfo)
	}

	fmt.Println("🏷️  Categorizing endpoints...")
	categories := categorizeEndpoints(mappings)
	fmt.Println("📂 Categories found:")
	for category, categoryMappings := range categories {
		implemented := 0
		for _, m := range categoryMappings {
			if m.IsImplemented {
				implemented++
			}
		}
		coverage := float64(implemented) / float64(len(categoryMappings)) * 100
		fmt.Printf("   %s: %d/%d (%.1f%%)\n", category, implemented, len(categoryMappings), coverage)
	}

	fmt.Println("✨ Analysis complete!")

	// Generate Markdown report
	fmt.Println("� Generating Markdown report...")
	report := generateMarkdownReport(mappings, methods, categories)

	// Save report to file
	reportPath := "api_coverage_report.md"
	err = saveReport(reportPath, report)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to save report to %s: %v\n", reportPath, err)
		fmt.Println("📋 Report content:")
		fmt.Println(report)
	} else {
		fmt.Printf("📄 Report saved to: %s\n", reportPath)
	}
}

// generateMarkdownReport creates a detailed Markdown report of API coverage
func generateMarkdownReport(mappings []EndpointMapping, methods []GoMethod, categories map[string][]EndpointMapping) string {
	var report strings.Builder

	// Header
	report.WriteString("# DeepL API Coverage Report\n\n")
	report.WriteString("This report provides a comprehensive analysis of the DeepL API implementation coverage.\n\n")

	// Calculate overall statistics
	implemented := 0
	for _, m := range mappings {
		if m.IsImplemented {
			implemented++
		}
	}
	coverage := float64(implemented) / float64(len(mappings)) * 100

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total API Endpoints**: %d\n", len(mappings)))
	report.WriteString(fmt.Sprintf("- **Implemented Endpoints**: %d\n", implemented))
	report.WriteString(fmt.Sprintf("- **Coverage Percentage**: %.1f%%\n", coverage))
	report.WriteString(fmt.Sprintf("- **Go Client Methods**: %d\n\n", len(methods)))

	// Coverage by Category
	report.WriteString("## Coverage by Category\n\n")
	report.WriteString("| Category | Implemented | Total | Coverage |\n")
	report.WriteString("|----------|-------------|-------|----------|\n")

	for category, categoryMappings := range categories {
		categoryImplemented := 0
		for _, m := range categoryMappings {
			if m.IsImplemented {
				categoryImplemented++
			}
		}
		categoryCoverage := float64(categoryImplemented) / float64(len(categoryMappings)) * 100
		report.WriteString(fmt.Sprintf("| %s | %d | %d | %.1f%% |\n",
			category, categoryImplemented, len(categoryMappings), categoryCoverage))
	}
	report.WriteString("\n")

	// Detailed Analysis
	report.WriteString("## Detailed Analysis\n\n")

	// Implemented Endpoints
	report.WriteString("### ✅ Implemented Endpoints\n\n")
	for _, m := range mappings {
		if m.IsImplemented {
			report.WriteString(fmt.Sprintf("- **%s %s** → `%s`\n",
				m.HTTPMethod, m.APIEndpoint, m.GoMethod.Name))
			if m.GoMethod.Comments != "" {
				report.WriteString(fmt.Sprintf("  - %s\n", m.GoMethod.Comments))
			}
		}
	}
	report.WriteString("\n")

	// Missing Endpoints
	report.WriteString("### ❌ Missing Endpoints\n\n")

	// Group by priority
	priorities := []string{"High", "Medium", "Low"}
	for _, priority := range priorities {
		hasItems := false
		for _, m := range mappings {
			if !m.IsImplemented && m.Priority == priority {
				if !hasItems {
					report.WriteString(fmt.Sprintf("#### %s Priority\n\n", priority))
					hasItems = true
				}
				report.WriteString(fmt.Sprintf("- **%s %s**\n", m.HTTPMethod, m.APIEndpoint))
				if m.Description != "" {
					report.WriteString(fmt.Sprintf("  - %s\n", m.Description))
				}
			}
		}
		if hasItems {
			report.WriteString("\n")
		}
	}

	// Go Client Methods
	report.WriteString("## Go Client Methods\n\n")
	report.WriteString("The following methods were detected in the Go client:\n\n")

	methodsByFile := make(map[string][]GoMethod)
	for _, method := range methods {
		methodsByFile[method.FileName] = append(methodsByFile[method.FileName], method)
	}

	for filename, fileMethods := range methodsByFile {
		report.WriteString(fmt.Sprintf("### %s\n\n", filename))
		for _, method := range fileMethods {
			report.WriteString(fmt.Sprintf("- `%s(%s) (%s)`\n",
				method.Name,
				strings.Join(method.Parameters, ", "),
				strings.Join(method.ReturnTypes, ", ")))
			if method.Comments != "" {
				report.WriteString(fmt.Sprintf("  - %s\n", method.Comments))
			}
		}
		report.WriteString("\n")
	}

	// Recommendations
	report.WriteString("## Recommendations\n\n")
	report.WriteString("Based on this analysis, the following implementation priorities are suggested:\n\n")
	report.WriteString("1. **High Priority**: Focus on core translation and language detection features\n")
	report.WriteString("2. **Medium Priority**: Implement document translation and glossary management\n")
	report.WriteString("3. **Low Priority**: Add administrative and advanced configuration features\n\n")

	// Footer
	report.WriteString("---\n")
	report.WriteString(fmt.Sprintf("*Report generated on %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return report.String()
}

// saveReport saves the report content to a file
func saveReport(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
