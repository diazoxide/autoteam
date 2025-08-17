package ports

import (
	"testing"
)

func TestPortManager_FindFreePort(t *testing.T) {
	pm := NewPortManager()

	// Test finding a single free port
	port, err := pm.FindFreePort()
	if err != nil {
		t.Fatalf("FindFreePort() returned error: %v", err)
	}

	if port < pm.StartPort || port > pm.EndPort {
		t.Errorf("FindFreePort() returned port %d, expected in range %d-%d", port, pm.StartPort, pm.EndPort)
	}

	// Verify port is marked as allocated
	allocated := pm.GetAllocatedPorts()
	if len(allocated) != 1 || allocated[0] != port {
		t.Errorf("Expected allocated ports to contain %d, got %v", port, allocated)
	}

	// Verify the same port is no longer available
	if pm.IsPortAvailable(port) {
		t.Errorf("Port %d should not be available after allocation", port)
	}
}

func TestPortManager_FindFreePorts(t *testing.T) {
	pm := NewPortManager()

	// Test finding multiple free ports
	count := 3
	ports, err := pm.FindFreePorts(count)
	if err != nil {
		t.Fatalf("FindFreePorts(%d) returned error: %v", count, err)
	}

	if len(ports) != count {
		t.Errorf("FindFreePorts(%d) returned %d ports, expected %d", count, len(ports), count)
	}

	// Verify all ports are unique
	portMap := make(map[int]bool)
	for _, port := range ports {
		if portMap[port] {
			t.Errorf("FindFreePorts() returned duplicate port %d", port)
		}
		portMap[port] = true

		// Verify port is in range
		if port < pm.StartPort || port > pm.EndPort {
			t.Errorf("Port %d is outside expected range %d-%d", port, pm.StartPort, pm.EndPort)
		}
	}

	// Verify all ports are marked as allocated
	allocated := pm.GetAllocatedPorts()
	if len(allocated) != count {
		t.Errorf("Expected %d allocated ports, got %d", count, len(allocated))
	}
}

func TestPortManager_AllocatePortsForServices(t *testing.T) {
	pm := NewPortManager()

	serviceNames := []string{"service1", "service2", "service3"}
	allocation, err := pm.AllocatePortsForServices(serviceNames)
	if err != nil {
		t.Fatalf("AllocatePortsForServices() returned error: %v", err)
	}

	// Verify allocation has entries for all services
	if len(allocation) != len(serviceNames) {
		t.Errorf("Expected allocation for %d services, got %d", len(serviceNames), len(allocation))
	}

	// Verify each service has a valid port
	for _, serviceName := range serviceNames {
		port, exists := allocation[serviceName]
		if !exists {
			t.Errorf("Service %s not found in allocation", serviceName)
			continue
		}

		if port < pm.StartPort || port > pm.EndPort {
			t.Errorf("Service %s allocated port %d outside range %d-%d", serviceName, port, pm.StartPort, pm.EndPort)
		}
	}

	// Verify all ports are unique
	usedPorts := make(map[int]string)
	for serviceName, port := range allocation {
		if existingService, exists := usedPorts[port]; exists {
			t.Errorf("Port %d allocated to both %s and %s", port, existingService, serviceName)
		}
		usedPorts[port] = serviceName
	}
}

func TestPortManager_WithCustomRange(t *testing.T) {
	// Test with custom port range
	startPort := 9000
	endPort := 9100
	pm := NewPortManagerWithRange(startPort, endPort)

	port, err := pm.FindFreePort()
	if err != nil {
		t.Fatalf("FindFreePort() with custom range returned error: %v", err)
	}

	if port < startPort || port > endPort {
		t.Errorf("FindFreePort() returned port %d, expected in custom range %d-%d", port, startPort, endPort)
	}
}

func TestPortManager_ReserveAndRelease(t *testing.T) {
	pm := NewPortManager()

	// Find an available port first
	port, err := pm.FindFreePort()
	if err != nil {
		t.Fatalf("FindFreePort() returned error: %v", err)
	}

	// Release it
	pm.ReleasePort(port)

	// Verify it's available again
	if !pm.IsPortAvailable(port) {
		t.Errorf("Port %d should be available after release", port)
	}

	// Reserve it again
	if err := pm.ReservePort(port); err != nil {
		t.Errorf("ReservePort(%d) returned error: %v", port, err)
	}

	// Verify it's not available
	if pm.IsPortAvailable(port) {
		t.Errorf("Port %d should not be available after reservation", port)
	}
}
