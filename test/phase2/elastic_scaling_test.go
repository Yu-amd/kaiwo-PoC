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

package phase2

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
	"github.com/silogen/kaiwo/pkg/scaling"
)

// MockMetricsCollector implements the MetricsCollector interface for testing
type MockMetricsCollector struct {
	metrics map[string][]scaling.ScalingMetric
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string][]scaling.ScalingMetric),
	}
}

func (m *MockMetricsCollector) CollectMetrics(workload *v1alpha1.KaiwoJob) ([]scaling.ScalingMetric, error) {
	workloadID := fmt.Sprintf("%s/%s", workload.Namespace, workload.Name)
	if metrics, exists := m.metrics[workloadID]; exists {
		return metrics, nil
	}

	// Return default metrics
	return []scaling.ScalingMetric{
		{
			Name:      "cpu",
			Type:      scaling.ScalingMetricTypeCPU,
			Value:     75.0,
			Threshold: 70.0,
			Unit:      "percent",
			Timestamp: time.Now(),
		},
		{
			Name:      "memory",
			Type:      scaling.ScalingMetricTypeMemory,
			Value:     60.0,
			Threshold: 80.0,
			Unit:      "percent",
			Timestamp: time.Now(),
		},
	}, nil
}

func (m *MockMetricsCollector) GetHistoricalMetrics(workloadID string, duration time.Duration) ([]scaling.ScalingMetric, error) {
	return m.metrics[workloadID], nil
}

func (m *MockMetricsCollector) RegisterCustomMetric(name string, collector func() (float64, error)) error {
	return nil
}

func (m *MockMetricsCollector) SetMetrics(workloadID string, metrics []scaling.ScalingMetric) {
	m.metrics[workloadID] = metrics
}

