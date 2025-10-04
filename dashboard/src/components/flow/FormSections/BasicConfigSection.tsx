import React from "react";
import {
  Box,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Autocomplete,
  Chip,
} from "@mui/material";
import { Controller, Control } from "react-hook-form";
import { components } from "../../../types/generated/api";

type FlowStepInput = components["schemas"]["FlowStepInput"];

interface BasicConfigSectionProps {
  control: Control<any>;
  stepIndex: number;
  availableSteps: string[];
}

const AGENT_TYPES = ["claude", "qwen", "gemini", "debug"];
const DEPENDENCY_POLICIES: Array<FlowStepInput["dependency_policy"]> = [
  "fail_fast",
  "all_success",
  "all_complete",
  "any_success",
];

export const BasicConfigSection: React.FC<BasicConfigSectionProps> = ({
  control,
  stepIndex,
  availableSteps,
}) => {
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <Controller
        name={`settings.flow.${stepIndex}.name`}
        control={control}
        rules={{
          required: "Step name is required",
          minLength: {
            value: 1,
            message: "Step name cannot be empty",
          },
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
            helperText={
              fieldState.error?.message || "Unique identifier for this step"
            }
            fullWidth
            size="small"
          />
        )}
      />

      <Controller
        name={`settings.flow.${stepIndex}.type`}
        control={control}
        rules={{ required: "Agent type is required" }}
        render={({ field, fieldState }) => (
          <FormControl fullWidth size="small" error={!!fieldState.error}>
            <InputLabel>Agent Type</InputLabel>
            <Select {...field} label="Agent Type">
              {AGENT_TYPES.map((type) => (
                <MenuItem key={type} value={type}>
                  {type}
                </MenuItem>
              ))}
            </Select>
            {fieldState.error && (
              <Box sx={{ color: "error.main", fontSize: "0.75rem", mt: 1 }}>
                {fieldState.error.message}
              </Box>
            )}
          </FormControl>
        )}
      />

      <Controller
        name={`settings.flow.${stepIndex}.dependency_policy`}
        control={control}
        render={({ field }) => (
          <FormControl fullWidth size="small">
            <InputLabel>Dependency Policy</InputLabel>
            <Select {...field} label="Dependency Policy">
              {DEPENDENCY_POLICIES.map((policy) => (
                <MenuItem key={policy} value={policy}>
                  {policy}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        )}
      />

      <Controller
        name={`settings.flow.${stepIndex}.depends_on`}
        control={control}
        render={({ field }) => (
          <Autocomplete
            {...field}
            multiple
            options={availableSteps}
            renderTags={(value, getTagProps) =>
              value.map((option, index) => (
                <Chip
                  variant="outlined"
                  label={option}
                  {...getTagProps({ index })}
                  key={option}
                />
              ))
            }
            renderInput={(params) => (
              <TextField
                {...params}
                label="Dependencies"
                helperText="Select steps that must complete before this step runs"
                size="small"
              />
            )}
            onChange={(_, value) => field.onChange(value)}
            value={field.value || []}
          />
        )}
      />
    </Box>
  );
};
