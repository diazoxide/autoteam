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
      field: "status",
      headerName: "Health Status",
      minWidth: 120,
      renderCell: ({ row }) => {
        const isReachable = row.status === "reachable";

        return (
          <Chip
            icon={isReachable ? <CheckCircleIcon /> : <ErrorIcon />}
            label={isReachable ? "Reachable" : "Unreachable"}
            color={isReachable ? "success" : "error"}
            variant="outlined"
          />
        );
      },
    },
    {
      field: "last_seen",
      headerName: "Last Seen",
      minWidth: 180,
      renderCell: ({ row }) => {
        const workerHealth = healthData?.data?.workers_health?.find(
          (w: any) => w.worker_id === row.id
        );

        if (workerHealth?.last_check) {
          const lastSeen = new Date(workerHealth.last_check);
          return (
            <Typography variant="body2">
              {lastSeen.toLocaleString()}
            </Typography>
          );
        }

        return <Typography variant="body2" color="textSecondary">Never</Typography>;
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
