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

interface WorkerOverviewProps {
  worker: any;
  healthData: any;
  statusData: any;
  flowStepsData: any;
  metricsData: any;
  healthLoading: boolean;
  statusLoading: boolean;
}

export const WorkerOverview: React.FC<WorkerOverviewProps> = ({
  worker,
  healthData,
  statusData,
  flowStepsData,
  metricsData,
  healthLoading,
  statusLoading,
}) => {
  const getStatusColor = (status: string) => {
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
        return statusData?.data?.active ? <PlayArrowIcon /> : <PauseIcon />;
    }
  };

  return (
    <Stack spacing={3}>
      {/* Worker Header */}
      <Card>
        <CardContent>
          <Stack direction="row" spacing={2} alignItems="center">
            <Avatar sx={{ bgcolor: "primary.main", width: 64, height: 64 }}>
              {worker?.data?.id?.charAt(0).toUpperCase() || "W"}
            </Avatar>
            <Box sx={{ flexGrow: 1 }}>
              <Typography variant="h4" gutterBottom>
                {worker?.data?.id || "Worker"}
              </Typography>
              <Stack direction="row" spacing={1}>
                <Chip
                  icon={getStatusIcon(healthData?.data?.status)}
                  label={healthData?.data?.status || "Unknown"}
                  color={getStatusColor(healthData?.data?.status) as any}
                  size="medium"
                />
                <Chip
                  label={statusData?.data?.active ? "Active" : "Inactive"}
                  color={statusData?.data?.active ? "success" : "default"}
                  size="medium"
                />
              </Stack>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      {/* Quick Stats */}
      <Grid container spacing={3}>
        <Grid item xs={12} sm={3}>
          <Paper sx={{ p: 2, textAlign: "center", bgcolor: "success.light", color: "white" }}>
            <Typography variant="h6">
              {healthLoading ? "..." : healthData?.data?.status || "Unknown"}
            </Typography>
            <Typography variant="body2">Health Status</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper sx={{ p: 2, textAlign: "center", bgcolor: "primary.light", color: "white" }}>
            <Typography variant="h6">
              {statusLoading ? "..." : statusData?.data?.active ? "Active" : "Inactive"}
            </Typography>
            <Typography variant="body2">Worker Status</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper sx={{ p: 2, textAlign: "center", bgcolor: "info.light", color: "white" }}>
            <Typography variant="h6">{flowStepsData?.data?.steps?.length || "0"}</Typography>
            <Typography variant="body2">Flow Steps</Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={3}>
          <Paper sx={{ p: 2, textAlign: "center", bgcolor: "warning.light", color: "white" }}>
            <Typography variant="h6">
              {metricsData?.data?.uptime ? `${Math.round(metricsData.data.uptime / 3600)}h` : "N/A"}
            </Typography>
            <Typography variant="body2">Uptime</Typography>
          </Paper>
        </Grid>
      </Grid>
    </Stack>
  );
};