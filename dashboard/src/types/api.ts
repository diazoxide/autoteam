/**
 * API Types - Re-exported from generated OpenAPI types
 * This file provides convenient type exports for use throughout the application
 */

import type { components, operations } from "./generated/api";

// Schema types
export type ControlPlaneHealthResponse =
  components["schemas"]["ControlPlaneHealthResponse"];
export type WorkersResponse = components["schemas"]["WorkersResponse"];
export type WorkerDetailsResponse =
  components["schemas"]["WorkerDetailsResponse"];
export type WorkerDetails = components["schemas"]["WorkerDetails"];
export type HealthResponse = components["schemas"]["HealthResponse"];
export type StatusResponse = components["schemas"]["StatusResponse"];
export type LogsResponse = components["schemas"]["LogsResponse"];
export type MetricsResponse = components["schemas"]["MetricsResponse"];
export type ConfigResponse = components["schemas"]["ConfigResponse"];
export type FlowResponse = components["schemas"]["FlowResponse"];
export type FlowStepsResponse = components["schemas"]["FlowStepsResponse"];
export type WorkerInfo = components["schemas"]["WorkerInfo"];
export type HealthCheck = components["schemas"]["HealthCheck"];
export type LogFile = components["schemas"]["LogFile"];
export type WorkerMetrics = components["schemas"]["WorkerMetrics"];
export type WorkerConfig = components["schemas"]["WorkerConfig"];
export type FlowInfo = components["schemas"]["FlowInfo"];
export type FlowStepInfo = components["schemas"]["FlowStepInfo"];
export type ErrorResponse = components["schemas"]["ErrorResponse"];

// Operation types
export type GetHealthOperation = operations["getHealth"];
export type GetWorkersOperation = operations["getWorkers"];
export type GetWorkerOperation = operations["getWorker"];
export type GetWorkerRuntimeHealthOperation =
  operations["getWorkerRuntimeHealth"];
export type GetWorkerRuntimeStatusOperation =
  operations["getWorkerRuntimeStatus"];
export type GetWorkerRuntimeConfigOperation =
  operations["getWorkerRuntimeConfig"];
export type GetWorkerRuntimeLogsOperation = operations["getWorkerRuntimeLogs"];
export type GetWorkerRuntimeLogFileOperation =
  operations["getWorkerRuntimeLogFile"];
export type GetWorkerRuntimeFlowOperation = operations["getWorkerRuntimeFlow"];
export type GetWorkerRuntimeFlowStepsOperation =
  operations["getWorkerRuntimeFlowSteps"];
export type GetWorkerRuntimeMetricsOperation =
  operations["getWorkerRuntimeMetrics"];

// Helper type for API responses
export type ApiResponse<T> =
  | {
      data: T;
      error?: never;
    }
  | {
      data?: never;
      error: ErrorResponse;
    };

// Convenience type aliases for common use cases
export type Worker = components["schemas"]["WorkerResponse"];
export type WorkerStatus = StatusResponse["status"];
export type WorkerMode = StatusResponse["mode"];
export type HealthStatus = HealthResponse["status"];
