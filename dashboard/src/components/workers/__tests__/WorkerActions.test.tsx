import React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { fireEvent, waitFor, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WorkerActions } from "../WorkerActions";
import {
  render,
  mockUseCustomMutation,
  mockUseWorkerRuntime,
} from "../../../test/utils/test-utils";

// Mock the hooks
vi.mock("@refinedev/core", () => ({
  useCustomMutation: () => mockUseCustomMutation(),
}));

vi.mock("../../../hooks/api/useWorkerApi", () => ({
  useWorkerRuntime: (workerId: string, options: any) =>
    mockUseWorkerRuntime(workerId, options),
}));

describe("WorkerActions", () => {
  const defaultProps = {
    workerId: "test-worker-id",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("Component Rendering", () => {
    it("should render compact version with deploy, stop, and restart buttons", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} compact={true} />);

      // Should show 3 buttons in compact mode (deploy, stop, restart)
      const buttons = screen.getAllByRole("button");
      expect(buttons).toHaveLength(3);

      // Check tooltips are present
      expect(screen.getByLabelText("Deploy")).toBeInTheDocument();
      expect(screen.getByLabelText("Stop")).toBeInTheDocument();
      expect(screen.getByLabelText("Restart")).toBeInTheDocument();
    });

    it("should render full version with all action buttons", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} compact={false} />);

      // Should show all 5 action buttons in full mode
      expect(screen.getByText("Deploy")).toBeInTheDocument();
      expect(screen.getByText("Stop")).toBeInTheDocument();
      expect(screen.getByText("Restart")).toBeInTheDocument();
      expect(screen.getByText("Pause")).toBeInTheDocument();
      expect(screen.getByText("Unpause")).toBeInTheDocument();
    });

    it("should use provided status when available", () => {
      const { rerender } = render(
        <WorkerActions {...defaultProps} status="running" compact={true} />
      );

      // Should not call useWorkerRuntime when status is provided
      expect(mockUseWorkerRuntime).toHaveBeenCalledWith(defaultProps.workerId, {
        enabled: false,
      });

      // Test different status values
      rerender(
        <WorkerActions {...defaultProps} status="not_deployed" compact={true} />
      );
      rerender(
        <WorkerActions {...defaultProps} status="paused" compact={true} />
      );
    });
  });

  describe("Button States Based on Worker Status", () => {
    it("should disable deploy button when worker is running", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "running" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      expect(deployButton).toBeDisabled();
    });

    it("should disable stop and restart buttons when worker is not deployed", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "not_deployed" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const stopButton = screen.getByText("Stop");
      const restartButton = screen.getByText("Restart");

      expect(stopButton).toBeDisabled();
      expect(restartButton).toBeDisabled();
    });

    it("should disable pause button when worker is not running", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "not_deployed" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const pauseButton = screen.getByText("Pause");
      expect(pauseButton).toBeDisabled();
    });

    it("should disable unpause button when worker is not paused", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "running" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const unpauseButton = screen.getByText("Unpause");
      expect(unpauseButton).toBeDisabled();
    });

    it("should enable appropriate buttons based on worker status", () => {
      // Test running status
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "running" } },
        isLoading: false,
        error: null,
      });

      const { rerender } = render(<WorkerActions {...defaultProps} />);

      expect(screen.getByText("Stop")).not.toBeDisabled();
      expect(screen.getByText("Restart")).not.toBeDisabled();
      expect(screen.getByText("Pause")).not.toBeDisabled();

      // Test paused status
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "paused" } },
        isLoading: false,
        error: null,
      });

      rerender(<WorkerActions {...defaultProps} />);

      expect(screen.getByText("Unpause")).not.toBeDisabled();
    });
  });

  describe("Action Handling", () => {
    const mockMutate = vi.fn();

    beforeEach(() => {
      mockUseCustomMutation.mockReturnValue({
        mutate: mockMutate,
        isLoading: false,
        error: null,
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });
    });

    it("should call deploy action when deploy button is clicked", async () => {
      const user = userEvent.setup();
      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      expect(mockMutate).toHaveBeenCalledWith(
        {
          url: `/workers/${defaultProps.workerId}/actions/deploy`,
          method: "post",
          values: {},
          successNotification: false,
          errorNotification: false,
        },
        expect.objectContaining({
          onSuccess: expect.any(Function),
          onError: expect.any(Function),
        })
      );
    });

    it("should call stop action when stop button is clicked", async () => {
      const user = userEvent.setup();
      render(<WorkerActions {...defaultProps} />);

      const stopButton = screen.getByText("Stop");
      await user.click(stopButton);

      expect(mockMutate).toHaveBeenCalledWith(
        {
          url: `/workers/${defaultProps.workerId}/actions/stop`,
          method: "post",
          values: {},
          successNotification: false,
          errorNotification: false,
        },
        expect.objectContaining({
          onSuccess: expect.any(Function),
          onError: expect.any(Function),
        })
      );
    });

    it("should show success notification on successful action", async () => {
      const user = userEvent.setup();
      mockMutate.mockImplementation((_, callbacks) => {
        callbacks.onSuccess();
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      await waitFor(() => {
        expect(
          screen.getByText("Worker deployment started")
        ).toBeInTheDocument();
      });
    });

    it("should show error notification on failed action", async () => {
      const user = userEvent.setup();
      const errorMessage = "Deployment failed";
      mockMutate.mockImplementation((_, callbacks) => {
        callbacks.onError(new Error(errorMessage));
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      await waitFor(() => {
        expect(
          screen.getByText(`Failed to deploy worker: ${errorMessage}`)
        ).toBeInTheDocument();
      });
    });

    it("should handle unknown error types gracefully", async () => {
      const user = userEvent.setup();
      mockMutate.mockImplementation((_, callbacks) => {
        callbacks.onError("string error");
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      await waitFor(() => {
        expect(
          screen.getByText("Failed to deploy worker: Unknown error")
        ).toBeInTheDocument();
      });
    });
  });

  describe("Loading States", () => {
    it("should show loading indicator when action is in progress", () => {
      mockUseCustomMutation.mockReturnValue({
        mutate: vi.fn(),
        isLoading: true,
        error: null,
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      // Should have CircularProgress components
      const progressElements = screen.getAllByRole("progressbar");
      expect(progressElements.length).toBeGreaterThan(0);
    });

    it("should disable all buttons when any action is loading", () => {
      mockUseCustomMutation.mockReturnValue({
        mutate: vi.fn(),
        isLoading: true,
        error: null,
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const buttons = screen.getAllByRole("button");
      buttons.forEach((button) => {
        expect(button).toBeDisabled();
      });
    });
  });

  describe("Notification Handling", () => {
    it("should close notification when close button is clicked", async () => {
      const user = userEvent.setup();
      const mockMutate = vi.fn((_, callbacks) => {
        callbacks.onSuccess();
      });

      mockUseCustomMutation.mockReturnValue({
        mutate: mockMutate,
        isLoading: false,
        error: null,
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      await waitFor(() => {
        expect(
          screen.getByText("Worker deployment started")
        ).toBeInTheDocument();
      });

      // Find and click the close button in the alert
      const closeButton = screen.getByLabelText("Close");
      await user.click(closeButton);

      await waitFor(() => {
        expect(
          screen.queryByText("Worker deployment started")
        ).not.toBeInTheDocument();
      });
    });

    it("should auto-hide notification after 4 seconds", async () => {
      vi.useFakeTimers();

      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
      const mockMutate = vi.fn((_, callbacks) => {
        callbacks.onSuccess();
      });

      mockUseCustomMutation.mockReturnValue({
        mutate: mockMutate,
        isLoading: false,
        error: null,
      });

      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      const deployButton = screen.getByText("Deploy");
      await user.click(deployButton);

      await waitFor(() => {
        expect(
          screen.getByText("Worker deployment started")
        ).toBeInTheDocument();
      });

      // Fast-forward time by 4 seconds
      vi.advanceTimersByTime(4000);

      await waitFor(() => {
        expect(
          screen.queryByText("Worker deployment started")
        ).not.toBeInTheDocument();
      });

      vi.useRealTimers();
    });
  });

  describe("Edge Cases", () => {
    it("should handle missing runtime data gracefully", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: undefined,
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      // Should render without crashing
      expect(screen.getByText("Deploy")).toBeInTheDocument();
    });

    it("should handle malformed runtime data", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: {} }, // Missing status
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} />);

      // Should render without crashing
      expect(screen.getByText("Deploy")).toBeInTheDocument();
    });

    it("should handle empty worker ID", () => {
      render(<WorkerActions workerId="" />);

      // Should render without crashing
      expect(screen.getByText("Deploy")).toBeInTheDocument();
    });

    it("should handle null/undefined status gracefully", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: null } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} status={undefined} />);

      // Should render without crashing
      expect(screen.getByText("Deploy")).toBeInTheDocument();
    });
  });

  describe("Accessibility", () => {
    it("should have proper ARIA labels and roles", () => {
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} compact={true} />);

      // Check for proper button roles
      const buttons = screen.getAllByRole("button");
      expect(buttons.length).toBeGreaterThan(0);

      // Check for tooltips
      expect(screen.getByLabelText("Deploy")).toBeInTheDocument();
      expect(screen.getByLabelText("Stop")).toBeInTheDocument();
      expect(screen.getByLabelText("Restart")).toBeInTheDocument();
    });

    it("should support keyboard navigation", async () => {
      const user = userEvent.setup();
      mockUseWorkerRuntime.mockReturnValue({
        data: { worker: { status: "unknown" } },
        isLoading: false,
        error: null,
      });

      render(<WorkerActions {...defaultProps} compact={true} />);

      // Tab through buttons
      await user.tab();
      const firstButton = screen.getAllByRole("button")[0];
      expect(firstButton).toHaveFocus();

      await user.tab();
      const secondButton = screen.getAllByRole("button")[1];
      expect(secondButton).toHaveFocus();
    });
  });
});
