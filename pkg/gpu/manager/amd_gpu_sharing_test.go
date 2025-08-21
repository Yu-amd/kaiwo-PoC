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
	"testing"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

func TestAMDGPUSharing(t *testing.T) {
	sharing := NewAMDGPUSharing()

	// Test capabilities
	capabilities := GetAMDGPUSharingCapabilities()
	if capabilities["hardware_partitioning"] != "advanced_on_mi300x" {
		t.Error("AMD GPU hardware partitioning should be advanced on MI300X")
	}
	if capabilities["time_slicing"] != true {
		t.Error("AMD GPUs should support time-slicing")
	}
	if capabilities["fractional_allocation"] != "hardware_on_mi300x" {
		t.Error("AMD GPU fractional allocation should be hardware-based on MI300X")
	}

	// Test allocation request
	request := &types.AllocationRequest{
		ID:            "test-allocation-1",
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
		GPURequest: &types.GPURequest{
			Fraction:       0.5,  // Used for scheduling priority, not hardware partitioning
			MemoryRequest:  2048, // 2GB
			IsolationType:  types.GPUIsolationTimeSlicing,
			SharingEnabled: true,
			Priority:       0,
		},
		Strategy:  types.AllocationStrategyFirstFit,
		Priority:  0,
		CreatedAt: time.Now(),
	}

	// Test allocation
	allocation, err := sharing.Allocate("card0", request)
	if err != nil {
		t.Fatalf("Failed to allocate GPU: %v", err)
	}

	if allocation.ID != request.ID {
		t.Errorf("Expected allocation ID '%s', got '%s'", request.ID, allocation.ID)
	}

	if allocation.Status != types.GPUAllocationStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", allocation.Status)
	}

	// Test memory usage tracking
	memoryUsage := sharing.GetMemoryUsage("card0")
	expectedMemory := int64(2048 * 1024 * 1024) // 2GB in bytes
	if memoryUsage != expectedMemory {
		t.Errorf("Expected memory usage %d, got %d", expectedMemory, memoryUsage)
	}

	// Test active allocations
	allocations := sharing.GetActiveAllocations("card0")
	if len(allocations) != 1 {
		t.Errorf("Expected 1 allocation, got %d", len(allocations))
	}

	// Test scheduler info
	scheduler := sharing.GetSchedulerInfo("card0")
	if scheduler == nil {
		t.Fatal("Expected scheduler info, got nil")
	}

	if len(scheduler.workloadQueue) != 1 {
		t.Errorf("Expected 1 workload in queue, got %d", len(scheduler.workloadQueue))
	}

	// Test time-slicing update
	// Initially, no workload should be active (time slice hasn't elapsed)
	sharing.UpdateScheduling("card0")

	// Check that the workload is in the queue but not yet active
	updatedScheduler := sharing.GetSchedulerInfo("card0")
	if updatedScheduler == nil {
		t.Fatal("Expected scheduler info, got nil")
	}

	if len(updatedScheduler.workloadQueue) != 1 {
		t.Errorf("Expected 1 workload in queue, got %d", len(updatedScheduler.workloadQueue))
	}

	// The workload should not be active yet (time slice hasn't elapsed)
	if updatedScheduler.activeWorkload != nil {
		t.Error("Expected no active workload initially (time slice hasn't elapsed)")
	}

	// Test release
	if err := sharing.Release("card0", request.ID); err != nil {
		t.Fatalf("Failed to release allocation: %v", err)
	}

	// Verify release
	allocations = sharing.GetActiveAllocations("card0")
	if len(allocations) != 0 {
		t.Errorf("Expected 0 allocations after release, got %d", len(allocations))
	}

	memoryUsage = sharing.GetMemoryUsage("card0")
	if memoryUsage != 0 {
		t.Errorf("Expected 0 memory usage after release, got %d", memoryUsage)
	}
}

