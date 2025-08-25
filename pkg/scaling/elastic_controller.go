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
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// ElasticController manages elastic scaling for Kaiwo workloads
type ElasticController struct {
	// kubeClient is the Kubernetes client
	kubeClient kubernetes.Interface

	// config contains the controller configuration
	config *ElasticControllerConfig

	// activeWorkloads tracks workloads that have elastic scaling enabled
	activeWorkloads map[string]*ElasticWorkload

	// metricsCollector collects metrics for scaling decisions
	metricsCollector MetricsCollector

	// scalingStrategy determines how to calculate desired replicas
	scalingStrategy ScalingStrategy

	// predictiveScaler provides predictive scaling capabilities
	predictiveScaler PredictiveScaler

	// scalingHistory stores scaling decisions and results
	scalingHistory map[string][]*ScalingDecision

	// mutex for thread safety
	mu sync.RWMutex

	// context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// metrics for the elastic controller
	metrics *ElasticControllerMetrics
}

// ElasticControllerConfig contains configuration for the elastic controller
type ElasticControllerConfig struct {
	// MetricsCollectionInterval is how often to collect metrics
	MetricsCollectionInterval time.Duration

	// EvaluationInterval is how often to evaluate scaling decisions
	EvaluationInterval time.Duration

	// DefaultPolicy is the default scaling policy for workloads
	DefaultPolicy *ScalingPolicy

	// MaxConcurrentScaling is the maximum number of concurrent scaling operations
	MaxConcurrentScaling int

	// ScalingHistoryLimit is the maximum number of scaling decisions to retain
	ScalingHistoryLimit int

	// EnablePredictiveScaling enables predictive scaling features
	EnablePredictiveScaling bool

	// PredictionWindow is the time window for predictive scaling
	PredictionWindow time.Duration

	// MinReplicasDefault is the default minimum replicas for elastic workloads
	MinReplicasDefault int

	// MaxReplicasDefault is the default maximum replicas for elastic workloads
	MaxReplicasDefault int
}

// ElasticWorkload represents a workload with elastic scaling enabled
type ElasticWorkload struct {
	// KaiwoJob is the reference to the KaiwoJob
	KaiwoJob *v1alpha1.KaiwoJob

	// Config is the elastic scaling configuration
	Config *v1alpha1.ElasticScalingSpec

	// CurrentReplicas is the current number of replicas
	CurrentReplicas int

	// LastScalingTime is when the last scaling operation occurred
	LastScalingTime time.Time

	// LastMetrics contains the most recent metrics
	LastMetrics []ScalingMetric

	// ScalingCooldown indicates if the workload is in cooldown period
	ScalingCooldown bool

	// PredictedLoad contains the predicted future load
	PredictedLoad *LoadPrediction

	// Status tracks the current scaling status
	Status ElasticWorkloadStatus
}

// ElasticWorkloadStatus represents the status of an elastic workload
type ElasticWorkloadStatus string

const (
	// ElasticWorkloadStatusActive indicates the workload is actively being monitored
	ElasticWorkloadStatusActive ElasticWorkloadStatus = "active"

	// ElasticWorkloadStatusScaling indicates the workload is currently scaling
	ElasticWorkloadStatusScaling ElasticWorkloadStatus = "scaling"

	// ElasticWorkloadStatusCooldown indicates the workload is in cooldown period
	ElasticWorkloadStatusCooldown ElasticWorkloadStatus = "cooldown"

	// ElasticWorkloadStatusError indicates there was an error with scaling
	ElasticWorkloadStatusError ElasticWorkloadStatus = "error"

	// ElasticWorkloadStatusPaused indicates scaling is paused for this workload
	ElasticWorkloadStatusPaused ElasticWorkloadStatus = "paused"
)

// ElasticControllerMetrics contains metrics for the elastic controller
type ElasticControllerMetrics struct {
	// TotalWorkloads is the total number of elastic workloads managed
	TotalWorkloads int64

	// ActiveScalingOperations is the number of active scaling operations
	ActiveScalingOperations int64

	// SuccessfulScalingOperations is the total number of successful scaling operations
	SuccessfulScalingOperations int64

	// FailedScalingOperations is the total number of failed scaling operations
	FailedScalingOperations int64

	// AverageScalingTime is the average time for scaling operations
	AverageScalingTime time.Duration

	// ResourceUtilizationAccuracy is the accuracy of resource utilization predictions
	ResourceUtilizationAccuracy float64

	// CostSavings is the estimated cost savings from elastic scaling
	CostSavings float64
}

