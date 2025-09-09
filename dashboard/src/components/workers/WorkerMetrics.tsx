import React from "react";
import {
  Card,
  CardContent,
  CardHeader,
  Grid,
  Typography,
  Paper,
  LinearProgress,
  Box,
  Stack,
  Chip,
  CircularProgress,
  Alert,
} from "@mui/material";
import MetricsIcon from "@mui/icons-material/Analytics";
import type { MetricsResponse, FlowResponse } from "../../types/api";

interface WorkerMetricsProps {
  metricsData: MetricsResponse | undefined;
  metricsLoading: boolean;
  flowData: FlowResponse | undefined;
}

export const WorkerMetrics: React.FC<WorkerMetricsProps> = ({
  metricsData,
  metricsLoading,
  flowData,
}) => {
  const formatUptime = (seconds: number) => {
    if (!seconds) return "N/A";
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  const formatBytes = (bytes: number) => {
    if (!bytes) return "0 B";
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${Math.round(bytes / Math.pow(1024, i) * 100) / 100} ${sizes[i]}`;
  };

  if (metricsLoading) {
    return <CircularProgress />;
  }

  const metrics = metricsData?.metrics || {};
  const flowMetrics = flowData?.flow || {};

  return (
    <Stack spacing={3}>
      {/* Execution Metrics */}
      <Card>
        <CardHeader title="Execution Metrics" avatar={<MetricsIcon />} />
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="primary">
                  {(flowMetrics as any)?.execution_count || 0}
                </Typography>
                <Typography variant="body2">Total Executions</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="success.main">
                  {Math.round(((flowMetrics as any)?.success_rate || 0) * 100)}%
                </Typography>
                <Typography variant="body2">Success Rate</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="info.main">
                  {(flowMetrics as any)?.total_steps || 0}
                </Typography>
                <Typography variant="body2">Total Steps</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="warning.main">
                  {(flowMetrics as any)?.enabled_steps || 0}
                </Typography>
                <Typography variant="body2">Enabled Steps</Typography>
              </Paper>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* System Metrics */}
      {metrics && Object.keys(metrics).length > 0 && (
        <Card>
          <CardHeader title="System Metrics" avatar={<MetricsIcon />} />
          <CardContent>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="subtitle2" gutterBottom>
                      Uptime
                    </Typography>
                    <Chip 
                      label={metrics.uptime || "N/A"} 
                      color="success" 
                    />
                  </Box>
                  
                  <Box>
                    <Typography variant="subtitle2" gutterBottom>
                      Average Execution Time
                    </Typography>
                    <Chip 
                      label={metrics.avg_execution_time || "N/A"} 
                      color="info" 
                    />
                  </Box>
                </Stack>
              </Grid>

              <Grid item xs={12} sm={6}>
                <Stack spacing={2}>
                  <Box>
                    <Typography variant="subtitle2" gutterBottom>
                      Last Activity
                    </Typography>
                    <Typography variant="body2">
                      {metrics.last_activity ? new Date(metrics.last_activity).toLocaleString() : "N/A"}
                    </Typography>
                  </Box>

                  {(flowMetrics as any)?.last_execution && (
                    <Box>
                      <Typography variant="subtitle2" gutterBottom>
                        Last Execution
                      </Typography>
                      <Typography variant="body2">
                        {new Date((flowMetrics as any)?.last_execution).toLocaleString()}
                      </Typography>
                    </Box>
                  )}
                </Stack>
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      )}

      {!metrics && !metricsLoading && (
        <Alert severity="info">
          No metrics data available
        </Alert>
      )}
    </Stack>
  );
};