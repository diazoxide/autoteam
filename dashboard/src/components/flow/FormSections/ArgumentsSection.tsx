import React from "react";
import { Box, TextField, Button, IconButton, Typography } from "@mui/material";
import { Add as AddIcon, Delete as DeleteIcon } from "@mui/icons-material";
import { Controller, useFieldArray, Control } from "react-hook-form";

interface ArgumentsSectionProps {
  control: Control<any>;
  stepIndex: number;
}

export const ArgumentsSection: React.FC<ArgumentsSectionProps> = ({
  control,
  stepIndex,
}) => {
  const {
    fields: argsFields,
    append: appendArg,
    remove: removeArg,
  } = useFieldArray({
    control,
    name: `settings.flow.${stepIndex}.args`,
  });

  const handleAddArg = () => {
    appendArg("");
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
        <Typography variant="subtitle2">Arguments</Typography>
        <Button
          size="small"
          startIcon={<AddIcon />}
          onClick={handleAddArg}
          variant="outlined"
        >
          Add Argument
        </Button>
      </Box>

      {argsFields.length === 0 ? (
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ fontStyle: "italic" }}
        >
          No arguments configured
        </Typography>
      ) : (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
          {argsFields.map((field, argIndex) => (
            <Box
              key={field.id}
              sx={{ display: "flex", alignItems: "center", gap: 1 }}
            >
              <Controller
                name={`settings.flow.${stepIndex}.args.${argIndex}`}
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    label={`Argument ${argIndex + 1}`}
                    size="small"
                    fullWidth
                  />
                )}
              />
              <IconButton
                size="small"
                onClick={() => removeArg(argIndex)}
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
