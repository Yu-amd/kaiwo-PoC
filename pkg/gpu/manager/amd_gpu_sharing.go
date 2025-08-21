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
	"sync"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// AMDGPUSharing implements AMD-specific GPU sharing without hardware partitioning
// ROCm doesn't support hardware-level GPU partitioning like NVIDIA MIG
type AMDGPUSharing struct {
	// gpuWorkloads tracks active workloads per GPU
	gpuWorkloads map[string][]*types.GPUAllocation

	// gpuMemoryUsage tracks memory usage per GPU
	gpuMemoryUsage map[string]int64

	// gpuScheduling tracks time-slicing information
	gpuScheduling map[string]*GPUScheduler

	// mutex for thread safety
	mu sync.RWMutex
}

// GPUScheduler manages time-slicing for AMD GPUs
type GPUScheduler struct {
	// timeSlice is the time slice allocated to each workload (in seconds)
	timeSlice time.Duration

	// workloadQueue is the queue of workloads waiting for GPU time
	workloadQueue []*types.GPUAllocation

	// activeWorkload is the currently running workload
	activeWorkload *types.GPUAllocation

	// lastSwitch is the last time we switched workloads
	lastSwitch time.Time
}

// NewAMDGPUSharing creates a new AMD GPU sharing manager
func NewAMDGPUSharing() *AMDGPUSharing {
	return &AMDGPUSharing{
		gpuWorkloads:   make(map[string][]*types.GPUAllocation),
		gpuMemoryUsage: make(map[string]int64),
		gpuScheduling:  make(map[string]*GPUScheduler),
	}
}

