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

package types

import (
	"fmt"
	"time"
)

// AllocationStrategy represents the strategy for GPU allocation
type AllocationStrategy string

const (
	// AllocationStrategyFirstFit allocates to the first available GPU
	AllocationStrategyFirstFit AllocationStrategy = "first-fit"
	
	// AllocationStrategyBestFit allocates to the GPU with the best fit
	AllocationStrategyBestFit AllocationStrategy = "best-fit"
	
	// AllocationStrategyWorstFit allocates to the GPU with the worst fit
	AllocationStrategyWorstFit AllocationStrategy = "worst-fit"
	
	// AllocationStrategyRoundRobin allocates in round-robin fashion
	AllocationStrategyRoundRobin AllocationStrategy = "round-robin"
	
	// AllocationStrategyLoadBalanced allocates based on load balancing
	AllocationStrategyLoadBalanced AllocationStrategy = "load-balanced"
)

// AllocationRequest represents a request for GPU allocation
type AllocationRequest struct {
	// ID is the unique identifier for this request
	ID string `json:"id"`
	
	// PodName is the name of the requesting pod
	PodName string `json:"podName"`
	
	// Namespace is the namespace of the requesting pod
	Namespace string `json:"namespace"`
	
	// ContainerName is the name of the requesting container
	ContainerName string `json:"containerName"`
	
	// GPURequest is the GPU allocation request
	GPURequest *GPURequest `json:"gpuRequest"`
	
	// Strategy is the allocation strategy to use
	Strategy AllocationStrategy `json:"strategy"`
	
	// Priority is the allocation priority (higher values = higher priority)
	Priority int `json:"priority"`
	
	// CreatedAt is the timestamp when the request was created
	CreatedAt time.Time `json:"createdAt"`
	
	// ExpiresAt is the timestamp when the request expires (nil for no expiration)
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	
	// NodeSelector is the node selector for allocation
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	
	// GPUType is the preferred GPU type
	GPUType GPUType `json:"gpuType,omitempty"`
}

// AllocationResult represents the result of a GPU allocation
type AllocationResult struct {
	// Success indicates if the allocation was successful
	Success bool `json:"success"`
	
	// Allocation is the GPU allocation (if successful)
	Allocation *GPUAllocation `json:"allocation,omitempty"`
	
	// Error is the error message (if unsuccessful)
	Error string `json:"error,omitempty"`
	
	// DeviceID is the allocated GPU device ID
	DeviceID string `json:"deviceId,omitempty"`
	
	// NodeName is the node where the GPU was allocated
	NodeName string `json:"nodeName,omitempty"`
	
	// AllocatedAt is the timestamp when the allocation was made
	AllocatedAt time.Time `json:"allocatedAt"`
}

// AllocationPool represents a pool of GPU allocations
type AllocationPool struct {
	// ID is the unique identifier for this pool
	ID string `json:"id"`
	
	// Name is the name of the pool
	Name string `json:"name"`
	
	// Description is the description of the pool
	Description string `json:"description,omitempty"`
	
	// GPUType is the GPU type for this pool
	GPUType GPUType `json:"gpuType"`
	
	// DeviceIDs is the list of GPU device IDs in this pool
	DeviceIDs []string `json:"deviceIds"`
	
	// TotalCapacity is the total capacity of the pool
	TotalCapacity int `json:"totalCapacity"`
	
	// AvailableCapacity is the available capacity of the pool
	AvailableCapacity int `json:"availableCapacity"`
	
	// Allocations is the list of active allocations in this pool
	Allocations []*GPUAllocation `json:"allocations"`
	
	// CreatedAt is the timestamp when the pool was created
	CreatedAt time.Time `json:"createdAt"`
	
	// UpdatedAt is the timestamp when the pool was last updated
	UpdatedAt time.Time `json:"updatedAt"`
}

