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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GPUType represents the type of GPU (AMD, NVIDIA, etc.)
type GPUType string

const (
	GPUTypeAMD     GPUType = "amd"
	GPUTypeNVIDIA  GPUType = "nvidia"
	GPUTypeUnknown GPUType = "unknown"
)

// GPUIsolationType represents the isolation mechanism for GPU sharing
type GPUIsolationType string

const (
	GPUIsolationTimeSlicing GPUIsolationType = "time-slicing"  // Time-slicing for AMD GPUs
	GPUIsolationMIG  GPUIsolationType = "mig"  // Multi-Instance GPU (NVIDIA)
	GPUIsolationNone GPUIsolationType = "none" // No isolation
)

// GPUInfo represents information about a GPU device
type GPUInfo struct {
	// DeviceID is the unique identifier for the GPU
	DeviceID string `json:"deviceId"`

	// Type is the GPU type (AMD, NVIDIA, etc.)
	Type GPUType `json:"type"`

	// Model is the GPU model name
	Model string `json:"model"`

	// TotalMemory is the total GPU memory in bytes
	TotalMemory int64 `json:"totalMemory"`

	// AvailableMemory is the available GPU memory in bytes
	AvailableMemory int64 `json:"availableMemory"`

	// Utilization is the current GPU utilization percentage (0-100)
	Utilization float64 `json:"utilization"`

	// Temperature is the current GPU temperature in Celsius
	Temperature float64 `json:"temperature"`

	// Power is the current GPU power consumption in watts
	Power float64 `json:"power"`

	// NodeName is the Kubernetes node where this GPU is located
	NodeName string `json:"nodeName"`

	// IsAvailable indicates if the GPU is available for allocation
	IsAvailable bool `json:"isAvailable"`

	// IsolationType is the current isolation mechanism
	IsolationType GPUIsolationType `json:"isolationType"`

	// ActiveAllocations is the number of active allocations on this GPU
	ActiveAllocations int `json:"activeAllocations"`
}

// GPUAllocation represents a GPU allocation request
type GPUAllocation struct {
	// ID is the unique identifier for this allocation
	ID string `json:"id"`

	// DeviceID is the GPU device being allocated
	DeviceID string `json:"deviceId"`

	// Fraction is the fractional allocation (0.1 to 1.0)
	Fraction float64 `json:"fraction"`

	// MemoryRequest is the requested GPU memory in bytes
	MemoryRequest int64 `json:"memoryRequest"`

	// IsolationType is the requested isolation mechanism
	IsolationType GPUIsolationType `json:"isolationType"`

	// PodName is the pod requesting the allocation
	PodName string `json:"podName"`

	// Namespace is the namespace of the requesting pod
	Namespace string `json:"namespace"`

	// ContainerName is the container requesting the allocation
	ContainerName string `json:"containerName"`

	// Status is the current status of the allocation
	Status GPUAllocationStatus `json:"status"`

	// CreatedAt is the timestamp when the allocation was created
	CreatedAt int64 `json:"createdAt"`

	// ExpiresAt is the timestamp when the allocation expires (0 for no expiration)
	ExpiresAt int64 `json:"expiresAt"`
}

// GPUAllocationStatus represents the status of a GPU allocation
type GPUAllocationStatus string

const (
	GPUAllocationStatusPending   GPUAllocationStatus = "pending"
	GPUAllocationStatusActive    GPUAllocationStatus = "active"
	GPUAllocationStatusCompleted GPUAllocationStatus = "completed"
	GPUAllocationStatusFailed    GPUAllocationStatus = "failed"
	GPUAllocationStatusExpired   GPUAllocationStatus = "expired"
)

