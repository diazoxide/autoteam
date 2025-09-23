import React, { useState } from "react";
import {
  Box,
  Button,
  ButtonGroup,
  CircularProgress,
  Tooltip,
  Alert,
  Snackbar,
} from "@mui/material";
import {
  PlayArrow as DeployIcon,
  Stop as StopIcon,
  RestartAlt as RestartIcon,
  Pause as PauseIcon,
  PlayCircle as UnpauseIcon,
} from "@mui/icons-material";
import { useCustomMutation } from "@refinedev/core";

interface WorkerActionsProps {
  workerId: string;
  status?: string;
  compact?: boolean;
}

export const WorkerActions: React.FC<WorkerActionsProps> = ({
  workerId,
  status,
  compact = false,
}) => {
  const [notification, setNotification] = useState<{
    message: string;
    severity: "success" | "error";
  } | null>(null);

  const { mutate: deployWorker, isLoading: isDeploying } = useCustomMutation();
  const { mutate: stopWorker, isLoading: isStopping } = useCustomMutation();
  const { mutate: restartWorker, isLoading: isRestarting } = useCustomMutation();
  const { mutate: pauseWorker, isLoading: isPausing } = useCustomMutation();
  const { mutate: unpauseWorker, isLoading: isUnpausing } = useCustomMutation();

  const handleAction = (action: string, actionFn: (args: unknown, options?: unknown) => void, successMessage: string) => {
    actionFn({
      url: `/workers/${workerId}/actions/${action}`,
      method: "post",
      values: {},
      successNotification: false,
      errorNotification: false,
    }, {
      onSuccess: () => {
        setNotification({
          message: successMessage,
          severity: "success",
        });
      },
      onError: (error: Error) => {
        setNotification({
          message: `Failed to ${action} worker: ${error?.message || 'Unknown error'}`,
          severity: "error",
        });
      },
    });
  };

  const isLoading = isDeploying || isStopping || isRestarting || isPausing || isUnpausing;

  const actions = [
    {
      key: "deploy",
      label: "Deploy",
      icon: <DeployIcon />,
      color: "primary" as const,
      action: () => handleAction("deploy", deployWorker, "Worker deployment started"),
      loading: isDeploying,
      disabled: status === "running",
    },
    {
      key: "stop",
      label: "Stop",
      icon: <StopIcon />,
      color: "error" as const,
      action: () => handleAction("stop", stopWorker, "Worker stopped"),
      loading: isStopping,
      disabled: status === "not_deployed",
    },
    {
      key: "restart",
      label: "Restart",
      icon: <RestartIcon />,
      color: "warning" as const,
      action: () => handleAction("restart", restartWorker, "Worker restarted"),
      loading: isRestarting,
      disabled: status === "not_deployed",
    },
    {
      key: "pause",
      label: "Pause",
      icon: <PauseIcon />,
      color: "secondary" as const,
      action: () => handleAction("pause", pauseWorker, "Worker paused"),
      loading: isPausing,
      disabled: status !== "running",
    },
    {
      key: "unpause",
      label: "Unpause",
      icon: <UnpauseIcon />,
      color: "success" as const,
      action: () => handleAction("unpause", unpauseWorker, "Worker unpaused"),
      loading: isUnpausing,
      disabled: status !== "paused",
    },
  ];

  if (compact) {
    return (
      <Box>
        <ButtonGroup size="small" variant="outlined">
          {actions.slice(0, 3).map((actionItem) => (
            <Tooltip key={actionItem.key} title={actionItem.label}>
              <span>
                <Button
                  onClick={actionItem.action}
                  disabled={actionItem.disabled || isLoading}
                  color={actionItem.color}
                  size="small"
                  sx={{ minWidth: 'auto', px: 1 }}
                >
                  {actionItem.loading ? (
                    <CircularProgress size={16} />
                  ) : (
                    actionItem.icon
                  )}
                </Button>
              </span>
            </Tooltip>
          ))}
        </ButtonGroup>

        <Snackbar
          open={!!notification}
          autoHideDuration={4000}
          onClose={() => setNotification(null)}
        >
          <Alert
            onClose={() => setNotification(null)}
            severity={notification?.severity}
            sx={{ width: '100%' }}
          >
            {notification?.message}
          </Alert>
        </Snackbar>
      </Box>
    );
  }

  return (
    <Box>
      <ButtonGroup orientation="vertical" variant="outlined" fullWidth>
        {actions.map((actionItem) => (
          <Button
            key={actionItem.key}
            onClick={actionItem.action}
            disabled={actionItem.disabled || isLoading}
            color={actionItem.color}
            startIcon={
              actionItem.loading ? (
                <CircularProgress size={16} />
              ) : (
                actionItem.icon
              )
            }
            sx={{ justifyContent: 'flex-start' }}
          >
            {actionItem.label}
          </Button>
        ))}
      </ButtonGroup>

      <Snackbar
        open={!!notification}
        autoHideDuration={4000}
        onClose={() => setNotification(null)}
      >
        <Alert
          onClose={() => setNotification(null)}
          severity={notification?.severity}
          sx={{ width: '100%' }}
        >
          {notification?.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};