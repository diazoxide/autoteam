import React from "react";
import {
  useDataGrid,
  ShowButton,
  List,
} from "@refinedev/mui";
import { useCustom } from "@refinedev/core";
import { DataGrid, GridColDef } from "@mui/x-data-grid";
import { Chip, Box, Typography } from "@mui/material";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";

// Removed unused interface - using inline types as needed

export const WorkersList = () => {
  const { dataGridProps } = useDataGrid({
    syncWithLocation: true,
  });

  // Get workers health status
  const { data: healthData } = useCustom({
    url: "/health",
    method: "get",
    queryOptions: {
      refetchInterval: 5000, // Refresh every 5 seconds
    },
  });

  const columns: GridColDef[] = [
    {
      field: "id",
      headerName: "Worker ID",
      type: "string",
      minWidth: 150,
    },
    {
      field: "url",
      headerName: "API URL",
      minWidth: 200,
      flex: 1,
    },
    {
      field: "last_check",
      headerName: "Last Check",
      minWidth: 180,
      renderCell: ({ row }) => {
        if (row.last_check) {
          const lastCheck = new Date(row.last_check);
          return (
            <>
              {lastCheck.toLocaleString()}
            </>
          );
        }

        return (
          <>
            Never
          </>
        );
      },
    },
    {
      field: "health_status",
      headerName: "Health Status",
      minWidth: 150,
      renderCell: ({ row }) => {
        const workersHealth = healthData?.data?.workers_health || {};
        const workerStatus = workersHealth[row.id];

        if (workerStatus) {
          return (
            <Chip
              icon={workerStatus === "reachable" ? <CheckCircleIcon /> : <ErrorIcon />}
              label={workerStatus === "reachable" ? "Healthy" : "Unhealthy"}
              color={workerStatus === "reachable" ? "success" : "error"}
              size="small"
            />
          );
        }

        return (
          <Chip
            label="Unknown"
            color="default"
            size="small"
          />
        );
      },
    },
    {
      field: "actions",
      headerName: "Actions",
      sortable: false,
      renderCell: ({ row }) => (
        <Box>
          <ShowButton hideText recordItemId={row.id} />
        </Box>
      ),
      align: "center",
      headerAlign: "center",
      minWidth: 80,
    },
  ];

  return (
    <List>
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
