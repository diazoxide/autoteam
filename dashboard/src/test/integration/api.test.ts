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
import { createMockWorker, createMockRuntimeData } from "../utils/test-utils";

// Mock API server setup
const server = setupServer();

// API base URL
const API_BASE_URL = "http://localhost:9090";

// Helper to create API URL
const apiUrl = (path: string) => `${API_BASE_URL}${path}`;

describe("API Integration Tests", () => {
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

  describe("Workers API", () => {
    describe("GET /workers", () => {
      it("should fetch workers list successfully", async () => {
        const mockWorkers = [
          createMockWorker({ id: "worker-1", name: "Worker 1" }),
          createMockWorker({ id: "worker-2", name: "Worker 2" }),
        ];

        server.use(
          http.get(apiUrl("/workers"), () => {
            return HttpResponse.json(mockWorkers);
          })
        );

        const response = await fetch(apiUrl("/workers"));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockWorkers);
        expect(data).toHaveLength(2);
      });

      it("should handle empty workers list", async () => {
        server.use(
          http.get(apiUrl("/workers"), () => {
            return HttpResponse.json([]);
          })
        );

        const response = await fetch(apiUrl("/workers"));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual([]);
        expect(data).toHaveLength(0);
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

        const response = await fetch(apiUrl("/workers"));
        const data = await response.json();

        expect(response.status).toBe(500);
        expect(data.error).toBe("Internal server error");
      });

      it("should handle network errors", async () => {
        server.use(
          http.get(apiUrl("/workers"), () => {
            return HttpResponse.error();
          })
        );

        await expect(fetch(apiUrl("/workers"))).rejects.toThrow();
      });
    });

    describe("GET /workers/:id", () => {
      it("should fetch single worker successfully", async () => {
        const mockWorker = createMockWorker({
          id: "worker-123",
          name: "Test Worker",
        });

        server.use(
          http.get(apiUrl("/workers/worker-123"), () => {
            return HttpResponse.json(mockWorker);
          })
        );

        const response = await fetch(apiUrl("/workers/worker-123"));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockWorker);
        expect(data.id).toBe("worker-123");
      });

      it("should handle worker not found", async () => {
        server.use(
          http.get(apiUrl("/workers/nonexistent"), () => {
            return HttpResponse.json(
              { error: "Worker not found" },
              { status: 404 }
            );
          })
        );

        const response = await fetch(apiUrl("/workers/nonexistent"));
        const data = await response.json();

        expect(response.status).toBe(404);
        expect(data.error).toBe("Worker not found");
      });

      it("should handle invalid worker ID format", async () => {
        server.use(
          http.get(apiUrl("/workers/invalid-uuid"), () => {
            return HttpResponse.json(
              { error: "Invalid worker ID format" },
              { status: 400 }
            );
          })
        );

        const response = await fetch(apiUrl("/workers/invalid-uuid"));
        const data = await response.json();

        expect(response.status).toBe(400);
        expect(data.error).toBe("Invalid worker ID format");
      });
    });

    describe("POST /workers", () => {
      it("should create worker successfully", async () => {
        const newWorker = {
          name: "New Test Worker",
          prompt: "This is a test prompt for the new worker",
          enabled: true,
        };

        const createdWorker = createMockWorker({
          ...newWorker,
          id: "new-worker-123",
        });

        server.use(
          http.post(apiUrl("/workers"), async ({ request }) => {
            const body = await request.json();
            expect(body).toEqual(newWorker);
            return HttpResponse.json(createdWorker, { status: 201 });
          })
        );

        const response = await fetch(apiUrl("/workers"), {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(newWorker),
        });

        const data = await response.json();

        expect(response.status).toBe(201);
        expect(data).toEqual(createdWorker);
        expect(data.name).toBe(newWorker.name);
      });

      it("should validate required fields", async () => {
        const invalidWorker = {
          name: "", // Empty name
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

        const response = await fetch(apiUrl("/workers"), {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(invalidWorker),
        });

        const data = await response.json();

        expect(response.status).toBe(400);
        expect(data.error).toBe("Validation failed");
        expect(data.details).toContainEqual({
          field: "name",
          message: "Name is required",
        });
      });

      it("should handle malformed JSON", async () => {
        server.use(
          http.post(apiUrl("/workers"), () => {
            return HttpResponse.json(
              { error: "Invalid JSON format" },
              { status: 400 }
            );
          })
        );

        const response = await fetch(apiUrl("/workers"), {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: "invalid-json",
        });

        const data = await response.json();

        expect(response.status).toBe(400);
        expect(data.error).toBe("Invalid JSON format");
      });
    });

    describe("PUT /workers/:id", () => {
      it("should update worker successfully", async () => {
        const workerId = "worker-123";
        const updateData = {
          name: "Updated Worker Name",
          prompt: "Updated prompt content",
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

        const response = await fetch(apiUrl(`/workers/${workerId}`), {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(updateData),
        });

        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(updatedWorker);
        expect(data.name).toBe(updateData.name);
      });

      it("should handle partial updates", async () => {
        const workerId = "worker-123";
        const partialUpdate = {
          name: "New Name Only",
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

        const response = await fetch(apiUrl(`/workers/${workerId}`), {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(partialUpdate),
        });

        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data.name).toBe(partialUpdate.name);
      });

      it("should validate update data", async () => {
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

        const response = await fetch(apiUrl(`/workers/${workerId}`), {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(invalidUpdate),
        });

        const data = await response.json();

        expect(response.status).toBe(400);
        expect(data.error).toBe("Validation failed");
      });
    });

    describe("DELETE /workers/:id", () => {
      it("should delete worker successfully", async () => {
        const workerId = "worker-123";

        server.use(
          http.delete(apiUrl(`/workers/${workerId}`), () => {
            return HttpResponse.json({
              message: "Worker deleted successfully",
            });
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}`), {
          method: "DELETE",
        });

        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data.message).toBe("Worker deleted successfully");
      });

      it("should handle delete of nonexistent worker", async () => {
        const workerId = "nonexistent-worker";

        server.use(
          http.delete(apiUrl(`/workers/${workerId}`), () => {
            return HttpResponse.json(
              { error: "Worker not found" },
              { status: 404 }
            );
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}`), {
          method: "DELETE",
        });

        const data = await response.json();

        expect(response.status).toBe(404);
        expect(data.error).toBe("Worker not found");
      });
    });
  });

  describe("Worker Actions API", () => {
    describe("POST /workers/:id/actions/:action", () => {
      const workerId = "worker-123";

      const actionTests = [
        { action: "deploy", successMessage: "Worker deployment started" },
        { action: "stop", successMessage: "Worker stopped successfully" },
        { action: "restart", successMessage: "Worker restarted successfully" },
        { action: "pause", successMessage: "Worker paused successfully" },
        { action: "unpause", successMessage: "Worker unpaused successfully" },
      ];

      actionTests.forEach(({ action, successMessage }) => {
        it(`should handle ${action} action successfully`, async () => {
          server.use(
            http.post(apiUrl(`/workers/${workerId}/actions/${action}`), () => {
              return HttpResponse.json({ message: successMessage });
            })
          );

          const response = await fetch(
            apiUrl(`/workers/${workerId}/actions/${action}`),
            {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify({}),
            }
          );

          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.message).toBe(successMessage);
        });

        it(`should handle ${action} action failure`, async () => {
          server.use(
            http.post(apiUrl(`/workers/${workerId}/actions/${action}`), () => {
              return HttpResponse.json(
                { error: `Failed to ${action} worker` },
                { status: 500 }
              );
            })
          );

          const response = await fetch(
            apiUrl(`/workers/${workerId}/actions/${action}`),
            {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify({}),
            }
          );

          const data = await response.json();

          expect(response.status).toBe(500);
          expect(data.error).toBe(`Failed to ${action} worker`);
        });
      });

      it("should handle invalid actions", async () => {
        const invalidAction = "invalid-action";

        server.use(
          http.post(
            apiUrl(`/workers/${workerId}/actions/${invalidAction}`),
            () => {
              return HttpResponse.json(
                { error: "Invalid action" },
                { status: 400 }
              );
            }
          )
        );

        const response = await fetch(
          apiUrl(`/workers/${workerId}/actions/${invalidAction}`),
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({}),
          }
        );

        const data = await response.json();

        expect(response.status).toBe(400);
        expect(data.error).toBe("Invalid action");
      });
    });
  });

  describe("Worker Status API", () => {
    describe("GET /workers/:id/status", () => {
      it("should fetch worker status successfully", async () => {
        const workerId = "worker-123";
        const mockStatus = { status: "running" };

        server.use(
          http.get(apiUrl(`/workers/${workerId}/status`), () => {
            return HttpResponse.json(mockStatus);
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}/status`));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockStatus);
      });

      it("should handle different status values", async () => {
        const workerId = "worker-123";
        const statusValues = [
          "running",
          "stopped",
          "paused",
          "error",
          "unknown",
        ];

        for (const status of statusValues) {
          server.use(
            http.get(apiUrl(`/workers/${workerId}/status`), () => {
              return HttpResponse.json({ status });
            })
          );

          const response = await fetch(apiUrl(`/workers/${workerId}/status`));
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.status).toBe(status);
        }
      });
    });

    describe("GET /workers/:id/runtime", () => {
      it("should fetch worker runtime data successfully", async () => {
        const workerId = "worker-123";
        const mockRuntimeData = createMockRuntimeData({ status: "reachable" });

        server.use(
          http.get(apiUrl(`/workers/${workerId}/runtime`), () => {
            return HttpResponse.json(mockRuntimeData);
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}/runtime`));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockRuntimeData);
      });

      it("should handle runtime data with different statuses", async () => {
        const workerId = "worker-123";
        const runtimeStatuses = ["reachable", "unreachable", "unknown"];

        for (const status of runtimeStatuses) {
          const mockRuntimeData = createMockRuntimeData({ status });

          server.use(
            http.get(apiUrl(`/workers/${workerId}/runtime`), () => {
              return HttpResponse.json(mockRuntimeData);
            })
          );

          const response = await fetch(apiUrl(`/workers/${workerId}/runtime`));
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.worker.status).toBe(status);
        }
      });
    });
  });

  describe("Worker Logs API", () => {
    describe("GET /workers/:id/logs", () => {
      it("should fetch worker logs successfully", async () => {
        const workerId = "worker-123";
        const mockLogs = {
          logs: [
            "2024-01-01 10:00:00 - Worker started",
            "2024-01-01 10:01:00 - Processing task",
            "2024-01-01 10:02:00 - Task completed",
          ],
        };

        server.use(
          http.get(apiUrl(`/workers/${workerId}/logs`), () => {
            return HttpResponse.json(mockLogs);
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}/logs`));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockLogs);
        expect(data.logs).toHaveLength(3);
      });

      it("should handle empty logs", async () => {
        const workerId = "worker-123";
        const emptyLogs = { logs: [] };

        server.use(
          http.get(apiUrl(`/workers/${workerId}/logs`), () => {
            return HttpResponse.json(emptyLogs);
          })
        );

        const response = await fetch(apiUrl(`/workers/${workerId}/logs`));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data.logs).toHaveLength(0);
      });

      it("should handle logs with query parameters", async () => {
        const workerId = "worker-123";
        const mockLogs = { logs: ["Recent log entry"] };

        server.use(
          http.get(apiUrl(`/workers/${workerId}/logs`), ({ request }) => {
            const url = new URL(request.url);
            const limit = url.searchParams.get("limit");
            const since = url.searchParams.get("since");

            expect(limit).toBe("100");
            expect(since).toBe("2024-01-01T00:00:00Z");

            return HttpResponse.json(mockLogs);
          })
        );

        const queryParams = new URLSearchParams({
          limit: "100",
          since: "2024-01-01T00:00:00Z",
        });

        const response = await fetch(
          apiUrl(`/workers/${workerId}/logs?${queryParams}`)
        );
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data).toEqual(mockLogs);
      });
    });
  });

  describe("Health Check API", () => {
    describe("GET /health", () => {
      it("should return healthy status", async () => {
        const healthStatus = {
          status: "healthy",
          timestamp: "2024-01-01T12:00:00Z",
          services: {
            database: "healthy",
            runtime: "healthy",
          },
        };

        server.use(
          http.get(apiUrl("/health"), () => {
            return HttpResponse.json(healthStatus);
          })
        );

        const response = await fetch(apiUrl("/health"));
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data.status).toBe("healthy");
        expect(data.services).toBeDefined();
      });

      it("should handle unhealthy status", async () => {
        const unhealthyStatus = {
          status: "unhealthy",
          timestamp: "2024-01-01T12:00:00Z",
          services: {
            database: "unhealthy",
            runtime: "healthy",
          },
          errors: ["Database connection failed"],
        };

        server.use(
          http.get(apiUrl("/health"), () => {
            return HttpResponse.json(unhealthyStatus, { status: 503 });
          })
        );

        const response = await fetch(apiUrl("/health"));
        const data = await response.json();

        expect(response.status).toBe(503);
        expect(data.status).toBe("unhealthy");
        expect(data.errors).toContain("Database connection failed");
      });
    });
  });

  describe("Error Handling", () => {
    it("should handle CORS errors", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(
            { error: "CORS error" },
            {
              status: 403,
              headers: {
                "Access-Control-Allow-Origin": "http://other-domain.com",
              },
            }
          );
        })
      );

      const response = await fetch(apiUrl("/workers"));
      expect(response.status).toBe(403);
    });

    it("should handle timeout errors", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(
            { error: "Request timeout" },
            { status: 408 }
          );
        })
      );

      const response = await fetch(apiUrl("/workers"));
      const data = await response.json();

      expect(response.status).toBe(408);
      expect(data.error).toBe("Request timeout");
    });

    it("should handle rate limiting", async () => {
      server.use(
        http.get(apiUrl("/workers"), () => {
          return HttpResponse.json(
            {
              error: "Rate limit exceeded",
              retryAfter: 60,
            },
            {
              status: 429,
              headers: {
                "Retry-After": "60",
              },
            }
          );
        })
      );

      const response = await fetch(apiUrl("/workers"));
      const data = await response.json();

      expect(response.status).toBe(429);
      expect(data.error).toBe("Rate limit exceeded");
      expect(response.headers.get("Retry-After")).toBe("60");
    });
  });

  describe("Data Consistency", () => {
    it("should maintain data consistency across operations", async () => {
      const workerId = "worker-123";
      const initialWorker = createMockWorker({
        id: workerId,
        name: "Initial Name",
      });
      const updatedWorker = { ...initialWorker, name: "Updated Name" };

      // Setup handlers for the sequence
      server.use(
        // Initial fetch
        http.get(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(initialWorker);
        }),
        // Update
        http.put(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(updatedWorker);
        }),
        // Fetch after update
        http.get(apiUrl(`/workers/${workerId}`), () => {
          return HttpResponse.json(updatedWorker);
        })
      );

      // Initial fetch
      let response = await fetch(apiUrl(`/workers/${workerId}`));
      let data = await response.json();
      expect(data.name).toBe("Initial Name");

      // Update
      response = await fetch(apiUrl(`/workers/${workerId}`), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "Updated Name" }),
      });
      data = await response.json();
      expect(data.name).toBe("Updated Name");

      // Verify update persisted
      response = await fetch(apiUrl(`/workers/${workerId}`));
      data = await response.json();
      expect(data.name).toBe("Updated Name");
    });
  });
});