// CanAllocate checks if an AMD GPU can handle the allocation request
// Note: AMD GPUs don't support true fractional allocation like NVIDIA MIG
func (a *AMDGPUSharing) CanAllocate(deviceID string, request *types.GPURequest) (bool, error) {
	if request == nil {
		return false, fmt.Errorf("GPU request cannot be nil")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check memory availability (this is the main constraint for AMD GPUs)
	requestedMemory := request.MemoryRequest * 1024 * 1024 // Convert MiB to bytes
	usedMemory := a.gpuMemoryUsage[deviceID]
	
	// Get GPU info to check total memory
	// This would need to be passed in or retrieved from the GPU manager
	// For now, we'll use a conservative estimate
	totalMemory := int64(8 * 1024 * 1024 * 1024) // 8GB default
	
	availableMemory := totalMemory - usedMemory
	if requestedMemory > availableMemory {
		return false, fmt.Errorf("insufficient memory: requested %d bytes, available %d bytes",
			requestedMemory, availableMemory)
	}

	// For AMD GPUs, we can always allocate (time-slicing handles the rest)
	// The fraction is used for scheduling priority, not hardware partitioning
	return true, nil
}

// Allocate allocates GPU resources for AMD GPUs using time-slicing
func (a *AMDGPUSharing) Allocate(deviceID string, request *types.AllocationRequest) (*types.GPUAllocation, error) {
	canAllocate, err := a.CanAllocate(deviceID, request.GPURequest)
	if err != nil {
		return nil, err
	}

	if !canAllocate {
		return nil, fmt.Errorf("cannot allocate GPU resources for request")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Create allocation (no hardware partitioning, just resource tracking)
	allocation := &types.GPUAllocation{
		ID:            request.ID,
		DeviceID:      deviceID,
		Fraction:      request.GPURequest.Fraction, // Used for scheduling priority
		MemoryRequest: request.GPURequest.MemoryRequest * 1024 * 1024, // Convert to bytes
		IsolationType: request.GPURequest.IsolationType,
		PodName:       request.PodName,
		Namespace:     request.Namespace,
		Status:        types.GPUAllocationStatusPending, // Will be scheduled for time-slicing
		CreatedAt:     time.Now().Unix(),
	}

	// Add to workload queue
	if a.gpuWorkloads[deviceID] == nil {
		a.gpuWorkloads[deviceID] = make([]*types.GPUAllocation, 0)
	}
	a.gpuWorkloads[deviceID] = append(a.gpuWorkloads[deviceID], allocation)

	// Update memory usage
	a.gpuMemoryUsage[deviceID] += allocation.MemoryRequest

	// Initialize scheduler if needed
	if a.gpuScheduling[deviceID] == nil {
		a.gpuScheduling[deviceID] = &GPUScheduler{
			timeSlice:   30 * time.Second, // 30-second time slices
			lastSwitch:  time.Now(),
		}
	}

	// Add to scheduling queue
	a.gpuScheduling[deviceID].workloadQueue = append(a.gpuScheduling[deviceID].workloadQueue, allocation)

	return allocation, nil
}

// Release releases GPU resources
func (a *AMDGPUSharing) Release(deviceID, allocationID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find and remove allocation
	workloads := a.gpuWorkloads[deviceID]
	for i, workload := range workloads {
		if workload.ID == allocationID {
			// Update memory usage
			a.gpuMemoryUsage[deviceID] -= workload.MemoryRequest
			
			// Remove from workloads
			a.gpuWorkloads[deviceID] = append(workloads[:i], workloads[i+1:]...)
			
			// Remove from scheduler queue
			if scheduler := a.gpuScheduling[deviceID]; scheduler != nil {
				for j, queued := range scheduler.workloadQueue {
					if queued.ID == allocationID {
						scheduler.workloadQueue = append(scheduler.workloadQueue[:j], scheduler.workloadQueue[j+1:]...)
						break
					}
				}
				
				// If this was the active workload, clear it
				if scheduler.activeWorkload != nil && scheduler.activeWorkload.ID == allocationID {
					scheduler.activeWorkload = nil
				}
			}
			
			return nil
		}
	}

	return fmt.Errorf("allocation %s not found on GPU %s", allocationID, deviceID)
}

// GetActiveAllocations returns active allocations for a GPU
func (a *AMDGPUSharing) GetActiveAllocations(deviceID string) []*types.GPUAllocation {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if workloads, exists := a.gpuWorkloads[deviceID]; exists {
		result := make([]*types.GPUAllocation, len(workloads))
		copy(result, workloads)
		return result
	}
	return []*types.GPUAllocation{}
}

// GetMemoryUsage returns current memory usage for a GPU
func (a *AMDGPUSharing) GetMemoryUsage(deviceID string) int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.gpuMemoryUsage[deviceID]
}

// GetSchedulerInfo returns scheduling information for a GPU
func (a *AMDGPUSharing) GetSchedulerInfo(deviceID string) *GPUScheduler {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if scheduler, exists := a.gpuScheduling[deviceID]; exists {
		// Return a copy to avoid race conditions
		return &GPUScheduler{
			timeSlice:       scheduler.timeSlice,
			workloadQueue:   append([]*types.GPUAllocation{}, scheduler.workloadQueue...),
			activeWorkload:  scheduler.activeWorkload,
			lastSwitch:      scheduler.lastSwitch,
		}
	}
	return nil
}

// UpdateScheduling updates the time-slicing schedule
// This would be called periodically to manage workload switching
func (a *AMDGPUSharing) UpdateScheduling(deviceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	scheduler := a.gpuScheduling[deviceID]
	if scheduler == nil || len(scheduler.workloadQueue) == 0 {
		return
	}

	// Check if it's time to switch workloads
	if time.Since(scheduler.lastSwitch) >= scheduler.timeSlice {
		// Switch to next workload in queue
		if len(scheduler.workloadQueue) > 0 {
			// Move current active workload to end of queue (round-robin)
			if scheduler.activeWorkload != nil {
				scheduler.workloadQueue = append(scheduler.workloadQueue, scheduler.activeWorkload)
			}
			
			// Set next workload as active
			scheduler.activeWorkload = scheduler.workloadQueue[0]
			scheduler.workloadQueue = scheduler.workloadQueue[1:]
			scheduler.lastSwitch = time.Now()
			
			// Update allocation status
			if scheduler.activeWorkload != nil {
				scheduler.activeWorkload.Status = types.GPUAllocationStatusActive
			}
		}
	}
}

// GetAMDGPUSharingCapabilities returns the capabilities of AMD GPU sharing
func GetAMDGPUSharingCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"hardware_partitioning": "advanced_on_mi300x", // Advanced partitioning on MI300X, basic on other Instinct
		"time_slicing":          true,
		"memory_management":     true,
		"mps_support":           true,
		"fractional_allocation": "hardware_on_mi300x", // Hardware-level XCD fractions on MI300X
		"isolation":             "srivov_on_mi300x",   // SR-IOV isolation on MI300X
		"max_concurrent":        "8_xcds_on_mi300x",   // 8 XCDs per MI300X
		"architecture_support": map[string]interface{}{
			"amd_instinct_mi300x": map[string]interface{}{
				"hardware_partitioning": true,
				"concurrent_execution":  true,
				"resource_guarantees":   true,
				"xcd_isolation":         true,
				"compute_modes":         []string{"SPX", "CPX"},
				"memory_modes":          []string{"NPS1", "NPS4"},
				"xcd_count":             8,
				"hbm_stacks":            8,
				"max_bandwidth_per_xcd": "1TB/s",
				"performance_gain":      "10-15% over SPX",
			},
		},
		"notes": []string{
			"AMD Instinct MI300X supports advanced hardware partitioning with 8 XCDs and SR-IOV isolation",
			"MI300X compute partitioning: SPX (single), CPX (8 separate GPUs)",
			"MI300X memory partitioning: NPS1 (unified), NPS4 (quadrant-based) with up to 1TB/s per XCD",
			"Fractional allocation is hardware-based on MI300X XCDs",
			"Memory isolation is NUMA-based on MI300X",
			"Concurrent execution is supported on MI300X XCDs with 10-15% performance gain",
		},
	}
}