func TestElasticController_RegisterWorkload(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	tests := []struct {
		name          string
		kaiwoJob      *v1alpha1.KaiwoJob
		expectedError bool
	}{
		{
			name: "Valid workload with elastic scaling enabled",
			kaiwoJob: &v1alpha1.KaiwoJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Spec: v1alpha1.KaiwoJobSpec{
					CommonMetaSpec: v1alpha1.CommonMetaSpec{
						Replicas: intPtr(2),
						ElasticScaling: &v1alpha1.ElasticScalingSpec{
							Enabled:     true,
							MinReplicas: 1,
							MaxReplicas: 5,
							Metrics: []v1alpha1.ScalingMetricSpec{
								{
									Type:      "cpu",
									Threshold: 70.0,
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Workload without elastic scaling",
			kaiwoJob: &v1alpha1.KaiwoJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-no-scaling",
					Namespace: "default",
				},
				Spec: v1alpha1.KaiwoJobSpec{
					CommonMetaSpec: v1alpha1.CommonMetaSpec{
						Replicas: intPtr(2),
					},
				},
			},
			expectedError: true,
		},
		{
			name: "Workload with elastic scaling disabled",
			kaiwoJob: &v1alpha1.KaiwoJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job-disabled",
					Namespace: "default",
				},
				Spec: v1alpha1.KaiwoJobSpec{
					CommonMetaSpec: v1alpha1.CommonMetaSpec{
						Replicas: intPtr(2),
						ElasticScaling: &v1alpha1.ElasticScalingSpec{
							Enabled: false,
						},
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.RegisterWorkload(tt.kaiwoJob)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectedError {
				// Verify workload was registered
				workloadID := fmt.Sprintf("%s/%s", tt.kaiwoJob.Namespace, tt.kaiwoJob.Name)
				workload, err := controller.GetWorkloadStatus(workloadID)
				if err != nil {
					t.Errorf("Failed to get workload status: %v", err)
				}
				if workload.KaiwoJob.Name != tt.kaiwoJob.Name {
					t.Errorf("Expected workload name %s, got %s", tt.kaiwoJob.Name, workload.KaiwoJob.Name)
				}
			}
		})
	}
}

func TestElasticController_EvaluateScaling(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	// Create a test workload
	kaiwoJob := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scaling-test-job",
			Namespace: "default",
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Replicas: intPtr(2),
				ElasticScaling: &v1alpha1.ElasticScalingSpec{
					Enabled:     true,
					MinReplicas: 1,
					MaxReplicas: 5,
					ScalingPolicy: &v1alpha1.ScalingPolicySpec{
						ScaleUpRate:   2,
						ScaleDownRate: 1,
						Cooldown:      &metav1.Duration{Duration: 5 * time.Minute},
					},
				},
			},
		},
	}

	err := controller.RegisterWorkload(kaiwoJob)
	if err != nil {
		t.Fatalf("Failed to register workload: %v", err)
	}

	workloadID := fmt.Sprintf("%s/%s", kaiwoJob.Namespace, kaiwoJob.Name)

	tests := []struct {
		name                string
		metrics             []scaling.ScalingMetric
		expectedDirection   scaling.ScalingDirection
		expectedMinReplicas int
		expectedMaxReplicas int
	}{
		{
			name: "High CPU utilization - scale up",
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     90.0,
					Threshold: 70.0,
					Unit:      "percent",
					Timestamp: time.Now(),
				},
			},
			expectedDirection:   scaling.ScalingDirectionUp,
			expectedMinReplicas: 2,
			expectedMaxReplicas: 5,
		},
		{
			name: "Low CPU utilization - scale down",
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     30.0,
					Threshold: 70.0,
					Unit:      "percent",
					Timestamp: time.Now(),
				},
			},
			expectedDirection:   scaling.ScalingDirectionDown,
			expectedMinReplicas: 1,
			expectedMaxReplicas: 2,
		},
		{
			name: "Normal CPU utilization - no scaling",
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     70.0,
					Threshold: 70.0,
					Unit:      "percent",
					Timestamp: time.Now(),
				},
			},
			expectedDirection:   scaling.ScalingDirectionNone,
			expectedMinReplicas: 2,
			expectedMaxReplicas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set metrics for the workload
			metricsCollector.SetMetrics(workloadID, tt.metrics)

			// Evaluate scaling
			decision, err := controller.EvaluateScaling(workloadID)
			if err != nil {
				t.Errorf("Failed to evaluate scaling: %v", err)
				return
			}

			if decision.WorkloadID != workloadID {
				t.Errorf("Expected workload ID %s, got %s", workloadID, decision.WorkloadID)
			}

			if decision.ScalingDirection != tt.expectedDirection {
				t.Errorf("Expected scaling direction %v, got %v", tt.expectedDirection, decision.ScalingDirection)
			}

			if decision.DesiredReplicas < tt.expectedMinReplicas || decision.DesiredReplicas > tt.expectedMaxReplicas {
				t.Errorf("Desired replicas %d not within expected range [%d, %d]",
					decision.DesiredReplicas, tt.expectedMinReplicas, tt.expectedMaxReplicas)
			}

			if decision.CurrentReplicas != 2 {
				t.Errorf("Expected current replicas to be 2, got %d", decision.CurrentReplicas)
			}

			if decision.Confidence <= 0 || decision.Confidence > 1 {
				t.Errorf("Expected confidence to be between 0 and 1, got %f", decision.Confidence)
			}
		})
	}
}

