import React from 'react';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  Divider,
  Stack,
  Chip,
  Card,
  CardContent,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Paper,
  useTheme,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PauseIcon from '@mui/icons-material/Pause';
import ScheduleIcon from '@mui/icons-material/Schedule';
import SettingsIcon from '@mui/icons-material/Settings';
import CodeIcon from '@mui/icons-material/Code';
import LinkIcon from '@mui/icons-material/Link';
import InfoIcon from '@mui/icons-material/Info';
import RefreshIcon from '@mui/icons-material/Refresh';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import FlashOnIcon from '@mui/icons-material/FlashOn';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import CheckBoxOutlineBlankIcon from '@mui/icons-material/CheckBoxOutlineBlank';
import Grid from '@mui/material/Grid';
import LinearProgress from '@mui/material/LinearProgress';
import Alert from '@mui/material/Alert';
import type { FlowStepInfo } from '../../types/api';

interface FlowStepDetailsDrawerProps {
  open: boolean;
  step: FlowStepInfo | null;
  onClose: () => void;
}

// Helper functions
const getDependencyPolicyDescription = (policy?: string): string => {
  switch (policy) {
    case 'fail_fast':
      return 'Stops all parallel steps immediately if any dependency fails (default behavior).';
    case 'all_success':
      return 'Waits for all parallel steps to complete. Proceeds only if all dependencies succeed.';
    case 'all_complete':
      return 'Waits for all dependencies to complete, then proceeds regardless of success/failure.';
    case 'any_success':
      return 'Waits for all parallel dependencies to complete. Proceeds if at least one succeeds.';
    default:
      return 'Uses fail_fast behavior - stops immediately on dependency failure.';
  }
};

const getDependencyPolicyIcon = (policy?: string) => {
  switch (policy) {
    case 'fail_fast':
      return <FlashOnIcon />;
    case 'all_success':
      return <CheckCircleIcon />;
    case 'all_complete':
      return <CheckBoxOutlineBlankIcon />;
    case 'any_success':
      return <CheckBoxIcon />;
    default:
      return <FlashOnIcon />;
  }
};

const getDependencyPolicyColor = (policy?: string) => {
  switch (policy) {
    case 'fail_fast':
      return 'error';
    case 'all_success':
      return 'success';
    case 'all_complete':
      return 'info';
    case 'any_success':
      return 'warning';
    default:
      return 'error';
  }
};

const calculateNextRetryDelay = (
  retry?: { max_attempts?: number; delay?: number; backoff?: string; max_delay?: number },
  attemptNumber?: number
): number => {
  if (!retry || !retry.delay || !attemptNumber) return 0;
  
  const baseDelay = retry.delay;
  const maxDelay = retry.max_delay || 300;
  
  switch (retry.backoff) {
    case 'exponential':
      const expDelay = baseDelay * Math.pow(2, attemptNumber - 1);
      return Math.min(expDelay, maxDelay);
    case 'linear':
      const linDelay = baseDelay * attemptNumber;
      return Math.min(linDelay, maxDelay);
    default: // fixed
      return baseDelay;
  }
};

