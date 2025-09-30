import React from "react";
import { useForm } from "@refinedev/react-hook-form";
import {
  Box,
  TextField,
  FormControlLabel,
  Switch,
  Button,
  Typography,
  Alert,
  CircularProgress,
} from "@mui/material";
import { Save as SaveIcon } from "@mui/icons-material";
import { components } from "../../types/generated/api";

interface WorkerBasicSettingsFormProps {
  workerId?: string;
  initialData?: {
    name: string;
    prompt: string;
    enabled: boolean;
  };
  onSuccess?: (data?: any) => void;
  onError?: (error: any) => void;
}

export const WorkerBasicSettingsForm: React.FC<WorkerBasicSettingsFormProps> = ({
  workerId,
  initialData,
  onSuccess,
  onError,
}) => {
  const [submitError, setSubmitError] = React.useState<any>(null);

  const {
    saveButtonProps,
    refineCore: { formLoading },
    register,
    formState: { errors, isSubmitting },
    watch,
    setValue,
    reset,
    clearErrors,
  } = useForm<components["schemas"]["UpdateWorkerRequest"]>({
    defaultValues: {
      name: "",
      prompt: "",
      enabled: true,
    },
    refineCoreProps: {
      resource: "workers",
      action: workerId ? "edit" : "create",
      id: workerId,
      redirect: false,
      onMutationSuccess: (data) => {
        setSubmitError(null);
        onSuccess?.(data);
      },
      onMutationError: (error) => {
        setSubmitError(error);
        onError?.(error);
      },
    },
  });

  const enabled = watch("enabled");

  // Load initial data when available
  React.useEffect(() => {
    if (initialData) {
      reset(initialData);
    }
  }, [initialData, reset]);

  // Enhanced save button props with better error handling
  const enhancedSaveButtonProps = {
    ...saveButtonProps,
    onClick: async (event?: React.MouseEvent<HTMLButtonElement>) => {
      setSubmitError(null);
      clearErrors();

      try {
        if (saveButtonProps.onClick && event) {
          await saveButtonProps.onClick(event);
        }
      } catch (error: any) {
        console.error("Basic settings save error:", error);
        setSubmitError(error);
      }
    },
  };

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Basic Worker Settings
      </Typography>

      {/* Error Display */}
      {submitError && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => setSubmitError(null)}
        >
          Failed to save basic settings: {submitError?.message || "Unknown error"}
        </Alert>
      )}

      <Box
        component="form"
        sx={{ display: "flex", flexDirection: "column" }}
        autoComplete="off"
      >
        <TextField
          {...register("name", {
            required: "Worker name is required",
            minLength: {
              value: 1,
              message: "Worker name cannot be empty",
            },
            maxLength: {
              value: 100,
              message: "Worker name must be less than 100 characters",
            },
            pattern: {
              value: /^[a-zA-Z0-9_\-\s]+$/,
              message:
                "Worker name can only contain letters, numbers, spaces, hyphens, and underscores",
            },
            validate: (value: string) => {
              const trimmed = value?.trim();
              if (!trimmed) return "Worker name cannot be empty";
              if (trimmed.length < 1) return "Worker name cannot be empty";
              return true;
            },
          })}
          error={!!(errors as any)?.name}
          helperText={(errors as any)?.name?.message}
          margin="normal"
          fullWidth
          InputLabelProps={{ shrink: true }}
          type="text"
          label="Worker Name"
          name="name"
          placeholder="Enter a unique name for this worker"
        />

        <TextField
          {...register("prompt", {
            required: "Worker prompt is required",
            minLength: {
              value: 10,
              message: "Worker prompt must be at least 10 characters",
            },
            maxLength: {
              value: 5000,
              message: "Worker prompt must be less than 5000 characters",
            },
            validate: (value: string) => {
              const trimmed = value?.trim();
              if (!trimmed) return "Worker prompt cannot be empty";
              if (trimmed.length < 10)
                return "Worker prompt must be at least 10 characters";
              return true;
            },
          })}
          error={!!(errors as any)?.prompt}
          helperText={
            (errors as any)?.prompt?.message ||
            "Describe what this worker should do. Be specific about the worker's role and responsibilities."
          }
          margin="normal"
          fullWidth
          multiline
          rows={4}
          InputLabelProps={{ shrink: true }}
          type="text"
          label="Worker Prompt"
          name="prompt"
          placeholder="Enter a detailed description of what this worker should do..."
        />

        <FormControlLabel
          label="Enabled"
          control={
            <Switch
              checked={!!enabled}
              onChange={(event) => {
                setValue("enabled", event.target.checked, {
                  shouldValidate: true,
                });
              }}
            />
          }
          sx={{ mt: 2, mb: 3 }}
        />

        <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
          <Button
            {...enhancedSaveButtonProps}
            variant="contained"
            startIcon={
              isSubmitting ? (
                <CircularProgress size={20} color="inherit" />
              ) : (
                <SaveIcon />
              )
            }
            disabled={isSubmitting || formLoading}
          >
            {isSubmitting ? "Saving..." : workerId ? "Save Basic Settings" : "Create Worker"}
          </Button>
        </Box>
      </Box>
    </Box>
  );
};