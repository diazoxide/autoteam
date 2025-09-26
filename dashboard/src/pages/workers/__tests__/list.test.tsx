import React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WorkersList } from "../list";
import {
  render,
  mockUseDataGrid,
  mockUseCustom,
  mockUseWorkerRuntime,
  createMockWorker,
  createMockRuntimeData,
} from "../../../test/utils/test-utils";

// Mock the hooks
vi.mock("@refinedev/mui", () => ({
  useDataGrid: () => mockUseDataGrid(),
  List: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="list-container">{children}</div>
  ),
  ShowButton: ({ recordItemId }: { recordItemId: string }) => (
    <button data-testid={`show-${recordItemId}`}>Show</button>
  ),
  EditButton: ({ recordItemId }: { recordItemId: string }) => (
    <button data-testid={`edit-${recordItemId}`}>Edit</button>
  ),
  DeleteButton: ({ recordItemId }: { recordItemId: string }) => (
    <button data-testid={`delete-${recordItemId}`}>Delete</button>
  ),
}));

vi.mock("@refinedev/core", () => ({
  useCustom: () => mockUseCustom(),
}));

vi.mock("../../../hooks/api/useWorkerApi", () => ({
  useWorkerRuntime: (workerId: string, options: any) =>
    mockUseWorkerRuntime(workerId, options),
}));

vi.mock("../../../components/workers/WorkerActions", () => ({
  WorkerActions: ({
    workerId,
    compact,
  }: {
    workerId: string;
    compact: boolean;
  }) => (
    <div data-testid={`worker-actions-${workerId}`} data-compact={compact}>
      Worker Actions
    </div>
  ),
}));

// Mock DataGrid component
vi.mock("@mui/x-data-grid", () => ({
  DataGrid: ({ columns, rows, ...props }: any) => (
    <div
      data-testid="data-grid"
      data-columns={columns?.length || 0}
      data-rows={rows?.length || 0}
    >
      {rows?.map((row: any, index: number) => (
        <div key={row.id || index} data-testid={`row-${row.id || index}`}>
          {columns?.map((col: any) => (
            <div
              key={col.field}
              data-testid={`cell-${col.field}-${row.id || index}`}
            >
              {col.renderCell
                ? col.renderCell({ row, value: row[col.field] })
                : row[col.field] || ""}
            </div>
          ))}
        </div>
      ))}
    </div>
  ),
}));