// NewElasticController creates a new elastic controller
func NewElasticController(kubeClient kubernetes.Interface, metricsCollector MetricsCollector) *ElasticController {
	ctx, cancel := context.WithCancel(context.Background())

	config := &ElasticControllerConfig{
		MetricsCollectionInterval: 30 * time.Second,
		EvaluationInterval:        60 * time.Second,
		DefaultPolicy: &ScalingPolicy{
			ScaleUpPolicy: ScalingVelocityPolicy{
				Rate:     2,
				Period:   time.Minute,
				MaxBurst: 5,
			},
			ScaleDownPolicy: ScalingVelocityPolicy{
				Rate:     1,
				Period:   time.Minute,
				MaxBurst: 3,
			},
			StabilizationWindow: 3 * time.Minute,
			CooldownPeriod:      5 * time.Minute,
		},
		MaxConcurrentScaling:    10,
		ScalingHistoryLimit:     100,
		EnablePredictiveScaling: true,
		PredictionWindow:        30 * time.Minute,
		MinReplicasDefault:      1,
		MaxReplicasDefault:      10,
	}

	ec := &ElasticController{
		kubeClient:       kubeClient,
		config:           config,
		activeWorkloads:  make(map[string]*ElasticWorkload),
		metricsCollector: metricsCollector,
		scalingStrategy:  &ProportionalScalingStrategy{TargetUtilization: 70.0, Tolerance: 10.0},
		scalingHistory:   make(map[string][]*ScalingDecision),
		ctx:              ctx,
		cancel:           cancel,
		metrics:          &ElasticControllerMetrics{},
	}

	// Start the controller loops
	go ec.metricsCollectionLoop()
	go ec.scalingEvaluationLoop()

	return ec
}

// RegisterWorkload registers a workload for elastic scaling
func (ec *ElasticController) RegisterWorkload(kaiwoJob *v1alpha1.KaiwoJob) error {
	if kaiwoJob.Spec.ElasticScaling == nil || !kaiwoJob.Spec.ElasticScaling.Enabled {
		return fmt.Errorf("elastic scaling not enabled for workload %s", kaiwoJob.Name)
	}

	ec.mu.Lock()
	defer ec.mu.Unlock()

	workloadID := ec.getWorkloadID(kaiwoJob)

	elasticWorkload := &ElasticWorkload{
		KaiwoJob:        kaiwoJob,
		Config:          kaiwoJob.Spec.ElasticScaling,
		CurrentReplicas: ec.getCurrentReplicas(kaiwoJob),
		LastScalingTime: time.Now(),
		Status:          ElasticWorkloadStatusActive,
	}

	// Apply defaults if not specified
	if elasticWorkload.Config.MinReplicas == 0 {
		elasticWorkload.Config.MinReplicas = ec.config.MinReplicasDefault
	}
	if elasticWorkload.Config.MaxReplicas == 0 {
		elasticWorkload.Config.MaxReplicas = ec.config.MaxReplicasDefault
	}

	ec.activeWorkloads[workloadID] = elasticWorkload
	ec.metrics.TotalWorkloads++

	return nil
}

// UnregisterWorkload removes a workload from elastic scaling
func (ec *ElasticController) UnregisterWorkload(workloadID string) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if _, exists := ec.activeWorkloads[workloadID]; !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}

	delete(ec.activeWorkloads, workloadID)
	ec.metrics.TotalWorkloads--

	return nil
}

// EvaluateScaling evaluates scaling for a specific workload
func (ec *ElasticController) EvaluateScaling(workloadID string) (*ScalingDecision, error) {
	ec.mu.RLock()
	workload := ec.activeWorkloads[workloadID]
	ec.mu.RUnlock()

	if workload == nil {
		return nil, fmt.Errorf("workload %s not found", workloadID)
	}

	// Check if workload is in cooldown
	if ec.isInCooldown(workload) {
		return &ScalingDecision{
			WorkloadID:       workloadID,
			CurrentReplicas:  workload.CurrentReplicas,
			DesiredReplicas:  workload.CurrentReplicas,
			ScalingDirection: ScalingDirectionNone,
			Reason:           "Workload is in cooldown period",
			Timestamp:        time.Now(),
		}, nil
	}

	// Collect current metrics
	metrics, err := ec.metricsCollector.CollectMetrics(workload.KaiwoJob)
	if err != nil {
		return nil, fmt.Errorf("failed to collect metrics: %v", err)
	}

	// Update workload metrics
	ec.mu.Lock()
	workload.LastMetrics = metrics
	ec.mu.Unlock()

	// Calculate desired replicas
	policy := ec.getScalingPolicy(workload)
	desiredReplicas, err := ec.scalingStrategy.CalculateDesiredReplicas(
		workload.CurrentReplicas, metrics, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate desired replicas: %v", err)
	}

	// Apply min/max constraints
	desiredReplicas = ec.applyReplicaConstraints(workload, desiredReplicas)

	// Determine scaling direction
	scalingDirection := ec.determineScalingDirection(workload.CurrentReplicas, desiredReplicas)

	// Check if scaling is necessary
	if !ec.scalingStrategy.ShouldScale(workload.CurrentReplicas, desiredReplicas, metrics) {
		scalingDirection = ScalingDirectionNone
	}

	// Create scaling decision
	decision := &ScalingDecision{
		WorkloadID:       workloadID,
		CurrentReplicas:  workload.CurrentReplicas,
		DesiredReplicas:  desiredReplicas,
		ScalingDirection: scalingDirection,
		Reason:           ec.scalingStrategy.GetScalingReason(workload.CurrentReplicas, desiredReplicas, metrics),
		Metrics:          ec.metricsToMap(metrics),
		Timestamp:        time.Now(),
		Confidence:       ec.calculateConfidence(metrics),
	}

	return decision, nil
}

