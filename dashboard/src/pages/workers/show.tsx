import React from "react";
import {
  Show,
  TextFieldComponent as TextField,
} from "@refinedev/mui";
import { useShow, useCustom } from "@refinedev/core";
import {
  Stack,
  Typography,
  Card,
  CardContent,
  Grid,
  Chip,
  Box,
  Alert,
  CircularProgress,
} from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import { useParams } from "react-router";

export const WorkersShow = () => {
  const { id } = useParams();
  
  const { queryResult } = useShow({
    resource: "workers",
    id: id as string,
  });

  const { data: worker, isLoading, error } = queryResult;

  // Get worker health status
  const { data: healthData, isLoading: healthLoading } = useCustom({
    url: `/workers/${id}/health`,
    method: "get",
    queryOptions: {
      refetchInterval: 5000, // Refresh every 5 seconds
      enabled: !!id,
    },
  });

  // Get worker status details
  const { data: statusData, isLoading: statusLoading } = useCustom({
    url: `/workers/${id}/status`,
    method: "get",
    queryOptions: {
      refetchInterval: 10000, // Refresh every 10 seconds
      enabled: !!id,
    },
  });

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        Failed to load worker details: {error.message}
      </Alert>
    );
  }

  const isHealthy = healthData?.data?.status === "healthy";

  return (
    <Show isLoading={isLoading}>
      <Stack gap={2}>
        {/* Basic Information */}
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Worker Information
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12} md={6}>
                <TextField
                  value={worker?.data?.id}
                  label="Worker ID"
                />
              </Grid>
              <Grid item xs={12} md={6}>
                <TextField
                  value={worker?.data?.api_url}
                  label="API URL"
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>

        {/* Health Status */}
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Health Status
            </Typography>
            {healthLoading ? (
              <CircularProgress size={24} />
            ) : (
              <Box>
                <Chip
                  icon={isHealthy ? <CheckCircleIcon /> : <ErrorIcon />}
                  label={isHealthy ? "Healthy" : "Unhealthy"}
                  color={isHealthy ? "success" : "error"}
                  variant="outlined"
                  sx={{ mb: 2 }}
                />
                {healthData?.data?.last_check && (
                  <Typography variant="body2" color="textSecondary">
                    Last checked: {new Date(healthData.data.last_check).toLocaleString()}
                  </Typography>
                )}
                {healthData?.data?.error && (
                  <Alert severity="error" sx={{ mt: 2 }}>
                    {healthData.data.error}
                  </Alert>
                )}
              </Box>
            )}
          </CardContent>
        </Card>

        {/* Worker Status Details */}
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Worker Status
            </Typography>
            {statusLoading ? (
              <CircularProgress size={24} />
            ) : statusData?.data ? (
              <Grid container spacing={2}>
                <Grid item xs={12} md={6}>
                  <TextField
                    value={statusData.data.version || "Unknown"}
                    label="Version"
                  />
                </Grid>
                <Grid item xs={12} md={6}>
                  <TextField
                    value={statusData.data.uptime || "Unknown"}
                    label="Uptime"
                  />
                </Grid>
                {statusData.data.config && (
                  <>
                    <Grid item xs={12} md={6}>
                      <TextField
                        value={statusData.data.config.agent_name || "Unknown"}
                        label="Agent Name"
                      />
                    </Grid>
                    <Grid item xs={12} md={6}>
                      <TextField
                        value={statusData.data.config.github_user || "Unknown"}
                        label="GitHub User"
                      />
                    </Grid>
                  </>
                )}
              </Grid>
            ) : (
              <Alert severity="warning">
                Worker status information is not available
              </Alert>
            )}
          </CardContent>
        </Card>
      </Stack>
    </Show>
  );
};