// GPURequest represents a GPU allocation request from a pod
type GPURequest struct {
	// Fraction is the fractional GPU allocation (0.1 to 1.0)
	Fraction float64 `json:"fraction"`

	// MemoryRequest is the requested GPU memory in MiB
	MemoryRequest int64 `json:"memoryRequest"`

	// IsolationType is the requested isolation mechanism
	IsolationType GPUIsolationType `json:"isolationType"`

	// SharingEnabled indicates if GPU sharing is enabled
	SharingEnabled bool `json:"sharingEnabled"`

	// Priority is the allocation priority (higher values = higher priority)
	Priority int `json:"priority"`
}

// GPUAnnotations represents GPU-related annotations that can be applied to pods
type GPUAnnotations struct {
	// Fraction is the fractional GPU allocation
	Fraction *float64 `json:"fraction,omitempty"`

	// Memory is the memory-based allocation in MiB
	Memory *int64 `json:"memory,omitempty"`

	// SharingEnabled indicates if GPU sharing is enabled
	SharingEnabled *bool `json:"sharingEnabled,omitempty"`

	// IsolationType is the isolation mechanism
	IsolationType *GPUIsolationType `json:"isolationType,omitempty"`
}

// ParseGPUAnnotations parses GPU-related annotations from a pod
func ParseGPUAnnotations(pod *corev1.Pod, containerName string) (*GPUAnnotations, error) {
	annotations := &GPUAnnotations{}

	// Find the container
	var container *corev1.Container
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			container = &pod.Spec.Containers[i]
			break
		}
	}

	if container == nil {
		return nil, fmt.Errorf("container %s not found in pod", containerName)
	}

	// Parse GPU fraction annotation
	if fractionStr, exists := pod.Annotations["kaiwo.ai/gpu-fraction"]; exists {
		fraction, err := strconv.ParseFloat(fractionStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gpu-fraction annotation: %v", err)
		}
		if fraction < 0.1 || fraction > 1.0 {
			return nil, fmt.Errorf("gpu-fraction must be between 0.1 and 1.0, got %f", fraction)
		}
		annotations.Fraction = &fraction
	}

	// Parse GPU memory annotation
	if memoryStr, exists := pod.Annotations["kaiwo.ai/gpu-memory"]; exists {
		memory, err := strconv.ParseInt(memoryStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gpu-memory annotation: %v", err)
		}
		if memory <= 0 {
			return nil, fmt.Errorf("gpu-memory must be positive, got %d", memory)
		}
		annotations.Memory = &memory
	}

	// Parse GPU sharing annotation
	if sharingStr, exists := pod.Annotations["kaiwo.ai/gpu-sharing"]; exists {
		sharing := strings.ToLower(sharingStr) == "true"
		annotations.SharingEnabled = &sharing
	}

	// Parse GPU isolation annotation
	if isolationStr, exists := pod.Annotations["kaiwo.ai/gpu-isolation"]; exists {
		isolation := GPUIsolationType(strings.ToLower(isolationStr))
		switch isolation {
		case GPUIsolationTimeSlicing, GPUIsolationMIG, GPUIsolationNone:
			annotations.IsolationType = &isolation
		default:
			return nil, fmt.Errorf("invalid gpu-isolation annotation: %s", isolationStr)
		}
	}

	return annotations, nil
}

