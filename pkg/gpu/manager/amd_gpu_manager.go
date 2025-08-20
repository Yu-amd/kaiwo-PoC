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
	"strings"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// AMDGPUManager manages AMD GPUs
type AMDGPUManager struct {
	*BaseGPUManager
	gpus map[string]*types.GPUInfo
	lastUpdate time.Time
}

// NewAMDGPUManager creates a new AMD GPU manager
func NewAMDGPUManager(config *GPUManagerConfig) (*AMDGPUManager, error) {
	if config.GPUType != types.GPUTypeAMD {
		return nil, fmt.Errorf("expected AMD GPU type, got %s", config.GPUType)
	}
	
	if err := ValidateGPUManagerConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}
	
	return &AMDGPUManager{
		BaseGPUManager: NewBaseGPUManager(config),
		gpus: make(map[string]*types.GPUInfo),
		lastUpdate: time.Now(),
	}, nil
}

// Initialize initializes the AMD GPU manager
func (a *AMDGPUManager) Initialize(ctx context.Context) error {
	// Discover AMD GPUs
	if err := a.discoverGPUs(ctx); err != nil {
		return fmt.Errorf("failed to discover GPUs: %v", err)
	}
	
	// Start GPU monitoring
	go a.monitorGPUs(ctx)
	
	return nil
}

// Shutdown shuts down the AMD GPU manager
func (a *AMDGPUManager) Shutdown(ctx context.Context) error {
	// Release all allocations
	for allocationID := range a.BaseGPUManager.allocations {
		if err := a.ReleaseGPU(ctx, allocationID); err != nil {
			// Log error but continue
			fmt.Printf("Error releasing allocation %s: %v\n", allocationID, err)
		}
	}
	
	return nil
}

// ListGPUs lists all available AMD GPUs
func (a *AMDGPUManager) ListGPUs(ctx context.Context) ([]*types.GPUInfo, error) {
	// Update GPU information if needed
	if time.Since(a.lastUpdate) > a.config.PollingInterval {
		if err := a.updateGPUInfo(ctx); err != nil {
			return nil, fmt.Errorf("failed to update GPU info: %v", err)
		}
	}
	
	gpus := make([]*types.GPUInfo, 0, len(a.gpus))
	for _, gpu := range a.gpus {
		gpus = append(gpus, gpu)
	}
	
	return gpus, nil
}

	// GetGPUInfo gets information about a specific AMD GPU
	func (a *AMDGPUManager) GetGPUInfo(ctx context.Context, deviceID string) (*types.GPUInfo, error) {
		_, exists := a.gpus[deviceID]
		if !exists {
			return nil, fmt.Errorf("GPU %s not found", deviceID)
		}
		
		// Update GPU information
		if err := a.updateSingleGPUInfo(ctx, deviceID); err != nil {
			return nil, fmt.Errorf("failed to update GPU info: %v", err)
		}
		
		return a.gpus[deviceID], nil
	}

// AllocateGPU allocates an AMD GPU for a request
func (a *AMDGPUManager) AllocateGPU(ctx context.Context, request *types.AllocationRequest) (*types.AllocationResult, error) {
	// Validate the request
	if err := a.ValidateAllocation(ctx, request); err != nil {
		return nil, fmt.Errorf("invalid allocation request: %v", err)
	}
	
	// Find available GPU
	selectedGPU, err := a.findAvailableGPU(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to find available GPU: %v", err)
	}
	
	// Create allocation
	allocation := &types.GPUAllocation{
		ID:            request.ID,
		DeviceID:      selectedGPU.DeviceID,
		Fraction:      request.GPURequest.Fraction,
		MemoryRequest: request.GPURequest.MemoryRequest,
		IsolationType: request.GPURequest.IsolationType,
		PodName:       request.PodName,
		Namespace:     request.Namespace,
		ContainerName: request.ContainerName,
		Status:        types.GPUAllocationStatusActive,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     0, // No expiration by default
	}
	
	// Set expiration if specified
	if request.ExpiresAt != nil {
		allocation.ExpiresAt = request.ExpiresAt.Unix()
	}
	
	// Add allocation to manager
	a.addAllocation(allocation)
	
	// Update GPU information
	selectedGPU.ActiveAllocations++
	selectedGPU.IsAvailable = a.isGPUAvailable(selectedGPU)
	
	// Create result
	result := &types.AllocationResult{
		Success:      true,
		Allocation:   allocation,
		DeviceID:     selectedGPU.DeviceID,
		NodeName:     selectedGPU.NodeName,
		AllocatedAt:  time.Now(),
	}
	
	return result, nil
}

