import React from "react";
import { Box } from "@mui/material";

interface TitleProps {
  collapsed?: boolean;
}

export const Title: React.FC<TitleProps> = ({ collapsed }) => {
  return (
    <Box
      component="div"
      sx={{
        display: "flex",
        alignItems: "center",
        justifyContent: collapsed ? "center" : "flex-start",
        padding: collapsed ? "8px" : "16px",
        minHeight: "64px",
      }}
    >
      <Box
        component="img"
        src="/logo.svg"
        alt="AutoTeam"
        sx={{
          height: collapsed ? "32px" : "40px",
          width: "auto",
          transition: "all 0.2s ease",
        }}
      />
    </Box>
  );
};