// CreateGPURequest creates a GPURequest from annotations and container resources
func CreateGPURequest(pod *corev1.Pod, containerName string) (*GPURequest, error) {
	annotations, err := ParseGPUAnnotations(pod, containerName)
	if err != nil {
		return nil, err
	}

	// Find the container
	var container *corev1.Container
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			container = &pod.Spec.Containers[i]
			break
		}
	}

	if container == nil {
		return nil, fmt.Errorf("container %s not found in pod", containerName)
	}

	request := &GPURequest{
		Fraction:       1.0, // Default to full GPU
		MemoryRequest:  0,   // Default to no specific memory request
		IsolationType:  GPUIsolationNone,
		SharingEnabled: false,
		Priority:       0,
	}

	// Apply annotations
	if annotations.Fraction != nil {
		request.Fraction = *annotations.Fraction
	}

	if annotations.Memory != nil {
		request.MemoryRequest = *annotations.Memory
	}

	if annotations.IsolationType != nil {
		request.IsolationType = *annotations.IsolationType
	}

	if annotations.SharingEnabled != nil {
		request.SharingEnabled = *annotations.SharingEnabled
	}

	// Check for GPU resource requests in container spec
	if gpuResource, exists := container.Resources.Requests["nvidia.com/gpu"]; exists {
		// If GPU resource is requested, use it to determine fraction
		gpuQuantity := gpuResource.Value()
		if gpuQuantity > 0 {
			request.Fraction = float64(gpuQuantity) / 1000.0 // Convert millicores to fraction
		}
	}

	// Check for AMD GPU resource requests
	if gpuResource, exists := container.Resources.Requests["amd.com/gpu"]; exists {
		gpuQuantity := gpuResource.Value()
		if gpuQuantity > 0 {
			request.Fraction = float64(gpuQuantity) / 1000.0
		}
	}

	return request, nil
}

// ValidateGPURequest validates a GPU request
func ValidateGPURequest(request *GPURequest) error {
	if request.Fraction < 0.1 || request.Fraction > 1.0 {
		return fmt.Errorf("GPU fraction must be between 0.1 and 1.0, got %f", request.Fraction)
	}

	if request.MemoryRequest < 0 {
		return fmt.Errorf("GPU memory request must be non-negative, got %d", request.MemoryRequest)
	}

	if request.Priority < 0 {
		return fmt.Errorf("GPU priority must be non-negative, got %d", request.Priority)
	}

	return nil
}

// GPUResourceRequirements represents GPU resource requirements
type GPUResourceRequirements struct {
	// Requests is the requested GPU resources
	Requests GPUResourceList `json:"requests"`

	// Limits is the maximum GPU resources
	Limits GPUResourceList `json:"limits"`
}

// GPUResourceList represents a list of GPU resources
type GPUResourceList struct {
	// GPUs is the number of GPUs requested
	GPUs resource.Quantity `json:"gpus"`

	// Memory is the GPU memory requested in bytes
	Memory resource.Quantity `json:"memory"`

	// Fraction is the fractional GPU allocation
	Fraction resource.Quantity `json:"fraction"`
}

// GPUStats represents GPU statistics for a node or cluster
type GPUStats struct {
	// TotalGPUs is the total number of GPUs
	TotalGPUs int `json:"totalGpus"`

	// AvailableGPUs is the number of available GPUs
	AvailableGPUs int `json:"availableGpus"`

	// TotalMemory is the total GPU memory in bytes
	TotalMemory int64 `json:"totalMemory"`

	// AvailableMemory is the available GPU memory in bytes
	AvailableMemory int64 `json:"availableMemory"`

	// AverageUtilization is the average GPU utilization percentage
	AverageUtilization float64 `json:"averageUtilization"`

	// AverageTemperature is the average GPU temperature in Celsius
	AverageTemperature float64 `json:"averageTemperature"`

	// AveragePower is the average GPU power consumption in watts
	AveragePower float64 `json:"averagePower"`

	// ActiveAllocations is the number of active GPU allocations
	ActiveAllocations int `json:"activeAllocations"`
}



// ReservationStats contains statistics about GPU reservations
type ReservationStats struct {
	TotalReservations     int               `json:"total_reservations"`
	PendingReservations   int               `json:"pending_reservations"`
	ActiveReservations    int               `json:"active_reservations"`
	CompletedReservations int               `json:"completed_reservations"`
	CancelledReservations int               `json:"cancelled_reservations"`
	ExpiredReservations   int               `json:"expired_reservations"`
	ReservationsByGPU     map[string]int    `json:"reservations_by_gpu"`
	ReservationsByUser    map[string]int    `json:"reservations_by_user"`
	ReservationsByStatus  map[string]int    `json:"reservations_by_status"`
}
