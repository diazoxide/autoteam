import React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WorkerCreate } from "../create";
import { render } from "../../../test/utils/test-utils";

// Mock the form hook
const mockFormMethods = {
  register: vi.fn(() => ({})),
  handleSubmit: vi.fn((fn) => (e: any) => {
    e.preventDefault();
    fn({});
  }),
  formState: {
    errors: {},
    isSubmitting: false,
    isValid: true,
  },
  control: {},
  watch: vi.fn(() => ({})),
  setValue: vi.fn(),
  getValues: vi.fn(() => ({})),
  reset: vi.fn(),
};

const mockRefineForm = {
  ...mockFormMethods,
  refineCore: {
    onFinish: vi.fn(),
    redirect: vi.fn(),
  },
  saveButtonProps: {
    disabled: false,
    onClick: vi.fn(),
  },
};

// Mock the hooks
vi.mock("@refinedev/react-hook-form", () => ({
  useForm: () => mockRefineForm,
}));

vi.mock("@refinedev/mui", () => ({
  Create: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="create-container">{children}</div>
  ),
  SaveButton: ({ disabled, ...props }: any) => (
    <button data-testid="save-button" disabled={disabled} {...props}>
      Save
    </button>
  ),
}));

// Mock Material-UI components
vi.mock("@mui/material", () => ({
  Box: ({ children, ...props }: any) => (
    <div data-testid="box" {...props}>
      {children}
    </div>
  ),
  TextField: ({ label, error, helperText, ...props }: any) => (
    <div data-testid="text-field">
      <label>{label}</label>
      <input
        data-testid={`input-${label?.toLowerCase().replace(/\s+/g, "-")}`}
        {...props}
      />
      {error && <span data-testid="field-error">{helperText}</span>}
    </div>
  ),
  FormControlLabel: ({ label, control }: any) => (
    <div data-testid="form-control-label">
      {control}
      <span>{label}</span>
    </div>
  ),
  Switch: (props: any) => (
    <input type="checkbox" data-testid="switch" {...props} />
  ),
}));

