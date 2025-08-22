package enhanced

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// ResourceAllocator implements resource-aware allocation for KaiwoJobs
type ResourceAllocator struct {
	client      client.Client
	mu          sync.RWMutex
	allocations map[string]*ResourceAllocation
	metrics     *AllocationMetrics
}

// ResourceAllocation represents a resource allocation for a job
type ResourceAllocation struct {
	JobName      string
	Namespace    string
	AllocatedGPU int64
	AllocatedCPU resource.Quantity
	AllocatedMem resource.Quantity
	AllocatedAt  time.Time
	ExpiresAt    time.Time
}

// AllocationMetrics tracks allocation performance metrics
type AllocationMetrics struct {
	TotalAllocations      int64
	SuccessfulAllocations int64
	FailedAllocations     int64
	AverageAllocationTime time.Duration
	mu                    sync.RWMutex
}

// NewResourceAllocator creates a new resource allocator instance
func NewResourceAllocator(client client.Client) *ResourceAllocator {
	return &ResourceAllocator{
		client:      client,
		allocations: make(map[string]*ResourceAllocation),
		metrics: &AllocationMetrics{
			TotalAllocations:      0,
			SuccessfulAllocations: 0,
			FailedAllocations:     0,
		},
	}
}

// AllocateResources attempts to allocate resources for a job
func (ra *ResourceAllocator) AllocateResources(ctx context.Context, job *v1alpha1.KaiwoJob) (*ResourceAllocation, error) {
	startTime := time.Now()

	ra.mu.Lock()
	defer ra.mu.Unlock()

	// Update metrics
	ra.metrics.mu.Lock()
	ra.metrics.TotalAllocations++
	ra.metrics.mu.Unlock()

	// Calculate required resources
	requiredGPU := ra.calculateRequiredGPU(job)
	requiredCPU := ra.calculateRequiredCPU(job)
	requiredMem := ra.calculateRequiredMemory(job)

	// Check resource availability
	availableResources, err := ra.getAvailableResources(ctx)
	if err != nil {
		ra.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("failed to get available resources: %w", err)
	}

	// Check if sufficient resources are available
	if availableResources.GPU < requiredGPU {
		ra.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("insufficient GPU resources: required %d, available %d", requiredGPU, availableResources.GPU)
	}

	if availableResources.CPU.Cmp(requiredCPU) < 0 {
		ra.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("insufficient CPU resources: required %s, available %s", requiredCPU.String(), availableResources.CPU.String())
	}

	if availableResources.Memory.Cmp(requiredMem) < 0 {
		ra.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("insufficient memory resources: required %s, available %s", requiredMem.String(), availableResources.Memory.String())
	}

	// Create allocation
	allocation := &ResourceAllocation{
		JobName:      job.Name,
		Namespace:    job.Namespace,
		AllocatedGPU: requiredGPU,
		AllocatedCPU: requiredCPU,
		AllocatedMem: requiredMem,
		AllocatedAt:  time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour), // Default 24-hour allocation
	}

	// Store allocation
	allocationKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
	ra.allocations[allocationKey] = allocation

	// Update job status to starting
	job.Status.Status = v1alpha1.WorkloadStatusStarting
	job.Status.StartTime = &metav1.Time{Time: time.Now()}

	if err := ra.client.Status().Update(ctx, job); err != nil {
		ra.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Update successful metrics
	ra.updateSuccessfulMetrics(time.Since(startTime))

	return allocation, nil
}

// ReleaseResources releases allocated resources for a job
func (ra *ResourceAllocator) ReleaseResources(ctx context.Context, job *v1alpha1.KaiwoJob) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	allocationKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)

	if _, exists := ra.allocations[allocationKey]; exists {
		// Update job status to terminated
		job.Status.Status = v1alpha1.WorkloadStatusTerminated

		if err := ra.client.Status().Update(ctx, job); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}

		// Remove allocation
		delete(ra.allocations, allocationKey)

		return nil
	}

	return fmt.Errorf("no allocation found for job %s", job.Name)
}

// calculateRequiredGPU calculates the total GPU requirements for a job
func (ra *ResourceAllocator) calculateRequiredGPU(job *v1alpha1.KaiwoJob) int64 {
	// Use the Gpus field from the job spec
	return int64(job.Spec.Gpus)
}

