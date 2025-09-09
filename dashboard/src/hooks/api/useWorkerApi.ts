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

interface UseApiOptions {
  enabled?: boolean;
  refetchInterval?: number;
  onSuccess?: (data: any) => void;
  onError?: (error: any) => void;
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
 * Hook for worker health check
 */
export const useWorkerHealth = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<HealthResponse>({
    url: `/workers/${workerId}/health`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
      refetchInterval: options?.refetchInterval || 5000, // Default 5s refresh
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
 * Hook for worker status
 */
export const useWorkerStatus = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<StatusResponse>({
    url: `/workers/${workerId}/status`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
      refetchInterval: options?.refetchInterval || 10000, // Default 10s refresh
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
 * Hook for worker configuration
 */
export const useWorkerConfig = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<ConfigResponse>({
    url: `/workers/${workerId}/config`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
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
 * Hook for worker logs
 */
export const useWorkerLogs = (
  workerId: string | undefined,
  params?: { role?: "collector" | "executor" | "both"; limit?: number },
  options?: UseApiOptions
) => {
  const result = useCustom<LogsResponse>({
    url: `/workers/${workerId}/logs`,
    method: "get",
    config: {
      query: params,
    },
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
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
 * Hook for worker flow configuration
 */
export const useWorkerFlow = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<FlowResponse>({
    url: `/workers/${workerId}/flow`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
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
 * Hook for worker flow steps
 */
export const useWorkerFlowSteps = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<FlowStepsResponse>({
    url: `/workers/${workerId}/flow/steps`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
      refetchInterval: options?.refetchInterval || 5000, // Default 5s refresh
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
 * Hook for worker metrics
 */
export const useWorkerMetrics = (workerId: string | undefined, options?: UseApiOptions) => {
  const result = useCustom<MetricsResponse>({
    url: `/workers/${workerId}/metrics`,
    method: "get",
    queryOptions: {
      enabled: !!workerId && (options?.enabled !== false),
      refetchInterval: options?.refetchInterval || 30000, // Default 30s refresh
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
 * Hook for fetching worker log file content
 */
export const useWorkerLogFile = (
  workerId: string | undefined,
  filename: string | undefined,
  tail?: number,
  options?: UseApiOptions
) => {
  const result = useCustom({
    url: `/workers/${workerId}/logs/${filename}`,
    method: "get",
    config: {
      query: tail ? { tail } : undefined,
    },
    queryOptions: {
      enabled: !!workerId && !!filename && (options?.enabled !== false),
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