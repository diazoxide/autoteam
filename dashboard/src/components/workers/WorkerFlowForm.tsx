import React from "react";
import { useForm } from "@refinedev/react-hook-form";
import {
  Box,
  Button,
  Typography,
  Alert,
  CircularProgress,
} from "@mui/material";
import { Save as SaveIcon } from "@mui/icons-material";
import { FlowConfiguration } from "../flow";
import { useCustomMutation } from "@refinedev/core";

interface WorkerFlowFormProps {
  workerId?: string;
  initialFlowData?: any[];
  onSuccess?: () => void;
  onError?: (error: any) => void;
}

export const WorkerFlowForm: React.FC<WorkerFlowFormProps> = ({
  workerId,
  initialFlowData,
  onSuccess,
  onError,
}) => {
  const [submitError, setSubmitError] = React.useState<any>(null);

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
    reset,
    getValues,
  } = useForm<{ settings: { flow: any[] } }>({
    defaultValues: {
      settings: {
        flow: [],
      },
    },
  });

  // Custom mutation for updating worker settings
  const { mutate: updateWorkerSettings, isLoading: isSaving } = useCustomMutation();

  // Load initial flow data when available
  React.useEffect(() => {
    if (initialFlowData) {
      reset({
        settings: {
          flow: initialFlowData,
        },
      });
    }
  }, [initialFlowData, reset]);

  const onSubmit = async (formData: any) => {
    if (!workerId) {
      setSubmitError(new Error("Worker ID is required"));
      return;
    }

    setSubmitError(null);

    try {
      // Transform form data to API format
      const transformedFlow = formData.settings.flow.map((step: any) => {
        // Transform env from form array format [{key, value}] to API object format {KEY: "value"}
        let envObject: { [key: string]: string } = {};
        if (Array.isArray(step.env)) {
          step.env.forEach((envVar: { key: string; value: string }) => {
            if (envVar.key && envVar.key.trim()) {
              envObject[envVar.key] = envVar.value || "";
            }
          });
        } else if (step.env && typeof step.env === "object") {
          envObject = step.env;
        }

        return {
          ...step,
          env: envObject,
        };
      });

      const updateData = {
        settings: {
          flow: transformedFlow,
        },
      };

      await updateWorkerSettings(
        {
          url: `/workers/${workerId}`,
          method: "put",
          values: updateData,
        },
        {
          onSuccess: () => {
            console.log("Flow configuration saved successfully");
            onSuccess?.();
          },
          onError: (error) => {
            console.error("Flow configuration save error:", error);
            setSubmitError(error);
            onError?.(error);
          },
        }
      );
    } catch (error: any) {
      console.error("Flow configuration save error:", error);
      setSubmitError(error);
      onError?.(error);
    }
  };

  const handleSave = () => {
    handleSubmit(onSubmit)();
  };

  return (
    <Box>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          mb: 3,
        }}
      >
        <Box>
          <Typography variant="h6" gutterBottom>
            Flow Configuration
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define the workflow steps that this worker will execute. Each step
            represents an agent execution with specific configuration.
          </Typography>
        </Box>

        <Button
          onClick={handleSave}
          variant="contained"
          startIcon={
            isSaving ? (
              <CircularProgress size={20} color="inherit" />
            ) : (
              <SaveIcon />
            )
          }
          disabled={isSaving || !workerId}
        >
          {isSaving ? "Saving..." : "Save Flow Configuration"}
        </Button>
      </Box>

      {/* Error Display */}
      {submitError && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => setSubmitError(null)}
        >
          Failed to save flow configuration: {submitError?.message || "Unknown error"}
        </Alert>
      )}

      <FlowConfiguration control={control} />
    </Box>
  );
};