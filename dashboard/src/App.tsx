import { Refine } from "@refinedev/core";
import { DevtoolsPanel, DevtoolsProvider } from "@refinedev/devtools";
import { RefineKbar, RefineKbarProvider } from "@refinedev/kbar";

import {
  ErrorComponent,
  RefineSnackbarProvider,
  ThemedLayoutV2,
  useNotificationProvider,
} from "@refinedev/mui";

import CssBaseline from "@mui/material/CssBaseline";
import GlobalStyles from "@mui/material/GlobalStyles";
import routerBindings, {
  DocumentTitleHandler,
  NavigateToResource,
  UnsavedChangesNotifier,
} from "@refinedev/react-router";
import { BrowserRouter, Outlet, Route, Routes } from "react-router";
import { Header } from "./components/header";
import { ColorModeContextProvider } from "./contexts/color-mode";
import { ConfigProvider } from "./providers/ConfigProvider";
import { useConfig } from "./hooks/useConfig";
import { createControlPlaneDataProvider } from "./providers/dataProvider";
import {
  WorkersList,
  WorkersShow,
} from "./pages/workers";

function AppContent() {
  const { config } = useConfig();
  const dataProvider = createControlPlaneDataProvider(config.apiUrl);

  return (
    <Refine
      dataProvider={dataProvider}
      notificationProvider={useNotificationProvider}
      routerProvider={routerBindings}
      authProvider={{
        check: async () => ({ authenticated: true }),
        login: async () => ({ success: true }),
        logout: async () => ({ success: true }),
        onError: async () => ({}),
        getIdentity: async () => null,
        getPermissions: async () => null,
      }}
      resources={[
        {
          name: "workers",
          list: "/workers",
          show: "/workers/show/:id",
          meta: {
            canDelete: false,
            canCreate: false,
            canEdit: false,
          },
        },
      ]}
      options={{
        syncWithLocation: true,
        warnWhenUnsavedChanges: true,
        useNewQueryKeys: true,
        projectId: "09tCO1-vnSeiH-EyLItM",
      }}
    >
      <Routes>
        <Route
          element={
            <ThemedLayoutV2 Header={() => <Header sticky />}>
              <Outlet />
            </ThemedLayoutV2>
          }
        >
          <Route
            index
            element={<NavigateToResource resource="workers" />}
          />
          <Route path="/workers">
            <Route index element={<WorkersList />} />
            <Route path="show/:id" element={<WorkersShow />} />
          </Route>
          <Route path="*" element={<ErrorComponent />} />
        </Route>
      </Routes>

      <RefineKbar />
      <UnsavedChangesNotifier />
      <DocumentTitleHandler />
    </Refine>
  );
}

function App() {
  return (
    <BrowserRouter>
      <RefineKbarProvider>
        <ColorModeContextProvider>
          <CssBaseline />
          <GlobalStyles styles={{ html: { WebkitFontSmoothing: "auto" } }} />
          <RefineSnackbarProvider>
            <DevtoolsProvider>
              <ConfigProvider>
                <AppContent />
              </ConfigProvider>
              <DevtoolsPanel />
            </DevtoolsProvider>
          </RefineSnackbarProvider>
        </ColorModeContextProvider>
      </RefineKbarProvider>
    </BrowserRouter>
  );
}

export default App;