export const FlowStepDetailsDrawer: React.FC<FlowStepDetailsDrawerProps> = ({
  open,
  step,
  onClose,
}) => {
  const theme = useTheme();

  if (!step) return null;

  const getStepIcon = (step: FlowStepInfo) => {
    if (step.active) {
      return (
        <Box
          sx={{
            animation: 'blink 1s infinite',
            '@keyframes blink': {
              '0%, 50%': { opacity: 1 },
              '51%, 100%': { opacity: 0.3 },
            },
          }}
        >
          <PlayArrowIcon color="primary" />
        </Box>
      );
    }
    
    if (step.last_execution && !step.last_error) {
      return <CheckCircleIcon color="success" />;
    }
    if (step.last_error) {
      return <ErrorIcon color="error" />;
    }
    return <PauseIcon color="disabled" />;
  };

  const getStepStatus = (step: FlowStepInfo) => {
    if (step.active) {
      return { label: "Active", color: "primary" as const };
    }
    
    if (step.last_execution && !step.last_error) {
      return { label: "Success", color: "success" as const };
    }
    if (step.last_error) {
      return { label: "Error", color: "error" as const };
    }
    if ((step.execution_count ?? 0) > 0) {
      return { label: "Completed", color: "info" as const };
    }
    return { label: "Pending", color: "default" as const };
  };

  const calculateSuccessRate = (step: FlowStepInfo) => {
    if ((step.execution_count ?? 0) === 0) return 0;
    return Math.round(((step.success_count ?? 0) / (step.execution_count ?? 1)) * 100);
  };

  const status = getStepStatus(step);
  const successRate = calculateSuccessRate(step);

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{
        sx: {
          width: { xs: '100%', sm: 480, md: 560 },
          maxWidth: '100vw',
        },
      }}
    >
      <Box sx={{ p: 3, height: '100%', overflow: 'auto' }}>
        {/* Header */}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
          <Stack direction="row" alignItems="center" spacing={2}>
            {getStepIcon(step)}
            <Box>
              <Typography variant="h5" component="h2">
                {step.name}
              </Typography>
              <Typography variant="subtitle2" color="textSecondary">
                Flow Step Details
              </Typography>
            </Box>
          </Stack>
          <IconButton onClick={onClose} edge="end">
            <CloseIcon />
          </IconButton>
        </Box>

        <Divider sx={{ mb: 3 }} />

        {/* Status and Basic Info */}
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack spacing={2}>
              <Stack direction="row" alignItems="center" justifyContent="space-between">
                <Typography variant="h6" gutterBottom>
                  Status & Overview
                </Typography>
                <Stack direction="row" spacing={1}>
                  <Chip 
                    label={status.label} 
                    color={status.color}
                    size="medium"
                  />
                  <Chip 
                    label={step.type} 
                    variant="outlined" 
                    size="medium"
                  />
                </Stack>
              </Stack>

              {step.enabled !== undefined && (
                <Stack direction="row" alignItems="center" spacing={1}>
                  <Typography variant="body2" component="span">
                    Enabled:
                  </Typography>
                  <Chip 
                    label={step.enabled ? "Yes" : "No"}
                    color={step.enabled ? "success" : "default"}
                    size="small"
                  />
                </Stack>
              )}

              <Stack direction="row" spacing={3} alignItems="center">
                <Box>
                  <Typography variant="body2" color="textSecondary">
                    Executions
                  </Typography>
                  <Typography variant="h6">
                    {step.execution_count ?? 0}
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="body2" color="textSecondary">
                    Success Rate
                  </Typography>
                  <Typography variant="h6" color={successRate >= 80 ? "success.main" : successRate >= 50 ? "warning.main" : "error.main"}>
                    {successRate}%
                  </Typography>
                </Box>
                <Box>
                  <Typography variant="body2" color="textSecondary">
                    Successes
                  </Typography>
                  <Typography variant="h6">
                    {step.success_count ?? 0}
                  </Typography>
                </Box>
              </Stack>
            </Stack>
          </CardContent>
        </Card>

        {/* Dependencies */}
        {step.depends_on && step.depends_on.length > 0 && (
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
                <LinkIcon color="primary" />
                <Typography variant="h6">
                  Dependencies
                </Typography>
              </Stack>
              <List dense>
                {step.depends_on.map((dependency, index) => (
                  <ListItem key={index} sx={{ py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 32 }}>
                      <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'primary.main' }} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={dependency}
                      primaryTypographyProps={{ variant: 'body2' }}
                    />
                  </ListItem>
                ))}
              </List>
            </CardContent>
          </Card>
        )}

        {/* Dependency Policy */}
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
              <AccountTreeIcon color="primary" />
              <Typography variant="h6">
                Dependency Policy
              </Typography>
            </Stack>
            
            <Stack spacing={2}>
              <Box>
                <Chip 
                  label={step.dependency_policy || "fail_fast"}
                  color={getDependencyPolicyColor(step.dependency_policy) as any}
                  icon={getDependencyPolicyIcon(step.dependency_policy)}
                  sx={{ mb: 1 }}
                />
                <Typography variant="body2" color="textSecondary">
                  {getDependencyPolicyDescription(step.dependency_policy)}
                </Typography>
              </Box>
              
              {step.depends_on && step.depends_on.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Required Dependencies
                  </Typography>
                  <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {step.depends_on.map(dep => (
                      <Chip 
                        key={dep}
                        label={dep}
                        size="small"
                        variant="outlined"
                        icon={<LinkIcon sx={{ fontSize: 14 }} />}
                      />
                    ))}
                  </Stack>
                </Box>
              )}
            </Stack>
          </CardContent>
        </Card>

        {/* Retry Configuration */}
        {step.retry && step.retry.max_attempts && step.retry.max_attempts > 1 && (
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
                <RefreshIcon color="primary" />
                <Typography variant="h6">
                  Retry Configuration
                </Typography>
              </Stack>
              
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" gutterBottom>Max Attempts</Typography>
                  <Typography variant="h6">{step.retry.max_attempts}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" gutterBottom>Delay Strategy</Typography>
                  <Chip 
                    label={step.retry.backoff || 'fixed'}
                    color="info"
                    size="small"
                  />
                </Grid>
                <Grid item xs={6}>
                  <Typography variant="subtitle2" gutterBottom>Initial Delay</Typography>
                  <Typography>{step.retry.delay || 0}s</Typography>
                </Grid>
                {step.retry.max_delay && (
                  <Grid item xs={6}>
                    <Typography variant="subtitle2" gutterBottom>Max Delay</Typography>
                    <Typography>{step.retry.max_delay}s</Typography>
                  </Grid>
                )}
                
                {/* Current retry status */}
                {step.retry_attempt !== undefined && step.retry_attempt > 0 && (
                  <Grid item xs={12}>
                    <Typography variant="subtitle2" gutterBottom>
                      Retry Progress
                    </Typography>
                    <LinearProgress 
                      variant="determinate" 
                      value={(step.retry_attempt / step.retry.max_attempts) * 100}
                      color={step.retry_attempt >= step.retry.max_attempts ? "error" : "warning"}
                      sx={{ mb: 1 }}
                    />
                    <Typography variant="caption" color="textSecondary">
                      Attempt {step.retry_attempt} of {step.retry.max_attempts}
                    </Typography>
                  </Grid>
                )}
                
                {step.next_retry_time && (
                  <Grid item xs={12}>
                    <Alert severity="info" variant="outlined">
                      Next retry scheduled at: {new Date(step.next_retry_time).toLocaleString()}
                    </Alert>
                  </Grid>
                )}
              </Grid>
            </CardContent>
          </Card>
        )}

        {/* Configuration */}
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
              <SettingsIcon color="primary" />
              <Typography variant="h6">
                Configuration
              </Typography>
            </Stack>
            
            {step.args && step.args.length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Arguments
                </Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  {step.args.map((arg, index) => (
                    <Chip 
                      key={index}
                      label={arg}
                      variant="outlined"
                      size="small"
                    />
                  ))}
                </Stack>
              </Box>
            )}

            {step.env && Object.keys(step.env).length > 0 && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Environment Variables
                </Typography>
                <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'grey.50' }}>
                  {Object.entries(step.env).map(([key, value]) => (
                    <Typography key={key} variant="body2" fontFamily="monospace" component="div">
                      <Box component="span" sx={{ fontWeight: 'bold', color: 'primary.main' }}>
                        {key}
                      </Box>
                      =
                      <Box component="span" sx={{ color: 'text.secondary' }}>
                        {value}
                      </Box>
                    </Typography>
                  ))}
                </Paper>
              </Box>
            )}

            {step.skip_when && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Skip Condition
                </Typography>
                <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'grey.50' }}>
                  <Typography variant="body2" fontFamily="monospace">
                    {step.skip_when}
                  </Typography>
                </Paper>
              </Box>
            )}
          </CardContent>
        </Card>

        {/* Input/Output Templates */}
        {(step.input || step.output) && (
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
                <CodeIcon color="primary" />
                <Typography variant="h6">
                  Templates
                </Typography>
              </Stack>
              
              {step.input && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" gutterBottom>
                    Input Template
                  </Typography>
                  <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'grey.50' }}>
                    <Typography variant="body2" fontFamily="monospace" sx={{ whiteSpace: 'pre-wrap' }}>
                      {step.input}
                    </Typography>
                  </Paper>
                </Box>
              )}

              {step.output && (
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Output Template
                  </Typography>
                  <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'grey.50' }}>
                    <Typography variant="body2" fontFamily="monospace" sx={{ whiteSpace: 'pre-wrap' }}>
                      {step.output}
                    </Typography>
                  </Paper>
                </Box>
              )}
            </CardContent>
          </Card>
        )}

        {/* Execution History */}
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
              <ScheduleIcon color="primary" />
              <Typography variant="h6">
                Execution History
              </Typography>
            </Stack>
            
            {step.last_execution && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Last Execution
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  {new Date(step.last_execution).toLocaleString()}
                </Typography>
              </Box>
            )}

            {step.last_output && (
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Last Output
                </Typography>
                <Paper variant="outlined" sx={{ p: 1.5, bgcolor: 'success.50' }}>
                  <Typography variant="body2" fontFamily="monospace" sx={{ whiteSpace: 'pre-wrap' }}>
                    {step.last_output}
                  </Typography>
                </Paper>
              </Box>
            )}

            {step.last_error && (
              <Alert severity="error" sx={{ mt: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Last Error
                </Typography>
                <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                  {step.last_error}
                </Typography>
              </Alert>
            )}
          </CardContent>
        </Card>

        {/* Additional Info */}
        <Card>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
              <InfoIcon color="primary" />
              <Typography variant="h6">
                Additional Information
              </Typography>
            </Stack>
            
            <Stack spacing={1}>
              <Typography variant="body2" color="textSecondary">
                This step is part of the worker's flow configuration and executes based on its dependencies and conditions.
              </Typography>
              
              {step.active && (
                <Alert severity="info" variant="outlined">
                  This step is currently active and executing.
                </Alert>
              )}
              
              {!step.enabled && (
                <Alert severity="warning" variant="outlined">
                  This step is disabled and will not execute.
                </Alert>
              )}
            </Stack>
          </CardContent>
        </Card>
      </Box>
    </Drawer>
  );
};