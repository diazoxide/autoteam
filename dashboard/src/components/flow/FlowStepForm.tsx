import React from "react";
import {
  Box,
  TextField,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  IconButton,
  Divider,
  Tooltip,
} from "@mui/material";
import {
  ExpandMore as ExpandMoreIcon,
  Delete as DeleteIcon,
} from "@mui/icons-material";
import { Controller, Control } from "react-hook-form";
import {
  BasicConfigSection,
  ArgumentsSection,
  EnvironmentSection,
  PromptSection,
  RetryConfigSection,
} from "./FormSections";

interface FlowStepFormProps {
  control: Control<any>;
  stepIndex: number;
  onRemove: () => void;
  availableSteps: string[];
  isExpanded: boolean;
  onToggleExpand: () => void;
}

export const FlowStepForm: React.FC<FlowStepFormProps> = ({
  control,
  stepIndex,
  onRemove,
  availableSteps,
  isExpanded,
  onToggleExpand,
}) => {
  return (
    <Box sx={{ position: "relative" }}>
      {/* Remove Button - positioned absolutely outside the accordion */}
      <Tooltip title="Remove step">
        <IconButton
          color="error"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          sx={{
            position: "absolute",
            top: 8,
            right: 8,
            zIndex: 10,
            backgroundColor: "background.paper",
            border: "1px solid",
            borderColor: "divider",
            "&:hover": {
              backgroundColor: "error.light",
              color: "error.contrastText",
            },
          }}
        >
          <DeleteIcon />
        </IconButton>
      </Tooltip>

      <Accordion expanded={isExpanded} onChange={onToggleExpand}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 2,
              width: "100%",
              pr: 6, // Add padding to avoid overlap with remove button
            }}
          >
            <Controller
              name={`settings.flow.${stepIndex}.name`}
              control={control}
              rules={{
                required: "Step name is required",
                minLength: { value: 1, message: "Step name cannot be empty" },
                maxLength: {
                  value: 50,
                  message: "Step name must be less than 50 characters",
                },
                pattern: {
                  value: /^[a-zA-Z0-9_-]+$/,
                  message:
                    "Step name can only contain letters, numbers, hyphens, and underscores",
                },
              }}
              render={({ field, fieldState }) => (
                <TextField
                  {...field}
                  label="Step Name"
                  error={!!fieldState.error}
                  helperText={fieldState.error?.message}
                  size="small"
                  sx={{ flexGrow: 1 }}
                  onClick={(e) => e.stopPropagation()}
                />
              )}
            />
          </Box>
        </AccordionSummary>

        <AccordionDetails>
          <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            {/* Basic Configuration */}
            <BasicConfigSection
              control={control}
              stepIndex={stepIndex}
              availableSteps={availableSteps}
            />

            <Divider />

            {/* Input/Output Configuration */}
            <PromptSection control={control} stepIndex={stepIndex} />

            <Divider />

            {/* Arguments */}
            <ArgumentsSection control={control} stepIndex={stepIndex} />

            <Divider />

            {/* Environment Variables */}
            <EnvironmentSection control={control} stepIndex={stepIndex} />

            <Divider />

            {/* Retry Configuration */}
            <RetryConfigSection control={control} stepIndex={stepIndex} />
          </Box>
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};