// ExecuteScaling executes a scaling decision
func (ec *ElasticController) ExecuteScaling(decision *ScalingDecision) error {
	if decision.ScalingDirection == ScalingDirectionNone {
		return nil // No scaling needed
	}

	ec.mu.Lock()
	workload := ec.activeWorkloads[decision.WorkloadID]
	if workload == nil {
		ec.mu.Unlock()
		return fmt.Errorf("workload %s not found", decision.WorkloadID)
	}

	// Mark as scaling
	workload.Status = ElasticWorkloadStatusScaling
	ec.metrics.ActiveScalingOperations++
	ec.mu.Unlock()

	startTime := time.Now()

	// Execute the scaling operation
	err := ec.scaleWorkload(workload, decision.DesiredReplicas)

	duration := time.Since(startTime)

	ec.mu.Lock()
	defer ec.mu.Unlock()

	if err != nil {
		workload.Status = ElasticWorkloadStatusError
		ec.metrics.FailedScalingOperations++
		return fmt.Errorf("failed to scale workload: %v", err)
	}

	// Update workload state
	workload.CurrentReplicas = decision.DesiredReplicas
	workload.LastScalingTime = time.Now()
	workload.Status = ElasticWorkloadStatusCooldown

	// Update metrics
	ec.metrics.SuccessfulScalingOperations++
	ec.metrics.ActiveScalingOperations--
	ec.updateAverageScalingTime(duration)

	// Store scaling decision in history
	ec.addToScalingHistory(decision)

	return nil
}

// GetWorkloadStatus returns the status of an elastic workload
func (ec *ElasticController) GetWorkloadStatus(workloadID string) (*ElasticWorkload, error) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	workload := ec.activeWorkloads[workloadID]
	if workload == nil {
		return nil, fmt.Errorf("workload %s not found", workloadID)
	}

	return workload, nil
}

// GetMetrics returns the elastic controller metrics
func (ec *ElasticController) GetMetrics() *ElasticControllerMetrics {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Return a copy of the metrics
	return &ElasticControllerMetrics{
		TotalWorkloads:              ec.metrics.TotalWorkloads,
		ActiveScalingOperations:     ec.metrics.ActiveScalingOperations,
		SuccessfulScalingOperations: ec.metrics.SuccessfulScalingOperations,
		FailedScalingOperations:     ec.metrics.FailedScalingOperations,
		AverageScalingTime:          ec.metrics.AverageScalingTime,
		ResourceUtilizationAccuracy: ec.metrics.ResourceUtilizationAccuracy,
		CostSavings:                 ec.metrics.CostSavings,
	}
}

// Private methods

func (ec *ElasticController) metricsCollectionLoop() {
	ticker := time.NewTicker(ec.config.MetricsCollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ec.collectMetricsForAllWorkloads()
		case <-ec.ctx.Done():
			return
		}
	}
}

func (ec *ElasticController) scalingEvaluationLoop() {
	ticker := time.NewTicker(ec.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ec.evaluateScalingForAllWorkloads()
		case <-ec.ctx.Done():
			return
		}
	}
}

func (ec *ElasticController) collectMetricsForAllWorkloads() {
	ec.mu.RLock()
	workloads := make(map[string]*ElasticWorkload)
	for id, workload := range ec.activeWorkloads {
		workloads[id] = workload
	}
	ec.mu.RUnlock()

	for _, workload := range workloads {
		if workload.Status == ElasticWorkloadStatusActive || workload.Status == ElasticWorkloadStatusCooldown {
			metrics, err := ec.metricsCollector.CollectMetrics(workload.KaiwoJob)
			if err != nil {
				continue
			}

			ec.mu.Lock()
			workload.LastMetrics = metrics
			ec.mu.Unlock()
		}
	}
}

func (ec *ElasticController) evaluateScalingForAllWorkloads() {
	ec.mu.RLock()
	workloadIDs := make([]string, 0, len(ec.activeWorkloads))
	for id := range ec.activeWorkloads {
		workloadIDs = append(workloadIDs, id)
	}
	ec.mu.RUnlock()

	for _, workloadID := range workloadIDs {
		decision, err := ec.EvaluateScaling(workloadID)
		if err != nil {
			continue
		}

		if decision.ScalingDirection != ScalingDirectionNone {
			go func(d *ScalingDecision) {
				ec.ExecuteScaling(d)
			}(decision)
		}
	}
}

