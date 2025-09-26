import React, { useMemo } from "react";
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
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  Box,
  Paper,
  Typography,
  Chip,
  Stack,
  useTheme,
  Badge,
  alpha,
} from "@mui/material";
import {
  AccountTree as FlowIcon,
  PlayArrow as PlayIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Pause as PauseIcon,
  Refresh as RefreshIcon,
  CheckBox as CheckBoxIcon,
  CheckBoxOutlineBlank as CheckBoxOutlineBlankIcon,
  FlashOn as FlashOnIcon,
} from "@mui/icons-material";
import type { FlowStepInfo } from "../../types/api";
import { components } from "../../types/generated/api";

type FlowStepInput = components["schemas"]["FlowStepInput"];

// Union type for both step types
type UnifiedFlowStep = FlowStepInfo | FlowStepInput;

interface UnifiedFlowVisualizationProps {
  steps: UnifiedFlowStep[];
  mode: "runtime" | "configuration";
  compact?: boolean;
  onStepClick?: (step: UnifiedFlowStep) => void;
}

// Type guards
const isRuntimeStep = (step: UnifiedFlowStep): step is FlowStepInfo => {
  return (
    "active" in step || "execution_count" in step || "last_execution" in step
  );
};

// Helper to check if step is configuration type
// const isConfigurationStep = (step: UnifiedFlowStep): step is FlowStepInput => {
//   return !isRuntimeStep(step);
// };

// Custom node component
interface StepNodeData {
  step: UnifiedFlowStep;
  mode: "runtime" | "configuration";
  compact?: boolean;
  onStepClick?: (step: UnifiedFlowStep) => void;
}