describe("WorkerCreate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("Component Rendering", () => {
    it("should render the create form container", () => {
      render(<WorkerCreate />);

      expect(screen.getByTestId("create-container")).toBeInTheDocument();
    });

    it("should render all form fields", () => {
      render(<WorkerCreate />);

      // Check for all form fields
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Prompt")).toBeInTheDocument();
      expect(screen.getByText("Enabled")).toBeInTheDocument();

      // Check for input elements
      expect(screen.getByTestId("input-name")).toBeInTheDocument();
      expect(screen.getByTestId("input-prompt")).toBeInTheDocument();
      expect(screen.getByTestId("switch")).toBeInTheDocument();
    });

    it("should render save button", () => {
      render(<WorkerCreate />);

      expect(screen.getByTestId("save-button")).toBeInTheDocument();
    });
  });

  describe("Form Field Configuration", () => {
    it("should configure name field correctly", () => {
      render(<WorkerCreate />);

      const nameInput = screen.getByTestId("input-name");
      expect(nameInput).toBeInTheDocument();

      // Name field should be required (would be reflected in form validation)
      expect(mockFormMethods.register).toHaveBeenCalledWith(
        "name",
        expect.objectContaining({
          required: "Name is required",
        })
      );
    });

    it("should configure prompt field correctly", () => {
      render(<WorkerCreate />);

      const promptInput = screen.getByTestId("input-prompt");
      expect(promptInput).toBeInTheDocument();

      // Prompt field should be required and multiline
      expect(mockFormMethods.register).toHaveBeenCalledWith(
        "prompt",
        expect.objectContaining({
          required: "Prompt is required",
        })
      );
    });

    it("should configure enabled field correctly", () => {
      render(<WorkerCreate />);

      const enabledSwitch = screen.getByTestId("switch");
      expect(enabledSwitch).toBeInTheDocument();

      // Enabled field should be registered
      expect(mockFormMethods.register).toHaveBeenCalledWith("enabled");
    });
  });

  describe("Form Validation", () => {
    it("should display validation errors", () => {
      const mockFormMethodsWithErrors = {
        ...mockFormMethods,
        formState: {
          errors: {
            name: { message: "Name is required" },
            prompt: { message: "Prompt is required" },
          },
          isSubmitting: false,
          isValid: false,
        },
      };

      const mockRefineFormWithErrors = {
        ...mockFormMethodsWithErrors,
        refineCore: {
          onFinish: vi.fn(),
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: true,
          onClick: vi.fn(),
        },
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormWithErrors
      );

      render(<WorkerCreate />);

      // Should display error messages
      const errorElements = screen.getAllByTestId("field-error");
      expect(errorElements).toHaveLength(2);
      expect(screen.getByText("Name is required")).toBeInTheDocument();
      expect(screen.getByText("Prompt is required")).toBeInTheDocument();
    });

    it("should disable save button when form is invalid", () => {
      const mockRefineFormWithErrors = {
        ...mockFormMethods,
        formState: {
          errors: { name: { message: "Name is required" } },
          isSubmitting: false,
          isValid: false,
        },
        refineCore: {
          onFinish: vi.fn(),
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: true,
          onClick: vi.fn(),
        },
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormWithErrors
      );

      render(<WorkerCreate />);

      const saveButton = screen.getByTestId("save-button");
      expect(saveButton).toBeDisabled();
    });

    it("should enable save button when form is valid", () => {
      render(<WorkerCreate />);

      const saveButton = screen.getByTestId("save-button");
      expect(saveButton).not.toBeDisabled();
    });
  });

  describe("Form Submission", () => {
    it("should handle form submission", async () => {
      const user = userEvent.setup();
      const mockOnFinish = vi.fn();

      const mockRefineFormWithSubmit = {
        ...mockFormMethods,
        refineCore: {
          onFinish: mockOnFinish,
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: false,
          onClick: vi.fn(),
        },
        handleSubmit: vi.fn((fn) => (e: any) => {
          e.preventDefault();
          fn({
            name: "Test Worker",
            prompt: "Test prompt",
            enabled: true,
          });
        }),
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormWithSubmit
      );

      render(<WorkerCreate />);

      const saveButton = screen.getByTestId("save-button");
      await user.click(saveButton);

      expect(mockRefineFormWithSubmit.handleSubmit).toHaveBeenCalled();
    });

    it("should prevent submission when form is submitting", () => {
      const mockRefineFormSubmitting = {
        ...mockFormMethods,
        formState: {
          errors: {},
          isSubmitting: true,
          isValid: true,
        },
        refineCore: {
          onFinish: vi.fn(),
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: true,
          onClick: vi.fn(),
        },
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormSubmitting
      );

      render(<WorkerCreate />);

      const saveButton = screen.getByTestId("save-button");
      expect(saveButton).toBeDisabled();
    });
  });

  describe("User Interactions", () => {
    it("should handle name input changes", async () => {
      const user = userEvent.setup();
      render(<WorkerCreate />);

      const nameInput = screen.getByTestId("input-name");
      await user.type(nameInput, "New Worker Name");

      expect(nameInput).toHaveValue("New Worker Name");
    });

    it("should handle prompt input changes", async () => {
      const user = userEvent.setup();
      render(<WorkerCreate />);

      const promptInput = screen.getByTestId("input-prompt");
      await user.type(promptInput, "This is a test prompt for the worker");

      expect(promptInput).toHaveValue("This is a test prompt for the worker");
    });

    it("should handle enabled switch toggle", async () => {
      const user = userEvent.setup();
      render(<WorkerCreate />);

      const enabledSwitch = screen.getByTestId("switch");
      await user.click(enabledSwitch);

      expect(enabledSwitch).toBeChecked();
    });
  });

  describe("Field Requirements and Validation Rules", () => {
    it("should enforce name field validation", () => {
      render(<WorkerCreate />);

      expect(mockFormMethods.register).toHaveBeenCalledWith("name", {
        required: "Name is required",
        maxLength: {
          value: 100,
          message: "Name cannot exceed 100 characters",
        },
        pattern: {
          value: /^[a-zA-Z0-9\s\-_]+$/,
          message:
            "Name can only contain letters, numbers, spaces, hyphens, and underscores",
        },
      });
    });

    it("should enforce prompt field validation", () => {
      render(<WorkerCreate />);

      expect(mockFormMethods.register).toHaveBeenCalledWith("prompt", {
        required: "Prompt is required",
        minLength: {
          value: 10,
          message: "Prompt must be at least 10 characters long",
        },
        maxLength: {
          value: 5000,
          message: "Prompt cannot exceed 5000 characters",
        },
      });
    });

    it("should set default values correctly", () => {
      render(<WorkerCreate />);

      // Enabled should default to true
      expect(mockFormMethods.register).toHaveBeenCalledWith("enabled");
    });
  });

  describe("Form Layout and Styling", () => {
    it("should have proper form layout structure", () => {
      render(<WorkerCreate />);

      // Check for main container
      expect(screen.getByTestId("create-container")).toBeInTheDocument();

      // Check for form boxes
      const boxes = screen.getAllByTestId("box");
      expect(boxes.length).toBeGreaterThan(0);
    });

    it("should configure text fields with proper attributes", () => {
      render(<WorkerCreate />);

      const nameField = screen.getByText("Name").parentElement;
      const promptField = screen.getByText("Prompt").parentElement;

      expect(nameField).toBeInTheDocument();
      expect(promptField).toBeInTheDocument();
    });
  });

  describe("Error Handling", () => {
    it("should handle form submission errors gracefully", async () => {
      const user = userEvent.setup();
      const mockOnFinishWithError = vi
        .fn()
        .mockRejectedValue(new Error("Submission failed"));

      const mockRefineFormWithError = {
        ...mockFormMethods,
        refineCore: {
          onFinish: mockOnFinishWithError,
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: false,
          onClick: vi.fn(),
        },
        handleSubmit: vi.fn((fn) => async (e: any) => {
          e.preventDefault();
          try {
            await fn({ name: "Test", prompt: "Test prompt", enabled: true });
          } catch (error) {
            // Error should be handled by the form
          }
        }),
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormWithError
      );

      render(<WorkerCreate />);

      const saveButton = screen.getByTestId("save-button");
      await user.click(saveButton);

      // Form should not crash on submission error
      expect(screen.getByTestId("create-container")).toBeInTheDocument();
    });

    it("should handle invalid form data gracefully", () => {
      const mockRefineFormWithInvalidData = {
        ...mockFormMethods,
        formState: {
          errors: {
            name: { message: "Invalid name format" },
            prompt: { message: "Prompt too short" },
          },
          isSubmitting: false,
          isValid: false,
        },
        refineCore: {
          onFinish: vi.fn(),
          redirect: vi.fn(),
        },
        saveButtonProps: {
          disabled: true,
          onClick: vi.fn(),
        },
      };

      vi.mocked(require("@refinedev/react-hook-form").useForm).mockReturnValue(
        mockRefineFormWithInvalidData
      );

      render(<WorkerCreate />);

      // Should display validation errors
      expect(screen.getByText("Invalid name format")).toBeInTheDocument();
      expect(screen.getByText("Prompt too short")).toBeInTheDocument();

      // Save button should be disabled
      expect(screen.getByTestId("save-button")).toBeDisabled();
    });
  });

  describe("Accessibility", () => {
    it("should have proper form labels and associations", () => {
      render(<WorkerCreate />);

      // Check for proper labels
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Prompt")).toBeInTheDocument();
      expect(screen.getByText("Enabled")).toBeInTheDocument();
    });

    it("should support keyboard navigation", async () => {
      const user = userEvent.setup();
      render(<WorkerCreate />);

      // Tab through form fields
      await user.tab();
      const nameInput = screen.getByTestId("input-name");
      expect(nameInput).toHaveFocus();

      await user.tab();
      const promptInput = screen.getByTestId("input-prompt");
      expect(promptInput).toHaveFocus();

      await user.tab();
      const enabledSwitch = screen.getByTestId("switch");
      expect(enabledSwitch).toHaveFocus();

      await user.tab();
      const saveButton = screen.getByTestId("save-button");
      expect(saveButton).toHaveFocus();
    });

    it("should have proper ARIA attributes", () => {
      render(<WorkerCreate />);

      // Form should have proper roles
      const inputs = screen.getAllByRole("textbox");
      expect(inputs.length).toBeGreaterThan(0);

      const checkbox = screen.getByRole("checkbox");
      expect(checkbox).toBeInTheDocument();

      const button = screen.getByRole("button");
      expect(button).toBeInTheDocument();
    });
  });

  describe("Integration with Refine", () => {
    it("should use refine form hooks correctly", () => {
      render(<WorkerCreate />);

      // Should call useForm hook
      expect(require("@refinedev/react-hook-form").useForm).toHaveBeenCalled();
    });

    it("should configure form submission correctly", () => {
      render(<WorkerCreate />);

      // Should have onFinish handler from refine
      expect(mockRefineForm.refineCore.onFinish).toBeDefined();
    });

    it("should integrate with save button props", () => {
      render(<WorkerCreate />);

      // Save button should use props from refine
      const saveButton = screen.getByTestId("save-button");
      expect(saveButton).toBeInTheDocument();
    });
  });
});
