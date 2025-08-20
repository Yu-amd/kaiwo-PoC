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
	"fmt"
	"math"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// FractionalAllocator manages fractional GPU allocations
type FractionalAllocator struct {
	// allocations tracks fractional allocations per GPU
	allocations map[string][]*types.GPUAllocation
	
	// gpuCapacity tracks the total capacity of each GPU
	gpuCapacity map[string]float64
	
	// gpuMemoryCapacity tracks the memory capacity of each GPU
	gpuMemoryCapacity map[string]int64
}

// NewFractionalAllocator creates a new fractional allocator
func NewFractionalAllocator() *FractionalAllocator {
	return &FractionalAllocator{
		allocations:      make(map[string][]*types.GPUAllocation),
		gpuCapacity:      make(map[string]float64),
		gpuMemoryCapacity: make(map[string]int64),
	}
}

// RegisterGPU registers a GPU with the fractional allocator
func (f *FractionalAllocator) RegisterGPU(deviceID string, totalMemory int64) {
	f.gpuCapacity[deviceID] = 1.0 // Full GPU capacity
	f.gpuMemoryCapacity[deviceID] = totalMemory
	f.allocations[deviceID] = make([]*types.GPUAllocation, 0)
}

// UnregisterGPU unregisters a GPU from the fractional allocator
func (f *FractionalAllocator) UnregisterGPU(deviceID string) {
	delete(f.gpuCapacity, deviceID)
	delete(f.gpuMemoryCapacity, deviceID)
	delete(f.allocations, deviceID)
}

// CanAllocate checks if a fractional allocation is possible
func (f *FractionalAllocator) CanAllocate(deviceID string, request *types.GPURequest) (bool, error) {
	if request == nil {
		return false, fmt.Errorf("GPU request cannot be nil")
	}
	
	if err := types.ValidateGPURequest(request); err != nil {
		return false, fmt.Errorf("invalid GPU request: %v", err)
	}
	
	// Check if GPU is registered
	if _, exists := f.gpuCapacity[deviceID]; !exists {
		return false, fmt.Errorf("GPU %s is not registered", deviceID)
	}
	
	// Check fractional capacity
	availableFraction := f.getAvailableFraction(deviceID)
	if request.Fraction > availableFraction {
		return false, fmt.Errorf("insufficient fractional capacity: requested %f, available %f", 
			request.Fraction, availableFraction)
	}
	
	// Check memory capacity
	if request.MemoryRequest > 0 {
		availableMemory := f.getAvailableMemory(deviceID)
		if request.MemoryRequest*1024*1024 > availableMemory { // Convert MiB to bytes
			return false, fmt.Errorf("insufficient memory: requested %d MiB, available %d bytes", 
				request.MemoryRequest, availableMemory)
		}
	}
	
	return true, nil
}