// AllocationPolicy represents a policy for GPU allocation
type AllocationPolicy struct {
	// Name is the name of the policy
	Name string `json:"name"`
	
	// Description is the description of the policy
	Description string `json:"description,omitempty"`
	
	// Strategy is the default allocation strategy
	Strategy AllocationStrategy `json:"strategy"`
	
	// MaxFraction is the maximum fractional allocation allowed
	MaxFraction float64 `json:"maxFraction"`
	
	// MinFraction is the minimum fractional allocation allowed
	MinFraction float64 `json:"minFraction"`
	
	// MaxMemoryRequest is the maximum memory request allowed in MiB
	MaxMemoryRequest int64 `json:"maxMemoryRequest"`
	
	// AllowSharing indicates if GPU sharing is allowed
	AllowSharing bool `json:"allowSharing"`
	
	// AllowedIsolationTypes is the list of allowed isolation types
	AllowedIsolationTypes []GPUIsolationType `json:"allowedIsolationTypes"`
	
	// PriorityBoost is the priority boost for this policy
	PriorityBoost int `json:"priorityBoost"`
	
	// Timeout is the allocation timeout
	Timeout time.Duration `json:"timeout"`
}

// AllocationMetrics represents metrics for GPU allocation
type AllocationMetrics struct {
	// TotalRequests is the total number of allocation requests
	TotalRequests int64 `json:"totalRequests"`
	
	// SuccessfulAllocations is the number of successful allocations
	SuccessfulAllocations int64 `json:"successfulAllocations"`
	
	// FailedAllocations is the number of failed allocations
	FailedAllocations int64 `json:"failedAllocations"`
	
	// ActiveAllocations is the number of active allocations
	ActiveAllocations int64 `json:"activeAllocations"`
	
	// AverageAllocationTime is the average time to allocate a GPU
	AverageAllocationTime time.Duration `json:"averageAllocationTime"`
	
	// TotalAllocationTime is the total time spent on allocations
	TotalAllocationTime time.Duration `json:"totalAllocationTime"`
	
	// UtilizationRate is the GPU utilization rate
	UtilizationRate float64 `json:"utilizationRate"`
	
	// MemoryUtilizationRate is the GPU memory utilization rate
	MemoryUtilizationRate float64 `json:"memoryUtilizationRate"`
	
	// LastUpdated is the timestamp when metrics were last updated
	LastUpdated time.Time `json:"lastUpdated"`
}

