import { DataProvider } from "@refinedev/core";
import type { 
  WorkersResponse, 
  WorkerDetailsResponse,
  HealthResponse,
  StatusResponse,
  ConfigResponse,
  LogsResponse,
  FlowResponse,
  FlowStepsResponse,
  MetricsResponse,
  ErrorResponse,
  ControlPlaneHealthResponse
} from "../types/api";

// Custom error class for API errors
class ApiError extends Error {
  status: number;
  data?: ErrorResponse;

  constructor(status: number, message: string, data?: ErrorResponse) {
    super(message);
    this.status = status;
    this.data = data;
  }
}

// Type-safe API client
class ControlPlaneApiClient {
  private apiUrl: string;

  constructor(apiUrl: string) {
    this.apiUrl = apiUrl;
  }

  private async request<T>(
    path: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${this.apiUrl}${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });

    if (!response.ok) {
      let errorData: ErrorResponse | undefined;
      try {
        errorData = await response.json();
      } catch {
        // Ignore JSON parse errors
      }
      throw new ApiError(
        response.status,
        errorData?.error || `HTTP error! status: ${response.status}`,
        errorData
      );
    }

    return response.json();
  }

  // Control plane endpoints
  async getHealth(): Promise<ControlPlaneHealthResponse> {
    return this.request<ControlPlaneHealthResponse>("/health");
  }

  // Worker management
  async getWorkers(): Promise<WorkersResponse> {
    return this.request<WorkersResponse>("/workers");
  }

  async getWorker(workerId: string): Promise<WorkerDetailsResponse> {
    return this.request<WorkerDetailsResponse>(`/workers/${workerId}`);
  }

  // Worker API proxy endpoints
  async getWorkerHealth(workerId: string): Promise<HealthResponse> {
    return this.request<HealthResponse>(`/workers/${workerId}/health`);
  }

  async getWorkerStatus(workerId: string): Promise<StatusResponse> {
    return this.request<StatusResponse>(`/workers/${workerId}/status`);
  }

  async getWorkerConfig(workerId: string): Promise<ConfigResponse> {
    return this.request<ConfigResponse>(`/workers/${workerId}/config`);
  }

  async getWorkerLogs(
    workerId: string,
    params?: { role?: "collector" | "executor" | "both"; limit?: number }
  ): Promise<LogsResponse> {
    const queryString = params ? 
      `?${new URLSearchParams(params as any).toString()}` : "";
    return this.request<LogsResponse>(`/workers/${workerId}/logs${queryString}`);
  }

  async getWorkerLogFile(
    workerId: string,
    filename: string,
    tail?: number
  ): Promise<string> {
    const queryString = tail ? `?tail=${tail}` : "";
    const response = await fetch(
      `${this.apiUrl}/workers/${workerId}/logs/${filename}${queryString}`
    );
    if (!response.ok) {
      throw new ApiError(response.status, `Failed to fetch log file`);
    }
    return response.text();
  }

  async getWorkerFlow(workerId: string): Promise<FlowResponse> {
    return this.request<FlowResponse>(`/workers/${workerId}/flow`);
  }

  async getWorkerFlowSteps(workerId: string): Promise<FlowStepsResponse> {
    return this.request<FlowStepsResponse>(`/workers/${workerId}/flow/steps`);
  }

  async getWorkerMetrics(workerId: string): Promise<MetricsResponse> {
    return this.request<MetricsResponse>(`/workers/${workerId}/metrics`);
  }
}

// Create the Refine data provider
export const createControlPlaneDataProvider = (apiUrl: string): DataProvider => {
  const client = new ControlPlaneApiClient(apiUrl);

  return {
    getApiUrl: () => apiUrl,

    // Get list of workers
    getList: async ({ resource }) => {
      if (resource === "workers") {
        const data = await client.getWorkers();
        return {
          data: (data.workers || []) as any[],
          total: data.total || 0,
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Get single worker details
    getOne: async ({ resource, id }) => {
      if (resource === "workers") {
        const data = await client.getWorker(id as string);
        return {
          data: data.worker as any,
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Get multiple workers
    getMany: async ({ resource, ids }) => {
      if (resource === "workers") {
        const workers = await Promise.all(
          ids.map(async (id) => {
            try {
              const data = await client.getWorker(id as string);
              return data.worker;
            } catch (error) {
              console.error(`Failed to fetch worker ${id}:`, error);
              return null;
            }
          })
        );
        
        return {
          data: workers.filter(Boolean) as any[],
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Custom method for typed API endpoints
    custom: async ({ url, method = "GET", payload, query, headers }) => {
      // Parse the URL to determine which typed method to use
      const urlPath = url.startsWith("/") ? url : `/${url}`;
      
      // Use typed client methods based on URL pattern
      if (urlPath === "/health") {
        const data = await client.getHealth();
        return { data };
      }
      
      // Worker-specific endpoints
      const workerMatch = urlPath.match(/^\/workers\/([^/]+)\/(.+)$/);
      if (workerMatch) {
        const [, workerId, endpoint] = workerMatch;
        
        switch (endpoint) {
          case "health": {
            const healthData = await client.getWorkerHealth(workerId);
            return { data: healthData };
          }
          case "status": {
            const statusData = await client.getWorkerStatus(workerId);
            return { data: statusData };
          }
          case "config": {
            const configData = await client.getWorkerConfig(workerId);
            return { data: configData };
          }
          case "logs": {
            const logsData = await client.getWorkerLogs(workerId, query as Record<string, unknown>);
            return { data: logsData };
          }
          case "flow": {
            const flowData = await client.getWorkerFlow(workerId);
            return { data: flowData };
          }
          case "flow/steps": {
            const stepsData = await client.getWorkerFlowSteps(workerId);
            return { data: stepsData };
          }
          case "metrics": {
            const metricsData = await client.getWorkerMetrics(workerId);
            return { data: metricsData };
          }
          default:
            // Check if it's a log file request
            if (endpoint.startsWith("logs/")) {
              const filename = endpoint.substring(5);
              const content = await client.getWorkerLogFile(
                workerId,
                filename,
                (query as Record<string, unknown>)?.tail as number
              );
              return { data: content };
            }
        }
      }
      
      // Fallback to generic request for unknown endpoints
      const response = await fetch(`${apiUrl}${urlPath}`, {
        method,
        headers: {
          "Content-Type": "application/json",
          ...headers,
        },
        body: payload ? JSON.stringify(payload) : undefined,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      return { data };
    },

    // Not implemented for read-only API
    create: async () => {
      throw new Error("Create operation not supported by control plane API");
    },

    update: async () => {
      throw new Error("Update operation not supported by control plane API");
    },

    deleteOne: async () => {
      throw new Error("Delete operation not supported by control plane API");
    },
  };
};