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

interface WorkerMetricsProps {
  metricsData: any;
  metricsLoading: boolean;
  flowData: any;
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

  const metrics = metricsData?.data || {};
  const flowMetrics = flowData?.data?.flow || {};

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
                  {flowMetrics.execution_count || 0}
                </Typography>
                <Typography variant="body2">Total Executions</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="success.main">
                  {Math.round((flowMetrics.success_rate || 0) * 100)}%
                </Typography>
                <Typography variant="body2">Success Rate</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="info.main">
                  {flowMetrics.total_steps || 0}
                </Typography>
                <Typography variant="body2">Total Steps</Typography>
              </Paper>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Paper sx={{ p: 2, textAlign: "center" }}>
                <Typography variant="h4" color="warning.main">
                  {flowMetrics.enabled_steps || 0}
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
                      label={formatUptime(metrics.uptime)} 
                      color="success" 
                    />
                  </Box>
                  
                  {metrics.memory_usage && (
                    <Box>
                      <Typography variant="subtitle2" gutterBottom>
                        Memory Usage
                      </Typography>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <LinearProgress
                          variant="determinate"
                          value={metrics.memory_usage.percent || 0}
                          sx={{ flexGrow: 1, height: 8 }}
                          color={
                            (metrics.memory_usage.percent || 0) > 80
                              ? "error"
                              : (metrics.memory_usage.percent || 0) > 60
                              ? "warning"
                              : "primary"
                          }
                        />
                        <Typography variant="body2">
                          {metrics.memory_usage.percent || 0}%
                        </Typography>
                      </Stack>
                      <Typography variant="body2" color="textSecondary">
                        {formatBytes(metrics.memory_usage.used)} / {formatBytes(metrics.memory_usage.total)}
                      </Typography>
                    </Box>
                  )}
                  
                  {metrics.cpu_usage && (
                    <Box>
                      <Typography variant="subtitle2" gutterBottom>
                        CPU Usage
                      </Typography>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <LinearProgress
                          variant="determinate"
                          value={metrics.cpu_usage || 0}
                          sx={{ flexGrow: 1, height: 8 }}
                          color={
                            metrics.cpu_usage > 80
                              ? "error"
                              : metrics.cpu_usage > 60
                              ? "warning"
                              : "primary"
                          }
                        />
                        <Typography variant="body2">
                          {metrics.cpu_usage || 0}%
                        </Typography>
                      </Stack>
                    </Box>
                  )}
                </Stack>
              </Grid>

              <Grid item xs={12} sm={6}>
                <Stack spacing={2}>
                  {metrics.disk_usage && (
                    <Box>
                      <Typography variant="subtitle2" gutterBottom>
                        Disk Usage
                      </Typography>
                      <Stack direction="row" spacing={1} alignItems="center">
                        <LinearProgress
                          variant="determinate"
                          value={metrics.disk_usage.percent || 0}
                          sx={{ flexGrow: 1, height: 8 }}
                          color={
                            (metrics.disk_usage.percent || 0) > 80
                              ? "error"
                              : (metrics.disk_usage.percent || 0) > 60
                              ? "warning"
                              : "primary"
                          }
                        />
                        <Typography variant="body2">
                          {metrics.disk_usage.percent || 0}%
                        </Typography>
                      </Stack>
                      <Typography variant="body2" color="textSecondary">
                        {formatBytes(metrics.disk_usage.used)} / {formatBytes(metrics.disk_usage.total)}
                      </Typography>
                    </Box>
                  )}

                  {flowMetrics.last_execution && (
                    <Box>
                      <Typography variant="subtitle2" gutterBottom>
                        Last Execution
                      </Typography>
                      <Typography variant="body2">
                        {new Date(flowMetrics.last_execution).toLocaleString()}
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