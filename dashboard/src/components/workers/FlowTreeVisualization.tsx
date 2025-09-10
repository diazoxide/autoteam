import React, { useMemo } from 'react';
import {
  ReactFlow,
  Node,
  Edge,
  Panel,
  MiniMap,
  Controls,
  Background,
  BackgroundVariant,
  MarkerType,
  Handle,
  Position,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { 
  Box, 
  Paper, 
  Typography,
  Chip,
  Stack,
  useTheme
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import PauseIcon from '@mui/icons-material/Pause';
import type { FlowStepInfo } from '../../types/api';

interface FlowTreeVisualizationProps {
  steps: FlowStepInfo[];
  onStepClick?: (step: FlowStepInfo) => void;
}

// Custom node component
interface StepNodeData {
  step: FlowStepInfo;
  onStepClick?: (step: FlowStepInfo) => void;
}

const StepNode = ({ data }: { data: StepNodeData }) => {
  const theme = useTheme();
  const step: FlowStepInfo = data.step;
  
  const getStatusColor = () => {
    if (step.active) return theme.palette.primary.main;
    
    // Check for explicit success (last execution with no error)
    if (step.last_execution && !step.last_error) return theme.palette.success.main;
    if (step.last_error) return theme.palette.error.main;
    return theme.palette.grey[500];
  };

  const getStatusIcon = () => {
    if (step.active) return <PlayArrowIcon sx={{ color: 'white', fontSize: 16 }} />;
    
    // Check for explicit success (last execution with no error)
    if (step.last_execution && !step.last_error) return <CheckCircleIcon sx={{ color: 'white', fontSize: 16 }} />;
    if (step.last_error) return <ErrorIcon sx={{ color: 'white', fontSize: 16 }} />;
    return <PauseIcon sx={{ color: 'white', fontSize: 16 }} />;
  };

  const successRate = (step.execution_count ?? 0) > 0 ? Math.round(((step.success_count ?? 0) / (step.execution_count ?? 1)) * 100) : 0;

  return (
    <>
      {/* Input handle (left side) */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          background: getStatusColor(),
          width: 10,
          height: 10,
          border: '2px solid white',
        }}
      />
      
      <Paper
        elevation={3}
        onClick={() => data.onStepClick?.(step)}
        sx={{
          minWidth: 200,
          maxWidth: 250,
          p: 2,
          bgcolor: 'background.paper',
          border: `2px solid ${getStatusColor()}`,
          borderRadius: 2,
          cursor: data.onStepClick ? 'pointer' : 'default',
          '&:hover': data.onStepClick ? {
            boxShadow: theme.shadows[6],
            transform: 'translateY(-2px)',
            transition: 'all 0.2s ease-in-out',
          } : {},
        }}
      >
        <Stack spacing={1}>
          <Stack direction="row" alignItems="center" spacing={1}>
            <Box
              sx={{
                width: 24,
                height: 24,
                borderRadius: '50%',
                bgcolor: getStatusColor(),
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                ...(step.active && {
                  animation: 'blink 1s infinite',
                  '@keyframes blink': {
                    '0%, 50%': { opacity: 1 },
                    '51%, 100%': { opacity: 0.3 },
                  },
                }),
              }}
            >
              {getStatusIcon()}
            </Box>
            <Typography variant="subtitle2" fontWeight="bold" noWrap>
              {step.name}
            </Typography>
          </Stack>
          
          <Chip 
            label={step.type} 
            size="small" 
            variant="outlined"
            sx={{ alignSelf: 'flex-start' }}
          />
          
          <Stack direction="row" spacing={1}>
            <Chip 
              label={`${step.execution_count} runs`} 
              size="small" 
              color="info"
            />
            <Chip 
              label={`${successRate}%`} 
              size="small" 
              color={successRate >= 80 ? "success" : successRate >= 50 ? "warning" : "error"}
            />
          </Stack>
          
          {step.last_error && (
            <Typography variant="caption" color="error" noWrap title={step.last_error} component="div">
              Error: {step.last_error}
            </Typography>
          )}
        </Stack>
      </Paper>
      
      {/* Output handle (right side) */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          background: getStatusColor(),
          width: 10,
          height: 10,
          border: '2px solid white',
        }}
      />
    </>
  );
};