// GetGPUStats gets AMD GPU statistics
func (a *AMDGPUManager) GetGPUStats(ctx context.Context) (*types.GPUStats, error) {
	gpus, err := a.ListGPUs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list GPUs: %v", err)
	}
	
	stats := &types.GPUStats{
		TotalGPUs:           len(gpus),
		AvailableGPUs:       0,
		TotalMemory:         0,
		AvailableMemory:     0,
		AverageUtilization:  0,
		AverageTemperature:  0,
		AveragePower:        0,
		ActiveAllocations:   int(a.metrics.ActiveAllocations),
	}
	
	if len(gpus) == 0 {
		return stats, nil
	}
	
	var totalUtilization, totalTemperature, totalPower float64
	
	for _, gpu := range gpus {
		if gpu.IsAvailable {
			stats.AvailableGPUs++
		}
		
		stats.TotalMemory += gpu.TotalMemory
		stats.AvailableMemory += gpu.AvailableMemory
		totalUtilization += gpu.Utilization
		totalTemperature += gpu.Temperature
		totalPower += gpu.Power
	}
	
	stats.AverageUtilization = totalUtilization / float64(len(gpus))
	stats.AverageTemperature = totalTemperature / float64(len(gpus))
	stats.AveragePower = totalPower / float64(len(gpus))
	
	return stats, nil
}

// UpdateGPUInfo updates AMD GPU information
func (a *AMDGPUManager) UpdateGPUInfo(ctx context.Context, deviceID string) error {
	return a.updateSingleGPUInfo(ctx, deviceID)
}

// discoverGPUs discovers AMD GPUs in the system
func (a *AMDGPUManager) discoverGPUs(ctx context.Context) error {
	// This is a simplified implementation
	// In practice, you would use AMD ROCm tools or system calls to discover GPUs
	
	// For now, we'll create some mock GPUs for testing
	mockGPUs := []*types.GPUInfo{
		{
			DeviceID:           "card0",
			Type:               types.GPUTypeAMD,
			Model:              "AMD Instinct MI250X",
			TotalMemory:        128 * 1024 * 1024 * 1024, // 128 GB
			AvailableMemory:    128 * 1024 * 1024 * 1024,
			Utilization:        0.0,
			Temperature:        45.0,
			Power:              0.0,
			NodeName:           "node-1",
			IsAvailable:        true,
			IsolationType:      types.GPUIsolationNone,
			ActiveAllocations:  0,
		},
		{
			DeviceID:           "card1",
			Type:               types.GPUTypeAMD,
			Model:              "AMD Instinct MI250X",
			TotalMemory:        128 * 1024 * 1024 * 1024, // 128 GB
			AvailableMemory:    128 * 1024 * 1024 * 1024,
			Utilization:        0.0,
			Temperature:        45.0,
			Power:              0.0,
			NodeName:           "node-1",
			IsAvailable:        true,
			IsolationType:      types.GPUIsolationNone,
			ActiveAllocations:  0,
		},
	}
	
	for _, gpu := range mockGPUs {
		a.gpus[gpu.DeviceID] = gpu
	}
	
	return nil
}