// Allocate performs a fractional allocation
func (f *FractionalAllocator) Allocate(deviceID string, request *types.AllocationRequest) (*types.GPUAllocation, error) {
	canAllocate, err := f.CanAllocate(deviceID, request.GPURequest)
	if err != nil {
		return nil, err
	}
	
	if !canAllocate {
		return nil, fmt.Errorf("cannot allocate on GPU %s", deviceID)
	}
	
	// Create allocation
	allocation := &types.GPUAllocation{
		ID:            request.ID,
		DeviceID:      deviceID,
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
	
	// Add allocation to the GPU
	f.allocations[deviceID] = append(f.allocations[deviceID], allocation)
	
	return allocation, nil
}

// Release releases a fractional allocation
func (f *FractionalAllocator) Release(allocationID string) error {
	for deviceID, allocations := range f.allocations {
		for i, allocation := range allocations {
			if allocation.ID == allocationID {
				// Remove allocation from slice
				f.allocations[deviceID] = append(allocations[:i], allocations[i+1:]...)
				return nil
			}
		}
	}
	
	return fmt.Errorf("allocation %s not found", allocationID)
}

// GetAvailableFraction returns the available fractional capacity for a GPU
func (f *FractionalAllocator) getAvailableFraction(deviceID string) float64 {
	totalCapacity := f.gpuCapacity[deviceID]
	usedCapacity := f.getUsedFraction(deviceID)
	
	available := totalCapacity - usedCapacity
	if available < 0 {
		available = 0
	}
	
	return available
}

// GetAvailableMemory returns the available memory for a GPU
func (f *FractionalAllocator) getAvailableMemory(deviceID string) int64 {
	totalMemory := f.gpuMemoryCapacity[deviceID]
	usedMemory := f.getUsedMemory(deviceID)
	
	available := totalMemory - usedMemory
	if available < 0 {
		available = 0
	}
	
	return available
}

// GetUsedFraction returns the used fractional capacity for a GPU
func (f *FractionalAllocator) getUsedFraction(deviceID string) float64 {
	allocations := f.allocations[deviceID]
	var used float64
	
	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			used += allocation.Fraction
		}
	}
	
	return used
}

// GetUsedMemory returns the used memory for a GPU
func (f *FractionalAllocator) getUsedMemory(deviceID string) int64 {
	allocations := f.allocations[deviceID]
	var used int64
	
	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			used += allocation.MemoryRequest * 1024 * 1024 // Convert MiB to bytes
		}
	}
	
	return used
}

// GetGPUUtilization returns the utilization statistics for a GPU
func (f *FractionalAllocator) GetGPUUtilization(deviceID string) *GPUUtilizationStats {
	allocations := f.allocations[deviceID]
	
	stats := &GPUUtilizationStats{
		DeviceID:           deviceID,
		TotalCapacity:      f.gpuCapacity[deviceID],
		TotalMemory:        f.gpuMemoryCapacity[deviceID],
		UsedFraction:       f.getUsedFraction(deviceID),
		UsedMemory:         f.getUsedMemory(deviceID),
		ActiveAllocations:  0,
		UtilizationRate:    0.0,
		MemoryUtilizationRate: 0.0,
	}
	
	// Count active allocations
	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			stats.ActiveAllocations++
		}
	}
	
	// Calculate utilization rates
	if stats.TotalCapacity > 0 {
		stats.UtilizationRate = stats.UsedFraction / stats.TotalCapacity
	}
	
	if stats.TotalMemory > 0 {
		stats.MemoryUtilizationRate = float64(stats.UsedMemory) / float64(stats.TotalMemory)
	}
	
	return stats
}

// FindBestFitGPU finds the GPU with the best fit for the allocation request
func (f *FractionalAllocator) FindBestFitGPU(request *types.GPURequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("GPU request cannot be nil")
	}
	
	var bestGPU string
	var bestScore float64 = math.MaxFloat64
	
	for deviceID := range f.gpuCapacity {
		canAllocate, err := f.CanAllocate(deviceID, request)
		if err != nil {
			continue // Skip this GPU if there's an error
		}
		
		if !canAllocate {
			continue // Skip this GPU if allocation is not possible
		}
		
		score := f.calculateFitScore(deviceID, request)
		if score < bestScore {
			bestScore = score
			bestGPU = deviceID
		}
	}
	
	if bestGPU == "" {
		return "", fmt.Errorf("no suitable GPU found for allocation")
	}
	
	return bestGPU, nil
}

// FindLoadBalancedGPU finds the GPU with the best load balance
func (f *FractionalAllocator) FindLoadBalancedGPU(request *types.GPURequest) (string, error) {
	if request == nil {
		return "", fmt.Errorf("GPU request cannot be nil")
	}
	
	var bestGPU string
	var bestLoad float64 = math.MaxFloat64
	
	for deviceID := range f.gpuCapacity {
		canAllocate, err := f.CanAllocate(deviceID, request)
		if err != nil {
			continue
		}
		
		if !canAllocate {
			continue
		}
		
		load := f.calculateLoadScore(deviceID)
		if load < bestLoad {
			bestLoad = load
			bestGPU = deviceID
		}
	}
	
	if bestGPU == "" {
		return "", fmt.Errorf("no suitable GPU found for allocation")
	}
	
	return bestGPU, nil
}