func TestElasticController_ExecuteScaling(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	// Create a test workload
	kaiwoJob := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "execution-test-job",
			Namespace: "default",
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Replicas: intPtr(2),
				ElasticScaling: &v1alpha1.ElasticScalingSpec{
					Enabled:     true,
					MinReplicas: 1,
					MaxReplicas: 5,
				},
			},
		},
	}

	err := controller.RegisterWorkload(kaiwoJob)
	if err != nil {
		t.Fatalf("Failed to register workload: %v", err)
	}

	workloadID := fmt.Sprintf("%s/%s", kaiwoJob.Namespace, kaiwoJob.Name)

	tests := []struct {
		name          string
		decision      *scaling.ScalingDecision
		expectedError bool
	}{
		{
			name: "Valid scale up decision",
			decision: &scaling.ScalingDecision{
				WorkloadID:       workloadID,
				CurrentReplicas:  2,
				DesiredReplicas:  4,
				ScalingDirection: scaling.ScalingDirectionUp,
				Reason:           "High CPU utilization",
				Timestamp:        time.Now(),
				Confidence:       0.9,
			},
			expectedError: false,
		},
		{
			name: "Valid scale down decision",
			decision: &scaling.ScalingDecision{
				WorkloadID:       workloadID,
				CurrentReplicas:  4,
				DesiredReplicas:  2,
				ScalingDirection: scaling.ScalingDirectionDown,
				Reason:           "Low CPU utilization",
				Timestamp:        time.Now(),
				Confidence:       0.8,
			},
			expectedError: false,
		},
		{
			name: "No scaling needed",
			decision: &scaling.ScalingDecision{
				WorkloadID:       workloadID,
				CurrentReplicas:  2,
				DesiredReplicas:  2,
				ScalingDirection: scaling.ScalingDirectionNone,
				Reason:           "No scaling needed",
				Timestamp:        time.Now(),
				Confidence:       0.7,
			},
			expectedError: false,
		},
		{
			name: "Invalid workload ID",
			decision: &scaling.ScalingDecision{
				WorkloadID:       "non-existent/workload",
				CurrentReplicas:  2,
				DesiredReplicas:  4,
				ScalingDirection: scaling.ScalingDirectionUp,
				Reason:           "Test",
				Timestamp:        time.Now(),
				Confidence:       0.9,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.ExecuteScaling(tt.decision)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectedError && tt.decision.ScalingDirection != scaling.ScalingDirectionNone {
				// Verify the workload was updated (in a real implementation)
				workload, err := controller.GetWorkloadStatus(tt.decision.WorkloadID)
				if err != nil {
					t.Errorf("Failed to get workload status: %v", err)
				}

				// The scaling simulation should complete quickly
				if workload.Status == scaling.ElasticWorkloadStatusScaling {
					// Wait a moment for the scaling to complete
					time.Sleep(200 * time.Millisecond)
					workload, _ = controller.GetWorkloadStatus(tt.decision.WorkloadID)
				}

				if workload.Status == scaling.ElasticWorkloadStatusError {
					t.Errorf("Scaling failed - workload status is error")
				}
			}
		})
	}
}

func TestElasticController_GetMetrics(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	metrics := controller.GetMetrics()
	if metrics == nil {
		t.Errorf("Expected metrics but got nil")
	}

	// Initial metrics should be zero
	if metrics.TotalWorkloads != 0 {
		t.Errorf("Expected TotalWorkloads to be 0, got %d", metrics.TotalWorkloads)
	}
	if metrics.ActiveScalingOperations != 0 {
		t.Errorf("Expected ActiveScalingOperations to be 0, got %d", metrics.ActiveScalingOperations)
	}
	if metrics.SuccessfulScalingOperations != 0 {
		t.Errorf("Expected SuccessfulScalingOperations to be 0, got %d", metrics.SuccessfulScalingOperations)
	}
	if metrics.FailedScalingOperations != 0 {
		t.Errorf("Expected FailedScalingOperations to be 0, got %d", metrics.FailedScalingOperations)
	}
}

