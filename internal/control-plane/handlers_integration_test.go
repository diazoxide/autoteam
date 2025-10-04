package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	controlplaneapi "autoteam/api/control-plane"
	"autoteam/internal/config"
	"autoteam/internal/logger"
	"autoteam/internal/types"
	"autoteam/internal/worker"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockWorkerRepo implements the Repository interface for testing
type MockWorkerRepo struct {
	mock.Mock
}

func (m *MockWorkerRepo) Create(ctx context.Context, worker *worker.Worker) error {
	args := m.Called(ctx, worker)
	return args.Error(0)
}

func (m *MockWorkerRepo) Update(ctx context.Context, worker *worker.Worker) error {
	args := m.Called(ctx, worker)
	return args.Error(0)
}

func (m *MockWorkerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWorkerRepo) GetByID(ctx context.Context, id uuid.UUID) (*worker.Worker, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*worker.Worker), args.Error(1)
}

func (m *MockWorkerRepo) GetByName(ctx context.Context, name string) (*worker.Worker, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*worker.Worker), args.Error(1)
}

func (m *MockWorkerRepo) List(ctx context.Context, filters ...worker.Filter) ([]*worker.Worker, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*worker.Worker), args.Error(1)
}

func (m *MockWorkerRepo) ListEnabled(ctx context.Context) ([]*worker.Worker, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*worker.Worker), args.Error(1)
}

func (m *MockWorkerRepo) CreateOrUpdateSettings(ctx context.Context, settings *worker.WorkerSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockWorkerRepo) GetSettingsByWorkerID(ctx context.Context, workerID uuid.UUID) (*worker.WorkerSettings, error) {
	args := m.Called(ctx, workerID)
	return args.Get(0).(*worker.WorkerSettings), args.Error(1)
}

func (m *MockWorkerRepo) DeleteSettings(ctx context.Context, workerID uuid.UUID) error {
	args := m.Called(ctx, workerID)
	return args.Error(0)
}

func (m *MockWorkerRepo) CreateFlowSteps(ctx context.Context, steps []worker.FlowStep) error {
	args := m.Called(ctx, steps)
	return args.Error(0)
}

func (m *MockWorkerRepo) UpdateFlowSteps(ctx context.Context, workerID uuid.UUID, steps []worker.FlowStep) error {
	args := m.Called(ctx, workerID, steps)
	return args.Error(0)
}

func (m *MockWorkerRepo) DeleteFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) error {
	args := m.Called(ctx, workerID)
	return args.Error(0)
}

func (m *MockWorkerRepo) GetFlowStepsByWorkerID(ctx context.Context, workerID uuid.UUID) ([]worker.FlowStep, error) {
	args := m.Called(ctx, workerID)
	return args.Get(0).([]worker.FlowStep), args.Error(1)
}

func (m *MockWorkerRepo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkerRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockWorkerRepo) Count(ctx context.Context, filters ...worker.Filter) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func setupTestHandler() (*Handlers, *MockWorkerRepo) {
	mockRepo := &MockWorkerRepo{}

	handlers := &Handlers{
		workerRepo: mockRepo,
		config:     &config.Config{}, // Basic config
		registry:   nil,              // Will be handled in handlers
		runtime:    nil,              // Will be handled in handlers
	}

	return handlers, mockRepo
}

func TestCreateWorker_MinimalRequest(t *testing.T) {
	h, mockRepo := setupTestHandler()

	// Setup expectations
	mockRepo.On("ExistsByName", mock.Anything, "test-worker").Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*worker.Worker")).Return(nil)
	mockRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&worker.Worker{
		ID:      uuid.New(),
		Name:    "test-worker",
		Prompt:  "test prompt",
		Enabled: true,
	}, nil)

	// Create minimal request
	reqBody := controlplaneapi.CreateWorkerRequest{
		Name:   "test-worker",
		Prompt: "test prompt",
	}

	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(reqBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	log, _ := logger.NewLogger(logger.InfoLevel)
	c.Set("logger", log)

	// Execute
	err := h.CreateWorker(c)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response controlplaneapi.WorkerResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-worker", response.Name)
	assert.Equal(t, "test prompt", response.Prompt)
	assert.True(t, response.Enabled)

	mockRepo.AssertExpectations(t)
}

