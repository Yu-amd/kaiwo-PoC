package mps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// AMDMPSServer represents an AMD MPS server instance
type AMDMPSServer struct {
	ID           string
	GPUID        string
	Port         int
	Process      *os.Process
	StartTime    time.Time
	Status       MPSServerStatus
	Config       MPSServerConfig
	mu           sync.RWMutex
}

// MPSServerStatus represents the status of an MPS server
type MPSServerStatus string

const (
	MPSServerStatusStarting MPSServerStatus = "starting"
	MPSServerStatusRunning  MPSServerStatus = "running"
	MPSServerStatusStopped  MPSServerStatus = "stopped"
	MPSServerStatusError    MPSServerStatus = "error"
)

// MPSServerConfig contains configuration for an MPS server
type MPSServerConfig struct {
	GPUID           string
	Port            int
	LogLevel        string
	MaxConnections  int
	MemoryLimit     int64 // in bytes
	Timeout         time.Duration
	LogDirectory    string
	EnvironmentVars map[string]string
}

// AMDMPSManager manages AMD MPS servers for GPU sharing
type AMDMPSManager struct {
	servers map[string]*AMDMPSServer
	config  MPSManagerConfig
	mu      sync.RWMutex
}

// MPSManagerConfig contains configuration for the MPS manager
type MPSManagerConfig struct {
	DefaultPort         int
	DefaultLogLevel     string
	DefaultMaxConnections int
	DefaultMemoryLimit  int64
	DefaultTimeout      time.Duration
	LogBaseDirectory    string
	EnableMetrics       bool
	HealthCheckInterval time.Duration
}

// NewAMDMPSManager creates a new AMD MPS manager
func NewAMDMPSManager(config MPSManagerConfig) *AMDMPSManager {
	if config.DefaultPort == 0 {
		config.DefaultPort = 29500
	}
	if config.DefaultLogLevel == "" {
		config.DefaultLogLevel = "info"
	}
	if config.DefaultMaxConnections == 0 {
		config.DefaultMaxConnections = 16
	}
	if config.DefaultMemoryLimit == 0 {
		config.DefaultMemoryLimit = 8 * 1024 * 1024 * 1024 // 8GB
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.LogBaseDirectory == "" {
		config.LogBaseDirectory = "/tmp/kaiwo-mps"
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 10 * time.Second
	}

	manager := &AMDMPSManager{
		servers: make(map[string]*AMDMPSServer),
		config:  config,
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogBaseDirectory, 0755); err != nil {
		// Log error but continue - MPS can still work without log directory
		fmt.Printf("Warning: Failed to create MPS log directory %s: %v\n", config.LogBaseDirectory, err)
	}

	return manager
}

// StartMPSServer starts an MPS server for a specific GPU
func (m *AMDMPSManager) StartMPSServer(ctx context.Context, gpuID string, config *MPSServerConfig) (*AMDMPSServer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if server already exists for this GPU
	if server, exists := m.servers[gpuID]; exists && server.Status == MPSServerStatusRunning {
		return server, nil
	}

	// Use default config if not provided
	if config == nil {
		config = &MPSServerConfig{
			GPUID:           gpuID,
			Port:            m.config.DefaultPort,
			LogLevel:        m.config.DefaultLogLevel,
			MaxConnections:  m.config.DefaultMaxConnections,
			MemoryLimit:     m.config.DefaultMemoryLimit,
			Timeout:         m.config.DefaultTimeout,
			LogDirectory:    filepath.Join(m.config.LogBaseDirectory, gpuID),
			EnvironmentVars: make(map[string]string),
		}
	}

	// Create log directory for this GPU
	if err := os.MkdirAll(config.LogDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create MPS log directory: %w", err)
	}

	// Generate unique port if not specified
	if config.Port == 0 {
		config.Port = m.findAvailablePort()
	}

	server := &AMDMPSServer{
		ID:        fmt.Sprintf("mps-%s-%d", gpuID, config.Port),
		GPUID:     gpuID,
		Port:      config.Port,
		StartTime: time.Now(),
		Status:    MPSServerStatusStarting,
		Config:    *config,
	}

	// Set environment variables for MPS
	env := os.Environ()
	env = append(env, fmt.Sprintf("HIP_VISIBLE_DEVICES=%s", gpuID))
	env = append(env, fmt.Sprintf("ROCM_VISIBLE_DEVICES=%s", gpuID))
	env = append(env, fmt.Sprintf("HIP_MPS_ENABLE=1"))
	env = append(env, fmt.Sprintf("HIP_MPS_PORT=%d", config.Port))
	env = append(env, fmt.Sprintf("HIP_MPS_LOG_LEVEL=%s", config.LogLevel))
	env = append(env, fmt.Sprintf("HIP_MPS_MAX_CONNECTIONS=%d", config.MaxConnections))
	env = append(env, fmt.Sprintf("HIP_MPS_MEMORY_LIMIT=%d", config.MemoryLimit))

	// Add custom environment variables
	for key, value := range config.EnvironmentVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Start MPS server process
	cmd := exec.CommandContext(ctx, "hip-mps-server")
	cmd.Env = env
	cmd.Dir = config.LogDirectory

	// Redirect output to log files
	logFile, err := os.Create(filepath.Join(config.LogDirectory, "mps-server.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start MPS server: %w", err)
	}

	server.Process = cmd.Process
	m.servers[gpuID] = server

	// Start health check goroutine
	go m.healthCheck(server)

	// Wait a bit for server to start
	time.Sleep(2 * time.Second)

	// Check if server is running
	if server.Process != nil && server.Process.Pid > 0 {
		server.Status = MPSServerStatusRunning
	} else {
		server.Status = MPSServerStatusError
		return nil, fmt.Errorf("MPS server failed to start properly")
	}

	return server, nil
}

// StopMPSServer stops an MPS server for a specific GPU
func (m *AMDMPSManager) StopMPSServer(gpuID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, exists := m.servers[gpuID]
	if !exists {
		return fmt.Errorf("MPS server not found for GPU %s", gpuID)
	}

	if server.Process != nil {
		if err := server.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill MPS server process: %w", err)
		}
	}

	server.Status = MPSServerStatusStopped
	delete(m.servers, gpuID)

	return nil
}

