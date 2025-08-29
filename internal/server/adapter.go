package server

import (
	"strconv"

	workerapi "autoteam/api/worker"

	"github.com/labstack/echo/v4"
)

// APIAdapter adapts our existing handlers to implement the generated api.ServerInterface
type APIAdapter struct {
	handlers *Handlers
}

// NewAPIAdapter creates a new API adapter
func NewAPIAdapter(handlers *Handlers) *APIAdapter {
	return &APIAdapter{
		handlers: handlers,
	}
}

// Ensure APIAdapter implements workerapi.ServerInterface
var _ workerapi.ServerInterface = (*APIAdapter)(nil)

// GetHealth implements the generated ServerInterface
func (a *APIAdapter) GetHealth(ctx echo.Context) error {
	return a.handlers.GetHealth(ctx)
}

// GetStatus implements the generated ServerInterface
func (a *APIAdapter) GetStatus(ctx echo.Context) error {
	return a.handlers.GetStatus(ctx)
}

// GetLogs implements the generated ServerInterface
func (a *APIAdapter) GetLogs(ctx echo.Context, params workerapi.GetLogsParams) error {
	// Convert generated types back to our handler's expected format
	// We'll need to set query parameters manually
	if params.Role != nil {
		ctx.QueryParams().Set("role", string(*params.Role))
	}
	if params.Limit != nil {
		ctx.QueryParams().Set("limit", strconv.Itoa(*params.Limit))
	}
	return a.handlers.GetLogs(ctx)
}

// GetLogFile implements the generated ServerInterface
func (a *APIAdapter) GetLogFile(ctx echo.Context, filename string, params workerapi.GetLogFileParams) error {
	// Set the filename as a path parameter
	ctx.SetParamNames("filename")
	ctx.SetParamValues(filename)

	// Convert tail parameter
	if params.Tail != nil {
		ctx.QueryParams().Set("tail", strconv.Itoa(*params.Tail))
	}

	return a.handlers.GetLogFile(ctx)
}

// GetFlow implements the generated ServerInterface
func (a *APIAdapter) GetFlow(ctx echo.Context) error {
	return a.handlers.GetFlow(ctx)
}

// GetFlowSteps implements the generated ServerInterface
func (a *APIAdapter) GetFlowSteps(ctx echo.Context) error {
	return a.handlers.GetFlowSteps(ctx)
}

// GetMetrics implements the generated ServerInterface
func (a *APIAdapter) GetMetrics(ctx echo.Context) error {
	return a.handlers.GetMetrics(ctx)
}

// GetConfig implements the generated ServerInterface
func (a *APIAdapter) GetConfig(ctx echo.Context) error {
	return a.handlers.GetConfig(ctx)
}

// GetOpenAPISpec implements the generated ServerInterface
func (a *APIAdapter) GetOpenAPISpec(ctx echo.Context) error {
	return a.handlers.GetOpenAPISpec(ctx)
}

// GetSwaggerUI implements the generated ServerInterface
func (a *APIAdapter) GetSwaggerUI(ctx echo.Context) error {
	return a.handlers.GetSwaggerUI(ctx)
}