func TestCreateWorker_WithComplexFlow(t *testing.T) {
	h, mockRepo := setupTestHandler()

	// Setup expectations
	mockRepo.On("ExistsByName", mock.Anything, "complex-worker").Return(false, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*worker.Worker")).Return(nil)
	mockRepo.On("CreateOrUpdateSettings", mock.Anything, mock.AnythingOfType("*worker.WorkerSettings")).Return(nil)
	mockRepo.On("CreateFlowSteps", mock.Anything, mock.AnythingOfType("[]worker.FlowStep")).Return(nil)
	mockRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&worker.Worker{
		ID:      uuid.New(),
		Name:    "complex-worker",
		Prompt:  "complex prompt",
		Enabled: true,
		Settings: &worker.WorkerSettings{
			SleepDuration: 60,
			TeamName:      "test-team",
			InstallDeps:   true,
			MaxAttempts:   5,
		},
		FlowSteps: []worker.FlowStep{
			{
				Name:             "step1",
				Type:             "claude",
				Order:            1,
				Args:             []string{"--verbose"},
				Env:              map[string]string{"NODE_ENV": "test"},
				Input:            "Process this",
				Output:           "{{.stdout}}",
				DependencyPolicy: "fail_fast",
				Retry: &worker.RetryConfig{
					MaxAttempts: 3,
					Delay:       5,
					Backoff:     "exponential",
					MaxDelay:    60,
				},
			},
		},
	}, nil)

	// Create complex request
	enabled := true
	sleepDuration := 60
	teamName := "test-team"
	installDeps := true
	maxAttempts := 5

	argsList := []string{"--verbose"}
	envMap := map[string]string{"NODE_ENV": "test"}
	dependsOn := []string{}
	input := "Process this"
	output := "{{.stdout}}"
	skipWhen := ""
	dependencyPolicy := controlplaneapi.FlowStepInputDependencyPolicyFailFast

	retryMaxAttempts := 3
	retryDelay := 5
	retryBackoff := controlplaneapi.Exponential
	retryMaxDelay := 60

	reqBody := controlplaneapi.CreateWorkerRequest{
		Name:    "complex-worker",
		Prompt:  "complex prompt",
		Enabled: &enabled,
		Settings: &controlplaneapi.WorkerSettingsInput{
			SleepDuration: &sleepDuration,
			TeamName:      &teamName,
			InstallDeps:   &installDeps,
			MaxAttempts:   &maxAttempts,
			Flow: &[]controlplaneapi.FlowStepInput{
				{
					Name:             "step1",
					Type:             "claude",
					Args:             &argsList,
					Env:              &envMap,
					DependsOn:        &dependsOn,
					Input:            &input,
					Output:           &output,
					SkipWhen:         &skipWhen,
					DependencyPolicy: &dependencyPolicy,
					Retry: &controlplaneapi.RetryConfig{
						MaxAttempts: &retryMaxAttempts,
						Delay:       &retryDelay,
						Backoff:     &retryBackoff,
						MaxDelay:    &retryMaxDelay,
					},
				},
			},
		},
	}

	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(reqBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	log, _ := logger.NewLogger(logger.InfoLevel)
	c.Set("logger", log)

	// Execute
	err := h.CreateWorker(c)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response controlplaneapi.WorkerResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "complex-worker", response.Name)
	assert.Equal(t, "complex prompt", response.Prompt)
	assert.True(t, response.Enabled)
	assert.NotNil(t, response.Settings)
	assert.Equal(t, 60, response.Settings.SleepDuration)
	assert.Equal(t, "test-team", response.Settings.TeamName)
	assert.NotNil(t, response.Settings.Flow)
	assert.Len(t, *response.Settings.Flow, 1)

	flowStep := (*response.Settings.Flow)[0]
	assert.Equal(t, "step1", flowStep.Name)
	assert.Equal(t, "claude", flowStep.Type)
	assert.NotNil(t, flowStep.Args)
	assert.Contains(t, *flowStep.Args, "--verbose")
	assert.NotNil(t, flowStep.Env)
	assert.Equal(t, "test", (*flowStep.Env)["NODE_ENV"])

	mockRepo.AssertExpectations(t)
}

