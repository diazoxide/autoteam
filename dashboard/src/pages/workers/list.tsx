import React from "react";
import {
  useDataGrid,
  ShowButton,
  EditButton,
  DeleteButton,
  List,
} from "@refinedev/mui";
import { useCustom } from "@refinedev/core";
import { DataGrid, GridColDef } from "@mui/x-data-grid";
import { Chip, Box, CircularProgress } from "@mui/material";
import {
  PlayCircle as RunningIcon,
  Stop as StoppedIcon,
  CloudOff as NotDeployedIcon,
} from "@mui/icons-material";
import { WorkerActions } from "../../components/workers/WorkerActions";
import { useWorkerRuntime } from "../../hooks/api/useWorkerApi";

// Component to show deployment status for a single worker
const WorkerStatusChip: React.FC<{ workerId: string }> = ({ workerId }) => {
  const { data: runtimeData, isLoading } = useWorkerRuntime(workerId, {
    refetchInterval: 10000,
    enabled: true,
  });

  if (isLoading) {
    return (
      <Chip
        icon={<CircularProgress size={14} />}
        label="Checking..."
        size="small"
        variant="outlined"
      />
    );
  }

  const status = runtimeData?.worker?.status || "unknown";

  const getStatusConfig = (status: string) => {
    switch (status) {
      case "reachable":
        return {
          icon: <RunningIcon />,
          label: "Deployed",
          color: "success" as const,
        };
      case "unreachable":
        return {
          icon: <StoppedIcon />,
          label: "Unreachable",
          color: "error" as const,
        };
      case "unknown":
      default:
        return {
          icon: <NotDeployedIcon />,
          label: "Not Deployed",
          color: "default" as const,
        };
    }
  };

  const statusConfig = getStatusConfig(status);

  return (
    <Chip
      icon={statusConfig.icon}
      label={statusConfig.label}
      color={statusConfig.color}
      size="small"
    />
  );
};

export const WorkersList = () => {
  const { dataGridProps } = useDataGrid({
    syncWithLocation: true,
  });

  // Get workers health status for monitoring
  useCustom({
    url: "/health",
    method: "get",
    queryOptions: {
      refetchInterval: 5000, // Refresh every 5 seconds
    },
  });

  const columns: GridColDef[] = [
    {
      field: "name",
      headerName: "Worker Name",
      type: "string",
      minWidth: 100,
      flex: 1,
    },
    {
      field: "enabled",
      headerName: "Enabled",
      minWidth: 100,
      renderCell: ({ row }) => (
        <Chip
          label={row.enabled ? "Yes" : "No"}
          color={row.enabled ? "success" : "default"}
          size="small"
          variant="outlined"
        />
      ),
    },
    {
      field: "deployment_status",
      headerName: "Deployment Status",
      minWidth: 150,
      renderCell: ({ row }) => <WorkerStatusChip workerId={row.id} />,
      sortable: false,
    },
    {
      field: "settings",
      headerName: "Flow Steps",
      minWidth: 120,
      renderCell: ({ row }) => {
        // Check for flow steps in settings
        const flowStepsCount = row.settings?.flow?.length || 0;

        // Add some defensive checking and fallback display
        const hasSettings = !!row.settings;
        const hasFlow = !!row.settings?.flow;

        return (
          <Chip
            label={
              hasSettings
                ? hasFlow
                  ? `${flowStepsCount} steps`
                  : "No flow"
                : "No settings"
            }
            color={flowStepsCount > 0 ? "primary" : "default"}
            size="small"
            variant="outlined"
          />
        );
      },
    },
    {
      field: "created_at",
      headerName: "Created",
      minWidth: 180,
      renderCell: ({ row }) => {
        if (row.created_at) {
          const createdAt = new Date(row.created_at);
          return <>{createdAt.toLocaleString()}</>;
        }

        return <>Unknown</>;
      },
    },
    {
      field: "updated_at",
      headerName: "Last Updated",
      minWidth: 180,
      renderCell: ({ row }) => {
        if (row.updated_at) {
          const updatedAt = new Date(row.updated_at);
          return <>{updatedAt.toLocaleString()}</>;
        }

        return <>Unknown</>;
      },
    },
    {
      field: "actions",
      headerName: "Actions",
      sortable: false,
      renderCell: ({ row }) => (
        <Box sx={{ display: "flex", gap: 1, alignItems: "center" }}>
          <WorkerActions workerId={row.id} compact={true} />
          <ShowButton hideText recordItemId={row.id} />
          <EditButton hideText recordItemId={row.id} />
          <DeleteButton hideText recordItemId={row.id} />
        </Box>
      ),
      align: "center",
      headerAlign: "center",
      flex: 1,
      minWidth: 200,
    },
  ];

  return (
    <List createButtonProps={{ variant: "contained" }}>
      <DataGrid
        {...dataGridProps}
        columns={columns}
        autoHeight
        pageSizeOptions={[10, 20, 50]}
        disableRowSelectionOnClick
      />
    </List>
  );
};