const StepNode = ({ data }: { data: StepNodeData }) => {
  const theme = useTheme();
  const { step, mode, compact, onStepClick } = data;

  const getAgentTypeColor = (type: string) => {
    switch (type) {
      case "claude":
        return theme.palette.primary.main;
      case "qwen":
        return theme.palette.secondary.main;
      case "gemini":
        return theme.palette.success.main;
      case "debug":
        return theme.palette.warning.main;
      default:
        return theme.palette.grey[500];
    }
  };

  const getPolicyIcon = (policy?: string) => {
    switch (policy) {
      case "fail_fast":
        return <FlashOnIcon sx={{ fontSize: 14 }} />;
      case "all_success":
        return <CheckCircleIcon sx={{ fontSize: 14 }} />;
      case "all_complete":
        return <CheckBoxOutlineBlankIcon sx={{ fontSize: 14 }} />;
      case "any_success":
        return <CheckBoxIcon sx={{ fontSize: 14 }} />;
      default:
        return <FlashOnIcon sx={{ fontSize: 14 }} />;
    }
  };

  const getPolicyColor = (policy?: string) => {
    switch (policy) {
      case "fail_fast":
        return theme.palette.error.main;
      case "all_success":
        return theme.palette.success.main;
      case "all_complete":
        return theme.palette.info.main;
      case "any_success":
        return theme.palette.warning.main;
      default:
        return theme.palette.error.main;
    }
  };

  const getBorderStyle = () => {
    // Check if step has retry configuration
    if (step.retry && step.retry.max_attempts && step.retry.max_attempts > 1) {
      return "2px dashed";
    }
    // Check if dependency policy is not default
    if (step.dependency_policy && step.dependency_policy !== "fail_fast") {
      return "3px solid";
    }
    return "2px solid";
  };

  const getStatusColor = () => {
    const agentColor = getAgentTypeColor(step.type);

    if (isRuntimeStep(step)) {
      if (step.active) return theme.palette.primary.main;
      if (step.last_execution && !step.last_error)
        return theme.palette.success.main;
      if (step.last_error) return theme.palette.error.main;
      return theme.palette.grey[500];
    }

    return agentColor;
  };

  const getStatusIcon = () => {
    if (isRuntimeStep(step)) {
      if (step.active)
        return <PlayIcon sx={{ color: "white", fontSize: 16 }} />;
      if (step.last_execution && !step.last_error)
        return <CheckCircleIcon sx={{ color: "white", fontSize: 16 }} />;
      if (step.last_error)
        return <ErrorIcon sx={{ color: "white", fontSize: 16 }} />;
      return <PauseIcon sx={{ color: "white", fontSize: 16 }} />;
    }

    return <PlayIcon sx={{ color: "white", fontSize: 16 }} />;
  };

  const agentColor = getAgentTypeColor(step.type);
  const statusColor = getStatusColor();

  return (
    <>
      {/* Input handle (left side) */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          background: statusColor,
          width: 10,
          height: 10,
          border: "2px solid white",
        }}
      />

      <Paper
        elevation={3}
        onClick={() => onStepClick?.(step)}
        sx={{
          minWidth: compact ? 150 : 200,
          maxWidth: compact ? 180 : 250,
          p: compact ? 1 : 2,
          bgcolor: alpha(agentColor, 0.1),
          border: `${getBorderStyle()} ${statusColor}`,
          borderRadius: 2,
          cursor: onStepClick ? "pointer" : "default",
          position: "relative",
          "&:hover": onStepClick
            ? {
                bgcolor: alpha(agentColor, 0.2),
                transform: "translateY(-2px)",
                boxShadow: theme.shadows[6],
                transition: "all 0.2s ease-in-out",
              }
            : {},
          ...(isRuntimeStep(step) &&
            step.active && {
              "&::after": {
                content: '""',
                position: "absolute",
                top: -2,
                left: -2,
                right: -2,
                bottom: -2,
                background: `linear-gradient(45deg, ${statusColor}, transparent, ${statusColor})`,
                borderRadius: 2,
                zIndex: -1,
                animation: "pulse 2s infinite",
                "@keyframes pulse": {
                  "0%, 100%": { opacity: 0.3 },
                  "50%": { opacity: 0.8 },
                },
              },
            }),
        }}
      >
        <Stack spacing={compact ? 0.5 : 1}>
          <Stack direction="row" alignItems="center" spacing={1}>
            <Box
              sx={{
                width: compact ? 20 : 24,
                height: compact ? 20 : 24,
                borderRadius: "50%",
                bgcolor: statusColor,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                ...(isRuntimeStep(step) &&
                  step.active && {
                    animation: "blink 1s infinite",
                    "@keyframes blink": {
                      "0%, 50%": { opacity: 1 },
                      "51%, 100%": { opacity: 0.3 },
                    },
                  }),
              }}
            >
              {getStatusIcon()}
            </Box>
            <Typography
              variant={compact ? "caption" : "subtitle2"}
              fontWeight="bold"
              noWrap
              sx={{ overflow: "hidden", textOverflow: "ellipsis" }}
            >
              {step.name}
            </Typography>
          </Stack>

          <Chip
            label={step.type}
            size="small"
            sx={{
              alignSelf: "flex-start",
              bgcolor: agentColor,
              color: "white",
              fontSize: compact ? "0.6rem" : "0.75rem",
              height: compact ? 20 : 24,
            }}
          />

          {/* Runtime-specific metrics */}
          {mode === "runtime" && isRuntimeStep(step) && !compact && (
            <Stack direction="row" spacing={1} flexWrap="wrap">
              <Chip
                label={`${step.execution_count || 0} runs`}
                size="small"
                color="info"
                sx={{ fontSize: "0.65rem", height: 20 }}
              />
              {step.execution_count && step.execution_count > 0 && (
                <Chip
                  label={`${Math.round(((step.success_count || 0) / step.execution_count) * 100)}%`}
                  size="small"
                  color={
                    (step.success_count || 0) / step.execution_count >= 0.8
                      ? "success"
                      : (step.success_count || 0) / step.execution_count >= 0.5
                        ? "warning"
                        : "error"
                  }
                  sx={{ fontSize: "0.65rem", height: 20 }}
                />
              )}
            </Stack>
          )}

          {/* Configuration-specific indicators */}
          {mode === "configuration" && !compact && (
            <Box>
              {step.skip_when && (
                <Chip
                  label="Conditional"
                  size="small"
                  variant="outlined"
                  color="warning"
                  sx={{ mr: 0.5, mb: 0.5, fontSize: "0.6rem", height: 18 }}
                />
              )}
              {step.depends_on && step.depends_on.length > 0 && (
                <Chip
                  label={`${step.depends_on.length} deps`}
                  size="small"
                  variant="outlined"
                  color="secondary"
                  sx={{ mr: 0.5, mb: 0.5, fontSize: "0.6rem", height: 18 }}
                />
              )}
            </Box>
          )}

          {/* Error display for runtime */}
          {mode === "runtime" &&
            isRuntimeStep(step) &&
            step.last_error &&
            !compact && (
              <Typography
                variant="caption"
                color="error"
                noWrap
                title={step.last_error}
                component="div"
              >
                Error: {step.last_error}
              </Typography>
            )}

          {/* Policy and Retry Indicators */}
          {!compact && (
            <Stack
              direction="row"
              spacing={0.5}
              alignItems="center"
              flexWrap="wrap"
            >
              {step.dependency_policy &&
                step.dependency_policy !== "fail_fast" && (
                  <Chip
                    icon={getPolicyIcon(step.dependency_policy)}
                    label={step.dependency_policy.replace("_", " ")}
                    size="small"
                    variant="outlined"
                    sx={{
                      fontSize: "0.65rem",
                      height: 20,
                      color: getPolicyColor(step.dependency_policy),
                      borderColor: getPolicyColor(step.dependency_policy),
                    }}
                  />
                )}

              {step.retry &&
                step.retry.max_attempts &&
                step.retry.max_attempts > 1 && (
                  <Chip
                    icon={<RefreshIcon sx={{ fontSize: 12 }} />}
                    label={`${step.retry.max_attempts}x`}
                    size="small"
                    color="info"
                    sx={{ fontSize: "0.65rem", height: 20 }}
                  />
                )}
            </Stack>
          )}
        </Stack>

        {/* Retry indicator badge - positioned absolutely - runtime only */}
        {mode === "runtime" &&
          isRuntimeStep(step) &&
          step.retry_attempt !== undefined &&
          step.retry_attempt > 0 && (
            <Box
              sx={{
                position: "absolute",
                top: -8,
                right: -8,
                animation: step.active ? "pulse 1.5s infinite" : "none",
                "@keyframes pulse": {
                  "0%": { transform: "scale(1)" },
                  "50%": { transform: "scale(1.1)" },
                  "100%": { transform: "scale(1)" },
                },
              }}
            >
              <Badge
                badgeContent={`${step.retry_attempt}/${step.retry?.max_attempts || 1}`}
                color={
                  step.retry_attempt >= (step.retry?.max_attempts || 1)
                    ? "error"
                    : step.active
                      ? "warning"
                      : "default"
                }
                sx={{
                  "& .MuiBadge-badge": {
                    fontSize: "0.6rem",
                    minWidth: "16px",
                    height: "16px",
                  },
                }}
              >
                <RefreshIcon
                  fontSize="small"
                  sx={{
                    color: step.active
                      ? theme.palette.warning.main
                      : theme.palette.text.secondary,
                  }}
                />
              </Badge>
            </Box>
          )}
      </Paper>

      {/* Output handle (right side) */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          background: statusColor,
          width: 10,
          height: 10,
          border: "2px solid white",
        }}
      />
    </>
  );
};

