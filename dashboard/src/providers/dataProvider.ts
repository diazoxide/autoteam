import { DataProvider } from "@refinedev/core";
import type {
  HealthResponse,
  StatusResponse,
  ConfigResponse,
  LogsResponse,
  FlowResponse,
  FlowStepsResponse,
  MetricsResponse,
  ErrorResponse,
  ControlPlaneHealthResponse,
} from "../types/api";
import type { components } from "../types/generated/api";

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

  private async request<T>(path: string, options?: RequestInit): Promise<T> {
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

    // Handle 204 No Content responses (like DELETE operations)
    if (
      response.status === 204 ||
      response.headers.get("content-length") === "0"
    ) {
      return {} as T;
    }

    return response.json();
  }

  // Control plane endpoints
  async getHealth(): Promise<ControlPlaneHealthResponse> {
    return this.request<ControlPlaneHealthResponse>("/health");
  }

  // Worker management
  async getWorkers(): Promise<components["schemas"]["WorkersResponse"]> {
    return this.request<components["schemas"]["WorkersResponse"]>("/workers");
  }

  async getWorker(
    workerId: string
  ): Promise<components["schemas"]["WorkerResponse"]> {
    return this.request<components["schemas"]["WorkerResponse"]>(
      `/workers/${workerId}`
    );
  }

  // Worker API proxy endpoints
  // Runtime endpoints - require running worker containers
  async getWorkerRuntimeHealth(workerId: string): Promise<HealthResponse> {
    return this.request<HealthResponse>(`/workers/${workerId}/runtime/health`);
  }

  async getWorkerRuntimeStatus(workerId: string): Promise<StatusResponse> {
    return this.request<StatusResponse>(`/workers/${workerId}/runtime/status`);
  }

  async getWorkerRuntimeConfig(workerId: string): Promise<ConfigResponse> {
    return this.request<ConfigResponse>(`/workers/${workerId}/runtime/config`);
  }

  async getWorkerRuntimeLogs(
    workerId: string,
    params?: { role?: "collector" | "executor" | "both"; limit?: number }
  ): Promise<LogsResponse> {
    const queryString = params
      ? `?${new URLSearchParams(params as Record<string, string>).toString()}`
      : "";
    return this.request<LogsResponse>(
      `/workers/${workerId}/runtime/logs${queryString}`
    );
  }

  async getWorkerRuntimeLogFile(
    workerId: string,
    filename: string,
    tail?: number
  ): Promise<string> {
    const queryString = tail ? `?tail=${tail}` : "";
    const response = await fetch(
      `${this.apiUrl}/workers/${workerId}/runtime/logs/${filename}${queryString}`
    );
    if (!response.ok) {
      throw new ApiError(response.status, `Failed to fetch log file`);
    }
    return response.text();
  }

  async getWorkerRuntimeFlow(workerId: string): Promise<FlowResponse> {
    return this.request<FlowResponse>(`/workers/${workerId}/runtime/flow`);
  }

  async getWorkerRuntimeFlowSteps(
    workerId: string
  ): Promise<FlowStepsResponse> {
    return this.request<FlowStepsResponse>(
      `/workers/${workerId}/runtime/flow/steps`
    );
  }

  async getWorkerRuntimeMetrics(workerId: string): Promise<MetricsResponse> {
    return this.request<MetricsResponse>(
      `/workers/${workerId}/runtime/metrics`
    );
  }

  async getWorkerSettings(
    workerId: string
  ): Promise<components["schemas"]["WorkerSettingsResponse"]> {
    return this.request<components["schemas"]["WorkerSettingsResponse"]>(
      `/workers/${workerId}/settings`
    );
  }

  async updateWorkerSettings(
    workerId: string,
    request: components["schemas"]["UpdateWorkerSettingsRequest"]
  ): Promise<components["schemas"]["WorkerSettingsResponse"]> {
    return this.request<components["schemas"]["WorkerSettingsResponse"]>(
      `/workers/${workerId}/settings`,
      {
        method: "PUT",
        body: JSON.stringify(request),
      }
    );
  }

  async getWorkerRuntime(
    workerId: string
  ): Promise<components["schemas"]["WorkerDetailsResponse"]> {
    return this.request<components["schemas"]["WorkerDetailsResponse"]>(
      `/workers/${workerId}/runtime`
    );
  }

  // CRUD operations
  async createWorker(
    request: components["schemas"]["CreateWorkerRequest"]
  ): Promise<components["schemas"]["WorkerResponse"]> {
    return this.request<components["schemas"]["WorkerResponse"]>("/workers", {
      method: "POST",
      body: JSON.stringify(request),
    });
  }

  async updateWorker(
    workerId: string,
    request: components["schemas"]["UpdateWorkerRequest"]
  ): Promise<components["schemas"]["WorkerResponse"]> {
    return this.request<components["schemas"]["WorkerResponse"]>(
      `/workers/${workerId}`,
      {
        method: "PUT",
        body: JSON.stringify(request),
      }
    );
  }

  async deleteWorker(workerId: string): Promise<void> {
    await this.request<void>(`/workers/${workerId}`, {
      method: "DELETE",
    });
  }
}

