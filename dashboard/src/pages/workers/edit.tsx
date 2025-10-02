import React from "react";
import { Edit } from "@refinedev/mui";
import { useOne } from "@refinedev/core";
import {
  Box,
  Tabs,
  Tab,
  Typography,
  Alert,
} from "@mui/material";
import { WorkerBasicSettingsForm } from "../../components/workers/WorkerBasicSettingsForm";
import { WorkerFlowForm } from "../../components/workers/WorkerFlowForm";
import { useWorkerSettings } from "../../hooks/api/useWorkerApi";
import { useParams, useSearchParams } from "react-router-dom";

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
  const [searchParams, setSearchParams] = useSearchParams();
  const [tabValue, setTabValue] = React.useState(() => {
    // Check if we should open flow tab (from create page redirect)
    return searchParams.get('tab') === 'flow' ? 1 : 0;
  });
  const [refreshKey, setRefreshKey] = React.useState(0);
  const [showSuccessMessage, setShowSuccessMessage] = React.useState(() => {
    // Show success message if coming from create page
    return searchParams.get('tab') === 'flow';
  });
  const { id } = useParams();

  // Clear the tab query parameter after reading it
  React.useEffect(() => {
    if (searchParams.get('tab') === 'flow') {
      setSearchParams({}, { replace: true });
    }
  }, [searchParams, setSearchParams]);

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
    const settings = workerSettings?.settings as { flow?: unknown[] } | undefined;
    if (!settings?.flow) return [];

    return settings.flow.map((step: unknown) => {
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
  }, [workerSettings?.settings]);

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
          {showSuccessMessage && (
            <Alert
              severity="success"
              sx={{ mb: 3 }}
              onClose={() => setShowSuccessMessage(false)}
            >
              Worker created successfully! Now you can configure the flow steps.
            </Alert>
          )}
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
