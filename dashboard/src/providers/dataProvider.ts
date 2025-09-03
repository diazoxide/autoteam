import { DataProvider } from "@refinedev/core";

export const createControlPlaneDataProvider = (apiUrl: string): DataProvider => {
  return {
    getApiUrl: () => apiUrl,

    // Get list of workers
    getList: async ({ resource }) => {
      if (resource === "workers") {
        const response = await fetch(`${apiUrl}/workers`);
        const data = await response.json();
        
        return {
          data: data.workers || [],
          total: data.workers?.length || 0,
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Get single worker details
    getOne: async ({ resource, id }) => {
      if (resource === "workers") {
        const response = await fetch(`${apiUrl}/workers/${id}`);
        const data = await response.json();
        
        return {
          data: data.worker,
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Get worker health status
    getMany: async ({ resource, ids }) => {
      if (resource === "workers") {
        const workers = await Promise.all(
          ids.map(async (id) => {
            try {
              const response = await fetch(`${apiUrl}/workers/${id}`);
              const data = await response.json();
              return data.worker;
            } catch (error) {
              console.error(`Failed to fetch worker ${id}:`, error);
              return null;
            }
          })
        );
        
        return {
          data: workers.filter(Boolean),
        };
      }
      
      throw new Error(`Resource ${resource} not supported`);
    },

    // Not implemented for read-only API
    create: async () => {
      throw new Error("Create operation not supported by control plane API");
    },

    update: async () => {
      throw new Error("Update operation not supported by control plane API");
    },

    deleteOne: async () => {
      throw new Error("Delete operation not supported by control plane API");
    },

    // Custom method to get worker health
    custom: async ({ url, method = "GET" }) => {
      const response = await fetch(`${apiUrl}${url}`, {
        method,
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    },
  };
};