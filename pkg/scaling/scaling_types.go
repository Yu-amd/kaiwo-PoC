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

package scaling

import (
	"fmt"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// ScalingDecision represents a decision to scale a workload
type ScalingDecision struct {
	// WorkloadID is the ID of the workload to scale
	WorkloadID string

	// CurrentReplicas is the current number of replicas
	CurrentReplicas int

	// DesiredReplicas is the desired number of replicas after scaling
	DesiredReplicas int

	// ScalingDirection indicates whether this is scaling up or down
	ScalingDirection ScalingDirection

	// Reason provides the reason for the scaling decision
	Reason string

	// Metrics contains the metrics that triggered the scaling decision
	Metrics map[string]float64

	// Timestamp is when the decision was made
	Timestamp time.Time

	// Confidence is the confidence level of the scaling decision (0.0 to 1.0)
	Confidence float64
}

// ScalingDirection indicates the direction of scaling
type ScalingDirection string

const (
	// ScalingDirectionUp indicates scaling up (increasing replicas)
	ScalingDirectionUp ScalingDirection = "up"

	// ScalingDirectionDown indicates scaling down (decreasing replicas)
	ScalingDirectionDown ScalingDirection = "down"

	// ScalingDirectionNone indicates no scaling is needed
	ScalingDirectionNone ScalingDirection = "none"
)

// ScalingMetric represents a metric used for scaling decisions
type ScalingMetric struct {
	// Name is the name of the metric
	Name string

	// Type is the type of metric (cpu, memory, gpu, custom)
	Type ScalingMetricType

	// Value is the current value of the metric
	Value float64

	// Threshold is the target value for the metric
	Threshold float64

	// Unit is the unit of measurement for the metric
	Unit string

	// Timestamp is when the metric was collected
	Timestamp time.Time
}

// ScalingMetricType defines the type of scaling metric
type ScalingMetricType string

const (
	// ScalingMetricTypeCPU represents CPU utilization metrics
	ScalingMetricTypeCPU ScalingMetricType = "cpu"

	// ScalingMetricTypeMemory represents memory utilization metrics
	ScalingMetricTypeMemory ScalingMetricType = "memory"

	// ScalingMetricTypeGPU represents GPU utilization metrics
	ScalingMetricTypeGPU ScalingMetricType = "gpu"

	// ScalingMetricTypeCustom represents custom application metrics
	ScalingMetricTypeCustom ScalingMetricType = "custom"

	// ScalingMetricTypeExternal represents external metrics
	ScalingMetricTypeExternal ScalingMetricType = "external"
)

// ScalingPolicy defines how scaling operations should be performed
type ScalingPolicy struct {
	// ScaleUpPolicy defines the policy for scaling up
	ScaleUpPolicy ScalingVelocityPolicy

	// ScaleDownPolicy defines the policy for scaling down
	ScaleDownPolicy ScalingVelocityPolicy

	// StabilizationWindow defines the time window for metric stabilization
	StabilizationWindow time.Duration

	// CooldownPeriod defines the minimum time between scaling operations
	CooldownPeriod time.Duration
}

// ScalingVelocityPolicy defines how quickly scaling operations should occur
type ScalingVelocityPolicy struct {
	// Rate is the maximum number of replicas to add/remove per period
	Rate int

	// Period is the time period for the rate limit
	Period time.Duration

	// MaxBurst is the maximum number of replicas to add/remove in a single operation
	MaxBurst int
}

// WorkloadScaler defines the interface for workload scaling operations
type WorkloadScaler interface {
	// EvaluateScaling evaluates whether a workload needs scaling
	EvaluateScaling(workload *v1alpha1.KaiwoJob, metrics []ScalingMetric) (*ScalingDecision, error)

	// ExecuteScaling executes a scaling decision
	ExecuteScaling(decision *ScalingDecision) error

	// GetCurrentMetrics retrieves current metrics for a workload
	GetCurrentMetrics(workload *v1alpha1.KaiwoJob) ([]ScalingMetric, error)

	// GetScalingHistory returns the scaling history for a workload
	GetScalingHistory(workloadID string) ([]*ScalingDecision, error)
}

// ScalingEvent represents a scaling event in the system
type ScalingEvent struct {
	// ID is the unique identifier for the event
	ID string

	// WorkloadID is the ID of the workload that was scaled
	WorkloadID string

	// EventType is the type of scaling event
	EventType ScalingEventType

	// Decision is the scaling decision that was executed
	Decision *ScalingDecision

	// Result is the result of the scaling operation
	Result *ScalingResult

	// Timestamp is when the event occurred
	Timestamp time.Time
}

// ScalingEventType defines the type of scaling event
type ScalingEventType string

const (
	// ScalingEventTypeDecisionMade indicates a scaling decision was made
	ScalingEventTypeDecisionMade ScalingEventType = "decision-made"

	// ScalingEventTypeExecutionStarted indicates scaling execution started
	ScalingEventTypeExecutionStarted ScalingEventType = "execution-started"

	// ScalingEventTypeExecutionCompleted indicates scaling execution completed
	ScalingEventTypeExecutionCompleted ScalingEventType = "execution-completed"

	// ScalingEventTypeExecutionFailed indicates scaling execution failed
	ScalingEventTypeExecutionFailed ScalingEventType = "execution-failed"
)

// ScalingResult represents the result of a scaling operation
type ScalingResult struct {
	// Success indicates whether the scaling operation was successful
	Success bool

	// ActualReplicas is the actual number of replicas after scaling
	ActualReplicas int

	// Error contains error information if scaling failed
	Error error

	// Duration is the time taken to complete the scaling operation
	Duration time.Duration

	// CompletedAt is when the scaling operation completed
	CompletedAt time.Time
}

// ScalingController manages elastic scaling for workloads
type ScalingController struct {
	// config contains the scaling controller configuration
	config *ScalingControllerConfig

	// scaler is the workload scaler implementation
	scaler WorkloadScaler

	// metricsCollector collects metrics for scaling decisions
	metricsCollector MetricsCollector

	// scalingHistory stores the history of scaling decisions
	scalingHistory map[string][]*ScalingDecision

	// lastScalingTime tracks the last scaling time for each workload
	lastScalingTime map[string]time.Time

	// scalingEvents stores scaling events
	scalingEvents []*ScalingEvent
}

// ScalingControllerConfig contains configuration for the scaling controller
type ScalingControllerConfig struct {
	// DefaultPolicy is the default scaling policy
	DefaultPolicy *ScalingPolicy

	// MetricsCollectionInterval is how often to collect metrics
	MetricsCollectionInterval time.Duration

	// EvaluationInterval is how often to evaluate scaling decisions
	EvaluationInterval time.Duration

	// MaxScalingHistory is the maximum number of scaling decisions to retain
	MaxScalingHistory int

	// EnablePredictiveScaling enables predictive scaling based on historical patterns
	EnablePredictiveScaling bool

	// PredictionWindow is the time window for predictive scaling
	PredictionWindow time.Duration
}

// MetricsCollector defines the interface for collecting scaling metrics
type MetricsCollector interface {
	// CollectMetrics collects current metrics for a workload
	CollectMetrics(workload *v1alpha1.KaiwoJob) ([]ScalingMetric, error)

	// GetHistoricalMetrics returns historical metrics for trend analysis
	GetHistoricalMetrics(workloadID string, duration time.Duration) ([]ScalingMetric, error)

	// RegisterCustomMetric registers a custom metric for collection
	RegisterCustomMetric(name string, collector func() (float64, error)) error
}

// PredictiveScaler defines the interface for predictive scaling
type PredictiveScaler interface {
	// PredictFutureLoad predicts future resource requirements
	PredictFutureLoad(workloadID string, timeWindow time.Duration) (*LoadPrediction, error)

	// UpdateModel updates the prediction model with new data
	UpdateModel(workloadID string, metrics []ScalingMetric) error

	// GetPredictionAccuracy returns the accuracy of recent predictions
	GetPredictionAccuracy(workloadID string) (float64, error)
}

// LoadPrediction represents a prediction of future resource requirements
type LoadPrediction struct {
	// PredictedLoad is the predicted resource load
	PredictedLoad map[string]float64

	// Confidence is the confidence level of the prediction (0.0 to 1.0)
	Confidence float64

	// TimeWindow is the time window for the prediction
	TimeWindow time.Duration

	// RecommendedReplicas is the recommended number of replicas
	RecommendedReplicas int

	// PredictionTime is when the prediction was made
	PredictionTime time.Time
}

// ResourceUtilization represents the current resource utilization of a workload
type ResourceUtilization struct {
	// WorkloadID is the ID of the workload
	WorkloadID string

	// CPU utilization as a percentage (0-100)
	CPU float64

	// Memory utilization as a percentage (0-100)
	Memory float64

	// GPU utilization as a percentage (0-100)
	GPU float64

	// NetworkIO utilization in bytes per second
	NetworkIO float64

	// DiskIO utilization in bytes per second
	DiskIO float64

	// CustomMetrics contains custom application-specific metrics
	CustomMetrics map[string]float64

	// Timestamp is when the utilization was measured
	Timestamp time.Time

	// Replicas is the current number of replicas
	Replicas int
}

// ScalingStrategy defines different strategies for scaling workloads
type ScalingStrategy interface {
	// CalculateDesiredReplicas calculates the desired number of replicas
	CalculateDesiredReplicas(current int, metrics []ScalingMetric, policy *ScalingPolicy) (int, error)

	// ShouldScale determines if scaling is necessary
	ShouldScale(current int, desired int, metrics []ScalingMetric) bool

	// GetScalingReason returns a human-readable reason for scaling
	GetScalingReason(current int, desired int, metrics []ScalingMetric) string
}

// ProportionalScalingStrategy implements proportional scaling based on metric ratios
type ProportionalScalingStrategy struct {
	// TargetUtilization is the target utilization percentage
	TargetUtilization float64

	// Tolerance is the tolerance around the target utilization
	Tolerance float64
}

// CalculateDesiredReplicas calculates the desired number of replicas based on metrics
func (p *ProportionalScalingStrategy) CalculateDesiredReplicas(current int, metrics []ScalingMetric, policy *ScalingPolicy) (int, error) {
	if len(metrics) == 0 {
		return current, nil
	}

	// Find the primary metric (typically CPU)
	var primaryMetric *ScalingMetric
	for _, metric := range metrics {
		if metric.Type == ScalingMetricTypeCPU {
			primaryMetric = &metric
			break
		}
	}

	if primaryMetric == nil {
		primaryMetric = &metrics[0] // Use first metric if no CPU metric found
	}

	// Calculate the scaling ratio based on current utilization vs target
	if primaryMetric.Threshold == 0 {
		return current, nil
	}

	utilizationRatio := primaryMetric.Value / primaryMetric.Threshold
	desiredReplicas := int(float64(current) * utilizationRatio)

	// Ensure we have at least 1 replica
	if desiredReplicas < 1 {
		desiredReplicas = 1
	}

	return desiredReplicas, nil
}

// ShouldScale determines if scaling is necessary based on tolerance
func (p *ProportionalScalingStrategy) ShouldScale(current int, desired int, metrics []ScalingMetric) bool {
	if len(metrics) == 0 {
		return false
	}

	// Find the primary metric
	var primaryMetric *ScalingMetric
	for _, metric := range metrics {
		if metric.Type == ScalingMetricTypeCPU {
			primaryMetric = &metric
			break
		}
	}

	if primaryMetric == nil {
		return false
	}

	// Calculate the deviation from target
	deviation := abs(primaryMetric.Value - primaryMetric.Threshold)
	return deviation > p.Tolerance
}

// GetScalingReason returns a human-readable reason for scaling
func (p *ProportionalScalingStrategy) GetScalingReason(current int, desired int, metrics []ScalingMetric) string {
	if current == desired {
		return "No scaling needed"
	}

	if len(metrics) == 0 {
		return "No metrics available"
	}

	var primaryMetric *ScalingMetric
	for _, metric := range metrics {
		if metric.Type == ScalingMetricTypeCPU {
			primaryMetric = &metric
			break
		}
	}

	if primaryMetric == nil {
		return "Primary metric not found"
	}

	if desired > current {
		return fmt.Sprintf("Scaling up: %s utilization (%.1f%%) exceeds target (%.1f%%)",
			primaryMetric.Name, primaryMetric.Value, primaryMetric.Threshold)
	}

	return fmt.Sprintf("Scaling down: %s utilization (%.1f%%) below target (%.1f%%)",
		primaryMetric.Name, primaryMetric.Value, primaryMetric.Threshold)
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// StepScalingStrategy implements step-based scaling with predefined thresholds
type StepScalingStrategy struct {
	// Steps define the scaling steps based on metric values
	Steps []ScalingStep
}

// ScalingStep defines a scaling step for step-based scaling
type ScalingStep struct {
	// MetricThreshold is the threshold value for this step
	MetricThreshold float64

	// ScalingAdjustment is the number of replicas to add/remove
	ScalingAdjustment int

	// AdjustmentType defines how the adjustment is applied
	AdjustmentType ScalingAdjustmentType
}

// ScalingAdjustmentType defines how scaling adjustments are applied
type ScalingAdjustmentType string

const (
	// ScalingAdjustmentTypeAbsolute sets the absolute number of replicas
	ScalingAdjustmentTypeAbsolute ScalingAdjustmentType = "absolute"

	// ScalingAdjustmentTypeRelative adds/removes a relative number of replicas
	ScalingAdjustmentTypeRelative ScalingAdjustmentType = "relative"

	// ScalingAdjustmentTypePercentage scales by a percentage of current replicas
	ScalingAdjustmentTypePercentage ScalingAdjustmentType = "percentage"
)
