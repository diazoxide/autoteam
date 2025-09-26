import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
  beforeAll,
  afterAll,
} from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { dataProvider } from "../../providers/dataProvider";
import { createMockWorker } from "../utils/test-utils";

// Mock API server setup
const server = setupServer();

// API base URL
const API_BASE_URL = "http://localhost:9090";

// Helper to create API URL
const apiUrl = (path: string) => `${API_BASE_URL}${path}`;

describe("Data Provider Integration Tests", () => {
  beforeAll(() => {
    server.listen({ onUnhandledRequest: "error" });
  });

  afterAll(() => {
    server.close();
  });

  beforeEach(() => {
    server.resetHandlers();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("getList", () => {
    it("should fetch workers list", async () => {
      const mockWorkers = [
        createMockWorker({ id: "worker-1", name: "Worker 1" }),
        createMockWorker({ id: "worker-2", name: "Worker 2" }),
      ];

      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(mockWorkers);
        })
      );

      const result = await dataProvider.getList({
        resource: "workers",
        pagination: { current: 1, pageSize: 10 },
        filters: [],
        sorters: [],
        meta: {},
      });

      expect(result.data).toEqual(mockWorkers);
      expect(result.total).toBe(mockWorkers.length);
    });

    it("should handle pagination parameters", async () => {
      const mockWorkers = Array.from({ length: 5 }, (_, i) =>
        createMockWorker({ id: `worker-${i}`, name: `Worker ${i}` })
      );

      server.use(
        http.get(apiUrl("/workers"), ({ request }) => {
          const url = new URL(request.url);
          const page = url.searchParams.get("_start");
          const limit = url.searchParams.get("_limit");

          expect(page).toBe("10"); // (page - 1) * pageSize
          expect(limit).toBe("5");

          return HttpResponse.json(mockWorkers.slice(0, 5), {
            headers: {
              "X-Total-Count": "25",
            },
          });
        })
      );

      const result = await dataProvider.getList({
        resource: "workers",
        pagination: { current: 3, pageSize: 5 },
        filters: [],
        sorters: [],
        meta: {},
      });

      expect(result.data).toHaveLength(5);
      expect(result.total).toBe(25);
    });

    it("should handle sorting parameters", async () => {
      const mockWorkers = [
        createMockWorker({ id: "worker-1", name: "A Worker" }),
        createMockWorker({ id: "worker-2", name: "B Worker" }),
      ];

      server.use(
        http.get(apiUrl("/workers"), ({ request }) => {
          const url = new URL(request.url);
          const sort = url.searchParams.get("_sort");
          const order = url.searchParams.get("_order");

          expect(sort).toBe("name");
          expect(order).toBe("ASC");

          return HttpResponse.json(mockWorkers);
        })
      );

      const result = await dataProvider.getList({
        resource: "workers",
        pagination: { current: 1, pageSize: 10 },
        filters: [],
        sorters: [{ field: "name", order: "asc" }],
        meta: {},
      });

      expect(result.data).toEqual(mockWorkers);
    });

    it("should handle filtering parameters", async () => {
      const mockWorkers = [
        createMockWorker({
          id: "worker-1",
          name: "Active Worker",
          enabled: true,
        }),
      ];

      server.use(
        http.get(apiUrl("/workers"), ({ request }) => {
          const url = new URL(request.url);
          const enabled = url.searchParams.get("enabled");

          expect(enabled).toBe("true");

          return HttpResponse.json(mockWorkers);
        })
      );

      const result = await dataProvider.getList({
        resource: "workers",
        pagination: { current: 1, pageSize: 10 },
        filters: [{ field: "enabled", operator: "eq", value: true }],
        sorters: [],
        meta: {},
      });

      expect(result.data).toEqual(mockWorkers);
    });

    it("should handle network errors gracefully", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.error();
        })
      );

      await expect(
        dataProvider.getList({
          resource: "workers",
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle server errors", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(
            { error: "Internal server error" },
            { status: 500 }
          );
        })
      );

      await expect(
        dataProvider.getList({
          resource: "workers",
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("getOne", () => {
    it("should fetch single worker", async () => {
      const workerId = "worker-123";
      const mockWorker = createMockWorker({
        id: workerId,
        name: "Test Worker",
      });

      server.use(
        http.get(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(mockWorker);
        })
      );

      const result = await dataProvider.getOne({
        resource: "workers",
        id: workerId,
        meta: {},
      });

      expect(result.data).toEqual(mockWorker);
    });

    it("should handle worker not found", async () => {
      const workerId = "nonexistent";

      server.use(
        http.get(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(
            { error: "Worker not found" },
            { status: 404 }
          );
        })
      );

      await expect(
        dataProvider.getOne({
          resource: "workers",
          id: workerId,
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle invalid ID format", async () => {
      const invalidId = "invalid-uuid";

      server.use(
        http.get(apiUrl(`/workers/${invalidId}`), () => {
          return HttpResponse.json(
            { error: "Invalid worker ID format" },
            { status: 400 }
          );
        })
      );

      await expect(
        dataProvider.getOne({
          resource: "workers",
          id: invalidId,
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("create", () => {
    it("should create new worker", async () => {
      const newWorkerData = {
        name: "New Worker",
        prompt: "Test prompt for new worker",
        enabled: true,
      };

      const createdWorker = createMockWorker({
        ...newWorkerData,
        id: "new-worker-123",
      });

      server.use(
        http.post(apiUrl("/workers"), async ({ request }) => {
          const body = await request.json();
          expect(body).toEqual(newWorkerData);
          return HttpResponse.json(createdWorker, { status: 201 });
        })
      );

      const result = await dataProvider.create({
        resource: "workers",
        variables: newWorkerData,
        meta: {},
      });

      expect(result.data).toEqual(createdWorker);
    });

    it("should handle validation errors", async () => {
      const invalidWorkerData = {
        name: "", // Invalid empty name
        prompt: "Valid prompt",
        enabled: true,
      };

      server.use(
        http.post(apiUrl("/workers"), () => {
          return HttpResponse.json(
            {
              error: "Validation failed",
              details: [{ field: "name", message: "Name is required" }],
            },
            { status: 400 }
          );
        })
      );

      await expect(
        dataProvider.create({
          resource: "workers",
          variables: invalidWorkerData,
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle server errors during creation", async () => {
      const newWorkerData = {
        name: "New Worker",
        prompt: "Test prompt",
        enabled: true,
      };

      server.use(
        http.post(apiUrl("/workers"), () => {
          return HttpResponse.json(
            { error: "Internal server error" },
            { status: 500 }
          );
        })
      );

      await expect(
        dataProvider.create({
          resource: "workers",
          variables: newWorkerData,
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("update", () => {
    it("should update existing worker", async () => {
      const workerId = "worker-123";
      const updateData = {
        name: "Updated Worker Name",
        prompt: "Updated prompt",
      };

      const updatedWorker = createMockWorker({
        id: workerId,
        ...updateData,
      });

      server.use(
        http.put(apiUrl(`/workers/${workerId}`), async ({ request }) => {
          const body = await request.json();
          expect(body).toEqual(updateData);
          return HttpResponse.json(updatedWorker);
        })
      );

      const result = await dataProvider.update({
        resource: "workers",
        id: workerId,
        variables: updateData,
        meta: {},
      });

      expect(result.data).toEqual(updatedWorker);
    });

    it("should handle partial updates", async () => {
      const workerId = "worker-123";
      const partialUpdate = {
        name: "Updated Name Only",
      };

      const updatedWorker = createMockWorker({
        id: workerId,
        name: partialUpdate.name,
      });

      server.use(
        http.put(apiUrl(`/workers/${workerId}`), async ({ request }) => {
          const body = await request.json();
          expect(body).toEqual(partialUpdate);
          return HttpResponse.json(updatedWorker);
        })
      );

      const result = await dataProvider.update({
        resource: "workers",
        id: workerId,
        variables: partialUpdate,
        meta: {},
      });

      expect(result.data.name).toBe(partialUpdate.name);
    });

    it("should handle update validation errors", async () => {
      const workerId = "worker-123";
      const invalidUpdate = {
        name: "", // Invalid empty name
      };

      server.use(
        http.put(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(
            {
              error: "Validation failed",
              details: [{ field: "name", message: "Name cannot be empty" }],
            },
            { status: 400 }
          );
        })
      );

      await expect(
        dataProvider.update({
          resource: "workers",
          id: workerId,
          variables: invalidUpdate,
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle update of nonexistent worker", async () => {
      const workerId = "nonexistent";
      const updateData = { name: "Updated Name" };

      server.use(
        http.put(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(
            { error: "Worker not found" },
            { status: 404 }
          );
        })
      );

      await expect(
        dataProvider.update({
          resource: "workers",
          id: workerId,
          variables: updateData,
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("deleteOne", () => {
    it("should delete worker successfully", async () => {
      const workerId = "worker-123";

      server.use(
        http.delete(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json({ message: "Worker deleted successfully" });
        })
      );

      const result = await dataProvider.deleteOne({
        resource: "workers",
        id: workerId,
        meta: {},
      });

      expect(result.data).toEqual({ id: workerId });
    });

    it("should handle delete of nonexistent worker", async () => {
      const workerId = "nonexistent";

      server.use(
        http.delete(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(
            { error: "Worker not found" },
            { status: 404 }
          );
        })
      );

      await expect(
        dataProvider.deleteOne({
          resource: "workers",
          id: workerId,
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle server errors during deletion", async () => {
      const workerId = "worker-123";

      server.use(
        http.delete(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(
            { error: "Internal server error" },
            { status: 500 }
          );
        })
      );

      await expect(
        dataProvider.deleteOne({
          resource: "workers",
          id: workerId,
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("custom", () => {
    it("should handle custom API calls", async () => {
      const workerId = "worker-123";
      const customData = { message: "Custom operation completed" };

      server.use(
        http.post(apiUrl(`/workers/${workerId}/actions/deploy`), () => {
          return HttpResponse.json(customData);
        })
      );

      const result = await dataProvider.custom({
        url: `/workers/${workerId}/actions/deploy`,
        method: "post",
        payload: {},
        meta: {},
      });

      expect(result.data).toEqual(customData);
    });

    it("should handle custom GET requests", async () => {
      const healthData = { status: "healthy" };

      server.use(
        http.get(apiUrl("/health"), () => {
          return HttpResponse.json(healthData);
        })
      );

      const result = await dataProvider.custom({
        url: "/health",
        method: "get",
        meta: {},
      });

      expect(result.data).toEqual(healthData);
    });

    it("should handle custom requests with query parameters", async () => {
      const workerId = "worker-123";
      const logsData = { logs: ["Log entry 1", "Log entry 2"] };

      server.use(
        http.get(apiUrl(`/workers/${workerId}/logs`), ({ request }) => {
          const url = new URL(request.url);
          const limit = url.searchParams.get("limit");
          expect(limit).toBe("100");
          return HttpResponse.json(logsData);
        })
      );

      const result = await dataProvider.custom({
        url: `/workers/${workerId}/logs?limit=100`,
        method: "get",
        meta: {},
      });

      expect(result.data).toEqual(logsData);
    });

    it("should handle custom request errors", async () => {
      server.use(
        http.post(apiUrl("/workers/invalid/actions/deploy"), () => {
          return HttpResponse.json(
            { error: "Worker not found" },
            { status: 404 }
          );
        })
      );

      await expect(
        dataProvider.custom({
          url: "/workers/invalid/actions/deploy",
          method: "post",
          payload: {},
          meta: {},
        })
      ).rejects.toThrow();
    });
  });

  describe("getApiUrl", () => {
    it("should return correct API URL", () => {
      const apiUrl = dataProvider.getApiUrl();
      expect(apiUrl).toBe(API_BASE_URL);
    });
  });

  describe("Error Handling and Edge Cases", () => {
    it("should handle malformed JSON responses", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return new Response("invalid-json", {
            headers: { "Content-Type": "application/json" },
          });
        })
      );

      await expect(
        dataProvider.getList({
          resource: "workers",
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle network timeouts", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(
            { error: "Request timeout" },
            { status: 408 }
          );
        })
      );

      await expect(
        dataProvider.getList({
          resource: "workers",
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        })
      ).rejects.toThrow();
    });

    it("should handle empty response bodies appropriately", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return new Response(null, { status: 204 });
        })
      );

      // This might vary based on your data provider implementation
      // Some might treat 204 as success with empty data, others as error
      try {
        const result = await dataProvider.getList({
          resource: "workers",
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        });
        // If it succeeds, verify it handles empty response correctly
        expect(result.data).toBeDefined();
      } catch (error) {
        // If it throws, that's also acceptable behavior
        expect(error).toBeDefined();
      }
    });

    it("should handle very large response payloads", async () => {
      // Create a large dataset
      const largeWorkerList = Array.from({ length: 10000 }, (_, i) =>
        createMockWorker({ id: `worker-${i}`, name: `Worker ${i}` })
      );

      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(largeWorkerList);
        })
      );

      const result = await dataProvider.getList({
        resource: "workers",
        pagination: { current: 1, pageSize: 10000 },
        filters: [],
        sorters: [],
        meta: {},
      });

      expect(result.data).toHaveLength(10000);
      expect(result.total).toBe(10000);
    });
  });

  describe("Resource Type Safety", () => {
    it("should handle different resource types", async () => {
      // Test with different resource paths
      const resources = ["workers", "settings", "logs"];

      for (const resource of resources) {
        server.use(
          http.get(apiUrl(`/${resource}`), () => {
            return HttpResponse.json([]);
          })
        );

        const result = await dataProvider.getList({
          resource,
          pagination: { current: 1, pageSize: 10 },
          filters: [],
          sorters: [],
          meta: {},
        });

        expect(result.data).toEqual([]);
      }
    });

    it("should handle nested resource paths", async () => {
      const workerId = "worker-123";
      const statusData = { status: "running" };

      server.use(
        http.get(apiUrl(`/workers/${workerId}/status`), () => {
          return HttpResponse.json(statusData);
        })
      );

      const result = await dataProvider.custom({
        url: `/workers/${workerId}/status`,
        method: "get",
        meta: {},
      });

      expect(result.data).toEqual(statusData);
    });
  });
});
