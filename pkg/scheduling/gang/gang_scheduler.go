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

package gang

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/silogen/kaiwo/pkg/scheduling/enhanced"
)

// GangScheduler implements gang scheduling for distributed workloads
type GangScheduler struct {
	// kubernetes client for API operations
	kubeClient kubernetes.Interface

	// priority scheduler for individual pod scheduling
	priorityScheduler *enhanced.PriorityScheduler

	// resource allocator for resource management
	resourceAllocator *enhanced.ResourceAllocator

	// load balancer for optimal node selection
	loadBalancer *enhanced.LoadBalancer

	// pending gangs waiting for scheduling
	pendingGangs map[string]*Gang

	// active gangs currently being scheduled
	activeGangs map[string]*Gang

	// completed gangs (for metrics and history)
	completedGangs map[string]*Gang

	// gang scheduling configuration
	config *GangSchedulerConfig

	// metrics for gang scheduling operations
	metrics *GangSchedulingMetrics

	// resource calculator for gang resource calculations
	resourceCalculator *ResourceCalculator

	// mutex for thread safety
	mu sync.RWMutex

	// context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// scheduling work queue
	schedulingQueue chan *Gang

	// worker pool size
	workerPoolSize int
}

// GangSchedulerConfig contains configuration for the gang scheduler
type GangSchedulerConfig struct {
	// MaxConcurrentGangs is the maximum number of gangs to schedule concurrently
	MaxConcurrentGangs int

	// DefaultTimeout is the default timeout for gang scheduling
	DefaultTimeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed gang scheduling
	MaxRetries int

	// RetryBackoff is the backoff duration between retry attempts
	RetryBackoff time.Duration

	// ResourceReservationTimeout is the timeout for resource reservations
	ResourceReservationTimeout time.Duration

	// EnablePreemption indicates whether preemption is enabled for gang scheduling
	EnablePreemption bool

	// EnableResourceReservation indicates whether resource reservation is enabled
	EnableResourceReservation bool

	// WorkerPoolSize is the number of worker goroutines for scheduling
	WorkerPoolSize int
}

// NewGangScheduler creates a new gang scheduler instance
func NewGangScheduler(kubeClient kubernetes.Interface, priorityScheduler *enhanced.PriorityScheduler,
	resourceAllocator *enhanced.ResourceAllocator, loadBalancer *enhanced.LoadBalancer) *GangScheduler {

	ctx, cancel := context.WithCancel(context.Background())

	config := &GangSchedulerConfig{
		MaxConcurrentGangs:         10,
		DefaultTimeout:             10 * time.Minute,
		MaxRetries:                 3,
		RetryBackoff:               30 * time.Second,
		ResourceReservationTimeout: 5 * time.Minute,
		EnablePreemption:           true,
		EnableResourceReservation:  true,
		WorkerPoolSize:             5,
	}

	gs := &GangScheduler{
		kubeClient:         kubeClient,
		priorityScheduler:  priorityScheduler,
		resourceAllocator:  resourceAllocator,
		loadBalancer:       loadBalancer,
		pendingGangs:       make(map[string]*Gang),
		activeGangs:        make(map[string]*Gang),
		completedGangs:     make(map[string]*Gang),
		config:             config,
		metrics:            &GangSchedulingMetrics{},
		resourceCalculator: &ResourceCalculator{},
		ctx:                ctx,
		cancel:             cancel,
		schedulingQueue:    make(chan *Gang, 100),
		workerPoolSize:     config.WorkerPoolSize,
	}

	// Start worker pool
	gs.startWorkerPool()

	return gs
}

