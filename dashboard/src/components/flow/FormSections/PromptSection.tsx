import React from "react";
import { Box, TextField } from "@mui/material";
import { Controller, Control } from "react-hook-form";

interface PromptSectionProps {
  control: Control<any>;
  stepIndex: number;
}

export const PromptSection: React.FC<PromptSectionProps> = ({
  control,
  stepIndex,
}) => {
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <Controller
        name={`settings.flow.${stepIndex}.input`}
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            label="Input Prompt"
            multiline
            rows={4}
            fullWidth
            helperText="Define the instructions for this step. Supports Go template syntax."
            placeholder="Enter the prompt that will be executed by the agent..."
          />
        )}
      />

      <Controller
        name={`settings.flow.${stepIndex}.output`}
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            label="Output Template"
            multiline
            rows={2}
            fullWidth
            helperText="Optional: Transform the output using Go template syntax"
            placeholder={'{{- .stdout | regexFind "PATTERN" -}}'}
          />
        )}
      />

      <Controller
        name={`settings.flow.${stepIndex}.skip_when`}
        control={control}
        render={({ field }) => (
          <TextField
            {...field}
            label="Skip Condition"
            fullWidth
            helperText="Optional: Condition to skip this step (Go template syntax)"
            placeholder={'{{- index .inputs 0 | trim | eq "0" -}}'}
          />
        )}
      />
    </Box>
  );
};
