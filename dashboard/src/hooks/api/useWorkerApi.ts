/**
 * Typed API hooks for worker operations
 * Provides type-safe hooks for all worker API endpoints
 */

import { useCustom } from "@refinedev/core";
import type {
  HealthResponse,
  StatusResponse,
  ConfigResponse,
  LogsResponse,
  FlowResponse,
  FlowStepsResponse,
  MetricsResponse,
  ControlPlaneHealthResponse,
} from "../../types/api";
import { components } from "../../types/generated/api";

interface UseApiOptions {
  enabled?: boolean;
  refetchInterval?: number;
  onSuccess?: (data: unknown) => void;
  onError?: (error: unknown) => void;
}

/**
 * Hook for control plane health check
 */
export const useControlPlaneHealth = (options?: UseApiOptions) => {
  const result = useCustom<ControlPlaneHealthResponse>({
    url: "/health",
    method: "get",
    queryOptions: {
      enabled: options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
  };
};

/**
 * Hook for worker runtime health check
 * Requires worker container to be running
 */
export const useWorkerRuntimeHealth = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<HealthResponse>({
    url: `/workers/${workerId}/runtime/health`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval || 5000, // Default 5s refresh
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker runtime status
 * Requires worker container to be running
 */
export const useWorkerRuntimeStatus = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<StatusResponse>({
    url: `/workers/${workerId}/runtime/status`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval || 10000, // Default 10s refresh
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker runtime configuration
 * Requires worker container to be running
 */
export const useWorkerRuntimeConfig = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<ConfigResponse>({
    url: `/workers/${workerId}/runtime/config`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker settings from database (CRUD)
 * Always available, doesn't require running container
 */
export const useWorkerSettings = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<components["schemas"]["WorkerSettingsResponse"]>({
    url: `/workers/${workerId}/settings`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
  };
};

/**
 * Hook for worker runtime information from registry
 */
export const useWorkerRuntime = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<components["schemas"]["WorkerDetailsResponse"]>({
    url: `/workers/${workerId}/runtime`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval || 10000, // Default 10s refresh for runtime data
      onSuccess: options?.onSuccess,
      onError: options?.onError,
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
  };
};

/**
 * Hook for worker runtime logs
 * Requires worker container to be running
 */
export const useWorkerRuntimeLogs = (
  workerId: string | undefined,
  params?: { role?: "collector" | "executor" | "both"; limit?: number },
  options?: UseApiOptions
) => {
  const result = useCustom<LogsResponse>({
    url: `/workers/${workerId}/runtime/logs`,
    method: "get",
    config: {
      query: params,
    },
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker runtime flow execution state
 * Requires worker container to be running
 */
export const useWorkerRuntimeFlow = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<FlowResponse>({
    url: `/workers/${workerId}/runtime/flow`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker runtime flow steps with execution details
 * Requires worker container to be running
 */
export const useWorkerRuntimeFlowSteps = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<FlowStepsResponse>({
    url: `/workers/${workerId}/runtime/flow/steps`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval || 5000, // Default 5s refresh
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for worker runtime metrics
 * Requires worker container to be running
 */
export const useWorkerRuntimeMetrics = (
  workerId: string | undefined,
  options?: UseApiOptions
) => {
  const result = useCustom<MetricsResponse>({
    url: `/workers/${workerId}/runtime/metrics`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && options?.enabled !== false,
      refetchInterval: options?.refetchInterval || 30000, // Default 30s refresh
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};

/**
 * Hook for fetching worker runtime log file content
 * Requires worker container to be running
 */
export const useWorkerRuntimeLogFile = (
  workerId: string | undefined,
  filename: string | undefined,
  tail?: number,
  options?: UseApiOptions
) => {
  const result = useCustom({
    url: `/workers/${workerId}/runtime/logs/${filename}`,
    method: "get",
    config: {
      query: tail ? { tail } : undefined,
    },
    queryOptions: {
      enabled: !!workerId && !!filename && options?.enabled !== false,
      refetchInterval: options?.refetchInterval,
      onSuccess: options?.onSuccess,
      onError: options?.onError,
      retry: false, // Don't retry if container not running
    },
  });

  return {
    data: result.data?.data,
    isLoading: result.isLoading,
    error: result.error,
    refetch: result.refetch,
    isContainerNotRunning: result.error?.status === 502,
  };
};
