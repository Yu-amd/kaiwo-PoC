package realtime

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

// MetricsCollector implements real-time metrics collection for KaiwoJobs
type MetricsCollector struct {
	client    client.Client
	mu        sync.RWMutex
	metrics   map[string]*JobMetrics
	collector *MetricsCollectorMetrics
}

// JobMetrics represents real-time metrics for a job
type JobMetrics struct {
	JobName     string
	Namespace   string
	Timestamp   time.Time
	CPUUsage    resource.Quantity
	MemoryUsage resource.Quantity
	GPUUsage    int64
	PodCount    int
	RunningPods int
	FailedPods  int
	PendingPods int
	Status      v1alpha1.WorkloadStatus
	Performance float64
	Efficiency  float64
}

// MetricsCollectorMetrics tracks metrics collection performance
type MetricsCollectorMetrics struct {
	TotalCollections      int64
	SuccessfulCollections int64
	FailedCollections     int64
	AverageCollectionTime time.Duration
	mu                    sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector(client client.Client) *MetricsCollector {
	return &MetricsCollector{
		client:  client,
		metrics: make(map[string]*JobMetrics),
		collector: &MetricsCollectorMetrics{
			TotalCollections:      0,
			SuccessfulCollections: 0,
			FailedCollections:     0,
		},
	}
}

// CollectMetrics collects real-time metrics for a job
func (mc *MetricsCollector) CollectMetrics(ctx context.Context, job *v1alpha1.KaiwoJob) (*JobMetrics, error) {
	startTime := time.Now()

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update metrics
	mc.collector.mu.Lock()
	mc.collector.TotalCollections++
	mc.collector.mu.Unlock()

	// Get job pods
	pods, err := mc.getJobPods(ctx, job)
	if err != nil {
		mc.updateFailedMetrics(time.Since(startTime))
		return nil, fmt.Errorf("failed to get job pods: %w", err)
	}

	// Calculate metrics
	metrics := &JobMetrics{
		JobName:   job.Name,
		Namespace: job.Namespace,
		Timestamp: time.Now(),
		Status:    job.Status.Status,
	}

	// Calculate pod statistics
	mc.calculatePodStats(pods, metrics)

	// Calculate resource usage
	mc.calculateResourceUsage(pods, metrics)

	// Calculate performance and efficiency
	mc.calculatePerformanceMetrics(metrics)

	// Store metrics
	metricsKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
	mc.metrics[metricsKey] = metrics

	// Update successful metrics
	mc.updateSuccessfulMetrics(time.Since(startTime))

	return metrics, nil
}

// getJobPods retrieves all pods associated with a job
func (mc *MetricsCollector) getJobPods(ctx context.Context, job *v1alpha1.KaiwoJob) ([]corev1.Pod, error) {
	var pods corev1.PodList
	if err := mc.client.List(ctx, &pods, client.MatchingLabels{"kaiwo.silogen.ai/name": job.Name}); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	return pods.Items, nil
}

// calculatePodStats calculates pod statistics
func (mc *MetricsCollector) calculatePodStats(pods []corev1.Pod, metrics *JobMetrics) {
	metrics.PodCount = len(pods)

	for _, pod := range pods {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			metrics.RunningPods++
		case corev1.PodFailed:
			metrics.FailedPods++
		case corev1.PodPending:
			metrics.PendingPods++
		}
	}
}

// calculateResourceUsage calculates resource usage from pods
func (mc *MetricsCollector) calculateResourceUsage(pods []corev1.Pod, metrics *JobMetrics) {
	totalCPU := resource.Quantity{}
	totalMemory := resource.Quantity{}
	totalGPU := int64(0)

	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			// Calculate resource usage from pod spec (requests)
					for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					totalCPU.Add(cpu)
				}
				if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					totalMemory.Add(mem)
				}
				if gpu, ok := container.Resources.Requests["amd.com/gpu"]; ok {
					totalGPU += gpu.Value()
				}
			}
		}
		}
	}

	metrics.CPUUsage = totalCPU
	metrics.MemoryUsage = totalMemory
	metrics.GPUUsage = totalGPU
}

// calculatePerformanceMetrics calculates performance and efficiency metrics
func (mc *MetricsCollector) calculatePerformanceMetrics(metrics *JobMetrics) {
	// Calculate performance based on pod status
	if metrics.PodCount == 0 {
		metrics.Performance = 0.0
		metrics.Efficiency = 0.0
		return
	}

	// Performance: ratio of running pods to total pods
	metrics.Performance = float64(metrics.RunningPods) / float64(metrics.PodCount)

	// Efficiency: resource utilization efficiency
	// This would typically be calculated from actual usage vs requests
	// For now, use a placeholder calculation
	if metrics.RunningPods > 0 {
		// Assume 80% efficiency for running pods
		metrics.Efficiency = 0.8
	} else {
		metrics.Efficiency = 0.0
	}
}