const nodeTypes = {
  stepNode: StepNode,
};

export const UnifiedFlowVisualization: React.FC<
  UnifiedFlowVisualizationProps
> = ({ steps, mode, compact = false, onStepClick }) => {
  const theme = useTheme();

  // Convert steps to nodes and edges
  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    const stepMap = new Map<string, UnifiedFlowStep>();
    steps.forEach((step) => stepMap.set(step.name, step));

    // Create nodes
    const nodes: Node[] = [];
    const edges: Edge[] = [];

    // Calculate positions using a simple tree layout
    const levelMap = new Map<string, number>();

    // Calculate levels for each step
    const calculateLevel = (
      stepName: string,
      visited = new Set<string>()
    ): number => {
      if (visited.has(stepName)) return 0; // Prevent cycles
      if (levelMap.has(stepName)) return levelMap.get(stepName) || 0;

      visited.add(stepName);
      const step = stepMap.get(stepName);

      if (!step || !step.depends_on || step.depends_on.length === 0) {
        levelMap.set(stepName, 0);
        return 0;
      }

      const maxParentLevel = Math.max(
        ...step.depends_on.map((dep: string) => calculateLevel(dep, visited))
      );
      const level = maxParentLevel + 1;
      levelMap.set(stepName, level);
      visited.delete(stepName);
      return level;
    };

    // Calculate levels for all steps
    steps.forEach((step) => calculateLevel(step.name));

    // Group steps by level
    const levelGroups = new Map<number, string[]>();
    levelMap.forEach((level, stepName) => {
      if (!levelGroups.has(level)) levelGroups.set(level, []);
      const group = levelGroups.get(level);
      if (group) group.push(stepName);
    });

    // Create nodes with positions
    const levelHeight = compact ? 120 : 150;
    const nodeWidth = compact ? 200 : 300;

    levelGroups.forEach((stepNames, level) => {
      stepNames.forEach((stepName, index) => {
        const step = stepMap.get(stepName);
        if (!step) return;
        const yOffset = (index - (stepNames.length - 1) / 2) * levelHeight;

        nodes.push({
          id: stepName,
          type: "stepNode",
          position: { x: level * nodeWidth, y: yOffset },
          data: { step, mode, compact, onStepClick },
        });
      });
    });

    // Create edges
    steps.forEach((step) => {
      if (step.depends_on && step.depends_on.length > 0) {
        step.depends_on.forEach((dependency) => {
          // Only create edge if source node exists
          if (stepMap.has(dependency)) {
            const isActive = isRuntimeStep(step) && step.active;

            edges.push({
              id: `${dependency}->${step.name}`,
              source: dependency,
              target: step.name,
              type: "smoothstep",
              animated: isActive,
              markerEnd: {
                type: MarkerType.ArrowClosed,
                color: isActive
                  ? theme.palette.primary.main
                  : theme.palette.grey[600],
              },
              style: {
                strokeWidth: 3,
                stroke: isActive
                  ? theme.palette.primary.main
                  : theme.palette.grey[600],
              },
            });
          }
        });
      }
    });

    return { nodes, edges };
  }, [steps, mode, compact, theme, onStepClick]);

  // Use static nodes and edges for read-only view
  const nodes = initialNodes;
  const edges = initialEdges;

  if (steps.length === 0) {
    return (
      <Box
        sx={{
          height: compact ? 200 : 400,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          bgcolor: "grey.50",
          borderRadius: 1,
          textAlign: "center",
        }}
      >
        <FlowIcon
          sx={{ fontSize: compact ? 32 : 48, color: "grey.400", mb: 2 }}
        />
        <Typography variant={compact ? "body2" : "h6"} color="text.secondary">
          No Flow Steps to Visualize
        </Typography>
      </Box>
    );
  }

  return (
    <Box
      sx={{
        height: compact ? 300 : 500,
        width: "100%",
        border: 1,
        borderColor: "divider",
        borderRadius: 1,
      }}
    >
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
              {mode === "runtime" ? "Flow Execution" : "Flow Configuration"} (
              {steps.length} steps)
            </Typography>
          </Paper>
        </Panel>

        {!compact && <Controls position="top-right" />}

        {!compact && (
          <MiniMap
            position="bottom-right"
            zoomable
            pannable
            style={{
              backgroundColor: theme.palette.background.paper,
              border: `1px solid ${theme.palette.divider}`,
            }}
          />
        )}

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
