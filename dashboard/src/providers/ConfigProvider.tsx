import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { Box, CircularProgress, Alert, Typography } from '@mui/material';

interface Config {
  apiUrl: string;
  title?: string;
  version?: string;
}

interface ConfigContextType {
  config: Config;
}

const ConfigContext = createContext<ConfigContextType | null>(null);

interface ConfigProviderProps {
  children: ReactNode;
}

export const ConfigProvider: React.FC<ConfigProviderProps> = ({ children }) => {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const response = await fetch('/config.json');
        if (!response.ok) {
          throw new Error(`Failed to load config: ${response.status}`);
        }
        const configData = await response.json();
        
        // Validate required fields
        if (!configData.apiUrl) {
          throw new Error('Config missing required field: apiUrl');
        }
        
        setConfig(configData);
        
        // Set document title if provided
        if (configData.title) {
          document.title = configData.title;
        }
      } catch (err) {
        console.error('Failed to load configuration:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        
        // Fallback configuration for development
        const fallbackConfig = {
          apiUrl: 'http://localhost:9090',
          title: 'AutoTeam Dashboard (Fallback)',
          version: 'dev'
        };
        
        console.warn('Using fallback configuration:', fallbackConfig);
        setConfig(fallbackConfig);
      } finally {
        setLoading(false);
      }
    };

    fetchConfig();
  }, []);

  if (loading) {
    return (
      <Box 
        display="flex" 
        flexDirection="column"
        justifyContent="center" 
        alignItems="center" 
        minHeight="100vh"
        gap={2}
      >
        <CircularProgress size={60} />
        <Typography variant="body1" color="textSecondary">
          Loading dashboard configuration...
        </Typography>
      </Box>
    );
  }

  if (!config) {
    return (
      <Box 
        display="flex" 
        justifyContent="center" 
        alignItems="center" 
        minHeight="100vh"
        p={4}
      >
        <Alert severity="error" sx={{ maxWidth: 600 }}>
          <Typography variant="h6" gutterBottom>
            Configuration Error
          </Typography>
          <Typography variant="body2">
            {error || 'Failed to load dashboard configuration. Please check the server status.'}
          </Typography>
        </Alert>
      </Box>
    );
  }

  return (
    <ConfigContext.Provider value={{ config }}>
      {children}
    </ConfigContext.Provider>
  );
};

export const useConfig = (): ConfigContextType => {
  const context = useContext(ConfigContext);
  if (!context) {
    throw new Error('useConfig must be used within a ConfigProvider');
  }
  return context;
};