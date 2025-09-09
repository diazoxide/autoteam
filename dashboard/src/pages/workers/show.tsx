import React, { useState } from "react";
import { Show } from "@refinedev/mui";
import { useShow } from "@refinedev/core";
import {
  Stack,
  Tabs,
  Tab,
  Box,
  CircularProgress,
  Alert,
} from "@mui/material";
import InfoIcon from "@mui/icons-material/Info";
import SettingsIcon from "@mui/icons-material/Settings";
import FlowIcon from "@mui/icons-material/AccountTree";
import MetricsIcon from "@mui/icons-material/Analytics";
import LogsIcon from "@mui/icons-material/Description";
import { useParams } from "react-router";

// Import modular components
import {
  WorkerOverview,
  WorkerConfiguration,
  WorkerFlowSteps,
  WorkerMetrics,
  WorkerLogs,
} from "../../components/workers";
import { TabPanel } from "../../components/common";
import { a11yProps } from "../../utils/tabUtils";
import {
  useWorkerHealth,
  useWorkerStatus,
  useWorkerConfig,
  useWorkerFlow,
  useWorkerFlowSteps,
  useWorkerMetrics,
} from "../../hooks/api/useWorkerApi";

export const WorkersShow = () => {
  const { id } = useParams();
  const [activeTab, setActiveTab] = useState(0);
  
  const { queryResult } = useShow({
    resource: "workers",
    id: id as string,
  });

  const { data: worker, isLoading, error } = queryResult;

  // Get worker health status
  const { data: healthData, isLoading: healthLoading } = useWorkerHealth(id);

  // Get worker status details
  const { data: statusData, isLoading: statusLoading } = useWorkerStatus(id);

  // Get worker configuration
  const { data: configData, isLoading: configLoading } = useWorkerConfig(id);

  // Get worker flow
  const { data: flowData } = useWorkerFlow(id);

  // Get worker flow steps
  const { data: flowStepsData } = useWorkerFlowSteps(id);

  // Get worker metrics
  const { data: metricsData, isLoading: metricsLoading } = useWorkerMetrics(id);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
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
            <Tab icon={<SettingsIcon />} label="Configuration" {...a11yProps(1)} />
            <Tab icon={<FlowIcon />} label="Flow" {...a11yProps(2)} />
            <Tab icon={<MetricsIcon />} label="Metrics" {...a11yProps(3)} />
            <Tab icon={<LogsIcon />} label="Logs" {...a11yProps(4)} />
          </Tabs>
        </Box>

        {/* Tab Panels */}
        <TabPanel value={activeTab} index={0}>
          <WorkerOverview
            worker={worker}
            healthData={healthData}
            statusData={statusData}
            flowStepsData={flowStepsData}
            metricsData={metricsData}
            healthLoading={healthLoading}
            statusLoading={statusLoading}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <WorkerConfiguration
            configData={{ data: configData }}
            configLoading={configLoading}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <WorkerFlowSteps workerId={id as string} />
        </TabPanel>

        <TabPanel value={activeTab} index={3}>
          <WorkerMetrics
            metricsData={metricsData}
            metricsLoading={metricsLoading}
            flowData={flowData}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={4}>
          <WorkerLogs workerId={id as string} />
        </TabPanel>
      </Stack>
    </Show>
  );
};