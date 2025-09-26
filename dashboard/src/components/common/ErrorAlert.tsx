import React from "react";
import { Alert, AlertTitle, Typography, Box } from "@mui/material";

interface ErrorAlertProps {
  error: any;
  title?: string;
  severity?: "error" | "warning" | "info" | "success";
  onClose?: () => void;
}

export const ErrorAlert: React.FC<ErrorAlertProps> = ({
  error,
  title = "Error",
  severity = "error",
  onClose,
}) => {
  if (!error) return null;

  const getErrorMessage = (error: any): string => {
    // If error is a string, return it directly
    if (typeof error === "string") return error;

    // Check for API error response structure
    if (error?.data?.error) return error.data.error;
    if (error?.data?.message) return error.data.message;

    // Check for standard error properties
    if (error?.message) return error.message;

    // Check for HTTP error status
    if (error?.status) {
      switch (error.status) {
        case 400:
          return "Invalid request. Please check your input and try again.";
        case 401:
          return "Unauthorized. Please check your credentials.";
        case 403:
          return "Forbidden. You do not have permission to perform this action.";
        case 404:
          return "Resource not found.";
        case 409:
          return "Conflict. The resource already exists or there is a naming conflict.";
        case 422:
          return "Validation error. Please check your input.";
        case 500:
          return "Internal server error. Please try again later.";
        default:
          return `HTTP Error ${error.status}`;
      }
    }

    // Fallback for unknown error types
    return "An unexpected error occurred. Please try again.";
  };

  const errorMessage = getErrorMessage(error);

  // Extract additional details if available
  const getErrorDetails = (error: any): string[] => {
    const details: string[] = [];

    if (error?.data?.details) {
      if (Array.isArray(error.data.details)) {
        details.push(...error.data.details);
      } else {
        details.push(error.data.details);
      }
    }

    // Add validation errors if present
    if (error?.data?.validation_errors) {
      details.push(...error.data.validation_errors);
    }

    return details;
  };

  const errorDetails = getErrorDetails(error);

  return (
    <Alert severity={severity} onClose={onClose} sx={{ mb: 2, mt: 1 }}>
      <AlertTitle>{title}</AlertTitle>
      <Typography variant="body2" sx={{ mb: errorDetails.length > 0 ? 1 : 0 }}>
        {errorMessage}
      </Typography>
      {errorDetails.length > 0 && (
        <Box component="ul" sx={{ margin: 0, paddingLeft: 2 }}>
          {errorDetails.map((detail, index) => (
            <li key={index}>
              <Typography variant="body2" color="inherit">
                {detail}
              </Typography>
            </li>
          ))}
        </Box>
      )}
    </Alert>
  );
};
