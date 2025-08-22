package enhanced

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// PriorityScheduler implements priority-based scheduling for KaiwoJobs
type PriorityScheduler struct {
	client    client.Client
	mu        sync.RWMutex
	jobQueue  []*v1alpha1.KaiwoJob
	metrics   *SchedulerMetrics
}

// SchedulerMetrics tracks scheduling performance metrics
type SchedulerMetrics struct {
	TotalJobsScheduled    int64
	AverageSchedulingTime time.Duration
	PriorityViolations    int64
	mu                    sync.RWMutex
}

// NewPriorityScheduler creates a new priority scheduler instance
func NewPriorityScheduler(client client.Client) *PriorityScheduler {
	return &PriorityScheduler{
		client:   client,
		jobQueue: make([]*v1alpha1.KaiwoJob, 0),
		metrics: &SchedulerMetrics{
			TotalJobsScheduled: 0,
			PriorityViolations: 0,
		},
	}
}

// ScheduleJob adds a job to the priority queue and schedules it
func (ps *PriorityScheduler) ScheduleJob(ctx context.Context, job *v1alpha1.KaiwoJob) error {
	startTime := time.Now()
	
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Add job to priority queue
	ps.jobQueue = append(ps.jobQueue, job)
	
	// Sort queue by priority (highest priority first)
	sort.Slice(ps.jobQueue, func(i, j int) bool {
		return ps.getJobPriority(ps.jobQueue[i]) > ps.getJobPriority(ps.jobQueue[j])
	})

	// Attempt to schedule jobs in priority order
	if err := ps.processQueue(ctx); err != nil {
		return fmt.Errorf("failed to process job queue: %w", err)
	}

	// Update metrics
	ps.updateMetrics(time.Since(startTime))
	
	return nil
}

// getJobPriority calculates the priority score for a job
func (ps *PriorityScheduler) getJobPriority(job *v1alpha1.KaiwoJob) int {
	priority := 0
	
	// Age-based priority boost (older jobs get higher priority)
	if job.CreationTimestamp.Time.Before(time.Now().Add(-1 * time.Hour)) {
		priority += 10
	}
	
	// GPU requirement priority (higher GPU needs get priority)
	if job.Spec.Gpus > 0 {
		priority += job.Spec.Gpus * 5
	}
	
	// Priority class boost
	if job.Spec.WorkloadPriorityClass != "" {
		priority += 20
	}
	
	return priority
}

// processQueue attempts to schedule all jobs in the priority queue
func (ps *PriorityScheduler) processQueue(ctx context.Context) error {
	var unscheduledJobs []*v1alpha1.KaiwoJob
	
	for _, job := range ps.jobQueue {
		if err := ps.attemptSchedule(ctx, job); err != nil {
			// Job couldn't be scheduled, keep it in queue
			unscheduledJobs = append(unscheduledJobs, job)
		}
	}
	
	// Update queue with unscheduled jobs
	ps.jobQueue = unscheduledJobs
	
	return nil
}

// attemptSchedule tries to schedule a single job
func (ps *PriorityScheduler) attemptSchedule(ctx context.Context, job *v1alpha1.KaiwoJob) error {
	// Check resource availability
	if !ps.checkResourceAvailability(ctx, job) {
		return fmt.Errorf("insufficient resources for job %s", job.Name)
	}
	
	// Update job status to starting
	job.Status.Status = v1alpha1.WorkloadStatusStarting
	job.Status.StartTime = &metav1.Time{Time: time.Now()}
	
	// Update job in Kubernetes
	if err := ps.client.Status().Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	
	ps.metrics.mu.Lock()
	ps.metrics.TotalJobsScheduled++
	ps.metrics.mu.Unlock()
	
	return nil
}

// checkResourceAvailability checks if sufficient resources are available
func (ps *PriorityScheduler) checkResourceAvailability(ctx context.Context, job *v1alpha1.KaiwoJob) bool {
	// Get available GPU resources from nodes
	var nodes corev1.NodeList
	if err := ps.client.List(ctx, &nodes); err != nil {
		return false
	}
	
	// Calculate required resources
	requiredGPU := ps.calculateRequiredGPU(job)
	
	// Check if sufficient resources are available
	availableGPU := int64(0)
	for _, node := range nodes.Items {
		if gpu, ok := node.Status.Capacity["amd.com/gpu"]; ok {
			availableGPU += gpu.Value()
		}
	}
	
	return availableGPU >= requiredGPU
}

// calculateRequiredGPU calculates the total GPU requirements for a job
func (ps *PriorityScheduler) calculateRequiredGPU(job *v1alpha1.KaiwoJob) int64 {
	// Use the Gpus field from the job spec
	return int64(job.Spec.Gpus)
}

// updateMetrics updates scheduling performance metrics
func (ps *PriorityScheduler) updateMetrics(schedulingTime time.Duration) {
	ps.metrics.mu.Lock()
	defer ps.metrics.mu.Unlock()
	
	// Update average scheduling time
	if ps.metrics.TotalJobsScheduled > 0 {
		totalTime := ps.metrics.AverageSchedulingTime * time.Duration(ps.metrics.TotalJobsScheduled-1)
		ps.metrics.AverageSchedulingTime = (totalTime + schedulingTime) / time.Duration(ps.metrics.TotalJobsScheduled)
	} else {
		ps.metrics.AverageSchedulingTime = schedulingTime
	}
}

// GetMetrics returns current scheduling metrics
func (ps *PriorityScheduler) GetMetrics() SchedulerMetrics {
	ps.metrics.mu.RLock()
	defer ps.metrics.mu.RUnlock()
	
	return *ps.metrics
}

// GetQueueLength returns the current length of the job queue
func (ps *PriorityScheduler) GetQueueLength() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	return len(ps.jobQueue)
}