func TestUpdateWorkerSettings_FlowStepsOnly(t *testing.T) {
	h, mockRepo := setupTestHandler()

	workerID := uuid.New()

	// Setup expectations
	mockRepo.On("GetByID", mock.Anything, workerID).Return(&worker.Worker{
		ID:      workerID,
		Name:    "test-worker",
		Prompt:  "test prompt",
		Enabled: true,
	}, nil)
	mockRepo.On("CreateOrUpdateSettings", mock.Anything, mock.AnythingOfType("*worker.WorkerSettings")).Return(nil)
	mockRepo.On("UpdateFlowSteps", mock.Anything, workerID, mock.AnythingOfType("[]worker.FlowStep")).Return(nil)
	mockRepo.On("GetSettingsByWorkerID", mock.Anything, workerID).Return(&worker.WorkerSettings{
		ID:            uuid.New(),
		WorkerID:      workerID,
		SleepDuration: 30,
		TeamName:      "autoteam",
		InstallDeps:   false,
		MaxAttempts:   3,
	}, nil)
	// Note: GetFlowStepsByWorkerID is not called when Flow is provided in request

	// Create update request with only flow steps
	envMap := map[string]string{"DEBUG": "true"}
	dependsOn := []string{}
	input := "Updated input"

	reqBody := controlplaneapi.UpdateWorkerSettingsRequest{
		Flow: &[]controlplaneapi.FlowStepInput{
			{
				Name:      "updated-step",
				Type:      "claude",
				Env:       &envMap,
				DependsOn: &dependsOn,
				Input:     &input,
			},
		},
	}

	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/workers/%s/settings", workerID), bytes.NewReader(reqBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetParamNames("worker_id")
	c.SetParamValues(workerID.String())
	log, _ := logger.NewLogger(logger.InfoLevel)
	c.Set("logger", log)

	// Execute
	err := h.UpdateWorkerSettings(c, workerID.String())

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response controlplaneapi.WorkerSettingsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 30, response.Settings.SleepDuration)
	assert.Equal(t, "autoteam", response.Settings.TeamName)
	assert.NotNil(t, response.Settings.Flow)
	assert.Len(t, *response.Settings.Flow, 1)

	flowStep := (*response.Settings.Flow)[0]
	assert.Equal(t, "updated-step", flowStep.Name)
	assert.Equal(t, "claude", flowStep.Type)
	assert.NotNil(t, flowStep.Env)
	assert.Equal(t, "true", (*flowStep.Env)["DEBUG"])

	mockRepo.AssertExpectations(t)
}

func TestCreateWorker_ValidationErrors(t *testing.T) {
	h, mockRepo := setupTestHandler()

	tests := []struct {
		name           string
		request        controlplaneapi.CreateWorkerRequest
		expectedStatus int
		setupMock      func()
	}{
		{
			name: "duplicate name",
			request: controlplaneapi.CreateWorkerRequest{
				Name:   "duplicate-worker",
				Prompt: "test prompt",
			},
			expectedStatus: http.StatusConflict,
			setupMock: func() {
				mockRepo.On("ExistsByName", mock.Anything, "duplicate-worker").Return(true, nil)
			},
		},
		{
			name: "empty name",
			request: controlplaneapi.CreateWorkerRequest{
				Name:   "",
				Prompt: "test prompt",
			},
			expectedStatus: http.StatusBadRequest,
			setupMock: func() {
				// No mock needed - validation should fail before DB call
			},
		},
		{
			name: "empty prompt",
			request: controlplaneapi.CreateWorkerRequest{
				Name:   "test-worker",
				Prompt: "",
			},
			expectedStatus: http.StatusBadRequest,
			setupMock: func() {
				// No mock needed - validation should fail before DB call
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.Mock = mock.Mock{}
			tt.setupMock()

			reqBytes, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/workers", bytes.NewReader(reqBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			e := echo.New()
			c := e.NewContext(req, rec)
			log, _ := logger.NewLogger(logger.InfoLevel)
			c.Set("logger", log)

			// Execute
			err := h.CreateWorker(c)

			// Verify
			if tt.expectedStatus >= 400 {
				var httpErr *echo.HTTPError
				require.ErrorAs(t, err, &httpErr)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateWorker_ValidationErrors(t *testing.T) {
	h, mockRepo := setupTestHandler()

	workerID := uuid.New()

	tests := []struct {
		name           string
		request        types.UpdateWorkerRequest
		expectedStatus int
		setupMock      func()
	}{
		{
			name: "empty name",
			request: types.UpdateWorkerRequest{
				Name: stringPtr(""),
			},
			expectedStatus: http.StatusBadRequest,
			setupMock: func() {
				// No mock needed - validation should fail before DB call
			},
		},
		{
			name: "empty prompt",
			request: types.UpdateWorkerRequest{
				Prompt: stringPtr(""),
			},
			expectedStatus: http.StatusBadRequest,
			setupMock: func() {
				// No mock needed - validation should fail before DB call
			},
		},
		{
			name: "worker not found",
			request: types.UpdateWorkerRequest{
				Name: stringPtr("updated-worker"),
			},
			expectedStatus: http.StatusNotFound,
			setupMock: func() {
				mockRepo.On("GetByID", mock.Anything, workerID).Return((*worker.Worker)(nil), fmt.Errorf("worker not found: %s", workerID))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.Mock = mock.Mock{}
			tt.setupMock()

			reqBytes, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/workers/%s", workerID), bytes.NewReader(reqBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			e := echo.New()
			c := e.NewContext(req, rec)
			c.SetParamNames("worker_id")
			c.SetParamValues(workerID.String())
			log, _ := logger.NewLogger(logger.InfoLevel)
			c.Set("logger", log)

			// Execute
			err := h.UpdateWorker(c, workerID.String())

			// Verify
			if tt.expectedStatus >= 400 {
				var httpErr *echo.HTTPError
				require.ErrorAs(t, err, &httpErr)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
