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
	"testing"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

func TestAMDGPUManager(t *testing.T) {
	// Create configuration
	config := &GPUManagerConfig{
		GPUType:               types.GPUTypeAMD,
		PollingInterval:       30 * time.Second,
		AllocationTimeout:     5 * time.Minute,
		DefaultStrategy:       types.AllocationStrategyFirstFit,
		EnableSharing:         true,
		MaxFraction:           1.0,
		MinFraction:           0.1,
		AllowedIsolationTypes: []types.GPUIsolationType{types.GPUIsolationMPS, types.GPUIsolationNone},
	}
	
	// Create AMD GPU manager
	manager, err := NewAMDGPUManager(config)
	if err != nil {
		t.Fatalf("Failed to create AMD GPU manager: %v", err)
	}
	
	// Initialize manager
	ctx := context.Background()
	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Test listing GPUs
	gpus, err := manager.ListGPUs(ctx)
	if err != nil {
		t.Fatalf("Failed to list GPUs: %v", err)
	}
	
	if len(gpus) == 0 {
		t.Fatal("Expected at least one GPU, got none")
	}
	
	// Test getting GPU info
	gpuInfo, err := manager.GetGPUInfo(ctx, "card0")
	if err != nil {
		t.Fatalf("Failed to get GPU info: %v", err)
	}
	
	if gpuInfo.DeviceID != "card0" {
		t.Errorf("Expected device ID 'card0', got '%s'", gpuInfo.DeviceID)
	}
	
	if gpuInfo.Type != types.GPUTypeAMD {
		t.Errorf("Expected GPU type AMD, got %s", gpuInfo.Type)
	}
	
	// Test allocation request
	request := &types.AllocationRequest{
		ID:            "test-allocation-1",
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
		GPURequest: &types.GPURequest{
			Fraction:        0.5,
			MemoryRequest:   4000, // 4GB
			IsolationType:   types.GPUIsolationMPS,
			SharingEnabled:  true,
			Priority:        0,
		},
		Strategy: types.AllocationStrategyFirstFit,
		Priority: 0,
		CreatedAt: time.Now(),
	}
	
	// Validate allocation
	if err := manager.ValidateAllocation(ctx, request); err != nil {
		t.Fatalf("Failed to validate allocation: %v", err)
	}
	
	// Allocate GPU
	result, err := manager.AllocateGPU(ctx, request)
	if err != nil {
		t.Fatalf("Failed to allocate GPU: %v", err)
	}
	
	if !result.Success {
		t.Fatalf("Allocation failed: %s", result.Error)
	}
	
	if result.Allocation == nil {
		t.Fatal("Expected allocation in result, got nil")
	}
	
	if result.Allocation.ID != request.ID {
		t.Errorf("Expected allocation ID '%s', got '%s'", request.ID, result.Allocation.ID)
	}
	
	// Test getting allocation
	allocation, err := manager.GetAllocation(ctx, request.ID)
	if err != nil {
		t.Fatalf("Failed to get allocation: %v", err)
	}
	
	if allocation.ID != request.ID {
		t.Errorf("Expected allocation ID '%s', got '%s'", request.ID, allocation.ID)
	}
	
	// Test listing allocations
	allocations, err := manager.ListAllocations(ctx)
	if err != nil {
		t.Fatalf("Failed to list allocations: %v", err)
	}
	
	if len(allocations) != 1 {
		t.Errorf("Expected 1 allocation, got %d", len(allocations))
	}
	
	// Test getting metrics
	metrics, err := manager.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	
	if metrics.ActiveAllocations != 1 {
		t.Errorf("Expected 1 active allocation, got %d", metrics.ActiveAllocations)
	}
	
	// Test getting GPU stats
	stats, err := manager.GetGPUStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get GPU stats: %v", err)
	}
	
	if stats.TotalGPUs == 0 {
		t.Fatal("Expected total GPUs > 0, got 0")
	}
	
	// Test releasing allocation
	if err := manager.ReleaseGPU(ctx, request.ID); err != nil {
		t.Fatalf("Failed to release GPU: %v", err)
	}
	
	// Verify allocation is released
	allocations, err = manager.ListAllocations(ctx)
	if err != nil {
		t.Fatalf("Failed to list allocations after release: %v", err)
	}
	
	if len(allocations) != 0 {
		t.Errorf("Expected 0 allocations after release, got %d", len(allocations))
	}
	
	// Shutdown manager
	if err := manager.Shutdown(ctx); err != nil {
		t.Fatalf("Failed to shutdown manager: %v", err)
	}
}

