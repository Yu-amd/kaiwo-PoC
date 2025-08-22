package optimization

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// DynamicAllocator implements dynamic resource allocation for KaiwoJobs
type DynamicAllocator struct {
	client      client.Client
	mu          sync.RWMutex
	allocations map[string]*DynamicAllocation
	metrics     *DynamicAllocatorMetrics
}

// DynamicAllocation represents a dynamic resource allocation for a job
type DynamicAllocation struct {
	JobName     string
	Namespace   string
	CurrentGPU  int64
	CurrentCPU  resource.Quantity
	CurrentMem  resource.Quantity
	OptimalGPU  int64
	OptimalCPU  resource.Quantity
	OptimalMem  resource.Quantity
	Performance float64
	LastUpdated time.Time
	Adjustments []ResourceAdjustment
}

// ResourceAdjustment represents a resource adjustment recommendation
type ResourceAdjustment struct {
	Type      string
	From      resource.Quantity
	To        resource.Quantity
	Reason    string
	Timestamp time.Time
}

// DynamicAllocatorMetrics tracks dynamic allocation performance metrics
type DynamicAllocatorMetrics struct {
	TotalAdjustments      int64
	SuccessfulAdjustments int64
	FailedAdjustments     int64
	AverageAdjustmentTime time.Duration
	mu                    sync.RWMutex
}

// NewDynamicAllocator creates a new dynamic allocator instance
func NewDynamicAllocator(client client.Client) *DynamicAllocator {
	return &DynamicAllocator{
		client:      client,
		allocations: make(map[string]*DynamicAllocation),
		metrics: &DynamicAllocatorMetrics{
			TotalAdjustments:      0,
			SuccessfulAdjustments: 0,
			FailedAdjustments:     0,
		},
	}
}

// AnalyzeJob analyzes a job's resource usage and performance
func (da *DynamicAllocator) AnalyzeJob(ctx context.Context, job *v1alpha1.KaiwoJob) error {
	startTime := time.Now()

	da.mu.Lock()
	defer da.mu.Unlock()

	// Update metrics
	da.metrics.mu.Lock()
	da.metrics.TotalAdjustments++
	da.metrics.mu.Unlock()

	// Get current resource allocation
	allocationKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
	currentAllocation := da.allocations[allocationKey]

	if currentAllocation == nil {
		// Create new allocation
		currentAllocation = &DynamicAllocation{
			JobName:     job.Name,
			Namespace:   job.Namespace,
			CurrentGPU:  int64(job.Spec.Gpus),
			LastUpdated: time.Now(),
			Adjustments: make([]ResourceAdjustment, 0),
		}

		// Set initial CPU and memory
		if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
			if cpu, ok := job.Spec.Resources.Requests[corev1.ResourceCPU]; ok {
				currentAllocation.CurrentCPU = cpu
			}
			if mem, ok := job.Spec.Resources.Requests[corev1.ResourceMemory]; ok {
				currentAllocation.CurrentMem = mem
			}
		}

		da.allocations[allocationKey] = currentAllocation
	}

	// Analyze performance metrics
	performance := da.calculatePerformance(ctx, job)
	currentAllocation.Performance = performance

	// Determine optimal resource allocation
	optimalGPU, optimalCPU, optimalMem := da.calculateOptimalResources(job, performance)

	// Check if adjustment is needed
	if da.shouldAdjustResources(currentAllocation, optimalGPU, optimalCPU, optimalMem) {
		if err := da.adjustResources(ctx, job, currentAllocation, optimalGPU, optimalCPU, optimalMem); err != nil {
			da.updateFailedMetrics(time.Since(startTime))
			return fmt.Errorf("failed to adjust resources: %w", err)
		}
	}

	// Update successful metrics
	da.updateSuccessfulMetrics(time.Since(startTime))

	return nil
}