// calculateRequiredCPU calculates the total CPU requirements for a job
func (ra *ResourceAllocator) calculateRequiredCPU(job *v1alpha1.KaiwoJob) resource.Quantity {
	// Use default CPU requirements from job spec resources
	if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
		if cpu, ok := job.Spec.Resources.Requests[corev1.ResourceCPU]; ok {
			return cpu
		}
	}
	// Default CPU requirement
	return resource.MustParse("1")
}

// calculateRequiredMemory calculates the total memory requirements for a job
func (ra *ResourceAllocator) calculateRequiredMemory(job *v1alpha1.KaiwoJob) resource.Quantity {
	// Use default memory requirements from job spec resources
	if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
		if mem, ok := job.Spec.Resources.Requests[corev1.ResourceMemory]; ok {
			return mem
		}
	}
	// Default memory requirement
	return resource.MustParse("4Gi")
}

// AvailableResources represents available cluster resources
type AvailableResources struct {
	GPU    int64
	CPU    resource.Quantity
	Memory resource.Quantity
}

// getAvailableResources gets the current available resources in the cluster
func (ra *ResourceAllocator) getAvailableResources(ctx context.Context) (*AvailableResources, error) {
	// Get nodes to understand current resource usage
	var nodes corev1.NodeList
	if err := ra.client.List(ctx, &nodes); err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Calculate total capacity and current usage
	totalGPU := int64(0)
	totalCPU := resource.Quantity{}
	totalMem := resource.Quantity{}

	usedGPU := int64(0)
	usedCPU := resource.Quantity{}
	usedMem := resource.Quantity{}

	// Calculate from nodes
	for _, node := range nodes.Items {
		// Add to total capacity
		if gpu, ok := node.Status.Capacity["amd.com/gpu"]; ok {
			totalGPU += gpu.Value()
		}
		if cpu, ok := node.Status.Capacity[corev1.ResourceCPU]; ok {
			totalCPU.Add(cpu)
		}
		if mem, ok := node.Status.Capacity[corev1.ResourceMemory]; ok {
			totalMem.Add(mem)
		}

		// Add to used resources
		if cpu, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
			usedCPU.Add(cpu)
		}
		if mem, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
			usedMem.Add(mem)
		}
	}

	// Subtract current allocations
	for _, allocation := range ra.allocations {
		usedGPU += allocation.AllocatedGPU
		usedCPU.Add(allocation.AllocatedCPU)
		usedMem.Add(allocation.AllocatedMem)
	}

	// Calculate available resources
	availableGPU := totalGPU - usedGPU
	availableCPU := totalCPU.DeepCopy()
	availableCPU.Sub(usedCPU)
	availableMem := totalMem.DeepCopy()
	availableMem.Sub(usedMem)

	return &AvailableResources{
		GPU:    availableGPU,
		CPU:    availableCPU,
		Memory: availableMem,
	}, nil
}

// updateSuccessfulMetrics updates metrics for successful allocations
func (ra *ResourceAllocator) updateSuccessfulMetrics(allocationTime time.Duration) {
	ra.metrics.mu.Lock()
	defer ra.metrics.mu.Unlock()

	ra.metrics.SuccessfulAllocations++

	// Update average allocation time
	if ra.metrics.SuccessfulAllocations > 0 {
		totalTime := ra.metrics.AverageAllocationTime * time.Duration(ra.metrics.SuccessfulAllocations-1)
		ra.metrics.AverageAllocationTime = (totalTime + allocationTime) / time.Duration(ra.metrics.SuccessfulAllocations)
	} else {
		ra.metrics.AverageAllocationTime = allocationTime
	}
}

// updateFailedMetrics updates metrics for failed allocations
func (ra *ResourceAllocator) updateFailedMetrics(allocationTime time.Duration) {
	ra.metrics.mu.Lock()
	defer ra.metrics.mu.Unlock()

	ra.metrics.FailedAllocations++
}

// GetMetrics returns current allocation metrics
func (ra *ResourceAllocator) GetMetrics() AllocationMetrics {
	ra.metrics.mu.RLock()
	defer ra.metrics.mu.RUnlock()

	return *ra.metrics
}

// GetAllocations returns all current resource allocations
func (ra *ResourceAllocator) GetAllocations() map[string]*ResourceAllocation {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	// Return a copy to avoid race conditions
	allocations := make(map[string]*ResourceAllocation)
	for k, v := range ra.allocations {
		allocations[k] = v
	}

	return allocations
}