// calculateFitScore calculates a fit score for a GPU (lower is better)
func (f *FractionalAllocator) calculateFitScore(deviceID string, request *types.GPURequest) float64 {
	stats := f.GetGPUUtilization(deviceID)
	
	// Calculate fit score based on utilization and available resources
	utilizationScore := stats.UtilizationRate
	memoryScore := stats.MemoryUtilizationRate
	
	// Weight the scores (you can adjust these weights)
	fitScore := utilizationScore*0.6 + memoryScore*0.4
	
	return fitScore
}

// calculateLoadScore calculates a load score for a GPU (lower is better)
func (f *FractionalAllocator) calculateLoadScore(deviceID string) float64 {
	stats := f.GetGPUUtilization(deviceID)
	
	// Calculate load score based on utilization and number of allocations
	utilizationScore := stats.UtilizationRate
	allocationScore := float64(stats.ActiveAllocations) / 10.0 // Normalize to 0-1
	
	// Weight the scores
	loadScore := utilizationScore*0.7 + allocationScore*0.3
	
	return loadScore
}

// CleanupExpiredAllocations removes expired allocations
func (f *FractionalAllocator) CleanupExpiredAllocations() {
	now := time.Now().Unix()
	
	for deviceID, allocations := range f.allocations {
		var validAllocations []*types.GPUAllocation
		
		for _, allocation := range allocations {
			if allocation.ExpiresAt > 0 && allocation.ExpiresAt <= now {
				// Mark as expired
				allocation.Status = types.GPUAllocationStatusExpired
			} else {
				validAllocations = append(validAllocations, allocation)
			}
		}
		
		f.allocations[deviceID] = validAllocations
	}
}

// GetGPUAllocations returns all allocations for a GPU
func (f *FractionalAllocator) GetGPUAllocations(deviceID string) []*types.GPUAllocation {
	allocations, exists := f.allocations[deviceID]
	if !exists {
		return []*types.GPUAllocation{}
	}
	
	// Return a copy to avoid external modifications
	result := make([]*types.GPUAllocation, len(allocations))
	copy(result, allocations)
	
	return result
}

// GetAllGPUAllocations returns all allocations across all GPUs
func (f *FractionalAllocator) GetAllGPUAllocations() map[string][]*types.GPUAllocation {
	result := make(map[string][]*types.GPUAllocation)
	
	for deviceID, allocations := range f.allocations {
		result[deviceID] = make([]*types.GPUAllocation, len(allocations))
		copy(result[deviceID], allocations)
	}
	
	return result
}

// GPUUtilizationStats represents utilization statistics for a GPU
type GPUUtilizationStats struct {
	DeviceID              string  `json:"deviceId"`
	TotalCapacity         float64 `json:"totalCapacity"`
	TotalMemory           int64   `json:"totalMemory"`
	UsedFraction          float64 `json:"usedFraction"`
	UsedMemory            int64   `json:"usedMemory"`
	ActiveAllocations     int     `json:"activeAllocations"`
	UtilizationRate       float64 `json:"utilizationRate"`
	MemoryUtilizationRate float64 `json:"memoryUtilizationRate"`
}

// GetUtilizationStats returns utilization statistics for all GPUs
func (f *FractionalAllocator) GetUtilizationStats() map[string]*GPUUtilizationStats {
	stats := make(map[string]*GPUUtilizationStats)
	
	for deviceID := range f.gpuCapacity {
		stats[deviceID] = f.GetGPUUtilization(deviceID)
	}
	
	return stats
}
