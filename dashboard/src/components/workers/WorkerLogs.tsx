import React, { useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  TextField,
  Stack,
  Box,
  Typography,
  Switch,
  FormControlLabel,
  IconButton,
  Tooltip,
  Alert,
} from "@mui/material";
import LogsIcon from "@mui/icons-material/Description";
import RefreshIcon from "@mui/icons-material/Refresh";
import ClearIcon from "@mui/icons-material/Clear";

interface WorkerLogsProps {
  workerId: string;
}

export const WorkerLogs: React.FC<WorkerLogsProps> = ({ workerId }) => {
  const [logs, setLogs] = useState("");
  const [autoRefresh, setAutoRefresh] = useState(false);

  const handleRefresh = () => {
    // TODO: Implement log fetching from API
    setLogs("Log fetching not yet implemented...");
  };

  const handleClear = () => {
    setLogs("");
  };

  return (
    <Stack spacing={2}>
      <Card>
        <CardHeader 
          title="Worker Logs"
          avatar={<LogsIcon />}
          action={
            <Stack direction="row" spacing={1} alignItems="center">
              <FormControlLabel
                control={
                  <Switch
                    checked={autoRefresh}
                    onChange={(e) => setAutoRefresh(e.target.checked)}
                    size="small"
                  />
                }
                label="Auto Refresh"
              />
              <Tooltip title="Refresh logs">
                <IconButton onClick={handleRefresh}>
                  <RefreshIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title="Clear logs">
                <IconButton onClick={handleClear}>
                  <ClearIcon />
                </IconButton>
              </Tooltip>
            </Stack>
          }
        />
        <CardContent>
          <Stack spacing={2}>
            <Alert severity="info">
              Log viewing functionality will be implemented in a future update.
            </Alert>
            
            <TextField
              multiline
              rows={15}
              fullWidth
              value={logs}
              placeholder="Worker logs will appear here..."
              InputProps={{
                readOnly: true,
                style: { 
                  fontFamily: 'monospace', 
                  fontSize: '0.875rem',
                  backgroundColor: '#f5f5f5'
                }
              }}
              variant="outlined"
            />
            
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography variant="body2" color="textSecondary">
                Worker: {workerId}
              </Typography>
              <Typography variant="body2" color="textSecondary">
                Last updated: {new Date().toLocaleTimeString()}
              </Typography>
            </Box>
          </Stack>
        </CardContent>
      </Card>
    </Stack>
  );
};