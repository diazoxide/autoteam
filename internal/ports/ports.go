package ports

import (
	"fmt"
	"net"
	"sort"
)

// PortManager handles dynamic port allocation
type PortManager struct {
	// Starting port range (default 8080-8999)
	StartPort int
	EndPort   int
	// Track allocated ports to avoid conflicts
	allocatedPorts map[int]bool
}

// NewPortManager creates a new port manager with default range (using less popular ports)
func NewPortManager() *PortManager {
	return &PortManager{
		StartPort:      45000,
		EndPort:        45999,
		allocatedPorts: make(map[int]bool),
	}
}

// NewPortManagerWithRange creates a port manager with custom range
func NewPortManagerWithRange(startPort, endPort int) *PortManager {
	return &PortManager{
		StartPort:      startPort,
		EndPort:        endPort,
		allocatedPorts: make(map[int]bool),
	}
}

// IsPortAvailable checks if a port is available on localhost
func (pm *PortManager) IsPortAvailable(port int) bool {
	// Check if already allocated
	if pm.allocatedPorts[port] {
		return false
	}

	// Try to listen on the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// FindFreePort finds the next available port in the range
func (pm *PortManager) FindFreePort() (int, error) {
	for port := pm.StartPort; port <= pm.EndPort; port++ {
		if pm.IsPortAvailable(port) {
			pm.allocatedPorts[port] = true
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports available in range %d-%d", pm.StartPort, pm.EndPort)
}

// FindFreePorts finds multiple free ports
func (pm *PortManager) FindFreePorts(count int) ([]int, error) {
	var ports []int

	for i := 0; i < count; i++ {
		port, err := pm.FindFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find %d free ports, only found %d: %w", count, len(ports), err)
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// ReservePort reserves a specific port if available
func (pm *PortManager) ReservePort(port int) error {
	if !pm.IsPortAvailable(port) {
		return fmt.Errorf("port %d is not available", port)
	}
	pm.allocatedPorts[port] = true
	return nil
}

// ReleasePort releases a previously allocated port
func (pm *PortManager) ReleasePort(port int) {
	delete(pm.allocatedPorts, port)
}

// GetAllocatedPorts returns a sorted list of allocated ports
func (pm *PortManager) GetAllocatedPorts() []int {
	var ports []int
	for port := range pm.allocatedPorts {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports
}

// Reset clears all allocated ports
func (pm *PortManager) Reset() {
	pm.allocatedPorts = make(map[int]bool)
}

// PortAllocation represents a mapping of service names to ports
type PortAllocation map[string]int

// AllocatePortsForServices allocates ports for a list of service names
func (pm *PortManager) AllocatePortsForServices(serviceNames []string) (PortAllocation, error) {
	allocation := make(PortAllocation)

	for _, serviceName := range serviceNames {
		port, err := pm.FindFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port for service '%s': %w", serviceName, err)
		}
		allocation[serviceName] = port
	}

	return allocation, nil
}