// calculatePerformance calculates the performance score for a job
func (da *DynamicAllocator) calculatePerformance(ctx context.Context, job *v1alpha1.KaiwoJob) float64 {
	// Get job pods to analyze performance
	var pods corev1.PodList
	if err := da.client.List(ctx, &pods, client.MatchingLabels{"kaiwo.silogen.ai/name": job.Name}); err != nil {
		return 0.0
	}

	if len(pods.Items) == 0 {
		return 0.0
	}

	// Calculate performance based on pod status and resource usage
	totalPerformance := 0.0
	podCount := 0

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Calculate resource utilization
			cpuUtilization := da.calculateCPUUtilization(&pod)
			memUtilization := da.calculateMemoryUtilization(&pod)
			gpuUtilization := da.calculateGPUUtilization(&pod)

			// Performance score based on resource utilization
			// Higher utilization with stable performance indicates good resource allocation
			performance := (cpuUtilization + memUtilization + gpuUtilization) / 3.0
			totalPerformance += performance
			podCount++
		}
	}

	if podCount == 0 {
		return 0.0
	}

	return totalPerformance / float64(podCount)
}

// calculateCPUUtilization calculates CPU utilization for a pod
func (da *DynamicAllocator) calculateCPUUtilization(pod *corev1.Pod) float64 {
	// This would typically get metrics from a metrics server
	// For now, return a placeholder value
	return 0.7 // 70% utilization
}

// calculateMemoryUtilization calculates memory utilization for a pod
func (da *DynamicAllocator) calculateMemoryUtilization(pod *corev1.Pod) float64 {
	// This would typically get metrics from a metrics server
	// For now, return a placeholder value
	return 0.6 // 60% utilization
}

// calculateGPUUtilization calculates GPU utilization for a pod
func (da *DynamicAllocator) calculateGPUUtilization(pod *corev1.Pod) float64 {
	// This would typically get metrics from a metrics server
	// For now, return a placeholder value
	return 0.8 // 80% utilization
}

// calculateOptimalResources calculates optimal resource allocation based on performance
func (da *DynamicAllocator) calculateOptimalResources(job *v1alpha1.KaiwoJob, performance float64) (int64, resource.Quantity, resource.Quantity) {
	currentGPU := int64(job.Spec.Gpus)
	currentCPU := resource.MustParse("1")
	currentMem := resource.MustParse("4Gi")

	if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
		if cpu, ok := job.Spec.Resources.Requests[corev1.ResourceCPU]; ok {
			currentCPU = cpu
		}
		if mem, ok := job.Spec.Resources.Requests[corev1.ResourceMemory]; ok {
			currentMem = mem
		}
	}

	// Adjust resources based on performance
	// Low performance might indicate insufficient resources
	// High performance might indicate over-allocation
	var optimalGPU int64
	var optimalCPU resource.Quantity
	var optimalMem resource.Quantity

	if performance < 0.5 {
		// Low performance - increase resources
		optimalGPU = currentGPU + 1
		optimalCPU = currentCPU.DeepCopy()
		optimalCPU.Add(resource.MustParse("1"))
		optimalMem = currentMem.DeepCopy()
		optimalMem.Add(resource.MustParse("2Gi"))
	} else if performance > 0.9 {
		// High performance - might be able to reduce resources
		if currentGPU > 1 {
			optimalGPU = currentGPU - 1
		} else {
			optimalGPU = currentGPU
		}
		optimalCPU = currentCPU.DeepCopy()
		optimalMem = currentMem.DeepCopy()
	} else {
		// Good performance - keep current allocation
		optimalGPU = currentGPU
		optimalCPU = currentCPU
		optimalMem = currentMem
	}

	return optimalGPU, optimalCPU, optimalMem
}

// shouldAdjustResources determines if resource adjustment is needed
func (da *DynamicAllocator) shouldAdjustResources(allocation *DynamicAllocation, optimalGPU int64, optimalCPU, optimalMem resource.Quantity) bool {
	// Check if optimal resources differ significantly from current
	gpuDiff := abs(int(optimalGPU - allocation.CurrentGPU))
	cpuDiff := optimalCPU.Cmp(allocation.CurrentCPU)
	memDiff := optimalMem.Cmp(allocation.CurrentMem)

	// Adjust if there's a significant difference
	return gpuDiff > 0 || cpuDiff != 0 || memDiff != 0
}