// GetMetrics returns metrics for a specific job
func (mc *MetricsCollector) GetMetrics(jobName, namespace string) (*JobMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metricsKey := fmt.Sprintf("%s/%s", namespace, jobName)
	metrics, exists := mc.metrics[metricsKey]
	if !exists {
		return nil, fmt.Errorf("no metrics found for job %s/%s", namespace, jobName)
	}

	return metrics, nil
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]*JobMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to avoid race conditions
	allMetrics := make(map[string]*JobMetrics)
	for k, v := range mc.metrics {
		allMetrics[k] = v
	}

	return allMetrics
}

// GetMetricsHistory returns historical metrics for a job
func (mc *MetricsCollector) GetMetricsHistory(jobName, namespace string, duration time.Duration) ([]*JobMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metricsKey := fmt.Sprintf("%s/%s", namespace, jobName)
	metrics, exists := mc.metrics[metricsKey]
	if !exists {
		return nil, fmt.Errorf("no metrics found for job %s/%s", namespace, jobName)
	}

	// For now, return the current metrics as a single entry
	// In a real implementation, this would return historical data
	cutoffTime := time.Now().Add(-duration)
	if metrics.Timestamp.After(cutoffTime) {
		return []*JobMetrics{metrics}, nil
	}

	return []*JobMetrics{}, nil
}

// GetClusterMetrics returns aggregated cluster metrics
func (mc *MetricsCollector) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	// Get all KaiwoJobs
	var jobs v1alpha1.KaiwoJobList
	if err := mc.client.List(ctx, &jobs); err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	clusterMetrics := &ClusterMetrics{
		Timestamp:          time.Now(),
		TotalJobs:          len(jobs.Items),
		RunningJobs:        0,
		PendingJobs:        0,
		FailedJobs:         0,
		TotalCPU:           resource.Quantity{},
		TotalMemory:        resource.Quantity{},
		TotalGPU:           0,
		AveragePerformance: 0.0,
	}

	// Aggregate metrics from all jobs
	totalPerformance := 0.0
	jobCount := 0

	for _, job := range jobs.Items {
		metrics, err := mc.GetMetrics(job.Name, job.Namespace)
		if err != nil {
			continue // Skip jobs without metrics
		}

		// Count job statuses
		switch job.Status.Status {
		case v1alpha1.WorkloadStatusRunning:
			clusterMetrics.RunningJobs++
		case v1alpha1.WorkloadStatusPending:
			clusterMetrics.PendingJobs++
		case v1alpha1.WorkloadStatusFailed:
			clusterMetrics.FailedJobs++
		}

		// Aggregate resources
		clusterMetrics.TotalCPU.Add(metrics.CPUUsage)
		clusterMetrics.TotalMemory.Add(metrics.MemoryUsage)
		clusterMetrics.TotalGPU += metrics.GPUUsage

		// Aggregate performance
		totalPerformance += metrics.Performance
		jobCount++
	}

	if jobCount > 0 {
		clusterMetrics.AveragePerformance = totalPerformance / float64(jobCount)
	}

	return clusterMetrics, nil
}

// ClusterMetrics represents aggregated cluster-level metrics
type ClusterMetrics struct {
	Timestamp          time.Time
	TotalJobs          int
	RunningJobs        int
	PendingJobs        int
	FailedJobs         int
	TotalCPU           resource.Quantity
	TotalMemory        resource.Quantity
	TotalGPU           int64
	AveragePerformance float64
}

// updateSuccessfulMetrics updates metrics for successful collections
func (mc *MetricsCollector) updateSuccessfulMetrics(collectionTime time.Duration) {
	mc.collector.mu.Lock()
	defer mc.collector.mu.Unlock()

	mc.collector.SuccessfulCollections++

	// Update average collection time
	if mc.collector.SuccessfulCollections > 0 {
		totalTime := mc.collector.AverageCollectionTime * time.Duration(mc.collector.SuccessfulCollections-1)
		mc.collector.AverageCollectionTime = (totalTime + collectionTime) / time.Duration(mc.collector.SuccessfulCollections)
	} else {
		mc.collector.AverageCollectionTime = collectionTime
	}
}

// updateFailedMetrics updates metrics for failed collections
func (mc *MetricsCollector) updateFailedMetrics(collectionTime time.Duration) {
	mc.collector.mu.Lock()
	defer mc.collector.mu.Unlock()

	mc.collector.FailedCollections++
}

// GetCollectorMetrics returns current collector metrics
func (mc *MetricsCollector) GetCollectorMetrics() MetricsCollectorMetrics {
	mc.collector.mu.RLock()
	defer mc.collector.mu.RUnlock()

	return *mc.collector
}