describe("WorkersList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("Component Rendering", () => {
    it("should render the list container and data grid", () => {
      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 0,
        },
      });

      mockUseCustom.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByTestId("list-container")).toBeInTheDocument();
      expect(screen.getByTestId("data-grid")).toBeInTheDocument();
    });

    it("should render column headers correctly", () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseCustom.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      const dataGrid = screen.getByTestId("data-grid");
      expect(dataGrid).toHaveAttribute("data-columns", "7"); // 7 columns defined
    });

    it("should display worker data correctly", () => {
      const mockWorker = createMockWorker({
        name: "Test Worker 1",
        enabled: true,
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseCustom.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByTestId(`row-${mockWorker.id}`)).toBeInTheDocument();
    });
  });

  describe("WorkerStatusChip Component", () => {
    it("should show loading state while fetching status", () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByText("Checking...")).toBeInTheDocument();
    });

    it('should show "Deployed" status for reachable workers', () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: createMockRuntimeData({ status: "reachable" }),
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByText("Deployed")).toBeInTheDocument();
    });

    it('should show "Unreachable" status for unreachable workers', () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: createMockRuntimeData({ status: "unreachable" }),
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByText("Unreachable")).toBeInTheDocument();
    });

    it('should show "Not Deployed" status for unknown workers', () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: createMockRuntimeData({ status: "unknown" }),
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByText("Not Deployed")).toBeInTheDocument();
    });

    it("should handle missing runtime data gracefully", () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(screen.getByText("Not Deployed")).toBeInTheDocument();
    });
  });

  describe("Worker Enabled Status", () => {
    it('should show "Yes" chip for enabled workers', () => {
      const mockWorker = createMockWorker({ enabled: true });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("Yes")).toBeInTheDocument();
    });

    it('should show "No" chip for disabled workers', () => {
      const mockWorker = createMockWorker({ enabled: false });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("No")).toBeInTheDocument();
    });
  });

  describe("Flow Steps Display", () => {
    it("should display flow steps count when settings exist", () => {
      const mockWorker = createMockWorker({
        settings: {
          flow: [
            { name: "step1", type: "agent" },
            { name: "step2", type: "agent" },
            { name: "step3", type: "agent" },
          ],
        },
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("3 steps")).toBeInTheDocument();
    });

    it('should show "No flow" when settings exist but no flow', () => {
      const mockWorker = createMockWorker({
        settings: {}, // No flow property
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("No flow")).toBeInTheDocument();
    });

    it('should show "No settings" when settings are missing', () => {
      const mockWorker = createMockWorker({
        settings: undefined,
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("No settings")).toBeInTheDocument();
    });

    it("should handle empty flow array", () => {
      const mockWorker = createMockWorker({
        settings: {
          flow: [],
        },
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      expect(screen.getByText("0 steps")).toBeInTheDocument();
    });
  });

  describe("Date Formatting", () => {
    it("should format created_at date correctly", () => {
      const testDate = "2024-01-15T10:30:00Z";
      const mockWorker = createMockWorker({
        created_at: testDate,
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Should contain formatted date (exact format depends on locale)
      const dateElement = screen.getByTestId(
        `cell-created_at-${mockWorker.id}`
      );
      expect(dateElement).toBeInTheDocument();
      expect(dateElement.textContent).not.toBe("Unknown");
    });

    it("should format updated_at date correctly", () => {
      const testDate = "2024-01-15T15:45:00Z";
      const mockWorker = createMockWorker({
        updated_at: testDate,
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      const dateElement = screen.getByTestId(
        `cell-updated_at-${mockWorker.id}`
      );
      expect(dateElement).toBeInTheDocument();
      expect(dateElement.textContent).not.toBe("Unknown");
    });

    it('should show "Unknown" for missing dates', () => {
      const mockWorker = createMockWorker({
        created_at: undefined,
        updated_at: null,
      });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      const createdCell = screen.getByTestId(
        `cell-created_at-${mockWorker.id}`
      );
      const updatedCell = screen.getByTestId(
        `cell-updated_at-${mockWorker.id}`
      );

      expect(createdCell.textContent).toBe("Unknown");
      expect(updatedCell.textContent).toBe("Unknown");
    });
  });

  describe("Action Buttons", () => {
    it("should render all action buttons for each worker", () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Check for WorkerActions component
      expect(
        screen.getByTestId(`worker-actions-${mockWorker.id}`)
      ).toBeInTheDocument();
      expect(
        screen.getByTestId(`worker-actions-${mockWorker.id}`)
      ).toHaveAttribute("data-compact", "true");

      // Check for CRUD buttons
      expect(screen.getByTestId(`show-${mockWorker.id}`)).toBeInTheDocument();
      expect(screen.getByTestId(`edit-${mockWorker.id}`)).toBeInTheDocument();
      expect(screen.getByTestId(`delete-${mockWorker.id}`)).toBeInTheDocument();
    });

    it("should handle clicks on CRUD buttons", async () => {
      const user = userEvent.setup();
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      const showButton = screen.getByTestId(`show-${mockWorker.id}`);
      const editButton = screen.getByTestId(`edit-${mockWorker.id}`);
      const deleteButton = screen.getByTestId(`delete-${mockWorker.id}`);

      // These should be clickable without throwing errors
      await user.click(showButton);
      await user.click(editButton);
      await user.click(deleteButton);
    });
  });

  describe("Multiple Workers", () => {
    it("should render multiple workers correctly", () => {
      const mockWorkers = [
        createMockWorker({ id: "worker-1", name: "Worker 1" }),
        createMockWorker({ id: "worker-2", name: "Worker 2" }),
        createMockWorker({ id: "worker-3", name: "Worker 3" }),
      ];

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: mockWorkers,
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 3,
        },
      });

      render(<WorkersList />);

      mockWorkers.forEach((worker) => {
        expect(screen.getByTestId(`row-${worker.id}`)).toBeInTheDocument();
      });
    });

    it("should handle mixed worker states", () => {
      const mockWorkers = [
        createMockWorker({
          id: "worker-1",
          name: "Enabled Worker",
          enabled: true,
          settings: { flow: [{ name: "step1", type: "agent" }] },
        }),
        createMockWorker({
          id: "worker-2",
          name: "Disabled Worker",
          enabled: false,
          settings: undefined,
        }),
        createMockWorker({
          id: "worker-3",
          name: "Worker with No Flow",
          enabled: true,
          settings: {},
        }),
      ];

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: mockWorkers,
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 3,
        },
      });

      render(<WorkersList />);

      // Should render all workers without errors
      expect(screen.getByTestId("row-worker-1")).toBeInTheDocument();
      expect(screen.getByTestId("row-worker-2")).toBeInTheDocument();
      expect(screen.getByTestId("row-worker-3")).toBeInTheDocument();
    });
  });

  describe("Health Monitoring", () => {
    it("should set up health monitoring with correct interval", () => {
      mockUseCustom.mockReturnValue({
        data: { status: "healthy" },
        isLoading: false,
        error: null,
      });

      render(<WorkersList />);

      expect(mockUseCustom).toHaveBeenCalledWith({
        url: "/health",
        method: "get",
        queryOptions: {
          refetchInterval: 5000,
        },
      });
    });

    it("should handle health monitoring errors gracefully", () => {
      mockUseCustom.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: new Error("Health check failed"),
      });

      render(<WorkersList />);

      // Should render without crashing despite health check error
      expect(screen.getByTestId("list-container")).toBeInTheDocument();
    });
  });

  describe("Edge Cases and Error Handling", () => {
    it("should handle empty worker list", () => {
      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 0,
        },
      });

      render(<WorkersList />);

      const dataGrid = screen.getByTestId("data-grid");
      expect(dataGrid).toHaveAttribute("data-rows", "0");
    });

    it("should handle loading state", () => {
      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [],
          loading: true,
          pageSize: 10,
          page: 0,
          rowCount: 0,
        },
      });

      render(<WorkersList />);

      // Should render data grid even in loading state
      expect(screen.getByTestId("data-grid")).toBeInTheDocument();
    });

    it("should handle malformed worker data", () => {
      const malformedWorker = {
        id: "malformed-worker",
        // Missing required fields
        name: null,
        enabled: undefined,
        settings: "invalid-settings",
      };

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [malformedWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Should render without crashing
      expect(
        screen.getByTestId(`row-${malformedWorker.id}`)
      ).toBeInTheDocument();
    });

    it("should handle null or undefined worker IDs", () => {
      const workerWithoutId = createMockWorker({ id: undefined });

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [workerWithoutId],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Should render with index fallback
      expect(screen.getByTestId("row-0")).toBeInTheDocument();
    });
  });

  describe("Accessibility", () => {
    it("should have proper ARIA labels and roles", () => {
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Check for button roles
      const buttons = screen.getAllByRole("button");
      expect(buttons.length).toBeGreaterThan(0);
    });

    it("should support keyboard navigation", async () => {
      const user = userEvent.setup();
      const mockWorker = createMockWorker();

      mockUseDataGrid.mockReturnValue({
        dataGridProps: {
          rows: [mockWorker],
          loading: false,
          pageSize: 10,
          page: 0,
          rowCount: 1,
        },
      });

      render(<WorkersList />);

      // Tab through interactive elements
      await user.tab();
      const firstButton = screen.getAllByRole("button")[0];
      expect(firstButton).toHaveFocus();
    });
  });
});
