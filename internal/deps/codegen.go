// Package deps ensures code generation dependencies are preserved in go.mod
// These imports are required by generated files but would be removed by go mod tidy
// since the generated files are in .gitignore
package deps

import (
	// Required by generated API client code
	_ "github.com/oapi-codegen/runtime"
	
	// Required by OpenAPI spec processing
	_ "github.com/getkin/kin-openapi/openapi3"
	
	// Required by YAML parsing in generated code
	_ "gopkg.in/yaml.v2"
)