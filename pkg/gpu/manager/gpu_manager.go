// Copyright 2025 Advanced Micro Devices, Inc.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// GPUManager is the main interface for GPU management
type GPUManager interface {
	// Initialize initializes the GPU manager
	Initialize(ctx context.Context) error

	// Shutdown shuts down the GPU manager
	Shutdown(ctx context.Context) error

	// GetGPUType returns the GPU type managed by this manager
	GetGPUType() types.GPUType

	// ListGPUs lists all available GPUs
	ListGPUs(ctx context.Context) ([]*types.GPUInfo, error)

	// GetGPUInfo gets information about a specific GPU
	GetGPUInfo(ctx context.Context, deviceID string) (*types.GPUInfo, error)

	// AllocateGPU allocates a GPU for a request
	AllocateGPU(ctx context.Context, request *types.AllocationRequest) (*types.AllocationResult, error)

	// ReleaseGPU releases a GPU allocation
	ReleaseGPU(ctx context.Context, allocationID string) error

	// GetGPUStats gets GPU statistics
	GetGPUStats(ctx context.Context) (*types.GPUStats, error)

	// UpdateGPUInfo updates GPU information
	UpdateGPUInfo(ctx context.Context, deviceID string) error

	// ValidateAllocation validates if an allocation is possible
	ValidateAllocation(ctx context.Context, request *types.AllocationRequest) error

	// GetAllocation gets information about a specific allocation
	GetAllocation(ctx context.Context, allocationID string) (*types.GPUAllocation, error)

	// ListAllocations lists all active allocations
	ListAllocations(ctx context.Context) ([]*types.GPUAllocation, error)

	// GetMetrics gets allocation metrics
	GetMetrics(ctx context.Context) (*types.AllocationMetrics, error)
}

