import React, { useState } from "react";
import {
  Box,
  Button,
  Typography,
  Alert,
  Divider,
  Fab,
  Tooltip,
} from "@mui/material";
import {
  Add as AddIcon,
  Visibility as VisibilityIcon,
} from "@mui/icons-material";
import { useFieldArray, Control, useWatch } from "react-hook-form";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  restrictToVerticalAxis,
  restrictToParentElement,
} from "@dnd-kit/modifiers";
import { UnifiedFlowVisualization } from "./UnifiedFlowVisualization";
import { SortableFlowStep } from "./SortableFlowStep";
import { components } from "../../types/generated/api";

type FlowStepInput = components["schemas"]["FlowStepInput"];

interface FlowConfigurationProps {
  control: Control<any>;
  disabled?: boolean;
}

export const FlowConfiguration: React.FC<FlowConfigurationProps> = ({
  control,
  disabled = false,
}) => {
  const [expandedStep, setExpandedStep] = useState<number | null>(0);
  const [showVisualization, setShowVisualization] = useState(false);

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const {
    fields: flowFields,
    append: appendStep,
    remove: removeStep,
    move: moveStep,
    replace: replaceFields,
  } = useFieldArray({
    control,
    name: "settings.flow",
  });

  // Watch flow steps for dependency management and visualization
  const flowSteps = useWatch({
    control,
    name: "settings.flow",
    defaultValue: [],
  }) as FlowStepInput[];

  // Sync flowFields with flowSteps when they differ (handles form reset)
  React.useEffect(() => {
    if (flowSteps.length !== flowFields.length && flowSteps.length > 0) {
      replaceFields(flowSteps);
    }
  }, [flowSteps, flowFields.length, replaceFields]);

  const handleAddStep = () => {
    const newStep: any = { // Use any type to allow form-compatible structure
      name: `step_${flowFields.length + 1}`,
      type: "claude",
      args: [],
      env: [], // Use array format for form compatibility - data provider converts to API format
      depends_on: [],
      input: "",
      output: "",
      dependency_policy: "fail_fast",
      retry: {
        max_attempts: 1,
        delay: 0,
        backoff: "fixed",
        max_delay: 300,
      },
    };
    appendStep(newStep);
    setExpandedStep(flowFields.length);
  };

  const handleRemoveStep = (index: number) => {
    removeStep(index);
    if (expandedStep === index) {
      setExpandedStep(null);
    } else if (expandedStep !== null && expandedStep > index) {
      setExpandedStep(expandedStep - 1);
    }
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (!over || disabled) {
      return;
    }

    const activeIndex = flowFields.findIndex((field) => field.id === active.id);
    const overIndex = flowFields.findIndex((field) => field.id === over.id);

    if (activeIndex !== overIndex && activeIndex !== -1 && overIndex !== -1) {
      moveStep(activeIndex, overIndex);

      // Update expanded step index after reordering
      if (expandedStep === activeIndex) {
        setExpandedStep(overIndex);
      } else if (
        expandedStep !== null &&
        expandedStep >= Math.min(activeIndex, overIndex) &&
        expandedStep <= Math.max(activeIndex, overIndex)
      ) {
        if (activeIndex < overIndex) {
          setExpandedStep(expandedStep - 1);
        } else {
          setExpandedStep(expandedStep + 1);
        }
      }
    }
  };

  const getAvailableStepNames = () => {
    return flowSteps.map((step) => step?.name || "").filter(Boolean);
  };

  const validateFlowSteps = () => {
    const errors: string[] = [];
    const stepNames = new Set<string>();

    flowSteps.forEach((step, index) => {
      if (!step?.name) {
        errors.push(`Step ${index + 1}: Name is required`);
      } else if (stepNames.has(step.name)) {
        errors.push(`Step ${index + 1}: Duplicate step name "${step.name}"`);
      } else {
        stepNames.add(step.name);
      }

      if (!step?.type) {
        errors.push(`Step ${index + 1}: Agent type is required`);
      }

      // Validate dependencies
      if (step?.depends_on) {
        step.depends_on.forEach((dep) => {
          if (!stepNames.has(dep)) {
            errors.push(`Step ${index + 1}: Dependency "${dep}" not found`);
          }
        });
      }
    });

    return errors;
  };

  const validationErrors = validateFlowSteps();

  return (
    <Box sx={{ width: "100%" }}>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          mb: 3,
        }}
      >
        <Typography variant="h6">Flow Configuration</Typography>
        <Box sx={{ display: "flex", gap: 1 }}>
          <Tooltip title="Visualize Flow">
            <Button
              startIcon={<VisibilityIcon />}
              onClick={() => setShowVisualization(!showVisualization)}
              variant="outlined"
              size="small"
            >
              {showVisualization ? "Hide" : "Show"} Visualization
            </Button>
          </Tooltip>
          <Button
            startIcon={<AddIcon />}
            onClick={handleAddStep}
            variant="contained"
            disabled={disabled}
          >
            Add Step
          </Button>
        </Box>
      </Box>

      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <Alert severity="error" sx={{ mb: 2 }}>
          <Typography variant="subtitle2" sx={{ mb: 1 }}>
            Flow Configuration Errors:
          </Typography>
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            {validationErrors.map((error, index) => (
              <li key={index}>{error}</li>
            ))}
          </ul>
        </Alert>
      )}

      {/* Flow Visualization */}
      {showVisualization && flowSteps.length > 0 && (
        <Box sx={{ mb: 3 }}>
          <UnifiedFlowVisualization
            steps={flowSteps}
            mode="configuration"
            compact={false}
          />
          <Divider sx={{ my: 2 }} />
        </Box>
      )}

      {/* Flow Steps */}
      {flowFields.length === 0 ? (
        <Box
          sx={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            py: 6,
            textAlign: "center",
            border: "2px dashed",
            borderColor: "grey.300",
            borderRadius: 2,
            bgcolor: "grey.50",
          }}
        >
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No Flow Steps Configured
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Add flow steps to define the worker's execution workflow
          </Typography>
          <Button
            startIcon={<AddIcon />}
            onClick={handleAddStep}
            variant="contained"
            disabled={disabled}
          >
            Add First Step
          </Button>
        </Box>
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
          modifiers={[restrictToVerticalAxis, restrictToParentElement]}
        >
          <SortableContext
            items={flowFields.map((field) => field.id)}
            strategy={verticalListSortingStrategy}
          >
            <Box>
              {flowFields.map((field, index) => (
                <SortableFlowStep
                  key={field.id}
                  id={field.id}
                  index={index}
                  disabled={disabled}
                  control={control}
                  onRemove={() => handleRemoveStep(index)}
                  availableSteps={getAvailableStepNames()}
                  isExpanded={expandedStep === index}
                  onToggleExpand={() =>
                    setExpandedStep(expandedStep === index ? null : index)
                  }
                />
              ))}
            </Box>
          </SortableContext>
        </DndContext>
      )}

      {/* Flow Summary */}
      {flowSteps.length > 0 && (
        <Box sx={{ mt: 3, p: 2, bgcolor: "grey.50", borderRadius: 1 }}>
          <Typography variant="subtitle2" gutterBottom>
            Flow Summary
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {flowSteps.length} step{flowSteps.length !== 1 ? "s" : ""}{" "}
            configured
            {validationErrors.length === 0 && <>, ready for execution</>}
          </Typography>
        </Box>
      )}

      {/* Floating Action Button for Quick Add */}
      {flowFields.length > 0 && !disabled && (
        <Fab
          color="primary"
          aria-label="add step"
          onClick={handleAddStep}
          sx={{
            position: "fixed",
            bottom: 16,
            right: 16,
            zIndex: 1000,
          }}
        >
          <AddIcon />
        </Fab>
      )}
    </Box>
  );
};