const nodeTypes = {
  stepNode: StepNode,
};

export const FlowTreeVisualization: React.FC<FlowTreeVisualizationProps> = ({ steps, onStepClick }) => {
  const theme = useTheme();

  // Convert steps to nodes and edges
  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    const stepMap = new Map<string, FlowStepInfo>();
    steps.forEach(step => stepMap.set(step.name, step));

    // Create nodes
    const nodes: Node[] = [];
    const edges: Edge[] = [];
    
    // Calculate positions using a simple tree layout
    const levelMap = new Map<string, number>();
    
    // Calculate levels for each step
    const calculateLevel = (stepName: string, visited = new Set<string>()): number => {
      if (visited.has(stepName)) return 0; // Prevent cycles
      if (levelMap.has(stepName)) return levelMap.get(stepName) || 0;
      
      visited.add(stepName);
      const step = stepMap.get(stepName);
      
      if (!step || !step.depends_on || step.depends_on.length === 0) {
        levelMap.set(stepName, 0);
        return 0;
      }
      
      const maxParentLevel = Math.max(...step.depends_on.map((dep: string) => calculateLevel(dep, visited)));
      const level = maxParentLevel + 1;
      levelMap.set(stepName, level);
      visited.delete(stepName);
      return level;
    };

    // Calculate levels for all steps
    steps.forEach(step => calculateLevel(step.name));
    
    // Group steps by level
    const levelGroups = new Map<number, string[]>();
    levelMap.forEach((level, stepName) => {
      if (!levelGroups.has(level)) levelGroups.set(level, []);
      const group = levelGroups.get(level);
      if (group) group.push(stepName);
    });

    // Create nodes with positions
    const levelHeight = 150;
    const nodeWidth = 300;
    
    levelGroups.forEach((stepNames, level) => {
      stepNames.forEach((stepName, index) => {
        const step = stepMap.get(stepName);
        if (!step) return;
        const yOffset = (index - (stepNames.length - 1) / 2) * levelHeight;
        
        nodes.push({
          id: stepName,
          type: 'stepNode',
          position: { x: level * nodeWidth, y: yOffset },
          data: { step, onStepClick },
        });
      });
    });

    // Create edges
    steps.forEach(step => {
      if (step.depends_on && step.depends_on.length > 0) {
        step.depends_on.forEach(dependency => {
          // Only create edge if source node exists
          if (stepMap.has(dependency)) {
            edges.push({
              id: `${dependency}->${step.name}`,
              source: dependency,
              target: step.name,
              type: 'smoothstep',
              animated: step.active,
              markerEnd: {
                type: MarkerType.ArrowClosed,
                color: step.active ? theme.palette.primary.main : theme.palette.grey[600],
              },
              style: {
                strokeWidth: 3,
                stroke: step.active ? theme.palette.primary.main : theme.palette.grey[600],
              },
            });
          }
        });
      }
    });

    return { nodes, edges };
  }, [steps, theme, onStepClick]);

  // Use static nodes and edges for read-only view
  const nodes = initialNodes;
  const edges = initialEdges;

  if (steps.length === 0) {
    return (
      <Box 
        sx={{ 
          height: 400, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          bgcolor: 'grey.50',
          borderRadius: 1,
        }}
      >
        <Typography variant="body1" color="textSecondary">
          No flow steps to visualize
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ height: 500, width: '100%', border: 1, borderColor: 'divider', borderRadius: 1 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={true}
        panOnDrag={true}
        zoomOnScroll={true}
        zoomOnPinch={true}
        fitView
        fitViewOptions={{ padding: 0.1 }}
        attributionPosition="bottom-left"
      >
        <Panel position="top-left">
          <Paper sx={{ p: 1 }}>
            <Typography variant="caption" color="textSecondary" component="div">
              Flow Dependencies ({steps.length} steps)
            </Typography>
          </Paper>
        </Panel>
        <Controls position="top-right" />
        <MiniMap 
          position="bottom-right"
          zoomable
          pannable
          style={{
            backgroundColor: theme.palette.background.paper,
            border: `1px solid ${theme.palette.divider}`,
          }}
        />
        <Background 
          variant={BackgroundVariant.Dots} 
          gap={20} 
          size={1} 
          color={theme.palette.divider}
        />
      </ReactFlow>
    </Box>
  );
};