// updateGPUInfo updates information for all GPUs
func (a *AMDGPUManager) updateGPUInfo(ctx context.Context) error {
	for deviceID := range a.gpus {
		if err := a.updateSingleGPUInfo(ctx, deviceID); err != nil {
			// Log error but continue with other GPUs
			fmt.Printf("Error updating GPU %s: %v\n", deviceID, err)
		}
	}
	
	a.lastUpdate = time.Now()
	return nil
}

// updateSingleGPUInfo updates information for a single GPU
func (a *AMDGPUManager) updateSingleGPUInfo(ctx context.Context, deviceID string) error {
	gpu, exists := a.gpus[deviceID]
	if !exists {
		return fmt.Errorf("GPU %s not found", deviceID)
	}
	
	// This is a simplified implementation
	// In practice, you would use AMD ROCm tools or system calls to get GPU information
	
	// Mock GPU information update
	gpu.Utilization = a.getMockGPUUtilization(deviceID)
	gpu.Temperature = a.getMockGPUTemperature(deviceID)
	gpu.Power = a.getMockGPUPower(deviceID)
	gpu.AvailableMemory = a.getMockGPUAvailableMemory(deviceID)
	gpu.IsAvailable = a.isGPUAvailable(gpu)
	
	return nil
}

// findAvailableGPU finds an available GPU for allocation
func (a *AMDGPUManager) findAvailableGPU(ctx context.Context, request *types.AllocationRequest) (*types.GPUInfo, error) {
	gpus, err := a.ListGPUs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list GPUs: %v", err)
	}
	
	// Filter available GPUs
	var availableGPUs []*types.GPUInfo
	for _, gpu := range gpus {
		if gpu.IsAvailable && a.canGPUHandleRequest(gpu, request) {
			availableGPUs = append(availableGPUs, gpu)
		}
	}
	
	if len(availableGPUs) == 0 {
		return nil, fmt.Errorf("no available GPUs found for request")
	}
	
	// Apply allocation strategy
	switch request.Strategy {
	case types.AllocationStrategyFirstFit:
		return availableGPUs[0], nil
	case types.AllocationStrategyBestFit:
		return a.findBestFitGPU(availableGPUs, request)
	case types.AllocationStrategyWorstFit:
		return a.findWorstFitGPU(availableGPUs, request)
	case types.AllocationStrategyRoundRobin:
		return a.findRoundRobinGPU(availableGPUs, request)
	case types.AllocationStrategyLoadBalanced:
		return a.findLoadBalancedGPU(availableGPUs, request)
	default:
		return availableGPUs[0], nil
	}
}

// canGPUHandleRequest checks if a GPU can handle the allocation request
func (a *AMDGPUManager) canGPUHandleRequest(gpu *types.GPUInfo, request *types.AllocationRequest) bool {
	// Check if GPU has enough memory
	if request.GPURequest.MemoryRequest > 0 {
		if gpu.AvailableMemory < request.GPURequest.MemoryRequest*1024*1024 { // Convert MiB to bytes
			return false
		}
	}
	
	// Check if GPU can handle the fraction
	// This is a simplified check - in practice, you'd need to check current allocations
	if request.GPURequest.Fraction > 1.0 {
		return false
	}
	
	return true
}

// isGPUAvailable checks if a GPU is available for allocation
func (a *AMDGPUManager) isGPUAvailable(gpu *types.GPUInfo) bool {
	// Check if GPU is healthy
	if gpu.Temperature > 90.0 { // Overheating
		return false
	}
	
	// Check if GPU has too many allocations
	if gpu.ActiveAllocations >= 10 { // Arbitrary limit
		return false
	}
	
	return true
}

// findBestFitGPU finds the GPU with the best fit for the request
func (a *AMDGPUManager) findBestFitGPU(gpus []*types.GPUInfo, request *types.AllocationRequest) (*types.GPUInfo, error) {
	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs available")
	}
	
	bestGPU := gpus[0]
	bestScore := a.calculateFitScore(bestGPU, request)
	
	for _, gpu := range gpus[1:] {
		score := a.calculateFitScore(gpu, request)
		if score < bestScore {
			bestScore = score
			bestGPU = gpu
		}
	}
	
	return bestGPU, nil
}

