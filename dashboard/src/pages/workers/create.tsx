import React from "react";
import { Create } from "@refinedev/mui";
import { useForm } from "@refinedev/react-hook-form";
import {
  Box,
  TextField,
  FormControlLabel,
  Switch,
  Tabs,
  Tab,
  Typography,
} from "@mui/material";
import { FlowConfiguration } from "../../components/flow";
import { ErrorAlert } from "../../components/common/ErrorAlert";
import { components } from "../../types/generated/api";

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`worker-tabpanel-${index}`}
      aria-labelledby={`worker-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
  );
}

export const WorkersCreate = () => {
  const [tabValue, setTabValue] = React.useState(0);
  const [submitError, setSubmitError] = React.useState<any>(null);

  const {
    saveButtonProps,
    refineCore: { formLoading, onFinish },
    register,
    control,
    formState: { errors, isSubmitting },
    watch,
    setValue,
    setError,
    clearErrors,
  } = useForm<
    components["schemas"]["CreateWorkerRequest"] & {
      settings?: { flow?: any[] };
    }
  >({
    defaultValues: {
      enabled: true,
      settings: {
        flow: [],
      },
    },
  });

  const enabled = watch("enabled");

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

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
        console.error("Form submission error:", error);
        setSubmitError(error);

        // Handle specific validation errors
        if (error?.status === 400 || error?.status === 422) {
          // Switch to the Basic Information tab if there are basic field errors
          const errorMessage = error?.data?.error || error?.message || "";
          if (
            errorMessage.includes("name") ||
            errorMessage.includes("prompt")
          ) {
            setTabValue(0);
          }
        }
      }
    },
  };

  return (
    <Create isLoading={formLoading} saveButtonProps={enhancedSaveButtonProps}>
      <Box sx={{ width: "100%" }}>
        {/* Global Error Display */}
        {submitError && (
          <ErrorAlert
            error={submitError}
            title="Failed to Create Worker"
            onClose={() => setSubmitError(null)}
          />
        )}
        <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
          <Tabs value={tabValue} onChange={handleTabChange}>
            <Tab label="Basic Information" />
            <Tab label="Flow Configuration" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
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
            />
          </Box>
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <Typography variant="h6" gutterBottom>
            Flow Configuration
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Define the workflow steps that this worker will execute. Each step
            represents an agent execution with specific configuration.
          </Typography>
          <FlowConfiguration control={control} />
        </TabPanel>
      </Box>
    </Create>
  );
};