// GPUManagerConfig represents configuration for a GPU manager
type GPUManagerConfig struct {
	// GPUType is the type of GPU to manage
	GPUType types.GPUType `json:"gpuType"`

	// PollingInterval is the interval for polling GPU information
	PollingInterval time.Duration `json:"pollingInterval"`

	// AllocationTimeout is the timeout for GPU allocations
	AllocationTimeout time.Duration `json:"allocationTimeout"`

	// DefaultStrategy is the default allocation strategy
	DefaultStrategy types.AllocationStrategy `json:"defaultStrategy"`

	// EnableSharing indicates if GPU sharing is enabled
	EnableSharing bool `json:"enableSharing"`

	// MaxFraction is the maximum fractional allocation
	MaxFraction float64 `json:"maxFraction"`

	// MinFraction is the minimum fractional allocation
	MinFraction float64 `json:"minFraction"`

	// AllowedIsolationTypes is the list of allowed isolation types
	AllowedIsolationTypes []types.GPUIsolationType `json:"allowedIsolationTypes"`

	// NodeSelector is the node selector for GPU discovery
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// GPUManagerFactory creates GPU managers
type GPUManagerFactory interface {
	// CreateManager creates a new GPU manager
	CreateManager(config *GPUManagerConfig) (GPUManager, error)

	// GetSupportedTypes returns the supported GPU types
	GetSupportedTypes() []types.GPUType
}

// BaseGPUManager provides common functionality for GPU managers
type BaseGPUManager struct {
	config      *GPUManagerConfig
	allocations map[string]*types.GPUAllocation
	metrics     *types.AllocationMetrics
}

// NewBaseGPUManager creates a new base GPU manager
func NewBaseGPUManager(config *GPUManagerConfig) *BaseGPUManager {
	return &BaseGPUManager{
		config:      config,
		allocations: make(map[string]*types.GPUAllocation),
		metrics: &types.AllocationMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// GetConfig returns the manager configuration
func (b *BaseGPUManager) GetConfig() *GPUManagerConfig {
	return b.config
}

// GetGPUType returns the GPU type
func (b *BaseGPUManager) GetGPUType() types.GPUType {
	return b.config.GPUType
}

// ValidateAllocation validates if an allocation is possible
func (b *BaseGPUManager) ValidateAllocation(ctx context.Context, request *types.AllocationRequest) error {
	if request == nil {
		return fmt.Errorf("allocation request cannot be nil")
	}

	if err := types.ValidateAllocationRequest(request); err != nil {
		return fmt.Errorf("invalid allocation request: %v", err)
	}

	// Check if GPU sharing is enabled if requested
	if request.GPURequest.SharingEnabled && !b.config.EnableSharing {
		return fmt.Errorf("GPU sharing is not enabled")
	}

	// Check fraction limits
	if request.GPURequest.Fraction < b.config.MinFraction {
		return fmt.Errorf("GPU fraction %f is below minimum %f", request.GPURequest.Fraction, b.config.MinFraction)
	}

	if request.GPURequest.Fraction > b.config.MaxFraction {
		return fmt.Errorf("GPU fraction %f is above maximum %f", request.GPURequest.Fraction, b.config.MaxFraction)
	}

	// Check isolation type
	if !b.isIsolationTypeAllowed(request.GPURequest.IsolationType) {
		return fmt.Errorf("isolation type %s is not allowed", request.GPURequest.IsolationType)
	}

	return nil
}

// GetAllocation gets information about a specific allocation
func (b *BaseGPUManager) GetAllocation(ctx context.Context, allocationID string) (*types.GPUAllocation, error) {
	allocation, exists := b.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("allocation %s not found", allocationID)
	}

	return allocation, nil
}

// ListAllocations lists all active allocations
func (b *BaseGPUManager) ListAllocations(ctx context.Context) ([]*types.GPUAllocation, error) {
	allocations := make([]*types.GPUAllocation, 0, len(b.allocations))
	for _, allocation := range b.allocations {
		allocations = append(allocations, allocation)
	}

	return allocations, nil
}

// GetMetrics gets allocation metrics
func (b *BaseGPUManager) GetMetrics(ctx context.Context) (*types.AllocationMetrics, error) {
	// Update metrics
	b.updateMetrics()

	return b.metrics, nil
}

// ReleaseGPU releases a GPU allocation
func (b *BaseGPUManager) ReleaseGPU(ctx context.Context, allocationID string) error {
	allocation, exists := b.allocations[allocationID]
	if !exists {
		return fmt.Errorf("allocation %s not found", allocationID)
	}

	// Update allocation status
	allocation.Status = types.GPUAllocationStatusCompleted

	// Remove from active allocations
	delete(b.allocations, allocationID)

	// Update metrics
	b.metrics.ActiveAllocations--

	return nil
}

// isIsolationTypeAllowed checks if an isolation type is allowed
func (b *BaseGPUManager) isIsolationTypeAllowed(isolationType types.GPUIsolationType) bool {
	for _, allowed := range b.config.AllowedIsolationTypes {
		if allowed == isolationType {
			return true
		}
	}
	return false
}

// updateMetrics updates allocation metrics
func (b *BaseGPUManager) updateMetrics() {
	b.metrics.ActiveAllocations = int64(len(b.allocations))
	b.metrics.LastUpdated = time.Now()
}

// addAllocation adds an allocation to the manager
func (b *BaseGPUManager) addAllocation(allocation *types.GPUAllocation) {
	b.allocations[allocation.ID] = allocation
	b.metrics.ActiveAllocations++
	b.metrics.SuccessfulAllocations++
}

// DefaultGPUManagerFactory is the default GPU manager factory
type DefaultGPUManagerFactory struct{}

// NewDefaultGPUManagerFactory creates a new default GPU manager factory
func NewDefaultGPUManagerFactory() *DefaultGPUManagerFactory {
	return &DefaultGPUManagerFactory{}
}

// CreateManager creates a new GPU manager
func (f *DefaultGPUManagerFactory) CreateManager(config *GPUManagerConfig) (GPUManager, error) {
	switch config.GPUType {
	case types.GPUTypeAMD:
		return NewAMDGPUManager(config)
	default:
		return nil, fmt.Errorf("unsupported GPU type: %s", config.GPUType)
	}
}

// GetSupportedTypes returns the supported GPU types
func (f *DefaultGPUManagerFactory) GetSupportedTypes() []types.GPUType {
	return []types.GPUType{
		types.GPUTypeAMD,
	}
}

// ValidateGPUManagerConfig validates GPU manager configuration
func ValidateGPUManagerConfig(config *GPUManagerConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	switch config.GPUType {
	case types.GPUTypeAMD:
		// Valid GPU type
	default:
		return fmt.Errorf("unsupported GPU type: %s", config.GPUType)
	}

	if config.PollingInterval <= 0 {
		return fmt.Errorf("polling interval must be positive, got %v", config.PollingInterval)
	}

	if config.AllocationTimeout <= 0 {
		return fmt.Errorf("allocation timeout must be positive, got %v", config.AllocationTimeout)
	}

	switch config.DefaultStrategy {
	case types.AllocationStrategyFirstFit, types.AllocationStrategyBestFit, types.AllocationStrategyWorstFit,
		types.AllocationStrategyRoundRobin, types.AllocationStrategyLoadBalanced:
		// Valid strategy
	default:
		return fmt.Errorf("invalid default strategy: %s", config.DefaultStrategy)
	}

	if config.MaxFraction < 0.1 || config.MaxFraction > 1.0 {
		return fmt.Errorf("max fraction must be between 0.1 and 1.0, got %f", config.MaxFraction)
	}

	if config.MinFraction < 0.1 || config.MinFraction > 1.0 {
		return fmt.Errorf("min fraction must be between 0.1 and 1.0, got %f", config.MinFraction)
	}

	if config.MinFraction > config.MaxFraction {
		return fmt.Errorf("min fraction cannot be greater than max fraction")
	}

	if len(config.AllowedIsolationTypes) == 0 {
		return fmt.Errorf("at least one isolation type must be allowed")
	}

	for _, isolationType := range config.AllowedIsolationTypes {
		switch isolationType {
		case types.GPUIsolationTimeSlicing, types.GPUIsolationMIG, types.GPUIsolationNone:
			// Valid isolation type
		default:
			return fmt.Errorf("invalid isolation type: %s", isolationType)
		}
	}

	return nil
}
