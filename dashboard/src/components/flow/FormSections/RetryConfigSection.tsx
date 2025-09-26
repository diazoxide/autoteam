import React from "react";
import {
  Box,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
} from "@mui/material";
import { Controller, Control } from "react-hook-form";
import { components } from "../../../types/generated/api";

type RetryConfig = components["schemas"]["RetryConfig"];

interface RetryConfigSectionProps {
  control: Control<any>;
  stepIndex: number;
}

const BACKOFF_STRATEGIES: Array<RetryConfig["backoff"]> = [
  "fixed",
  "exponential",
  "linear",
];

export const RetryConfigSection: React.FC<RetryConfigSectionProps> = ({
  control,
  stepIndex,
}) => {
  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>
        Retry Configuration
      </Typography>
      <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
        <Controller
          name={`settings.flow.${stepIndex}.retry.max_attempts`}
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Max Attempts"
              type="number"
              size="small"
              inputProps={{ min: 1, max: 10 }}
              helperText="Maximum number of retry attempts"
            />
          )}
        />

        <Controller
          name={`settings.flow.${stepIndex}.retry.delay`}
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Initial Delay (seconds)"
              type="number"
              size="small"
              inputProps={{ min: 0 }}
              helperText="Initial delay before first retry"
            />
          )}
        />

        <Controller
          name={`settings.flow.${stepIndex}.retry.backoff`}
          control={control}
          render={({ field }) => (
            <FormControl fullWidth size="small">
              <InputLabel>Backoff Strategy</InputLabel>
              <Select {...field} label="Backoff Strategy">
                {BACKOFF_STRATEGIES.map((strategy) => (
                  <MenuItem key={strategy} value={strategy}>
                    {strategy}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
        />

        <Controller
          name={`settings.flow.${stepIndex}.retry.max_delay`}
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              label="Max Delay (seconds)"
              type="number"
              size="small"
              inputProps={{ min: 0 }}
              helperText="Maximum delay between retries"
            />
          )}
        />
      </Box>
    </Box>
  );
};