// findWorstFitGPU finds the GPU with the worst fit for the request
func (a *AMDGPUManager) findWorstFitGPU(gpus []*types.GPUInfo, request *types.AllocationRequest) (*types.GPUInfo, error) {
	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs available")
	}
	
	worstGPU := gpus[0]
	worstScore := a.calculateFitScore(worstGPU, request)
	
	for _, gpu := range gpus[1:] {
		score := a.calculateFitScore(gpu, request)
		if score > worstScore {
			worstScore = score
			worstGPU = gpu
		}
	}
	
	return worstGPU, nil
}

// findRoundRobinGPU finds the next GPU in round-robin fashion
func (a *AMDGPUManager) findRoundRobinGPU(gpus []*types.GPUInfo, request *types.AllocationRequest) (*types.GPUInfo, error) {
	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs available")
	}
	
	// Simple round-robin implementation
	// In practice, you'd maintain a counter across requests
	return gpus[0], nil
}

// findLoadBalancedGPU finds the GPU with the best load balance
func (a *AMDGPUManager) findLoadBalancedGPU(gpus []*types.GPUInfo, request *types.AllocationRequest) (*types.GPUInfo, error) {
	if len(gpus) == 0 {
		return nil, fmt.Errorf("no GPUs available")
	}
	
	bestGPU := gpus[0]
	bestLoad := a.calculateLoadScore(bestGPU)
	
	for _, gpu := range gpus[1:] {
		load := a.calculateLoadScore(gpu)
		if load < bestLoad {
			bestLoad = load
			bestGPU = gpu
		}
	}
	
	return bestGPU, nil
}

// calculateFitScore calculates a fit score for a GPU (lower is better)
func (a *AMDGPUManager) calculateFitScore(gpu *types.GPUInfo, request *types.AllocationRequest) float64 {
	// Simple fit score based on utilization and available memory
	utilizationScore := gpu.Utilization / 100.0
	memoryScore := float64(gpu.AvailableMemory) / float64(gpu.TotalMemory)
	
	return utilizationScore + (1.0 - memoryScore)
}

// calculateLoadScore calculates a load score for a GPU (lower is better)
func (a *AMDGPUManager) calculateLoadScore(gpu *types.GPUInfo) float64 {
	// Simple load score based on utilization and active allocations
	utilizationScore := gpu.Utilization / 100.0
	allocationScore := float64(gpu.ActiveAllocations) / 10.0 // Normalize to 0-1
	
	return utilizationScore + allocationScore
}

// monitorGPUs monitors GPU health and performance
func (a *AMDGPUManager) monitorGPUs(ctx context.Context) {
	ticker := time.NewTicker(a.config.PollingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.updateGPUInfo(ctx); err != nil {
				fmt.Printf("Error updating GPU info: %v\n", err)
			}
		}
	}
}

// Mock functions for GPU information (replace with real implementations)

func (a *AMDGPUManager) getMockGPUUtilization(deviceID string) float64 {
	// Mock utilization based on device ID
	if strings.Contains(deviceID, "card0") {
		return 25.0
	}
	return 15.0
}

func (a *AMDGPUManager) getMockGPUTemperature(deviceID string) float64 {
	// Mock temperature based on device ID
	if strings.Contains(deviceID, "card0") {
		return 65.0
	}
	return 55.0
}

func (a *AMDGPUManager) getMockGPUPower(deviceID string) float64 {
	// Mock power consumption based on device ID
	if strings.Contains(deviceID, "card0") {
		return 150.0
	}
	return 120.0
}

func (a *AMDGPUManager) getMockGPUAvailableMemory(deviceID string) int64 {
	// Mock available memory based on device ID
	if strings.Contains(deviceID, "card0") {
		return 100 * 1024 * 1024 * 1024 // 100 GB
	}
	return 110 * 1024 * 1024 * 1024 // 110 GB
}
