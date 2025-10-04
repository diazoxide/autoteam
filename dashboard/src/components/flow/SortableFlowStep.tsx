import React from "react";
import { Box } from "@mui/material";
import { DragIndicator as DragIndicatorIcon } from "@mui/icons-material";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Control } from "react-hook-form";
import { FlowStepForm } from "./FlowStepForm";

interface SortableFlowStepProps {
  id: string;
  index: number;
  disabled?: boolean;
  control: Control<any>;
  onRemove: () => void;
  availableSteps: string[];
  isExpanded: boolean;
  onToggleExpand: () => void;
}

export const SortableFlowStep: React.FC<SortableFlowStepProps> = ({
  id,
  index,
  disabled = false,
  control,
  onRemove,
  availableSteps,
  isExpanded,
  onToggleExpand,
}) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id, disabled });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 1000 : 1,
    opacity: isDragging ? 0.8 : 1,
  };

  return (
    <Box
      ref={setNodeRef}
      style={style}
      sx={{
        mb: 2,
        position: "relative",
        ...(isDragging && {
          "& > *": {
            boxShadow: (theme) => theme.shadows[8],
          },
        }),
      }}
    >
      {/* Drag Handle */}
      <Box
        {...attributes}
        {...listeners}
        sx={{
          position: "absolute",
          left: -16,
          top: "50%",
          transform: "translateY(-50%)",
          zIndex: 10,
          cursor: disabled ? "default" : "grab",
          bgcolor: "background.paper",
          border: 1,
          borderColor: "divider",
          borderRadius: 1,
          p: 0.5,
          display: "flex",
          alignItems: "center",
          "&:active": {
            cursor: disabled ? "default" : "grabbing",
          },
          "&:hover": {
            bgcolor: disabled ? "background.paper" : "action.hover",
          },
        }}
      >
        <DragIndicatorIcon
          sx={{
            color: disabled ? "grey.400" : "grey.600",
            fontSize: 16,
          }}
        />
      </Box>

      {/* Step Number Badge */}
      <Box
        sx={{
          position: "absolute",
          left: -8,
          top: 8,
          zIndex: 5,
          bgcolor: "primary.main",
          color: "primary.contrastText",
          borderRadius: "50%",
          width: 24,
          height: 24,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontSize: "0.75rem",
          fontWeight: "bold",
        }}
      >
        {index + 1}
      </Box>

      {/* Flow Step Form */}
      <FlowStepForm
        control={control}
        stepIndex={index}
        onRemove={onRemove}
        availableSteps={availableSteps}
        isExpanded={isExpanded}
        onToggleExpand={onToggleExpand}
      />
    </Box>
  );
};