// Create the Refine data provider
export const createControlPlaneDataProvider = (
  apiUrl: string
): DataProvider => {
  const client = new ControlPlaneApiClient(apiUrl);

  return {
    getApiUrl: () => apiUrl,

    // Get list of workers
    getList: async ({ resource }) => {
      if (resource === "workers") {
        const data = await client.getWorkers();
        return {
          data: (data.workers || []) as any,
          total: data.total || 0,
        };
      }

      throw new Error(`Resource ${resource} not supported`);
    },

    // Get single worker details
    getOne: async ({ resource, id }) => {
      if (resource === "workers") {
        try {
          // Get CRUD worker data (name, prompt, enabled, settings, etc.)
          const workerData = await client.getWorker(id as string);

          return {
            data: workerData as any,
          };
        } catch (error) {
          console.error("Failed to fetch worker data:", error);
          throw error;
        }
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
              return data; // Return the WorkerResponse directly, not data.worker
            } catch (error) {
              console.error(`Failed to fetch worker ${id}:`, error);
              return null;
            }
          })
        );

        return {
          data: workers.filter(Boolean) as any,
        };
      }

      throw new Error(`Resource ${resource} not supported`);
    },

    // Custom method for typed API endpoints
    custom: async (params: any) => {
      console.log("custom() called with params:", JSON.stringify(params, null, 2));
      // useCustomMutation sends data in 'payload' or 'values' depending on version
      const { url, method = "GET", payload, values, query, headers } = params;
      const requestData = payload || values;

      // Parse the URL to determine which typed method to use
      const urlPath = url.startsWith("/") ? url : `/${url}`;

      // Use typed client methods based on URL pattern
      if (urlPath === "/health") {
        const healthData = await client.getHealth();
        return { data: healthData };
      }

      // Worker-specific endpoints
      const workerMatch = urlPath.match(/^\/workers\/([^/]+)\/(.+)$/);
      if (workerMatch) {
        const [, workerId, endpoint] = workerMatch;

        switch (endpoint) {
          case "runtime": {
            const runtimeData = await client.getWorkerRuntime(workerId);
            return { data: runtimeData };
          }
          case "runtime/health": {
            const healthData = await client.getWorkerRuntimeHealth(workerId);
            return { data: healthData };
          }
          case "runtime/status": {
            const statusData = await client.getWorkerRuntimeStatus(workerId);
            return { data: statusData };
          }
          case "runtime/config": {
            const configData = await client.getWorkerRuntimeConfig(workerId);
            return { data: configData };
          }
          case "runtime/logs": {
            const logsData = await client.getWorkerRuntimeLogs(
              workerId,
              query as Record<string, unknown>
            );
            return { data: logsData };
          }
          case "runtime/flow": {
            const flowData = await client.getWorkerRuntimeFlow(workerId);
            return { data: flowData };
          }
          case "runtime/flow/steps": {
            const stepsData = await client.getWorkerRuntimeFlowSteps(workerId);
            return { data: stepsData };
          }
          case "runtime/metrics": {
            const metricsData = await client.getWorkerRuntimeMetrics(workerId);
            return { data: metricsData };
          }
          case "settings": {
            if (method.toUpperCase() === "PUT") {
              // Handle PUT request for updating settings
              console.log("PUT /workers/*/settings - requestData:", JSON.stringify(requestData, null, 2));
              const settingsData = await client.updateWorkerSettings(
                workerId,
                requestData as components["schemas"]["UpdateWorkerSettingsRequest"]
              );
              return { data: settingsData };
            } else {
              // Handle GET request for fetching settings
              try {
                const settingsData = await client.getWorkerSettings(workerId);
                return { data: settingsData };
              } catch (error) {
                // 404 is expected for new workers without settings - return empty settings
                if (error instanceof ApiError && error.status === 404) {
                  return { data: { settings: {} } };
                }
                throw error;
              }
            }
          }
          default:
            // Check if it's a runtime log file request
            if (endpoint.startsWith("runtime/logs/")) {
              const filename = endpoint.substring(13); // Remove "runtime/logs/"
              const content = await client.getWorkerRuntimeLogFile(
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
        body: requestData ? JSON.stringify(requestData) : undefined,
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const responseData = await response.json();
      return { data: responseData };
    },

    // CRUD operations
    create: async ({ resource, variables }) => {
      if (resource === "workers") {
        const transformedVariables = { ...variables } as any;

        // Transform flow steps env field from form format to API format
        if (transformedVariables.settings?.flow) {
          transformedVariables.settings.flow =
            transformedVariables.settings.flow.map((step: any) => {
              // Transform env from form array format [{key, value}] to API object format {[key]: value}
              let envObject: Record<string, string> = {};
              if (Array.isArray(step.env)) {
                // Form format: [{key: "KEY1", value: "value1"}, {key: "KEY2", value: "value2"}]
                step.env.forEach((envItem: any) => {
                  if (
                    envItem &&
                    envItem.key &&
                    typeof envItem.key === "string"
                  ) {
                    envObject[envItem.key] = String(envItem.value || "");
                  }
                });
              } else if (step.env && typeof step.env === "object") {
                // Already in correct format: {KEY1: "value1", KEY2: "value2"}
                envObject = step.env;
              }

              return {
                ...step,
                env: envObject,
                args: Array.isArray(step.args) ? step.args : [],
                depends_on: Array.isArray(step.depends_on)
                  ? step.depends_on
                  : [],
              };
            });
        }

        const data = await client.createWorker(
          transformedVariables as components["schemas"]["CreateWorkerRequest"]
        );
        return {
          data: data as any,
        };
      }

      throw new Error(
        `Create operation not supported for resource ${resource}`
      );
    },

    update: async ({ resource, id, variables }) => {
      if (resource === "workers") {
        const transformedVariables = { ...variables } as any;

        // Separate basic worker fields from settings
        const { settings, ...workerFields } = transformedVariables;

        // Update basic worker info if there are basic fields to update
        let workerData;
        if (Object.keys(workerFields).length > 0) {
          workerData = await client.updateWorker(
            id as string,
            workerFields as components["schemas"]["UpdateWorkerRequest"]
          );
        }

        // Update settings if provided (including flow steps)
        if (settings) {
          // Transform flow steps from edit format to input format
          const settingsUpdate = { ...settings };
          if (settingsUpdate.flow) {
            settingsUpdate.flow = settingsUpdate.flow.map((step: any) => {
              // Transform env from form array format [{key, value}] to API object format {[key]: value}
              let envObject: Record<string, string> = {};
              if (Array.isArray(step.env)) {
                // Form format: [{key: "KEY1", value: "value1"}, {key: "KEY2", value: "value2"}]
                step.env.forEach((envItem: any) => {
                  if (
                    envItem &&
                    envItem.key &&
                    typeof envItem.key === "string"
                  ) {
                    envObject[envItem.key] = String(envItem.value || "");
                  }
                });
              } else if (step.env && typeof step.env === "object") {
                // Already in correct format: {KEY1: "value1", KEY2: "value2"}
                envObject = step.env;
              }

              return {
                name: step.name,
                type: step.type,
                args: Array.isArray(step.args) ? step.args : [],
                env: envObject,
                depends_on: Array.isArray(step.depends_on)
                  ? step.depends_on
                  : [],
                input: step.input || "",
                output: step.output || "",
                skip_when: step.skip_when || "",
                dependency_policy: step.dependency_policy || "fail_fast",
                retry: step.retry || undefined,
              };
            });
          }

          await client.updateWorkerSettings(
            id as string,
            settingsUpdate as components["schemas"]["UpdateWorkerSettingsRequest"]
          );
        }

        // Return the updated worker data (fetch fresh data to ensure consistency)
        const finalData = await client.getWorker(id as string);
        return {
          data: finalData as any,
        };
      }

      throw new Error(
        `Update operation not supported for resource ${resource}`
      );
    },

    deleteOne: async ({ resource, id }) => {
      if (resource === "workers") {
        await client.deleteWorker(id as string);
        return {
          data: { id } as any,
        };
      }

      throw new Error(
        `Delete operation not supported for resource ${resource}`
      );
    },
  };
};
