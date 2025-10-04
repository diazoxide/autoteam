import React from "react";
import { Box, TextField, Button, IconButton, Typography } from "@mui/material";
import { Add as AddIcon, Delete as DeleteIcon } from "@mui/icons-material";
import { Controller, useFieldArray, Control } from "react-hook-form";

interface EnvironmentSectionProps {
  control: Control<any>;
  stepIndex: number;
}

export const EnvironmentSection: React.FC<EnvironmentSectionProps> = ({
  control,
  stepIndex,
}) => {
  const {
    fields: envFields,
    append: appendEnv,
    remove: removeEnv,
  } = useFieldArray({
    control,
    name: `settings.flow.${stepIndex}.env`,
  });

  const handleAddEnv = () => {
    appendEnv({ key: "", value: "" });
  };

  return (
    <Box>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          mb: 2,
        }}
      >
        <Typography variant="subtitle2">Environment Variables</Typography>
        <Button
          size="small"
          startIcon={<AddIcon />}
          onClick={handleAddEnv}
          variant="outlined"
        >
          Add Variable
        </Button>
      </Box>

      {envFields.length === 0 ? (
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ fontStyle: "italic" }}
        >
          No environment variables configured
        </Typography>
      ) : (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
          {envFields.map((field, envIndex) => (
            <Box
              key={field.id}
              sx={{ display: "flex", alignItems: "center", gap: 1 }}
            >
              <Controller
                name={`settings.flow.${stepIndex}.env.${envIndex}.key`}
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label="Key"
                    size="small"
                    sx={{ minWidth: 120 }}
                  />
                )}
              />
              <Controller
                name={`settings.flow.${stepIndex}.env.${envIndex}.value`}
                control={control}
                render={({ field }) => (
                  <TextField {...field} label="Value" size="small" fullWidth />
                )}
              />
              <IconButton
                size="small"
                onClick={() => removeEnv(envIndex)}
                color="error"
              >
                <DeleteIcon />
              </IconButton>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
};
