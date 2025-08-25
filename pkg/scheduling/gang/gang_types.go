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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// GangSchedulingConfig defines the configuration for gang scheduling
type GangSchedulingConfig struct {
	// Enabled indicates whether gang scheduling is enabled
	Enabled bool `json:"enabled"`

	// MinMembers is the minimum number of members required for the gang
	MinMembers int `json:"minMembers"`

	// MaxMembers is the maximum number of members allowed in the gang (optional)
	MaxMembers int `json:"maxMembers,omitempty"`

	// Timeout is the maximum time to wait for all gang members to be scheduled
	Timeout metav1.Duration `json:"timeout"`

	// SchedulingPolicy defines how gang members should be scheduled
	SchedulingPolicy GangSchedulingPolicy `json:"schedulingPolicy,omitempty"`

	// ResourceRequirements defines the resource requirements for each gang member
	ResourceRequirements corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// GangSchedulingPolicy defines the policy for gang scheduling
type GangSchedulingPolicy string

const (
	// GangSchedulingPolicyStrict requires all members to be scheduled simultaneously
	GangSchedulingPolicyStrict GangSchedulingPolicy = "strict"

	// GangSchedulingPolicyBestEffort allows partial gang scheduling with graceful degradation
	GangSchedulingPolicyBestEffort GangSchedulingPolicy = "best-effort"

	// GangSchedulingPolicyAdaptive adjusts gang size based on available resources
	GangSchedulingPolicyAdaptive GangSchedulingPolicy = "adaptive"
)

// GangSchedulingRequest represents a request to schedule a gang of pods
type GangSchedulingRequest struct {
	// JobID is the unique identifier for the job requesting gang scheduling
	JobID string

	// JobName is the human-readable name of the job
	JobName string

	// Namespace is the Kubernetes namespace for the job
	Namespace string

	// MinMembers is the minimum number of gang members required
	MinMembers int

	// MaxMembers is the maximum number of gang members (0 means no limit)
	MaxMembers int

	// Timeout is the maximum time to wait for gang scheduling
	Timeout time.Duration

	// Priority is the scheduling priority for the gang
	Priority int32

	// Resources are the resource requirements for each gang member
	Resources corev1.ResourceRequirements

	// Constraints are additional scheduling constraints
	Constraints []GangConstraint

	// Labels are labels to be applied to gang member pods
	Labels map[string]string

	// Annotations are annotations to be applied to gang member pods
	Annotations map[string]string

	// CreatedAt is the timestamp when the request was created
	CreatedAt time.Time

	// KaiwoJob is the reference to the original KaiwoJob
	KaiwoJob *v1alpha1.KaiwoJob
}

// GangConstraint defines constraints for gang scheduling
type GangConstraint struct {
	// Type is the type of constraint
	Type GangConstraintType `json:"type"`

	// Key is the constraint key (e.g., node label key)
	Key string `json:"key"`

	// Values are the acceptable values for the constraint
	Values []string `json:"values"`

	// Required indicates if this constraint is required or preferred
	Required bool `json:"required"`
}

// GangConstraintType defines the type of gang constraint
type GangConstraintType string

const (
	// GangConstraintTypeNodeAffinity requires gang members to be scheduled on specific nodes
	GangConstraintTypeNodeAffinity GangConstraintType = "node-affinity"

	// GangConstraintTypePodAffinity requires gang members to be co-located
	GangConstraintTypePodAffinity GangConstraintType = "pod-affinity"

	// GangConstraintTypePodAntiAffinity requires gang members to be spread apart
	GangConstraintTypePodAntiAffinity GangConstraintType = "pod-anti-affinity"

	// GangConstraintTypeTopology requires gang members to follow topology constraints
	GangConstraintTypeTopology GangConstraintType = "topology"
)

// GangSchedulingResponse represents the response to a gang scheduling request
type GangSchedulingResponse struct {
	// RequestID is the ID of the original request
	RequestID string

	// Status is the status of the gang scheduling operation
	Status GangSchedulingStatus

	// ScheduledMembers is the number of successfully scheduled gang members
	ScheduledMembers int

	// TotalMembers is the total number of gang members requested
	TotalMembers int

	// ScheduledPods contains the details of scheduled pods
	ScheduledPods []ScheduledPod

	// Reason provides additional information about the scheduling result
	Reason string

	// Error contains error information if scheduling failed
	Error error

	// SchedulingTime is the time taken to complete gang scheduling
	SchedulingTime time.Duration

	// CompletedAt is the timestamp when gang scheduling completed
	CompletedAt time.Time
}

// GangSchedulingStatus represents the status of gang scheduling
type GangSchedulingStatus string

const (
	// GangSchedulingStatusPending indicates gang scheduling is pending
	GangSchedulingStatusPending GangSchedulingStatus = "pending"

	// GangSchedulingStatusScheduling indicates gang scheduling is in progress
	GangSchedulingStatusScheduling GangSchedulingStatus = "scheduling"

	// GangSchedulingStatusScheduled indicates all gang members are scheduled
	GangSchedulingStatusScheduled GangSchedulingStatus = "scheduled"

	// GangSchedulingStatusPartiallyScheduled indicates some gang members are scheduled
	GangSchedulingStatusPartiallyScheduled GangSchedulingStatus = "partially-scheduled"

	// GangSchedulingStatusFailed indicates gang scheduling failed
	GangSchedulingStatusFailed GangSchedulingStatus = "failed"

	// GangSchedulingStatusTimeout indicates gang scheduling timed out
	GangSchedulingStatusTimeout GangSchedulingStatus = "timeout"

	// GangSchedulingStatusCancelled indicates gang scheduling was cancelled
	GangSchedulingStatusCancelled GangSchedulingStatus = "cancelled"
)

// ScheduledPod contains information about a scheduled gang member pod
type ScheduledPod struct {
	// Name is the name of the scheduled pod
	Name string

	// NodeName is the name of the node where the pod is scheduled
	NodeName string

	// Resources are the allocated resources for the pod
	Resources corev1.ResourceList

	// ScheduledAt is the timestamp when the pod was scheduled
	ScheduledAt time.Time

	// Status is the current status of the pod
	Status corev1.PodPhase
}

// GangMember represents a single member of a gang
type GangMember struct {
	// ID is the unique identifier for the gang member
	ID string

	// Index is the index of the member within the gang (0-based)
	Index int

	// PodSpec is the pod specification for the gang member
	PodSpec corev1.PodSpec

	// NodeName is the assigned node (set after scheduling)
	NodeName string

	// Status is the current status of the gang member
	Status GangMemberStatus

	// Resources are the allocated resources
	Resources corev1.ResourceList

	// ScheduledAt is the timestamp when this member was scheduled
	ScheduledAt time.Time
}

// GangMemberStatus represents the status of a gang member
type GangMemberStatus string

const (
	// GangMemberStatusPending indicates the member is pending scheduling
	GangMemberStatusPending GangMemberStatus = "pending"

	// GangMemberStatusScheduled indicates the member is scheduled
	GangMemberStatusScheduled GangMemberStatus = "scheduled"

	// GangMemberStatusRunning indicates the member pod is running
	GangMemberStatusRunning GangMemberStatus = "running"

	// GangMemberStatusFailed indicates the member failed
	GangMemberStatusFailed GangMemberStatus = "failed"

	// GangMemberStatusCompleted indicates the member completed successfully
	GangMemberStatusCompleted GangMemberStatus = "completed"
)

// Gang represents a group of pods that must be scheduled together
type Gang struct {
	// ID is the unique identifier for the gang
	ID string

	// JobID is the ID of the job this gang belongs to
	JobID string

	// Request is the original scheduling request
	Request *GangSchedulingRequest

	// Members are the individual gang members
	Members []*GangMember

	// Status is the current status of the gang
	Status GangSchedulingStatus

	// CreatedAt is the timestamp when the gang was created
	CreatedAt time.Time

	// SchedulingStartedAt is the timestamp when scheduling started
	SchedulingStartedAt time.Time

	// SchedulingCompletedAt is the timestamp when scheduling completed
	SchedulingCompletedAt time.Time

	// ResourceReservations are the reserved resources for this gang
	ResourceReservations map[string]corev1.ResourceList

	// Metrics contains gang scheduling metrics
	Metrics *GangMetrics
}

// GangMetrics contains metrics for gang scheduling operations
type GangMetrics struct {
	// SchedulingAttempts is the number of scheduling attempts
	SchedulingAttempts int

	// SchedulingTime is the total time spent on scheduling
	SchedulingTime time.Duration

	// ResourceWaitTime is the time spent waiting for resources
	ResourceWaitTime time.Duration

	// SuccessfulMembers is the number of successfully scheduled members
	SuccessfulMembers int

	// FailedMembers is the number of failed members
	FailedMembers int

	// SchedulingEfficiency is the scheduling efficiency (0.0 to 1.0)
	SchedulingEfficiency float64
}

// GangSchedulerInterface defines the interface for gang scheduling
type GangSchedulerInterface interface {
	// ScheduleGang schedules a gang of pods
	ScheduleGang(ctx context.Context, request *GangSchedulingRequest) (*GangSchedulingResponse, error)

	// CancelGang cancels gang scheduling for the specified gang
	CancelGang(ctx context.Context, gangID string) error

	// GetGangStatus returns the current status of a gang
	GetGangStatus(ctx context.Context, gangID string) (*Gang, error)

	// ListGangs returns a list of all gangs
	ListGangs(ctx context.Context) ([]*Gang, error)

	// GetMetrics returns gang scheduling metrics
	GetMetrics(ctx context.Context) (*GangSchedulingMetrics, error)
}

// GangSchedulingMetrics contains overall gang scheduling metrics
type GangSchedulingMetrics struct {
	// TotalGangs is the total number of gangs processed
	TotalGangs int64

	// SuccessfulGangs is the number of successfully scheduled gangs
	SuccessfulGangs int64

	// FailedGangs is the number of failed gangs
	FailedGangs int64

	// AverageSchedulingTime is the average time to schedule a gang
	AverageSchedulingTime time.Duration

	// AverageGangSize is the average number of members per gang
	AverageGangSize float64

	// ResourceUtilization is the current resource utilization
	ResourceUtilization map[string]float64

	// SchedulingEfficiency is the overall scheduling efficiency
	SchedulingEfficiency float64
}

// ResourceCalculator provides utilities for resource calculations
type ResourceCalculator struct{}

// CalculateGangResources calculates the total resources required for a gang
func (rc *ResourceCalculator) CalculateGangResources(members int, memberResources corev1.ResourceRequirements) corev1.ResourceList {
	totalResources := make(corev1.ResourceList)

	// Calculate total requests
	for resourceName, quantity := range memberResources.Requests {
		totalQuantity := quantity.DeepCopy()
		totalQuantity.Set(totalQuantity.Value() * int64(members))
		totalResources[resourceName] = totalQuantity
	}

	return totalResources
}

// CalculateResourceEfficiency calculates resource efficiency for gang scheduling
func (rc *ResourceCalculator) CalculateResourceEfficiency(allocated, requested corev1.ResourceList) float64 {
	if len(requested) == 0 {
		return 1.0
	}

	totalEfficiency := 0.0
	resourceCount := 0

	for resourceName, requestedQuantity := range requested {
		if allocatedQuantity, exists := allocated[resourceName]; exists {
			if requestedQuantity.Value() > 0 {
				efficiency := float64(allocatedQuantity.Value()) / float64(requestedQuantity.Value())
				totalEfficiency += efficiency
				resourceCount++
			}
		}
	}

	if resourceCount == 0 {
		return 1.0
	}

	return totalEfficiency / float64(resourceCount)
}

// ValidateGangRequest validates a gang scheduling request
func ValidateGangRequest(request *GangSchedulingRequest) error {
	if request == nil {
		return fmt.Errorf("gang scheduling request cannot be nil")
	}

	if request.JobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	if request.MinMembers <= 0 {
		return fmt.Errorf("minimum members must be greater than 0")
	}

	if request.MaxMembers > 0 && request.MaxMembers < request.MinMembers {
		return fmt.Errorf("maximum members cannot be less than minimum members")
	}

	if request.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	// Validate resource requirements
	if len(request.Resources.Requests) == 0 {
		return fmt.Errorf("resource requests cannot be empty")
	}

	return nil
}
