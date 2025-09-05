import DarkModeOutlined from "@mui/icons-material/DarkModeOutlined";
import LightModeOutlined from "@mui/icons-material/LightModeOutlined";
import AppBar from "@mui/material/AppBar";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import { HamburgerMenu, RefineThemedLayoutV2HeaderProps } from "@refinedev/mui";
import React, { useContext } from "react";
import { ColorModeContext } from "../../contexts/color-mode";

export const Header: React.FC<RefineThemedLayoutV2HeaderProps> = ({
  sticky = true,
}) => {
  const { mode, setMode } = useContext(ColorModeContext);

  return (
    <AppBar position={sticky ? "sticky" : "relative"}>
      <Toolbar>
        <HamburgerMenu />
        <Typography variant="h6" component="div" sx={{ flexGrow: 1, ml: 2 }}>
          AutoTeam Dashboard
        </Typography>
        <Stack
          direction="row"
          alignItems="center"
        >
            <IconButton
              color="inherit"
              onClick={() => {
                setMode();
              }}
            >
              {mode === "dark" ? <LightModeOutlined /> : <DarkModeOutlined />}
            </IconButton>

        </Stack>
      </Toolbar>
    </AppBar>
  );
};