func TestAMDGPUSharingMultipleWorkloads(t *testing.T) {
	sharing := NewAMDGPUSharing()

	// Create multiple allocation requests
	requests := []*types.AllocationRequest{
		{
			ID:        "allocation-1",
			PodName:   "pod-1",
			Namespace: "default",
			GPURequest: &types.GPURequest{
				Fraction:       0.3,
				MemoryRequest:  1024, // 1GB
				IsolationType:  types.GPUIsolationTimeSlicing,
				SharingEnabled: true,
			},
		},
		{
			ID:        "allocation-2",
			PodName:   "pod-2",
			Namespace: "default",
			GPURequest: &types.GPURequest{
				Fraction:       0.7,
				MemoryRequest:  2048, // 2GB
				IsolationType:  types.GPUIsolationTimeSlicing,
				SharingEnabled: true,
			},
		},
		{
			ID:        "allocation-3",
			PodName:   "pod-3",
			Namespace: "default",
			GPURequest: &types.GPURequest{
				Fraction:       0.5,
				MemoryRequest:  1536, // 1.5GB
				IsolationType:  types.GPUIsolationTimeSlicing,
				SharingEnabled: true,
			},
		},
	}

	// Allocate all workloads
	for _, request := range requests {
		_, err := sharing.Allocate("card0", request)
		if err != nil {
			t.Fatalf("Failed to allocate %s: %v", request.ID, err)
		}
	}

	// Check total memory usage
	memoryUsage := sharing.GetMemoryUsage("card0")
	expectedMemory := int64((1024 + 2048 + 1536) * 1024 * 1024) // 4.5GB in bytes
	if memoryUsage != expectedMemory {
		t.Errorf("Expected memory usage %d, got %d", expectedMemory, memoryUsage)
	}

	// Check scheduler queue
	scheduler := sharing.GetSchedulerInfo("card0")
	if len(scheduler.workloadQueue) != 3 {
		t.Errorf("Expected 3 workloads in queue, got %d", len(scheduler.workloadQueue))
	}

	// Test time-slicing with multiple workloads
	// Note: In a real scenario, time-slicing would happen over longer periods
	// For testing, we just verify the queue structure
	sharing.UpdateScheduling("card0")

	// Verify that workloads are in the queue
	updatedScheduler := sharing.GetSchedulerInfo("card0")
	if updatedScheduler == nil {
		t.Fatal("Expected scheduler info, got nil")
	}

	// All workloads should be in the queue (time slice hasn't elapsed in test)
	if len(updatedScheduler.workloadQueue) != 3 {
		t.Errorf("Expected 3 workloads in queue, got %d", len(updatedScheduler.workloadQueue))
	}
}

func TestAMDGPUSharingMemoryLimits(t *testing.T) {
	sharing := NewAMDGPUSharing()

	// Try to allocate more memory than available
	largeRequest := &types.AllocationRequest{
		ID:        "large-allocation",
		PodName:   "large-pod",
		Namespace: "default",
		GPURequest: &types.GPURequest{
			Fraction:       1.0,
			MemoryRequest:  16384, // 16GB (more than our 8GB default)
			IsolationType:  types.GPUIsolationTimeSlicing,
			SharingEnabled: true,
		},
	}

	// This should fail due to insufficient memory
	canAllocate, err := sharing.CanAllocate("card0", largeRequest.GPURequest)
	if err == nil {
		t.Fatalf("Expected error due to insufficient memory, got nil")
	}

	if canAllocate {
		t.Error("Expected allocation to fail due to insufficient memory")
	}

	// Try to allocate the large request
	_, err = sharing.Allocate("card0", largeRequest)
	if err == nil {
		t.Error("Expected allocation to fail due to insufficient memory")
	}
}

func TestAMDGPUSharingCapabilities(t *testing.T) {
	capabilities := GetAMDGPUSharingCapabilities()

	// Test that capabilities correctly reflect AMD GPU architecture differences
	expectedCapabilities := map[string]interface{}{
		"hardware_partitioning": "advanced_on_mi300x",
		"time_slicing":          true,
		"memory_management":     true,
		"time_slicing_support":  true,
		"fractional_allocation": "hardware_on_mi300x",
		"isolation":             "srivov_on_mi300x",
	}

	for key, expectedValue := range expectedCapabilities {
		if capabilities[key] != expectedValue {
			t.Errorf("Expected capability %s to be %v, got %v", key, expectedValue, capabilities[key])
		}
	}

	// Test that notes explain the limitations
	notes, exists := capabilities["notes"].([]string)
	if !exists {
		t.Fatal("Expected notes in capabilities")
	}

	if len(notes) == 0 {
		t.Error("Expected non-empty notes explaining AMD GPU limitations")
	}

	// Check for key architecture notes
	expectedNotes := []string{
		"AMD Instinct MI300X supports advanced hardware partitioning with 8 XCDs and SR-IOV isolation",
		"MI300X compute partitioning: SPX (single), CPX (8 separate GPUs)",
	}

	for _, expectedNote := range expectedNotes {
		found := false
		for _, note := range notes {
			if note == expectedNote {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected note not found: %s", expectedNote)
		}
	}
}
