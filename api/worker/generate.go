package worker

// Generate server interface and types from OpenAPI specification
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config server.cfg.yaml openapi.yaml

// Generate client code for testing (optional) - disabled for now
// //go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config client.cfg.yaml openapi.yaml