// adjustResources adjusts the resources for a job
func (da *DynamicAllocator) adjustResources(ctx context.Context, job *v1alpha1.KaiwoJob, allocation *DynamicAllocation, optimalGPU int64, optimalCPU, optimalMem resource.Quantity) error {
	// Create adjustment record
	adjustment := ResourceAdjustment{
		Timestamp: time.Now(),
	}

	// Update GPU allocation
	if optimalGPU != allocation.CurrentGPU {
		adjustment.Type = "GPU"
		adjustment.From = *resource.NewQuantity(allocation.CurrentGPU, resource.DecimalSI)
		adjustment.To = *resource.NewQuantity(optimalGPU, resource.DecimalSI)
		adjustment.Reason = fmt.Sprintf("Performance-based adjustment: %f", allocation.Performance)

		allocation.Adjustments = append(allocation.Adjustments, adjustment)
		allocation.CurrentGPU = optimalGPU
	}

	// Update CPU allocation
	if optimalCPU.Cmp(allocation.CurrentCPU) != 0 {
		adjustment.Type = "CPU"
		adjustment.From = allocation.CurrentCPU
		adjustment.To = optimalCPU
		adjustment.Reason = fmt.Sprintf("Performance-based adjustment: %f", allocation.Performance)

		allocation.Adjustments = append(allocation.Adjustments, adjustment)
		allocation.CurrentCPU = optimalCPU
	}

	// Update memory allocation
	if optimalMem.Cmp(allocation.CurrentMem) != 0 {
		adjustment.Type = "Memory"
		adjustment.From = allocation.CurrentMem
		adjustment.To = optimalMem
		adjustment.Reason = fmt.Sprintf("Performance-based adjustment: %f", allocation.Performance)

		allocation.Adjustments = append(allocation.Adjustments, adjustment)
		allocation.CurrentMem = optimalMem
	}

	// Update job spec with new resources
	job.Spec.Gpus = int(optimalGPU)

	if job.Spec.Resources == nil {
		job.Spec.Resources = &corev1.ResourceRequirements{
			Requests: make(corev1.ResourceList),
		}
	}

	job.Spec.Resources.Requests[corev1.ResourceCPU] = optimalCPU
	job.Spec.Resources.Requests[corev1.ResourceMemory] = optimalMem

	// Update job in Kubernetes
	if err := da.client.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job resources: %w", err)
	}

	allocation.LastUpdated = time.Now()
	allocation.OptimalGPU = optimalGPU
	allocation.OptimalCPU = optimalCPU
	allocation.OptimalMem = optimalMem

	return nil
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// updateSuccessfulMetrics updates metrics for successful adjustments
func (da *DynamicAllocator) updateSuccessfulMetrics(adjustmentTime time.Duration) {
	da.metrics.mu.Lock()
	defer da.metrics.mu.Unlock()

	da.metrics.SuccessfulAdjustments++

	// Update average adjustment time
	if da.metrics.SuccessfulAdjustments > 0 {
		totalTime := da.metrics.AverageAdjustmentTime * time.Duration(da.metrics.SuccessfulAdjustments-1)
		da.metrics.AverageAdjustmentTime = (totalTime + adjustmentTime) / time.Duration(da.metrics.SuccessfulAdjustments)
	} else {
		da.metrics.AverageAdjustmentTime = adjustmentTime
	}
}

// updateFailedMetrics updates metrics for failed adjustments
func (da *DynamicAllocator) updateFailedMetrics(adjustmentTime time.Duration) {
	da.metrics.mu.Lock()
	defer da.metrics.mu.Unlock()

	da.metrics.FailedAdjustments++
}

// GetMetrics returns current dynamic allocator metrics
func (da *DynamicAllocator) GetMetrics() DynamicAllocatorMetrics {
	da.metrics.mu.RLock()
	defer da.metrics.mu.RUnlock()

	return *da.metrics
}

// GetAllocations returns all current dynamic allocations
func (da *DynamicAllocator) GetAllocations() map[string]*DynamicAllocation {
	da.mu.RLock()
	defer da.mu.RUnlock()

	// Return a copy to avoid race conditions
	allocations := make(map[string]*DynamicAllocation)
	for k, v := range da.allocations {
		allocations[k] = v
	}

	return allocations
}
