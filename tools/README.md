# Development Tools

This directory contains development tools for the deepl-go library.

## API Coverage Analysis Tool

### Overview

The API coverage analysis system compares the official DeepL API specification with the currently implemented Go methods to identify coverage gaps and prioritize future development.

### Files

- **`gen_api_coverage.go`** - Main generator tool for API coverage analysis
- **`tools.go`** - Go module dependency management for development tools
- **`README.md`** - This documentation file

### Generated Files

- **`../testdata/openapi_spec.yaml`** - Cached copy of official DeepL OpenAPI specification
- **`../api_coverage_report.md`** - Comprehensive API coverage report

## Usage

### Generate Coverage Report

```bash
# From project root
go generate ./tools

# Or from tools directory
cd tools && go generate

# Or run directly
go run ./tools/gen_api_coverage.go
```

### Report Structure

The generated `api_coverage_report.md` includes:

- **Executive Summary**: Overall coverage statistics and metrics
- **Coverage by Category**: Breakdown by API categories (translation, languages, etc.)
- **Implemented Endpoints**: Currently working Go methods with their API mappings
- **Missing Endpoints**: Prioritized list of unimplemented features (High/Medium/Low priority)
- **Go Client Methods**: Detailed list of detected methods with signatures and documentation
- **Recommendations**: Implementation priorities and next steps

## How It Works

### 1. OpenAPI Specification Fetching
- Downloads latest official specification from DeepL's GitHub repository
- Caches locally in `testdata/openapi_spec.yaml` to avoid repeated downloads
- Includes HTTP timeout and retry logic for reliability

### 2. Go Source Code Analysis
- Uses Go AST parsing to detect all client methods automatically
- Extracts method signatures, parameters, return types, and documentation
- Filters for actual client methods (with `*Client` receiver)
- Processes all `.go` files except tests and tools

### 3. Intelligent Endpoint Mapping
- Maps API endpoints to Go methods using multiple strategies:
  - Operation ID matching (e.g., `translateText` → `TranslateText`)
  - Path-based pattern matching (e.g., `/translate` → methods containing "Translate")
- Assigns priority levels based on API endpoint importance and usage patterns

### 4. Report Generation
- Creates comprehensive Markdown report with detailed analysis
- Categorizes endpoints by functionality (translation, languages, usage, etc.)
- Provides actionable recommendations for future development

## Current Status

As of the latest analysis:

- **Total API Endpoints**: 25 official DeepL API endpoints
- **Implemented Endpoints**: 5 endpoints (20% coverage)
- **Go Client Methods**: 12 detected methods across multiple files
- **High Priority Missing**: Document translation, glossary management
- **Categories**: Full coverage for translation/languages/usage, gaps in utilities/administration

## Integration

### Development Workflow
1. Run coverage analysis to identify implementation gaps
2. Choose high-priority missing endpoints for implementation
3. Implement new client methods following existing patterns
4. Re-run analysis to verify coverage improvement
5. Commit updated coverage report with implementation

### CI/CD Integration
- Coverage analysis can be automated in CI pipelines
- Detects API specification changes automatically
- Provides coverage regression detection
- Supports automated documentation updates

## Technical Details

### Dependencies
- `gopkg.in/yaml.v3` for OpenAPI YAML parsing
- Go standard library packages for AST analysis and HTTP operations
- No external runtime dependencies for the main library

### Security Considerations
- HTTP requests only to trusted sources (github.com/DeepLcom)
- File operations restricted to project directories
- No sensitive data processing or storage
- Cached files are safe to commit and share

### Performance
- AST parsing is fast and efficient for the current codebase size
- HTTP caching reduces redundant API specification downloads
- Report generation typically completes in under 2 seconds

## Contributing

When adding new API endpoint implementations:

1. Implement the new client method following existing patterns
2. Add appropriate tests and documentation
3. Run `go generate ./tools` to update the coverage report
4. Include the updated report in your pull request
5. Verify coverage percentage improvement in the executive summary

The coverage analysis helps maintain high-quality API compatibility and guides strategic development decisions.