// ScheduleGang schedules a gang of pods according to the gang scheduling request
func (gs *GangScheduler) ScheduleGang(ctx context.Context, request *GangSchedulingRequest) (*GangSchedulingResponse, error) {
	startTime := time.Now()

	// Validate the request
	if err := ValidateGangRequest(request); err != nil {
		return &GangSchedulingResponse{
			RequestID: request.JobID,
			Status:    GangSchedulingStatusFailed,
			Reason:    fmt.Sprintf("Request validation failed: %v", err),
			Error:     err,
		}, err
	}

	// Create gang from request
	gang := gs.createGangFromRequest(request)

	// Add to pending gangs
	gs.mu.Lock()
	gs.pendingGangs[gang.ID] = gang
	gs.metrics.TotalGangs++
	gs.mu.Unlock()

	// Submit to scheduling queue
	select {
	case gs.schedulingQueue <- gang:
		// Successfully queued
	case <-ctx.Done():
		return &GangSchedulingResponse{
			RequestID: request.JobID,
			Status:    GangSchedulingStatusCancelled,
			Reason:    "Context cancelled before scheduling could begin",
			Error:     ctx.Err(),
		}, ctx.Err()
	}

	// Wait for scheduling completion or timeout
	timeout := request.Timeout
	if timeout == 0 {
		timeout = gs.config.DefaultTimeout
	}

	select {
	case <-time.After(timeout):
		// Timeout occurred
		gs.mu.Lock()
		gang.Status = GangSchedulingStatusTimeout
		delete(gs.pendingGangs, gang.ID)
		delete(gs.activeGangs, gang.ID)
		gs.completedGangs[gang.ID] = gang
		gs.metrics.FailedGangs++
		gs.mu.Unlock()

		return &GangSchedulingResponse{
			RequestID:        request.JobID,
			Status:           GangSchedulingStatusTimeout,
			ScheduledMembers: gs.countScheduledMembers(gang),
			TotalMembers:     len(gang.Members),
			Reason:           fmt.Sprintf("Gang scheduling timed out after %v", timeout),
			SchedulingTime:   time.Since(startTime),
			CompletedAt:      time.Now(),
		}, fmt.Errorf("gang scheduling timed out")

	case <-ctx.Done():
		// Context cancelled
		return &GangSchedulingResponse{
			RequestID: request.JobID,
			Status:    GangSchedulingStatusCancelled,
			Reason:    "Context cancelled during scheduling",
			Error:     ctx.Err(),
		}, ctx.Err()

	case <-gs.waitForGangCompletion(gang.ID):
		// Gang scheduling completed
		gs.mu.RLock()
		finalGang := gs.completedGangs[gang.ID]
		if finalGang == nil {
			finalGang = gs.activeGangs[gang.ID]
		}
		gs.mu.RUnlock()

		if finalGang == nil {
			return &GangSchedulingResponse{
				RequestID: request.JobID,
				Status:    GangSchedulingStatusFailed,
				Reason:    "Gang not found after completion",
				Error:     fmt.Errorf("gang not found"),
			}, fmt.Errorf("gang not found")
		}

		scheduledPods := gs.createScheduledPodsResponse(finalGang)

		return &GangSchedulingResponse{
			RequestID:        request.JobID,
			Status:           finalGang.Status,
			ScheduledMembers: gs.countScheduledMembers(finalGang),
			TotalMembers:     len(finalGang.Members),
			ScheduledPods:    scheduledPods,
			Reason:           gs.getGangStatusReason(finalGang),
			SchedulingTime:   time.Since(startTime),
			CompletedAt:      time.Now(),
		}, nil
	}
}

// CancelGang cancels gang scheduling for the specified gang
func (gs *GangScheduler) CancelGang(ctx context.Context, gangID string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Check if gang exists in pending or active gangs
	gang := gs.pendingGangs[gangID]
	if gang == nil {
		gang = gs.activeGangs[gangID]
	}

	if gang == nil {
		return fmt.Errorf("gang %s not found", gangID)
	}

	// Update status and move to completed
	gang.Status = GangSchedulingStatusCancelled
	delete(gs.pendingGangs, gangID)
	delete(gs.activeGangs, gangID)
	gs.completedGangs[gangID] = gang
	gs.metrics.FailedGangs++

	return nil
}

// GetGangStatus returns the current status of a gang
func (gs *GangScheduler) GetGangStatus(ctx context.Context, gangID string) (*Gang, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Check pending gangs
	if gang := gs.pendingGangs[gangID]; gang != nil {
		return gang, nil
	}

	// Check active gangs
	if gang := gs.activeGangs[gangID]; gang != nil {
		return gang, nil
	}

	// Check completed gangs
	if gang := gs.completedGangs[gangID]; gang != nil {
		return gang, nil
	}

	return nil, fmt.Errorf("gang %s not found", gangID)
}

// ListGangs returns a list of all gangs
func (gs *GangScheduler) ListGangs(ctx context.Context) ([]*Gang, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var gangs []*Gang

	// Add pending gangs
	for _, gang := range gs.pendingGangs {
		gangs = append(gangs, gang)
	}

	// Add active gangs
	for _, gang := range gs.activeGangs {
		gangs = append(gangs, gang)
	}

	// Add completed gangs
	for _, gang := range gs.completedGangs {
		gangs = append(gangs, gang)
	}

	return gangs, nil
}