// AllocationEvent represents an event related to GPU allocation
type AllocationEvent struct {
	// ID is the unique identifier for this event
	ID string `json:"id"`
	
	// Type is the type of event
	Type AllocationEventType `json:"type"`
	
	// AllocationID is the ID of the related allocation
	AllocationID string `json:"allocationId"`
	
	// PodName is the name of the related pod
	PodName string `json:"podName"`
	
	// Namespace is the namespace of the related pod
	Namespace string `json:"namespace"`
	
	// Message is the event message
	Message string `json:"message"`
	
	// Timestamp is the timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`
	
	// Metadata contains additional event metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AllocationEventType represents the type of allocation event
type AllocationEventType string

const (
	// AllocationEventTypeRequested indicates an allocation was requested
	AllocationEventTypeRequested AllocationEventType = "requested"
	
	// AllocationEventTypeAllocated indicates an allocation was successful
	AllocationEventTypeAllocated AllocationEventType = "allocated"
	
	// AllocationEventTypeFailed indicates an allocation failed
	AllocationEventTypeFailed AllocationEventType = "failed"
	
	// AllocationEventTypeReleased indicates an allocation was released
	AllocationEventTypeReleased AllocationEventType = "released"
	
	// AllocationEventTypeExpired indicates an allocation expired
	AllocationEventTypeExpired AllocationEventType = "expired"
	
	// AllocationEventTypeModified indicates an allocation was modified
	AllocationEventTypeModified AllocationEventType = "modified"
)

// ValidateAllocationRequest validates an allocation request
func ValidateAllocationRequest(request *AllocationRequest) error {
	if request.ID == "" {
		return fmt.Errorf("allocation request ID cannot be empty")
	}
	
	if request.PodName == "" {
		return fmt.Errorf("pod name cannot be empty")
	}
	
	if request.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	
	if request.ContainerName == "" {
		return fmt.Errorf("container name cannot be empty")
	}
	
	if request.GPURequest == nil {
		return fmt.Errorf("GPU request cannot be nil")
	}
	
	if err := ValidateGPURequest(request.GPURequest); err != nil {
		return fmt.Errorf("invalid GPU request: %v", err)
	}
	
	switch request.Strategy {
	case AllocationStrategyFirstFit, AllocationStrategyBestFit, AllocationStrategyWorstFit,
		 AllocationStrategyRoundRobin, AllocationStrategyLoadBalanced:
		// Valid strategy
	default:
		return fmt.Errorf("invalid allocation strategy: %s", request.Strategy)
	}
	
	if request.Priority < 0 {
		return fmt.Errorf("priority must be non-negative, got %d", request.Priority)
	}
	
	return nil
}

// ValidateAllocationPolicy validates an allocation policy
func ValidateAllocationPolicy(policy *AllocationPolicy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name cannot be empty")
	}
	
	switch policy.Strategy {
	case AllocationStrategyFirstFit, AllocationStrategyBestFit, AllocationStrategyWorstFit,
		 AllocationStrategyRoundRobin, AllocationStrategyLoadBalanced:
		// Valid strategy
	default:
		return fmt.Errorf("invalid allocation strategy: %s", policy.Strategy)
	}
	
	if policy.MaxFraction < 0.1 || policy.MaxFraction > 1.0 {
		return fmt.Errorf("max fraction must be between 0.1 and 1.0, got %f", policy.MaxFraction)
	}
	
	if policy.MinFraction < 0.1 || policy.MinFraction > 1.0 {
		return fmt.Errorf("min fraction must be between 0.1 and 1.0, got %f", policy.MinFraction)
	}
	
	if policy.MinFraction > policy.MaxFraction {
		return fmt.Errorf("min fraction cannot be greater than max fraction")
	}
	
	if policy.MaxMemoryRequest < 0 {
		return fmt.Errorf("max memory request must be non-negative, got %d", policy.MaxMemoryRequest)
	}
	
	if len(policy.AllowedIsolationTypes) == 0 {
		return fmt.Errorf("at least one isolation type must be allowed")
	}
	
	for _, isolationType := range policy.AllowedIsolationTypes {
		switch isolationType {
		case GPUIsolationMPS, GPUIsolationMIG, GPUIsolationNone:
			// Valid isolation type
		default:
			return fmt.Errorf("invalid isolation type: %s", isolationType)
		}
	}
	
	if policy.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative, got %v", policy.Timeout)
	}
	
	return nil
}

// CalculatePoolUtilization calculates the utilization rate of an allocation pool
func CalculatePoolUtilization(pool *AllocationPool) float64 {
	if pool.TotalCapacity == 0 {
		return 0.0
	}
	
	usedCapacity := pool.TotalCapacity - pool.AvailableCapacity
	return float64(usedCapacity) / float64(pool.TotalCapacity)
}

// IsPoolFull checks if an allocation pool is full
func IsPoolFull(pool *AllocationPool) bool {
	return pool.AvailableCapacity == 0
}

// CanAllocate checks if a pool can allocate the requested resources
func (pool *AllocationPool) CanAllocate(request *GPURequest) bool {
	if IsPoolFull(pool) {
		return false
	}
	
	// Check if the pool has enough capacity for the fraction
	if request.Fraction > 1.0 {
		return false
	}
	
	// Check if the pool has enough memory
	if request.MemoryRequest > 0 {
		// This is a simplified check - in practice, you'd need to check available memory
		// across all GPUs in the pool
		return true
	}
	
	return true
}
