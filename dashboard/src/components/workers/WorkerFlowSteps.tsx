import React, { useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  Badge,
  Tooltip,
  Chip,
  Stack,
  Box,
  Typography,
  CircularProgress,
  Alert,
  LinearProgress,
  ToggleButton,
  ToggleButtonGroup,
} from "@mui/material";
import FlowIcon from "@mui/icons-material/AccountTree";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import PauseIcon from "@mui/icons-material/Pause";
import ListIcon from "@mui/icons-material/List";
import AccountTreeIcon from "@mui/icons-material/AccountTree";
import { FlowTreeVisualization } from "./FlowTreeVisualization";

interface WorkerFlowStepsProps {
  flowStepsData: any;
  flowStepsLoading: boolean;
}

export const WorkerFlowSteps: React.FC<WorkerFlowStepsProps> = ({
  flowStepsData,
  flowStepsLoading,
}) => {
  const [viewType, setViewType] = useState<'list' | 'tree'>('list');
  const getStepIcon = (step: any) => {
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
    
    // Use new API field if available
    if (step.last_execution_success === true) {
      return <CheckCircleIcon color="success" />;
    }
    if (step.last_execution_success === false) {
      return <ErrorIcon color="error" />;
    }
    
    // Fallback to old logic if new field is not available
    // Check for explicit success (last execution with no error)
    if (step.last_execution && !step.last_error) {
      return <CheckCircleIcon color="success" />;
    }
    if (step.last_error) {
      return <ErrorIcon color="error" />;
    }
    return <PauseIcon color="disabled" />;
  };

  const getStepStatus = (step: any) => {
    if (step.active) {
      return { label: "Active", color: "primary" as const };
    }
    
    // Use new API field if available
    if (step.last_execution_success === true) {
      return { label: "Success", color: "success" as const };
    }
    if (step.last_execution_success === false) {
      return { label: "Error", color: "error" as const };
    }
    
    // Fallback to old logic if new field is not available
    // Check for explicit success (last execution with no error)
    if (step.last_execution && !step.last_error) {
      return { label: "Success", color: "success" as const };
    }
    if (step.last_error) {
      return { label: "Error", color: "error" as const };
    }
    if (step.execution_count > 0) {
      return { label: "Completed", color: "info" as const };
    }
    return { label: "Pending", color: "default" as const };
  };

  const calculateSuccessRate = (step: any) => {
    if (step.execution_count === 0) return 0;
    return Math.round((step.success_count / step.execution_count) * 100);
  };

  if (flowStepsLoading) {
    return <CircularProgress />;
  }

  if (!flowStepsData?.data?.steps || flowStepsData.data.steps.length === 0) {
    return (
      <Alert severity="info">
        No flow steps configured
      </Alert>
    );
  }

  const handleViewChange = (event: React.MouseEvent<HTMLElement>, newView: 'list' | 'tree') => {
    if (newView !== null) {
      setViewType(newView);
    }
  };

  const renderListView = () => (
    <List>
      {flowStepsData.data.steps.map((step: any, index: number) => {
        const status = getStepStatus(step);
        const successRate = calculateSuccessRate(step);

        return (
          <React.Fragment key={step.name || index}>
            <ListItem>
              <ListItemIcon>
                <Badge badgeContent={index + 1} color="primary">
                  {getStepIcon(step)}
                </Badge>
              </ListItemIcon>
              <ListItemText
                primary={
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="h6">{step.name}</Typography>
                    <Chip 
                      label={status.label} 
                      color={status.color}
                      size="small" 
                    />
                    <Chip 
                      label={step.type} 
                      variant="outlined" 
                      size="small" 
                    />
                  </Stack>
                }
                primaryTypographyProps={{ component: 'div' }}
                secondary={
                  <Stack spacing={1} sx={{ mt: 1 }}>
                    {step.depends_on && step.depends_on.length > 0 && (
                      <Box>
                        <Typography variant="body2" component="span">
                          Depends on: {step.depends_on.join(", ")}
                        </Typography>
                      </Box>
                    )}
                    <Stack direction="row" spacing={2} alignItems="center">
                      <Typography variant="body2" component="span">
                        Executions: {step.execution_count}
                      </Typography>
                      <Typography variant="body2" component="span">
                        Success Rate: {successRate}%
                      </Typography>
                    </Stack>
                    <Box sx={{ width: '100%', maxWidth: 200 }}>
                      <LinearProgress
                        variant="determinate"
                        value={successRate}
                        color={successRate >= 80 ? "success" : successRate >= 50 ? "warning" : "error"}
                      />
                    </Box>
                    {step.last_error && (
                      <Tooltip title={step.last_error} arrow>
                        <Typography variant="body2" color="error" noWrap component="div">
                          Last Error: {step.last_error}
                        </Typography>
                      </Tooltip>
                    )}
                    {step.last_execution && (
                      <Typography variant="body2" color="textSecondary" component="div">
                        Last Execution: {new Date(step.last_execution).toLocaleString()}
                      </Typography>
                    )}
                  </Stack>
                }
                secondaryTypographyProps={{ component: 'div' }}
              />
            </ListItem>
            {index < flowStepsData.data.steps.length - 1 && <Divider />}
          </React.Fragment>
        );
      })}
    </List>
  );

  const renderTreeView = () => (
    <FlowTreeVisualization steps={flowStepsData.data.steps} />
  );

  return (
    <Card>
      <CardHeader 
        title="Flow Steps" 
        avatar={<FlowIcon />}
        action={
          <ToggleButtonGroup
            value={viewType}
            exclusive
            onChange={handleViewChange}
            size="small"
            aria-label="view type"
          >
            <ToggleButton value="list" aria-label="list view">
              <Tooltip title="List View">
                <ListIcon />
              </Tooltip>
            </ToggleButton>
            <ToggleButton value="tree" aria-label="tree view">
              <Tooltip title="Tree View">
                <AccountTreeIcon />
              </Tooltip>
            </ToggleButton>
          </ToggleButtonGroup>
        }
      />
      <CardContent>
        {viewType === 'list' ? renderListView() : renderTreeView()}
      </CardContent>
    </Card>
  );
};