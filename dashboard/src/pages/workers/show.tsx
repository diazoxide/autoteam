import React, { useState } from "react";
import { Show } from "@refinedev/mui";
import { useShow } from "@refinedev/core";
import { Stack, Tabs, Tab, Box, CircularProgress, Alert } from "@mui/material";
import InfoIcon from "@mui/icons-material/Info";
import FlowIcon from "@mui/icons-material/AccountTree";
import MetricsIcon from "@mui/icons-material/Analytics";
import LogsIcon from "@mui/icons-material/Description";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import { useParams } from "react-router";

// Import modular components
import {
  WorkerOverview,
  WorkerFlowSteps,
  WorkerMetrics,
  WorkerLogs,
} from "../../components/workers";
import { WorkerActions } from "../../components/workers/WorkerActions";
import { TabPanel } from "../../components/common";
import { a11yProps } from "../../utils/tabUtils";
import {
  useWorkerRuntimeHealth,
  useWorkerRuntimeStatus,
  useWorkerRuntimeFlow,
  useWorkerRuntimeFlowSteps,
  useWorkerRuntimeMetrics,
  useWorkerRuntime,
} from "../../hooks/api/useWorkerApi";

export const WorkersShow = () => {
  const { id } = useParams();
  const [activeTab, setActiveTab] = useState(0);

  const { queryResult } = useShow({
    resource: "workers",
    id: id as string,
  });

  const { data: worker, isLoading, error } = queryResult;

  // Get worker runtime info (registry data)
  const { data: runtimeData, isLoading: runtimeLoading } = useWorkerRuntime(id);

  // Determine if worker is deployed based on runtime data
  const isWorkerDeployed = runtimeData?.worker?.status === "reachable";

  // Get worker health status - only if deployed
  const { data: healthData, isLoading: healthLoading } = useWorkerRuntimeHealth(
    id,
    { enabled: isWorkerDeployed }
  );

  // Get worker status details - only if deployed
  const { data: statusData, isLoading: statusLoading } = useWorkerRuntimeStatus(
    id,
    { enabled: isWorkerDeployed }
  );

  // Get worker flow - only if deployed
  const { data: flowData } = useWorkerRuntimeFlow(id, {
    enabled: isWorkerDeployed,
  });

  // Get worker flow steps - only if deployed
  const { data: flowStepsData } = useWorkerRuntimeFlowSteps(id, {
    enabled: isWorkerDeployed,
  });

  // Get worker metrics - only if deployed
  const { data: metricsData, isLoading: metricsLoading } =
    useWorkerRuntimeMetrics(id, { enabled: isWorkerDeployed });

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  if (isLoading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="400px"
      >
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        Failed to load worker details: {error.message}
      </Alert>
    );
  }

  return (
    <Show isLoading={isLoading}>
      <Stack spacing={3}>
        {/* Tabs Navigation */}
        <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            aria-label="worker details tabs"
            variant="scrollable"
            scrollButtons="auto"
          >
            <Tab icon={<InfoIcon />} label="Overview" {...a11yProps(0)} />
            <Tab icon={<FlowIcon />} label="Flow" {...a11yProps(1)} />
            <Tab icon={<MetricsIcon />} label="Metrics" {...a11yProps(2)} />
            <Tab icon={<LogsIcon />} label="Logs" {...a11yProps(3)} />
            <Tab icon={<PlayArrowIcon />} label="Actions" {...a11yProps(4)} />
          </Tabs>
        </Box>

        {/* Deployment Status Alert */}
        {!isWorkerDeployed && (
          <Alert severity="info" sx={{ mb: 2 }}>
            This worker is not currently deployed. Deploy it using the Actions
            tab to view runtime information such as metrics, logs, and flow
            execution status.
          </Alert>
        )}

        {/* Tab Panels */}
        <TabPanel value={activeTab} index={0}>
          <WorkerOverview
            worker={worker as any}
            runtimeData={runtimeData}
            healthData={healthData}
            statusData={statusData}
            flowStepsData={flowStepsData}
            metricsData={metricsData}
            runtimeLoading={runtimeLoading}
            healthLoading={healthLoading}
            statusLoading={statusLoading}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <WorkerFlowSteps workerId={id as string} />
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <WorkerMetrics
            metricsData={metricsData}
            metricsLoading={metricsLoading}
            flowData={flowData}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={3}>
          <WorkerLogs workerId={id as string} />
        </TabPanel>

        <TabPanel value={activeTab} index={4}>
          <Box sx={{ maxWidth: 400 }}>
            <WorkerActions
              workerId={id as string}
              status={statusData?.status}
              compact={false}
            />
          </Box>
        </TabPanel>
      </Stack>
    </Show>
  );
};
