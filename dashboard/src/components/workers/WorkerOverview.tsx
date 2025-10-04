import React from "react";
import {
  Card,
  CardContent,
  Grid,
  Typography,
  Paper,
  Chip,
  Stack,
  Avatar,
  Box,
} from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import PauseIcon from "@mui/icons-material/Pause";
import type {
  Worker,
  HealthResponse,
  StatusResponse,
  FlowStepsResponse,
  MetricsResponse,
} from "../../types/api";
import { components } from "../../types/generated/api";

interface WorkerOverviewProps {
  worker: { data?: Worker } | undefined;
  runtimeData: components["schemas"]["WorkerDetailsResponse"] | undefined;
  healthData: HealthResponse | undefined;
  statusData: StatusResponse | undefined;
  flowStepsData: FlowStepsResponse | undefined;
  metricsData: MetricsResponse | undefined;
  runtimeLoading: boolean;
  healthLoading: boolean;
  statusLoading: boolean;
}

export const WorkerOverview: React.FC<WorkerOverviewProps> = ({
  worker,
  runtimeData,
  healthData,
  statusData,
  metricsData,
  runtimeLoading,
  healthLoading,
}) => {
  const getStatusColor = (
    status: string
  ): "success" | "error" | "warning" | "default" => {
    switch (status?.toLowerCase()) {
      case "healthy":
      case "running":
      case "active":
        return "success";
      case "error":
      case "failed":
        return "error";
      case "warning":
        return "warning";
      default:
        return "default";
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status?.toLowerCase()) {
      case "healthy":
      case "running":
      case "active":
        return <CheckCircleIcon />;
      case "error":
      case "failed":
        return <ErrorIcon />;
      default:
        return statusData?.status === "running" ? (
          <PlayArrowIcon />
        ) : (
          <PauseIcon />
        );
    }
  };

  return (
    <Stack spacing={3}>
      {/* Worker Header */}
      <Card>
        <CardContent>
          <Stack direction="row" spacing={2} alignItems="center">
            <Avatar sx={{ bgcolor: "primary.main", width: 64, height: 64 }}>
              {(worker?.data?.name || worker?.data?.id)
                ?.charAt(0)
                .toUpperCase() || "W"}
            </Avatar>
            <Box sx={{ flexGrow: 1 }}>
              <Typography variant="h4" gutterBottom>
                {worker?.data?.name || "Worker"}
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                ID: {worker?.data?.id}
              </Typography>
              <Typography variant="body2" color="text.secondary" gutterBottom>
                Status: {worker?.data?.enabled ? "Enabled" : "Disabled"} •
                Runtime: {runtimeData?.worker?.status || "Unknown"}
              </Typography>
              <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                <Chip
                  icon={getStatusIcon(healthData?.status || "unknown")}
                  label={`Health: ${healthData?.status || "Unknown"}`}
                  color={getStatusColor(healthData?.status || "unknown")}
                  size="small"
                />
                <Chip
                  label={
                    runtimeData?.worker?.status === "reachable"
                      ? "Online"
                      : "Offline"
                  }
                  color={
                    runtimeData?.worker?.status === "reachable"
                      ? "success"
                      : "error"
                  }
                  size="small"
                />
                {statusData?.status && (
                  <Chip
                    label={`Status: ${statusData.status}`}
                    color={
                      statusData?.status === "running" ? "success" : "default"
                    }
                    size="small"
                  />
                )}
              </Stack>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      {/* Quick Stats */}
      <Grid container spacing={3}>
        <Grid item xs={12} sm={3}>
          <Paper
            sx={{
              p: 2,
              textAlign: "center",
              bgcolor: "success.light",
              color: "white",
            }}
          >
            <Typography variant="h6">
              {runtimeLoading
                ? "..."
                : runtimeData?.worker?.status || "Unknown"}
            </Typography>
            <Typography variant="body2">Registry Status</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper
            sx={{
              p: 2,
              textAlign: "center",
              bgcolor: "primary.light",
              color: "white",
            }}
          >
            <Typography variant="h6">
              {healthLoading ? "..." : healthData?.status || "Unknown"}
            </Typography>
            <Typography variant="body2">Health Check</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper
            sx={{
              p: 2,
              textAlign: "center",
              bgcolor: "info.light",
              color: "white",
            }}
          >
            <Typography variant="h6">
              {worker?.data?.settings?.flow?.length || "0"}
            </Typography>
            <Typography variant="body2">Configured Steps</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper
            sx={{
              p: 2,
              textAlign: "center",
              bgcolor: "warning.light",
              color: "white",
            }}
          >
            <Typography variant="h6">
              {metricsData?.metrics?.uptime || "N/A"}
            </Typography>
            <Typography variant="body2">Uptime</Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Configuration Details */}
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Configuration
              </Typography>
              <Stack spacing={2}>
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Worker Name
                  </Typography>
                  <Typography variant="body1">
                    {worker?.data?.name || "Unnamed Worker"}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Prompt
                  </Typography>
                  <Typography
                    variant="body1"
                    sx={{ maxHeight: 100, overflow: "auto" }}
                  >
                    {worker?.data?.prompt || "No prompt configured"}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Status
                  </Typography>
                  <Chip
                    label={worker?.data?.enabled ? "Enabled" : "Disabled"}
                    color={worker?.data?.enabled ? "success" : "default"}
                    size="small"
                  />
                </Box>
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Flow Steps
                  </Typography>
                  <Typography variant="body1">
                    {worker?.data?.settings?.flow?.length || 0} steps configured
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Runtime Information
              </Typography>
              <Stack spacing={2}>
                {runtimeData?.worker?.worker_info ? (
                  <>
                    <Box>
                      <Typography variant="body2" color="text.secondary">
                        Runtime Name
                      </Typography>
                      <Typography variant="body1">
                        {runtimeData.worker.worker_info.name || "N/A"}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary">
                        Version
                      </Typography>
                      <Typography variant="body1">
                        {runtimeData.worker.worker_info.version || "N/A"}
                      </Typography>
                    </Box>
                    <Box>
                      <Typography variant="body2" color="text.secondary">
                        Last Execution
                      </Typography>
                      <Typography variant="body1">
                        {metricsData?.metrics?.last_activity
                          ? new Date(
                              metricsData.metrics.last_activity
                            ).toLocaleString()
                          : "Never"}
                      </Typography>
                    </Box>
                  </>
                ) : (
                  <Typography variant="body2" color="text.secondary">
                    Worker not reachable - no runtime information available
                  </Typography>
                )}
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Registry URL
                  </Typography>
                  <Typography variant="body1" sx={{ wordBreak: "break-all" }}>
                    {runtimeData?.worker?.url || "N/A"}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="body2" color="text.secondary">
                    Last Check
                  </Typography>
                  <Typography variant="body1">
                    {runtimeData?.worker?.last_check
                      ? new Date(runtimeData.worker.last_check).toLocaleString()
                      : "Never"}
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Stack>
  );
};
