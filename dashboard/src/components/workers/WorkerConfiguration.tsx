import React, { useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  Typography,
  TextField,
  Stack,
  Box,
  Chip,
  Collapse,
  IconButton,
  List,
  ListItem,
  ListItemText,
  CircularProgress,
  Alert,
} from "@mui/material";
import CodeIcon from "@mui/icons-material/Code";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import ExpandLessIcon from "@mui/icons-material/ExpandLess";

interface WorkerConfigurationProps {
  configData: any;
  configLoading: boolean;
}

export const WorkerConfiguration: React.FC<WorkerConfigurationProps> = ({
  configData,
  configLoading,
}) => {
  const [expandedSections, setExpandedSections] = useState<{[key: string]: boolean}>({});

  const toggleExpanded = (section: string) => {
    setExpandedSections(prev => ({
      ...prev,
      [section]: !prev[section]
    }));
  };

  if (configLoading) {
    return <CircularProgress />;
  }

  return (
    <Stack spacing={2}>
      {/* Agent Configuration */}
      {configData?.data?.agent && (
        <Card>
          <CardHeader 
            title="Agent Configuration" 
            avatar={<CodeIcon />}
            action={
              <IconButton onClick={() => toggleExpanded('agent')}>
                {expandedSections['agent'] ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </IconButton>
            }
          />
          <Collapse in={expandedSections['agent']}>
            <CardContent>
              <Stack spacing={2}>
                <Box>
                  <Typography variant="subtitle2" gutterBottom>Name</Typography>
                  <Chip label={configData.data.agent.name || "Unknown"} />
                </Box>
                <Box>
                  <Typography variant="subtitle2" gutterBottom>Type</Typography>
                  <Chip label={configData.data.agent.type || "Unknown"} color="primary" />
                </Box>
                {configData.data.agent.args && (
                  <Box>
                    <Typography variant="subtitle2" gutterBottom>Arguments</Typography>
                    <List dense>
                      {configData.data.agent.args.map((arg: string, index: number) => (
                        <ListItem key={index}>
                          <ListItemText primary={arg} />
                        </ListItem>
                      ))}
                    </List>
                  </Box>
                )}
              </Stack>
            </CardContent>
          </Collapse>
        </Card>
      )}

      {/* Prompts */}
      {configData?.data?.prompts && (
        <Card>
          <CardHeader 
            title="Prompts" 
            avatar={<CodeIcon />}
            action={
              <IconButton onClick={() => toggleExpanded('prompts')}>
                {expandedSections['prompts'] ? <ExpandLessIcon /> : <ExpandMoreIcon />}
              </IconButton>
            }
          />
          <Collapse in={expandedSections['prompts']}>
            <CardContent>
              <Stack spacing={2}>
                {Object.entries(configData.data.prompts).map(([key, value]) => (
                  <Box key={key}>
                    <Typography variant="subtitle2" gutterBottom>
                      {key.replace(/_/g, ' ').toUpperCase()}
                    </Typography>
                    <TextField
                      multiline
                      rows={4}
                      fullWidth
                      value={value as string || ""}
                      InputProps={{
                        readOnly: true,
                        style: { fontFamily: 'monospace', fontSize: '0.875rem' }
                      }}
                      variant="outlined"
                    />
                  </Box>
                ))}
              </Stack>
            </CardContent>
          </Collapse>
        </Card>
      )}

      {!configData?.data && !configLoading && (
        <Alert severity="info">
          No configuration data available
        </Alert>
      )}
    </Stack>
  );
};