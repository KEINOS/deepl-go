//go:build tools

// This file ensures that tools dependencies are retained during `go mod tidy`.
// The `tools` build tag prevents this file from being included in regular builds.

package main

// Tool dependencies for code generation and analysis
import (
	_ "gopkg.in/yaml.v3" // Required for OpenAPI specification parsing in testdata/gen_api_coverage.go
)