// GetMetrics returns gang scheduling metrics
func (gs *GangScheduler) GetMetrics(ctx context.Context) (*GangSchedulingMetrics, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Calculate current metrics
	metrics := &GangSchedulingMetrics{
		TotalGangs:      gs.metrics.TotalGangs,
		SuccessfulGangs: gs.metrics.SuccessfulGangs,
		FailedGangs:     gs.metrics.FailedGangs,
	}

	// Calculate average scheduling time
	if gs.metrics.SuccessfulGangs > 0 {
		totalTime := time.Duration(0)
		count := int64(0)

		for _, gang := range gs.completedGangs {
			if gang.Status == GangSchedulingStatusScheduled && !gang.SchedulingCompletedAt.IsZero() && !gang.SchedulingStartedAt.IsZero() {
				totalTime += gang.SchedulingCompletedAt.Sub(gang.SchedulingStartedAt)
				count++
			}
		}

		if count > 0 {
			metrics.AverageSchedulingTime = totalTime / time.Duration(count)
		}
	}

	// Calculate average gang size
	if gs.metrics.TotalGangs > 0 {
		totalMembers := 0
		for _, gang := range gs.completedGangs {
			totalMembers += len(gang.Members)
		}
		metrics.AverageGangSize = float64(totalMembers) / float64(gs.metrics.TotalGangs)
	}

	// Calculate scheduling efficiency
	if gs.metrics.TotalGangs > 0 {
		metrics.SchedulingEfficiency = float64(gs.metrics.SuccessfulGangs) / float64(gs.metrics.TotalGangs)
	}

	return metrics, nil
}

// startWorkerPool starts the worker pool for gang scheduling
func (gs *GangScheduler) startWorkerPool() {
	for i := 0; i < gs.workerPoolSize; i++ {
		go gs.worker()
	}
}

// worker processes gang scheduling requests from the queue
func (gs *GangScheduler) worker() {
	for {
		select {
		case gang := <-gs.schedulingQueue:
			gs.processGang(gang)
		case <-gs.ctx.Done():
			return
		}
	}
}

// processGang processes a single gang scheduling request
func (gs *GangScheduler) processGang(gang *Gang) {
	// Move from pending to active
	gs.mu.Lock()
	delete(gs.pendingGangs, gang.ID)
	gs.activeGangs[gang.ID] = gang
	gang.Status = GangSchedulingStatusScheduling
	gang.SchedulingStartedAt = time.Now()
	gs.mu.Unlock()

	// Attempt to schedule the gang
	success := gs.attemptGangScheduling(gang)

	// Update final status
	gs.mu.Lock()
	delete(gs.activeGangs, gang.ID)
	gang.SchedulingCompletedAt = time.Now()

	if success {
		gang.Status = GangSchedulingStatusScheduled
		gs.metrics.SuccessfulGangs++
	} else {
		gang.Status = GangSchedulingStatusFailed
		gs.metrics.FailedGangs++
	}

	gs.completedGangs[gang.ID] = gang
	gs.mu.Unlock()
}

// attemptGangScheduling attempts to schedule all members of a gang
func (gs *GangScheduler) attemptGangScheduling(gang *Gang) bool {
	// Step 1: Check if enough resources are available
	totalResources := gs.resourceCalculator.CalculateGangResources(len(gang.Members), gang.Request.Resources)

	// Step 2: Reserve resources if enabled
	if gs.config.EnableResourceReservation {
		if !gs.reserveGangResources(gang, totalResources) {
			return false
		}
	}

	// Step 3: Schedule all gang members
	scheduledMembers := 0
	for _, member := range gang.Members {
		if gs.scheduleMember(gang, member) {
			member.Status = GangMemberStatusScheduled
			member.ScheduledAt = time.Now()
			scheduledMembers++
		} else {
			member.Status = GangMemberStatusFailed
		}
	}

	// Step 4: Check if minimum requirements are met
	minMembers := gang.Request.MinMembers
	if scheduledMembers >= minMembers {
		// Update metrics
		if gang.Metrics == nil {
			gang.Metrics = &GangMetrics{}
		}
		gang.Metrics.SuccessfulMembers = scheduledMembers
		gang.Metrics.FailedMembers = len(gang.Members) - scheduledMembers
		gang.Metrics.SchedulingEfficiency = float64(scheduledMembers) / float64(len(gang.Members))

		return true
	}

	// Step 5: If scheduling failed, clean up
	gs.cleanupFailedGang(gang)
	return false
}