func TestProportionalScalingStrategy(t *testing.T) {
	strategy := &scaling.ProportionalScalingStrategy{
		TargetUtilization: 70.0,
		Tolerance:         10.0,
	}

	tests := []struct {
		name            string
		currentReplicas int
		metrics         []scaling.ScalingMetric
		expectedMin     int
		expectedMax     int
		shouldScale     bool
	}{
		{
			name:            "High utilization - should scale up",
			currentReplicas: 2,
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     90.0,
					Threshold: 70.0,
				},
			},
			expectedMin: 3,
			expectedMax: 4,
			shouldScale: true,
		},
		{
			name:            "Low utilization - should scale down",
			currentReplicas: 4,
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     40.0,
					Threshold: 70.0,
				},
			},
			expectedMin: 2,
			expectedMax: 3,
			shouldScale: true,
		},
		{
			name:            "Normal utilization - no scaling",
			currentReplicas: 3,
			metrics: []scaling.ScalingMetric{
				{
					Name:      "cpu",
					Type:      scaling.ScalingMetricTypeCPU,
					Value:     70.0,
					Threshold: 70.0,
				},
			},
			expectedMin: 3,
			expectedMax: 3,
			shouldScale: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &scaling.ScalingPolicy{
				StabilizationWindow: 2 * time.Minute,
				CooldownPeriod:      5 * time.Minute,
			}

			desiredReplicas, err := strategy.CalculateDesiredReplicas(tt.currentReplicas, tt.metrics, policy)
			if err != nil {
				t.Errorf("Failed to calculate desired replicas: %v", err)
				return
			}

			if desiredReplicas < tt.expectedMin || desiredReplicas > tt.expectedMax {
				t.Errorf("Desired replicas %d not within expected range [%d, %d]",
					desiredReplicas, tt.expectedMin, tt.expectedMax)
			}

			shouldScale := strategy.ShouldScale(tt.currentReplicas, desiredReplicas, tt.metrics)
			if shouldScale != tt.shouldScale {
				t.Errorf("Expected should scale to be %v, got %v", tt.shouldScale, shouldScale)
			}

			reason := strategy.GetScalingReason(tt.currentReplicas, desiredReplicas, tt.metrics)
			if reason == "" {
				t.Errorf("Expected scaling reason but got empty string")
			}
		})
	}
}

// Benchmark tests for elastic scaling performance
func BenchmarkElasticController_EvaluateScaling(b *testing.B) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	// Create a test workload
	kaiwoJob := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-job",
			Namespace: "default",
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Replicas: intPtr(2),
				ElasticScaling: &v1alpha1.ElasticScalingSpec{
					Enabled:     true,
					MinReplicas: 1,
					MaxReplicas: 10,
				},
			},
		},
	}

	controller.RegisterWorkload(kaiwoJob)
	workloadID := fmt.Sprintf("%s/%s", kaiwoJob.Namespace, kaiwoJob.Name)

	// Set up metrics
	metrics := []scaling.ScalingMetric{
		{
			Name:      "cpu",
			Type:      scaling.ScalingMetricTypeCPU,
			Value:     75.0,
			Threshold: 70.0,
			Unit:      "percent",
			Timestamp: time.Now(),
		},
	}
	metricsCollector.SetMetrics(workloadID, metrics)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := controller.EvaluateScaling(workloadID)
		if err != nil {
			b.Errorf("Failed to evaluate scaling: %v", err)
		}
	}
}

func BenchmarkElasticController_ExecuteScaling(b *testing.B) {
	kubeClient := fake.NewSimpleClientset()
	metricsCollector := NewMockMetricsCollector()

	controller := scaling.NewElasticController(kubeClient, metricsCollector)

	// Create a test workload
	kaiwoJob := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-execution-job",
			Namespace: "default",
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Replicas: intPtr(2),
				ElasticScaling: &v1alpha1.ElasticScalingSpec{
					Enabled:     true,
					MinReplicas: 1,
					MaxReplicas: 10,
				},
			},
		},
	}

	controller.RegisterWorkload(kaiwoJob)
	workloadID := fmt.Sprintf("%s/%s", kaiwoJob.Namespace, kaiwoJob.Name)

	decision := &scaling.ScalingDecision{
		WorkloadID:       workloadID,
		CurrentReplicas:  2,
		DesiredReplicas:  4,
		ScalingDirection: scaling.ScalingDirectionUp,
		Reason:           "Benchmark test",
		Timestamp:        time.Now(),
		Confidence:       0.9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := controller.ExecuteScaling(decision)
		if err != nil {
			b.Errorf("Failed to execute scaling: %v", err)
		}
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}
