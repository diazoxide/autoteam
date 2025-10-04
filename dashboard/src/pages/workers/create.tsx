import React from "react";
import { Create } from "@refinedev/mui";
import { useNavigate } from "react-router-dom";
import {
  Box,
  Tabs,
  Tab,
  Typography,
  Alert,
} from "@mui/material";
import { WorkerBasicSettingsForm } from "../../components/workers/WorkerBasicSettingsForm";
import { WorkerFlowForm } from "../../components/workers/WorkerFlowForm";

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

export const WorkersCreate = () => {
  const [tabValue, setTabValue] = React.useState(0);
  const [createdWorkerId, setCreatedWorkerId] = React.useState<string | null>(null);
  const [showFlowTab, setShowFlowTab] = React.useState(false);
  const navigate = useNavigate();

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleBasicSettingsSuccess = (data?: unknown) => {
    console.log("Worker created successfully:", data);
    // Extract worker ID from the response to enable flow configuration
    const workerId = (data as { data?: { id?: string }; id?: string })?.data?.id || (data as { id?: string })?.id;
    if (workerId) {
      console.log("Navigating to edit page for worker:", workerId);
      // Navigate to edit page to configure flow
      navigate(`/workers/edit/${workerId}?tab=flow`, { replace: true });
    }
  };

  const handleFlowSuccess = () => {
    console.log("Flow configuration saved successfully");
    // Navigate to edit page after flow is configured
    if (createdWorkerId) {
      navigate(`/workers/edit/${createdWorkerId}`);
    }
  };

  const handleError = (error: unknown) => {
    console.error("Save error:", error);
  };

  return (
    <Create isLoading={false} saveButtonProps={{ style: { display: 'none' } }}>
      <Box sx={{ width: "100%" }}>
        <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
          <Tabs value={tabValue} onChange={handleTabChange}>
            <Tab label="Basic Settings" />
            <Tab
              label="Flow Configuration"
              disabled={!showFlowTab}
            />
          </Tabs>
        </Box>

        {!showFlowTab && (
          <Alert severity="info" sx={{ m: 3 }}>
            Create the basic worker settings first, then configure the flow.
          </Alert>
        )}

        <TabPanel value={tabValue} index={0}>
          <WorkerBasicSettingsForm
            initialData={{ name: "", prompt: "", enabled: true }}
            onSuccess={handleBasicSettingsSuccess}
            onError={handleError}
          />
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          {showFlowTab ? (
            <Box>
              <Alert severity="success" sx={{ mb: 3 }}>
                Worker created successfully! Now you can configure the flow steps.
              </Alert>
              <WorkerFlowForm
                workerId={createdWorkerId!}
                initialFlowData={[]}
                onSuccess={handleFlowSuccess}
                onError={handleError}
              />
            </Box>
          ) : (
            <Typography variant="body2" color="text.secondary">
              Please create the basic worker settings first.
            </Typography>
          )}
        </TabPanel>
      </Box>
    </Create>
  );
};