// createGangFromRequest creates a Gang from a GangSchedulingRequest
func (gs *GangScheduler) createGangFromRequest(request *GangSchedulingRequest) *Gang {
	gangID := fmt.Sprintf("%s-%d", request.JobID, time.Now().Unix())

	gang := &Gang{
		ID:                   gangID,
		JobID:                request.JobID,
		Request:              request,
		Status:               GangSchedulingStatusPending,
		CreatedAt:            time.Now(),
		Members:              make([]*GangMember, request.MinMembers),
		ResourceReservations: make(map[string]corev1.ResourceList),
		Metrics:              &GangMetrics{},
	}

	// Create gang members
	for i := 0; i < request.MinMembers; i++ {
		member := &GangMember{
			ID:     fmt.Sprintf("%s-member-%d", gangID, i),
			Index:  i,
			Status: GangMemberStatusPending,
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:      fmt.Sprintf("%s-container", request.JobName),
						Image:     "placeholder:latest", // This should come from KaiwoJob
						Resources: request.Resources,
					},
				},
			},
		}
		gang.Members[i] = member
	}

	return gang
}

// Helper methods

func (gs *GangScheduler) countScheduledMembers(gang *Gang) int {
	count := 0
	for _, member := range gang.Members {
		if member.Status == GangMemberStatusScheduled || member.Status == GangMemberStatusRunning {
			count++
		}
	}
	return count
}

func (gs *GangScheduler) createScheduledPodsResponse(gang *Gang) []ScheduledPod {
	var pods []ScheduledPod
	for _, member := range gang.Members {
		if member.Status == GangMemberStatusScheduled || member.Status == GangMemberStatusRunning {
			pod := ScheduledPod{
				Name:        member.ID,
				NodeName:    member.NodeName,
				Resources:   member.Resources,
				ScheduledAt: member.ScheduledAt,
				Status:      corev1.PodPending, // Default status
			}
			pods = append(pods, pod)
		}
	}
	return pods
}

func (gs *GangScheduler) getGangStatusReason(gang *Gang) string {
	switch gang.Status {
	case GangSchedulingStatusScheduled:
		return fmt.Sprintf("All %d gang members successfully scheduled", len(gang.Members))
	case GangSchedulingStatusPartiallyScheduled:
		scheduled := gs.countScheduledMembers(gang)
		return fmt.Sprintf("%d out of %d gang members scheduled", scheduled, len(gang.Members))
	case GangSchedulingStatusFailed:
		return "Gang scheduling failed - insufficient resources or constraints not met"
	case GangSchedulingStatusTimeout:
		return "Gang scheduling timed out"
	case GangSchedulingStatusCancelled:
		return "Gang scheduling was cancelled"
	default:
		return "Gang scheduling in progress"
	}
}

func (gs *GangScheduler) waitForGangCompletion(gangID string) <-chan bool {
	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				gs.mu.RLock()
				gang := gs.completedGangs[gangID]
				gs.mu.RUnlock()

				if gang != nil {
					done <- true
					return
				}
			case <-gs.ctx.Done():
				done <- false
				return
			}
		}
	}()
	return done
}

func (gs *GangScheduler) reserveGangResources(gang *Gang, resources corev1.ResourceList) bool {
	// Simplified resource reservation - in a real implementation,
	// this would interact with a resource manager
	gang.ResourceReservations["total"] = resources
	return true
}

func (gs *GangScheduler) scheduleMember(gang *Gang, member *GangMember) bool {
	// Simplified member scheduling - in a real implementation,
	// this would use the priority scheduler and resource allocator

	// For now, simulate successful scheduling
	member.NodeName = fmt.Sprintf("node-%d", member.Index%3) // Distribute across 3 nodes
	member.Resources = member.PodSpec.Containers[0].Resources.Requests
	return true
}

func (gs *GangScheduler) cleanupFailedGang(gang *Gang) {
	// Clean up any partially scheduled members
	for _, member := range gang.Members {
		if member.Status == GangMemberStatusScheduled {
			// In a real implementation, this would delete the scheduled pods
			member.Status = GangMemberStatusFailed
		}
	}
}

// Shutdown gracefully shuts down the gang scheduler
func (gs *GangScheduler) Shutdown() {
	gs.cancel()
	close(gs.schedulingQueue)
}
