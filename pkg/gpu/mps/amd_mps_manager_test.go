package mps

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewAMDMPSManager(t *testing.T) {
	config := MPSManagerConfig{
		DefaultPort:         29501,
		DefaultLogLevel:     "debug",
		DefaultMaxConnections: 8,
		DefaultMemoryLimit:  4 * 1024 * 1024 * 1024, // 4GB
		DefaultTimeout:      15 * time.Second,
		LogBaseDirectory:    "/tmp/test-mps",
		EnableMetrics:       true,
		HealthCheckInterval: 5 * time.Second,
	}

	manager := NewAMDMPSManager(config)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.config.DefaultPort != 29501 {
		t.Errorf("Expected default port 29501, got %d", manager.config.DefaultPort)
	}

	if manager.config.DefaultLogLevel != "debug" {
		t.Errorf("Expected default log level 'debug', got %s", manager.config.DefaultLogLevel)
	}

	if manager.config.DefaultMaxConnections != 8 {
		t.Errorf("Expected default max connections 8, got %d", manager.config.DefaultMaxConnections)
	}

	// Clean up
	os.RemoveAll("/tmp/test-mps")
}

func TestAMDMPSManagerDefaultConfig(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	if manager.config.DefaultPort != 29500 {
		t.Errorf("Expected default port 29500, got %d", manager.config.DefaultPort)
	}

	if manager.config.DefaultLogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", manager.config.DefaultLogLevel)
	}

	if manager.config.DefaultMaxConnections != 16 {
		t.Errorf("Expected default max connections 16, got %d", manager.config.DefaultMaxConnections)
	}

	if manager.config.DefaultMemoryLimit != 8*1024*1024*1024 {
		t.Errorf("Expected default memory limit 8GB, got %d", manager.config.DefaultMemoryLimit)
	}

	if manager.config.DefaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", manager.config.DefaultTimeout)
	}

	if manager.config.LogBaseDirectory != "/tmp/kaiwo-mps" {
		t.Errorf("Expected default log directory '/tmp/kaiwo-mps', got %s", manager.config.LogBaseDirectory)
	}

	if manager.config.HealthCheckInterval != 10*time.Second {
		t.Errorf("Expected default health check interval 10s, got %v", manager.config.HealthCheckInterval)
	}
}

func TestAMDMPSManagerGetMPSServer(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Test getting non-existent server
	server, exists := manager.GetMPSServer("card0")
	if exists {
		t.Error("Expected server to not exist")
	}
	if server != nil {
		t.Error("Expected nil server")
	}
}

func TestAMDMPSManagerListMPSServers(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Test empty list
	servers := manager.ListMPSServers()
	if len(servers) != 0 {
		t.Errorf("Expected empty server list, got %d servers", len(servers))
	}
}

func TestAMDMPSManagerGetMPSConnectionInfo(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Test getting connection info for non-existent server
	connInfo, err := manager.GetMPSConnectionInfo("card0")
	if err == nil {
		t.Error("Expected error for non-existent server")
	}
	if connInfo != nil {
		t.Error("Expected nil connection info")
	}
}

func TestAMDMPSManagerGetMPSStats(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	stats := manager.GetMPSStats()

	if stats.TotalServers != 0 {
		t.Errorf("Expected 0 total servers, got %d", stats.TotalServers)
	}

	if stats.RunningServers != 0 {
		t.Errorf("Expected 0 running servers, got %d", stats.RunningServers)
	}

	if stats.StoppedServers != 0 {
		t.Errorf("Expected 0 stopped servers, got %d", stats.StoppedServers)
	}

	if stats.ErrorServers != 0 {
		t.Errorf("Expected 0 error servers, got %d", stats.ErrorServers)
	}

	if stats.StartingServers != 0 {
		t.Errorf("Expected 0 starting servers, got %d", stats.StartingServers)
	}

	if len(stats.ServerDetails) != 0 {
		t.Errorf("Expected empty server details, got %d", len(stats.ServerDetails))
	}
}

func TestAMDMPSManagerStopMPSServer(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Test stopping non-existent server
	err := manager.StopMPSServer("card0")
	if err == nil {
		t.Error("Expected error when stopping non-existent server")
	}
}

func TestAMDMPSManagerCleanup(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Test cleanup with no servers
	err := manager.Cleanup()
	if err != nil {
		t.Errorf("Expected no error during cleanup, got %v", err)
	}

	stats := manager.GetMPSStats()
	if stats.TotalServers != 0 {
		t.Errorf("Expected 0 servers after cleanup, got %d", stats.TotalServers)
	}
}

func TestAMDMPSManagerFindAvailablePort(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	port := manager.findAvailablePort()
	if port != 29500 {
		t.Errorf("Expected port 29500, got %d", port)
	}

	// Test with existing servers
	manager.servers["card0"] = &AMDMPSServer{
		ID:    "test-server",
		GPUID: "card0",
		Port:  29500,
	}

	port = manager.findAvailablePort()
	if port != 29501 {
		t.Errorf("Expected port 29501, got %d", port)
	}
}

