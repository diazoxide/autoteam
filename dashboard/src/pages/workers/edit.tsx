import React from "react";
import { Edit } from "@refinedev/mui";
import { useOne } from "@refinedev/core";
import {
  Box,
  Tabs,
  Tab,
  Typography,
} from "@mui/material";
import { WorkerBasicSettingsForm } from "../../components/workers/WorkerBasicSettingsForm";
import { WorkerFlowForm } from "../../components/workers/WorkerFlowForm";
import { useWorkerSettings } from "../../hooks/api/useWorkerApi";
import { useParams } from "react-router";

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`worker-tabpanel-${index}`}
      aria-labelledby={`worker-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
  );
}

export const WorkersEdit = () => {
  const [tabValue, setTabValue] = React.useState(0);
  const [refreshKey, setRefreshKey] = React.useState(0);
  const { id } = useParams();

  // Fetch worker basic data
  const { data: workerData, isLoading: workerLoading, refetch: refetchWorker } = useOne({
    resource: "workers",
    id: id!,
  });

  // Fetch worker settings (including flow steps) from settings endpoint
  const { data: workerSettings, isLoading: settingsLoading, refetch: refetchSettings } =
    useWorkerSettings(id);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleBasicSettingsSuccess = () => {
    console.log("Basic settings saved successfully");
    refetchWorker();
    setRefreshKey(prev => prev + 1);
  };

  const handleFlowSuccess = () => {
    console.log("Flow configuration saved successfully");
    refetchSettings();
    setRefreshKey(prev => prev + 1);
  };

  const handleError = (error: unknown) => {
    console.error("Save error:", error);
  };

  // Prepare data for forms
  const basicSettingsData = workerData?.data ? {
    name: workerData.data.name,
    prompt: workerData.data.prompt,
    enabled: workerData.data.enabled,
  } : undefined;

  // Transform flow steps from API format to form format
  const flowData = React.useMemo(() => {
    if (!workerSettings?.settings?.flow) return [];

    return workerSettings.settings.flow.map((step: unknown) => {
      const stepData = step as { env?: unknown; [key: string]: unknown };
      // Transform env from API object format {KEY: "value"} to form array format [{key: "KEY", value: "value"}]
      let envArray: Array<{ key: string; value: string }> = [];
      if (
        stepData.env &&
        typeof stepData.env === "object" &&
        !Array.isArray(stepData.env)
      ) {
        envArray = Object.entries(stepData.env as Record<string, unknown>).map(([key, value]) => ({
          key,
          value: String(value || ""),
        }));
      }

      return {
        ...stepData,
        env: envArray,
      };
    });
  }, [workerSettings?.settings?.flow]);

  if (workerLoading || settingsLoading) {
    return (
      <Edit isLoading={true}>
        <Box sx={{ width: "100%" }}>
          <Typography>Loading worker data...</Typography>
        </Box>
      </Edit>
    );
  }

  return (
    <Edit isLoading={false} saveButtonProps={{ style: { display: 'none' } }}>
      <Box sx={{ width: "100%" }}>
        <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
          <Tabs value={tabValue} onChange={handleTabChange}>
            <Tab label="Basic Settings" />
            <Tab label="Flow Configuration" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
          <WorkerBasicSettingsForm
            key={`basic-${refreshKey}`}
            workerId={id}
            initialData={basicSettingsData}
            onSuccess={handleBasicSettingsSuccess}
            onError={handleError}
          />
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <WorkerFlowForm
            key={`flow-${refreshKey}`}
            workerId={id}
            initialFlowData={flowData}
            onSuccess={handleFlowSuccess}
            onError={handleError}
          />
        </TabPanel>
      </Box>
    </Edit>
  );
};
