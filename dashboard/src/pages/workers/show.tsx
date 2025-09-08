import React, { useState } from "react";
import { Show } from "@refinedev/mui";
import { useShow, useCustom } from "@refinedev/core";
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
import { TabPanel, a11yProps } from "../../components/common";

export const WorkersShow = () => {
  const { id } = useParams();
  const [activeTab, setActiveTab] = useState(0);
  
  const { queryResult } = useShow({
    resource: "workers",
    id: id as string,
  });

  const { data: worker, isLoading, error } = queryResult;

  // Get worker health status
  const { data: healthData, isLoading: healthLoading } = useCustom({
    url: `/workers/${id}/health`,
    method: "get",
    queryOptions: {
      refetchInterval: 5000,
      enabled: !!id,
    },
  });

  // Get worker status details
  const { data: statusData, isLoading: statusLoading } = useCustom({
    url: `/workers/${id}/status`,
    method: "get",
    queryOptions: {
      refetchInterval: 10000,
      enabled: !!id,
    },
  });

  // Get worker configuration
  const { data: configData, isLoading: configLoading } = useCustom({
    url: `/workers/${id}/config`,
    method: "get",
    queryOptions: {
      enabled: !!id,
    },
  });

  // Get worker flow
  const { data: flowData } = useCustom({
    url: `/workers/${id}/flow`,
    method: "get",
    queryOptions: {
      enabled: !!id,
    },
  });

  // Get worker flow steps
  const { data: flowStepsData, isLoading: flowStepsLoading } = useCustom({
    url: `/workers/${id}/flow/steps`,
    method: "get",
    queryOptions: {
      refetchInterval: 5000, // Refresh every 5 seconds
      enabled: !!id,
    },
  });

  // Get worker metrics
  const { data: metricsData, isLoading: metricsLoading } = useCustom({
    url: `/workers/${id}/metrics`,
    method: "get",
    queryOptions: {
      refetchInterval: 30000,
      enabled: !!id,
    },
  });

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
            configData={configData}
            configLoading={configLoading}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <WorkerFlowSteps
            flowStepsData={flowStepsData}
            flowStepsLoading={flowStepsLoading}
          />
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