//go:build tools

package main

// This file imports packages that are used when running go generate or other
// development tasks. This ensures these tools are available in CI environments
// and are tracked as dependencies.

import (
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
)
