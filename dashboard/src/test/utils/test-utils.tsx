import React, { ReactElement } from "react";
import { render, RenderOptions } from "@testing-library/react";
import { RefineContext } from "@refinedev/core";
import { BrowserRouter } from "react-router-dom";
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { vi } from "vitest";

// Mock theme for testing
const theme = createTheme();

// Mock dataProvider
const mockDataProvider = {
  getList: vi.fn(() => Promise.resolve({ data: [], total: 0 })),
  getOne: vi.fn(() => Promise.resolve({ data: {} })),
  getMany: vi.fn(() => Promise.resolve({ data: [] })),
  create: vi.fn(() => Promise.resolve({ data: {} })),
  update: vi.fn(() => Promise.resolve({ data: {} })),
  deleteOne: vi.fn(() => Promise.resolve({ data: {} })),
  getApiUrl: vi.fn(() => "http://localhost:3000"),
  custom: vi.fn(() => Promise.resolve({ data: {} })),
};

// Mock authProvider
const mockAuthProvider = {
  login: vi.fn(() => Promise.resolve({ success: true })),
  logout: vi.fn(() => Promise.resolve({ success: true })),
  check: vi.fn(() => Promise.resolve({ authenticated: true })),
  onError: vi.fn(() => Promise.resolve({})),
  getPermissions: vi.fn(() => Promise.resolve([])),
  getUserIdentity: vi.fn(() => Promise.resolve(null)),
};

// Mock routerProvider
const mockRouterProvider = {
  go: vi.fn(),
  back: vi.fn(),
  parse: vi.fn(() => ({ pathname: "/", params: {}, resource: undefined })),
};

// Mock notificationProvider
const mockNotificationProvider = {
  open: vi.fn(),
  close: vi.fn(),
};

// Custom render function with providers
const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
  return (
    <BrowserRouter>
      <ThemeProvider theme={theme}>
        <RefineContext.Provider
          value={{
            dataProvider: mockDataProvider,
            authProvider: mockAuthProvider,
            routerProvider: mockRouterProvider,
            notificationProvider: mockNotificationProvider,
            i18nProvider: undefined,
            accessControlProvider: undefined,
            liveProvider: undefined,
            auditLogProvider: undefined,
            options: {},
          }}
        >
          {children}
        </RefineContext.Provider>
      </ThemeProvider>
    </BrowserRouter>
  );
};

const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">
) => render(ui, { wrapper: AllTheProviders, ...options });

// Re-export everything
export * from "@testing-library/react";
export { customRender as render };

// Mock hooks for testing
export const mockUseDataGrid = vi.fn(() => ({
  dataGridProps: {
    rows: [],
    loading: false,
    pageSize: 10,
    page: 0,
    rowCount: 0,
  },
}));

export const mockUseCustom = vi.fn(() => ({
  data: undefined,
  isLoading: false,
  error: null,
}));

export const mockUseCustomMutation = vi.fn(() => ({
  mutate: vi.fn(),
  isLoading: false,
  error: null,
}));

export const mockUseWorkerRuntime = vi.fn(() => ({
  data: undefined,
  isLoading: false,
  error: null,
}));

// Helper to create mock worker data
export const createMockWorker = (overrides = {}) => ({
  id: "123e4567-e89b-12d3-a456-426614174000",
  name: "Test Worker",
  prompt: "Test prompt for worker",
  enabled: true,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T12:00:00Z",
  settings: {
    flow: [
      { name: "step1", type: "agent" },
      { name: "step2", type: "agent" },
    ],
  },
  ...overrides,
});

// Helper to create mock runtime data
export const createMockRuntimeData = (overrides = {}) => ({
  worker: {
    status: "reachable",
    ...overrides,
  },
});