// GetMPSServer returns the MPS server for a specific GPU
func (m *AMDMPSManager) GetMPSServer(gpuID string) (*AMDMPSServer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, exists := m.servers[gpuID]
	return server, exists
}

// ListMPSServers returns all MPS servers
func (m *AMDMPSManager) ListMPSServers() []*AMDMPSServer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]*AMDMPSServer, 0, len(m.servers))
	for _, server := range m.servers {
		servers = append(servers, server)
	}
	return servers
}

// GetMPSConnectionInfo returns connection information for MPS
func (m *AMDMPSManager) GetMPSConnectionInfo(gpuID string) (*types.MPSConnectionInfo, error) {
	server, exists := m.GetMPSServer(gpuID)
	if !exists {
		return nil, fmt.Errorf("MPS server not found for GPU %s", gpuID)
	}

	if server.Status != MPSServerStatusRunning {
		return nil, fmt.Errorf("MPS server is not running (status: %s)", server.Status)
	}

	return &types.MPSConnectionInfo{
		GPUID:     gpuID,
		Port:      server.Port,
		Host:      "localhost",
		Protocol:  "tcp",
		Status:    string(server.Status),
		StartTime: server.StartTime,
	}, nil
}

// healthCheck performs periodic health checks on MPS servers
func (m *AMDMPSManager) healthCheck(server *AMDMPSServer) {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			if server.Process != nil {
				// Check if process is still running
				if err := server.Process.Signal(os.Signal(nil)); err != nil {
					server.Status = MPSServerStatusError
					// Try to restart the server
					go m.restartMPSServer(server)
				}
			}
			m.mu.Unlock()
		}
	}
}

// restartMPSServer attempts to restart a failed MPS server
func (m *AMDMPSManager) restartMPSServer(server *AMDMPSServer) {
	// Remove from servers map
	m.mu.Lock()
	delete(m.servers, server.GPUID)
	m.mu.Unlock()

	// Wait a bit before restarting
	time.Sleep(5 * time.Second)

	// Try to restart
	_, err := m.StartMPSServer(context.Background(), server.GPUID, &server.Config)
	if err != nil {
		fmt.Printf("Failed to restart MPS server for GPU %s: %v\n", server.GPUID, err)
	}
}

// findAvailablePort finds an available port for MPS server
func (m *AMDMPSManager) findAvailablePort() int {
	// Simple port finding - start from default and increment
	port := m.config.DefaultPort
	for i := 0; i < 100; i++ {
		// Check if port is in use by checking if any server uses it
		portInUse := false
		for _, server := range m.servers {
			if server.Port == port {
				portInUse = true
				break
			}
		}
		if !portInUse {
			return port
		}
		port++
	}
	return m.config.DefaultPort // Fallback
}

// GetMPSStats returns statistics about MPS usage
func (m *AMDMPSManager) GetMPSStats() *types.MPSStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &types.MPSStats{
		TotalServers:    len(m.servers),
		RunningServers:  0,
		StoppedServers:  0,
		ErrorServers:    0,
		StartingServers: 0,
		ServerDetails:   make(map[string]types.MPSServerStats),
	}

	for gpuID, server := range m.servers {
		switch server.Status {
		case MPSServerStatusRunning:
			stats.RunningServers++
		case MPSServerStatusStopped:
			stats.StoppedServers++
		case MPSServerStatusError:
			stats.ErrorServers++
		case MPSServerStatusStarting:
			stats.StartingServers++
		}

		stats.ServerDetails[gpuID] = types.MPSServerStats{
			ID:        server.ID,
			Status:    string(server.Status),
			Port:      server.Port,
			StartTime: server.StartTime,
			Uptime:    time.Since(server.StartTime),
		}
	}

	return stats
}

// Cleanup stops all MPS servers and cleans up resources
func (m *AMDMPSManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for gpuID, server := range m.servers {
		if server.Process != nil {
			if err := server.Process.Kill(); err != nil {
				fmt.Printf("Warning: Failed to kill MPS server for GPU %s: %v\n", gpuID, err)
			}
		}
	}

	m.servers = make(map[string]*AMDMPSServer)
	return nil
}