func TestAMDMPSManagerStartMPSServerWithConfig(t *testing.T) {
	tempDir := t.TempDir()
	config := MPSManagerConfig{
		LogBaseDirectory: tempDir,
	}

	manager := NewAMDMPSManager(config)

	serverConfig := &MPSServerConfig{
		GPUID:           "card0",
		Port:            29501,
		LogLevel:        "info",
		MaxConnections:  4,
		MemoryLimit:     2 * 1024 * 1024 * 1024, // 2GB
		Timeout:         10 * time.Second,
		LogDirectory:    filepath.Join(tempDir, "card0"),
		EnvironmentVars: map[string]string{"TEST_VAR": "test_value"},
	}

	// Note: This test won't actually start a real MPS server since hip-mps-server
	// is not available in the test environment, but it tests the configuration logic
	ctx := context.Background()
	server, err := manager.StartMPSServer(ctx, "card0", serverConfig)

	// We expect an error because hip-mps-server is not available
	if err == nil {
		t.Log("MPS server started successfully (hip-mps-server available)")
		// If server started successfully, test its properties
		if server.GPUID != "card0" {
			t.Errorf("Expected GPU ID 'card0', got %s", server.GPUID)
		}
		if server.Port != 29501 {
			t.Errorf("Expected port 29501, got %d", server.Port)
		}
		if server.Config.LogLevel != "info" {
			t.Errorf("Expected log level 'info', got %s", server.Config.LogLevel)
		}
	} else {
		t.Logf("Expected error (hip-mps-server not available): %v", err)
	}
}

func TestAMDMPSManagerStartMPSServerDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	config := MPSManagerConfig{
		LogBaseDirectory: tempDir,
	}

	manager := NewAMDMPSManager(config)

	ctx := context.Background()
	server, err := manager.StartMPSServer(ctx, "card1", nil)

	// We expect an error because hip-mps-server is not available
	if err == nil {
		t.Log("MPS server started successfully with default config")
		// Test default configuration
		if server.GPUID != "card1" {
			t.Errorf("Expected GPU ID 'card1', got %s", server.GPUID)
		}
		if server.Port != 29500 {
			t.Errorf("Expected default port 29500, got %d", server.Port)
		}
		if server.Config.LogLevel != "info" {
			t.Errorf("Expected default log level 'info', got %s", server.Config.LogLevel)
		}
		if server.Config.MaxConnections != 16 {
			t.Errorf("Expected default max connections 16, got %d", server.Config.MaxConnections)
		}
	} else {
		t.Logf("Expected error (hip-mps-server not available): %v", err)
	}
}

func TestAMDMPSManagerRestartLogic(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{})

	// Create a mock server for testing restart logic
	server := &AMDMPSServer{
		ID:    "test-server",
		GPUID: "card0",
		Port:  29500,
		Config: MPSServerConfig{
			GPUID:    "card0",
			Port:     29500,
			LogLevel: "info",
		},
	}

	// Test restart logic (this won't actually restart since hip-mps-server is not available)
	manager.restartMPSServer(server)

	// Verify server was removed from the map
	_, exists := manager.GetMPSServer("card0")
	if exists {
		t.Error("Expected server to be removed after restart attempt")
	}
}

func TestAMDMPSManagerHealthCheck(t *testing.T) {
	manager := NewAMDMPSManager(MPSManagerConfig{
		HealthCheckInterval: 100 * time.Millisecond, // Fast for testing
	})

	// Create a mock server
	server := &AMDMPSServer{
		ID:    "test-server",
		GPUID: "card0",
		Port:  29500,
		Status: MPSServerStatusRunning,
	}

	// Add server to manager
	manager.servers["card0"] = server

	// Start health check
	go manager.healthCheck(server)

	// Wait a bit for health check to run
	time.Sleep(200 * time.Millisecond)

	// Verify server is still in the map
	_, exists := manager.GetMPSServer("card0")
	if !exists {
		t.Error("Expected server to still exist after health check")
	}
}

func TestAMDMPSManagerIntegration(t *testing.T) {
	tempDir := t.TempDir()
	config := MPSManagerConfig{
		LogBaseDirectory: tempDir,
	}

	manager := NewAMDMPSManager(config)

	// Test the complete flow
	ctx := context.Background()

	// Start multiple servers
	_, err1 := manager.StartMPSServer(ctx, "card0", nil)
	_, err2 := manager.StartMPSServer(ctx, "card1", nil)

	// We expect errors because hip-mps-server is not available
	if err1 == nil && err2 == nil {
		t.Log("Both MPS servers started successfully")

		// Test listing servers
		servers := manager.ListMPSServers()
		if len(servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(servers))
		}

		// Test getting connection info
		connInfo1, err := manager.GetMPSConnectionInfo("card0")
		if err != nil {
			t.Errorf("Failed to get connection info for card0: %v", err)
		} else {
			if connInfo1.GPUID != "card0" {
				t.Errorf("Expected GPU ID 'card0', got %s", connInfo1.GPUID)
			}
			if connInfo1.Host != "localhost" {
				t.Errorf("Expected host 'localhost', got %s", connInfo1.Host)
			}
			if connInfo1.Protocol != "tcp" {
				t.Errorf("Expected protocol 'tcp', got %s", connInfo1.Protocol)
			}
		}

		// Test stats
		stats := manager.GetMPSStats()
		if stats.TotalServers != 2 {
			t.Errorf("Expected 2 total servers, got %d", stats.TotalServers)
		}
		if stats.RunningServers != 2 {
			t.Errorf("Expected 2 running servers, got %d", stats.RunningServers)
		}

		// Test stopping servers
		err = manager.StopMPSServer("card0")
		if err != nil {
			t.Errorf("Failed to stop server card0: %v", err)
		}

		// Verify server was stopped
		_, exists := manager.GetMPSServer("card0")
		if exists {
			t.Error("Expected server card0 to be stopped")
		}

		// Test cleanup
		err = manager.Cleanup()
		if err != nil {
			t.Errorf("Failed to cleanup: %v", err)
		}

		// Verify all servers were cleaned up
		stats = manager.GetMPSStats()
		if stats.TotalServers != 0 {
			t.Errorf("Expected 0 servers after cleanup, got %d", stats.TotalServers)
		}
	} else {
		t.Logf("Expected errors (hip-mps-server not available): card0=%v, card1=%v", err1, err2)
	}
}