func (ec *ElasticController) getWorkloadID(kaiwoJob *v1alpha1.KaiwoJob) string {
	return fmt.Sprintf("%s/%s", kaiwoJob.Namespace, kaiwoJob.Name)
}

func (ec *ElasticController) getCurrentReplicas(kaiwoJob *v1alpha1.KaiwoJob) int {
	if kaiwoJob.Spec.Replicas != nil {
		return *kaiwoJob.Spec.Replicas
	}
	return 1 // Default replica count
}

func (ec *ElasticController) isInCooldown(workload *ElasticWorkload) bool {
	policy := ec.getScalingPolicy(workload)
	return time.Since(workload.LastScalingTime) < policy.CooldownPeriod
}

func (ec *ElasticController) getScalingPolicy(workload *ElasticWorkload) *ScalingPolicy {
	if workload.Config.ScalingPolicy != nil {
		// Convert from ElasticScalingSpec to ScalingPolicy
		policy := &ScalingPolicy{
			ScaleUpPolicy: ScalingVelocityPolicy{
				Rate:     workload.Config.ScalingPolicy.ScaleUpRate,
				Period:   time.Minute,
				MaxBurst: workload.Config.ScalingPolicy.ScaleUpRate * 2,
			},
			ScaleDownPolicy: ScalingVelocityPolicy{
				Rate:     workload.Config.ScalingPolicy.ScaleDownRate,
				Period:   time.Minute,
				MaxBurst: workload.Config.ScalingPolicy.ScaleDownRate * 2,
			},
		}

		// Handle potentially nil duration fields
		if workload.Config.ScalingPolicy.StabilizationWindow != nil {
			policy.StabilizationWindow = workload.Config.ScalingPolicy.StabilizationWindow.Duration
		} else {
			policy.StabilizationWindow = 2 * time.Minute // default
		}

		if workload.Config.ScalingPolicy.Cooldown != nil {
			policy.CooldownPeriod = workload.Config.ScalingPolicy.Cooldown.Duration
		} else {
			policy.CooldownPeriod = 5 * time.Minute // default
		}

		return policy
	}
	return ec.config.DefaultPolicy
}

func (ec *ElasticController) applyReplicaConstraints(workload *ElasticWorkload, desired int) int {
	if desired < workload.Config.MinReplicas {
		return workload.Config.MinReplicas
	}
	if desired > workload.Config.MaxReplicas {
		return workload.Config.MaxReplicas
	}
	return desired
}

func (ec *ElasticController) determineScalingDirection(current, desired int) ScalingDirection {
	if desired > current {
		return ScalingDirectionUp
	} else if desired < current {
		return ScalingDirectionDown
	}
	return ScalingDirectionNone
}

func (ec *ElasticController) metricsToMap(metrics []ScalingMetric) map[string]float64 {
	result := make(map[string]float64)
	for _, metric := range metrics {
		result[metric.Name] = metric.Value
	}
	return result
}

func (ec *ElasticController) calculateConfidence(metrics []ScalingMetric) float64 {
	// Simple confidence calculation based on metric stability
	// In a real implementation, this would be more sophisticated
	return 0.8
}

func (ec *ElasticController) scaleWorkload(workload *ElasticWorkload, desiredReplicas int) error {
	// In a real implementation, this would update the KaiwoJob replicas
	// and trigger the appropriate Kubernetes controllers

	// For now, just simulate the scaling operation
	time.Sleep(time.Millisecond * 100) // Simulate scaling time
	return nil
}

func (ec *ElasticController) addToScalingHistory(decision *ScalingDecision) {
	history := ec.scalingHistory[decision.WorkloadID]
	history = append(history, decision)

	// Keep only the last N decisions
	if len(history) > ec.config.ScalingHistoryLimit {
		history = history[len(history)-ec.config.ScalingHistoryLimit:]
	}

	ec.scalingHistory[decision.WorkloadID] = history
}

func (ec *ElasticController) updateAverageScalingTime(duration time.Duration) {
	// Simple average calculation
	if ec.metrics.SuccessfulScalingOperations == 1 {
		ec.metrics.AverageScalingTime = duration
	} else {
		totalTime := ec.metrics.AverageScalingTime * time.Duration(ec.metrics.SuccessfulScalingOperations-1)
		ec.metrics.AverageScalingTime = (totalTime + duration) / time.Duration(ec.metrics.SuccessfulScalingOperations)
	}
}

// Shutdown gracefully shuts down the elastic controller
func (ec *ElasticController) Shutdown() {
	ec.cancel()
}