func TestFractionalAllocator(t *testing.T) {
	// Create fractional allocator
	allocator := NewFractionalAllocator()
	
	// Register GPUs
	allocator.RegisterGPU("card0", 128*1024*1024*1024) // 128GB
	allocator.RegisterGPU("card1", 128*1024*1024*1024) // 128GB
	
	// Test allocation request
	request := &types.AllocationRequest{
		ID:            "test-fractional-1",
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
		GPURequest: &types.GPURequest{
			Fraction:        0.5,
			MemoryRequest:   4000, // 4GB
			IsolationType:   types.GPUIsolationMPS,
			SharingEnabled:  true,
			Priority:        0,
		},
		Strategy: types.AllocationStrategyFirstFit,
		Priority: 0,
		CreatedAt: time.Now(),
	}
	
	// Test can allocate
	canAllocate, err := allocator.CanAllocate("card0", request.GPURequest)
	if err != nil {
		t.Fatalf("Failed to check if can allocate: %v", err)
	}
	
	if !canAllocate {
		t.Fatal("Expected to be able to allocate on card0")
	}
	
	// Test allocation
	allocation, err := allocator.Allocate("card0", request)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}
	
	if allocation.ID != request.ID {
		t.Errorf("Expected allocation ID '%s', got '%s'", request.ID, allocation.ID)
	}
	
	if allocation.DeviceID != "card0" {
		t.Errorf("Expected device ID 'card0', got '%s'", allocation.DeviceID)
	}
	
	// Test utilization stats
	stats := allocator.GetGPUUtilization("card0")
	if stats.UsedFraction != 0.5 {
		t.Errorf("Expected used fraction 0.5, got %f", stats.UsedFraction)
	}
	
	if stats.UtilizationRate != 0.5 {
		t.Errorf("Expected utilization rate 0.5, got %f", stats.UtilizationRate)
	}
	
	// Test finding best fit GPU
	bestGPU, err := allocator.FindBestFitGPU(request.GPURequest)
	if err != nil {
		t.Fatalf("Failed to find best fit GPU: %v", err)
	}
	
	if bestGPU != "card1" {
		t.Errorf("Expected best fit GPU 'card1', got '%s'", bestGPU)
	}
	
	// Test release
	if err := allocator.Release(request.ID); err != nil {
		t.Fatalf("Failed to release allocation: %v", err)
	}
	
	// Verify release
	stats = allocator.GetGPUUtilization("card0")
	if stats.UsedFraction != 0.0 {
		t.Errorf("Expected used fraction 0.0 after release, got %f", stats.UsedFraction)
	}
}

func TestGPUManagerFactory(t *testing.T) {
	// Create factory
	factory := NewDefaultGPUManagerFactory()
	
	// Test supported types
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) != 1 {
		t.Errorf("Expected 1 supported type, got %d", len(supportedTypes))
	}
	
	if supportedTypes[0] != types.GPUTypeAMD {
		t.Errorf("Expected supported type AMD, got %s", supportedTypes[0])
	}
	
	// Test creating AMD manager
	config := &GPUManagerConfig{
		GPUType:               types.GPUTypeAMD,
		PollingInterval:       30 * time.Second,
		AllocationTimeout:     5 * time.Minute,
		DefaultStrategy:       types.AllocationStrategyFirstFit,
		EnableSharing:         true,
		MaxFraction:           1.0,
		MinFraction:           0.1,
		AllowedIsolationTypes: []types.GPUIsolationType{types.GPUIsolationMPS, types.GPUIsolationNone},
	}
	
	manager, err := factory.CreateManager(config)
	if err != nil {
		t.Fatalf("Failed to create AMD manager: %v", err)
	}
	
	if manager.GetGPUType() != types.GPUTypeAMD {
		t.Errorf("Expected GPU type AMD, got %s", manager.GetGPUType())
	}
	
	// Test creating unsupported manager
	unsupportedConfig := &GPUManagerConfig{
		GPUType: types.GPUTypeNVIDIA,
	}
	
	_, err = factory.CreateManager(unsupportedConfig)
	if err == nil {
		t.Fatal("Expected error for unsupported GPU type")
	}
}

func TestGPUManagerConfigValidation(t *testing.T) {
	// Test valid configuration
	validConfig := &GPUManagerConfig{
		GPUType:               types.GPUTypeAMD,
		PollingInterval:       30 * time.Second,
		AllocationTimeout:     5 * time.Minute,
		DefaultStrategy:       types.AllocationStrategyFirstFit,
		EnableSharing:         true,
		MaxFraction:           1.0,
		MinFraction:           0.1,
		AllowedIsolationTypes: []types.GPUIsolationType{types.GPUIsolationMPS, types.GPUIsolationNone},
	}
	
	if err := ValidateGPUManagerConfig(validConfig); err != nil {
		t.Fatalf("Valid config should not return error: %v", err)
	}
	
	// Test invalid GPU type
	invalidGPUConfig := &GPUManagerConfig{
		GPUType: types.GPUTypeNVIDIA,
	}
	
	if err := ValidateGPUManagerConfig(invalidGPUConfig); err == nil {
		t.Fatal("Expected error for unsupported GPU type")
	}
	
	// Test invalid polling interval
	invalidPollingConfig := &GPUManagerConfig{
		GPUType:        types.GPUTypeAMD,
		PollingInterval: 0,
	}
	
	if err := ValidateGPUManagerConfig(invalidPollingConfig); err == nil {
		t.Fatal("Expected error for invalid polling interval")
	}
	
	// Test invalid fraction range
	invalidFractionConfig := &GPUManagerConfig{
		GPUType:        types.GPUTypeAMD,
		PollingInterval: 30 * time.Second,
		MaxFraction:    2.0, // Invalid
		MinFraction:    0.1,
	}
	
	if err := ValidateGPUManagerConfig(invalidFractionConfig); err == nil {
		t.Fatal("Expected error for invalid fraction range")
	}
}
