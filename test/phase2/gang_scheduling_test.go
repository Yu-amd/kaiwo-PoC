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
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/silogen/kaiwo/pkg/scheduling/enhanced"
	"github.com/silogen/kaiwo/pkg/scheduling/gang"
)

func TestGangScheduler_ScheduleGang(t *testing.T) {
	// Create fake Kubernetes client
	kubeClient := fake.NewSimpleClientset()

	// Create mock components
	priorityScheduler := &enhanced.PriorityScheduler{}
	resourceAllocator := &enhanced.ResourceAllocator{}
	loadBalancer := &enhanced.LoadBalancer{}

	// Create gang scheduler
	gangScheduler := gang.NewGangScheduler(kubeClient, priorityScheduler, resourceAllocator, loadBalancer)

	tests := []struct {
		name           string
		request        *gang.GangSchedulingRequest
		expectedStatus gang.GangSchedulingStatus
		expectedError  bool
	}{
		{
			name: "Valid gang scheduling request",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job-1",
				JobName:    "distributed-training",
				Namespace:  "default",
				MinMembers: 4,
				MaxMembers: 4,
				Timeout:    30 * time.Second,
				Priority:   100,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"amd.com/gpu":         resource.MustParse("1"),
					},
				},
				Labels: map[string]string{
					"app": "distributed-training",
				},
				CreatedAt: time.Now(),
			},
			expectedStatus: gang.GangSchedulingStatusScheduled,
			expectedError:  false,
		},
		{
			name: "Invalid gang request - no job ID",
			request: &gang.GangSchedulingRequest{
				JobName:    "test-job",
				MinMembers: 2,
				Timeout:    30 * time.Second,
			},
			expectedStatus: gang.GangSchedulingStatusFailed,
			expectedError:  true,
		},
		{
			name: "Invalid gang request - zero min members",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job-2",
				JobName:    "test-job",
				MinMembers: 0,
				Timeout:    30 * time.Second,
			},
			expectedStatus: gang.GangSchedulingStatusFailed,
			expectedError:  true,
		},
		{
			name: "Invalid gang request - zero timeout",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job-3",
				JobName:    "test-job",
				MinMembers: 2,
				Timeout:    0,
			},
			expectedStatus: gang.GangSchedulingStatusFailed,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			response, err := gangScheduler.ScheduleGang(ctx, tt.request)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if response.Status != tt.expectedStatus {
					t.Errorf("Expected status %v, got %v", tt.expectedStatus, response.Status)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if response.Status != tt.expectedStatus {
					t.Errorf("Expected status %v, got %v", tt.expectedStatus, response.Status)
				}
				if response.RequestID != tt.request.JobID {
					t.Errorf("Expected request ID %s, got %s", tt.request.JobID, response.RequestID)
				}
			}
		})
	}
}

func TestGangScheduler_CancelGang(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	priorityScheduler := &enhanced.PriorityScheduler{}
	resourceAllocator := &enhanced.ResourceAllocator{}
	loadBalancer := &enhanced.LoadBalancer{}

	gangScheduler := gang.NewGangScheduler(kubeClient, priorityScheduler, resourceAllocator, loadBalancer)

	// First, schedule a gang
	request := &gang.GangSchedulingRequest{
		JobID:      "test-job-cancel",
		JobName:    "test-job",
		Namespace:  "default",
		MinMembers: 2,
		Timeout:    30 * time.Second,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("1"),
			},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	_, err := gangScheduler.ScheduleGang(ctx, request)
	if err != nil {
		t.Fatalf("Failed to schedule gang: %v", err)
	}

	// Wait a moment for the gang to be processed
	time.Sleep(100 * time.Millisecond)

	// Test cancelling existing gang
	t.Run("Cancel existing gang", func(t *testing.T) {
		// We need to get the actual gang ID, which would be generated
		gangs, err := gangScheduler.ListGangs(ctx)
		if err != nil {
			t.Fatalf("Failed to list gangs: %v", err)
		}

		if len(gangs) == 0 {
			t.Skip("No gangs found to cancel")
		}

		gangID := gangs[0].ID
		err = gangScheduler.CancelGang(ctx, gangID)
		if err != nil {
			t.Errorf("Failed to cancel gang: %v", err)
		}

		// Verify gang status
		gangStatus, err := gangScheduler.GetGangStatus(ctx, gangID)
		if err != nil {
			t.Errorf("Failed to get gang status: %v", err)
		}
		if gangStatus.Status != gang.GangSchedulingStatusCancelled {
			t.Errorf("Expected gang status to be cancelled, got %v", gangStatus.Status)
		}
	})

	// Test cancelling non-existent gang
	t.Run("Cancel non-existent gang", func(t *testing.T) {
		err := gangScheduler.CancelGang(ctx, "non-existent-gang")
		if err == nil {
			t.Errorf("Expected error when cancelling non-existent gang")
		}
	})
}

func TestGangScheduler_GetMetrics(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	priorityScheduler := &enhanced.PriorityScheduler{}
	resourceAllocator := &enhanced.ResourceAllocator{}
	loadBalancer := &enhanced.LoadBalancer{}

	gangScheduler := gang.NewGangScheduler(kubeClient, priorityScheduler, resourceAllocator, loadBalancer)

	ctx := context.Background()
	metrics, err := gangScheduler.GetMetrics(ctx)
	if err != nil {
		t.Errorf("Failed to get metrics: %v", err)
	}

	if metrics == nil {
		t.Errorf("Expected metrics but got nil")
	}

	// Initial metrics should be zero
	if metrics.TotalGangs != 0 {
		t.Errorf("Expected TotalGangs to be 0, got %d", metrics.TotalGangs)
	}
	if metrics.SuccessfulGangs != 0 {
		t.Errorf("Expected SuccessfulGangs to be 0, got %d", metrics.SuccessfulGangs)
	}
	if metrics.FailedGangs != 0 {
		t.Errorf("Expected FailedGangs to be 0, got %d", metrics.FailedGangs)
	}
}

func TestValidateGangRequest(t *testing.T) {
	tests := []struct {
		name          string
		request       *gang.GangSchedulingRequest
		expectedError bool
	}{
		{
			name: "Valid request",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job",
				MinMembers: 2,
				Timeout:    30 * time.Second,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			},
			expectedError: false,
		},
		{
			name:          "Nil request",
			request:       nil,
			expectedError: true,
		},
		{
			name: "Empty job ID",
			request: &gang.GangSchedulingRequest{
				JobID:      "",
				MinMembers: 2,
				Timeout:    30 * time.Second,
			},
			expectedError: true,
		},
		{
			name: "Zero min members",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job",
				MinMembers: 0,
				Timeout:    30 * time.Second,
			},
			expectedError: true,
		},
		{
			name: "Max members less than min members",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job",
				MinMembers: 4,
				MaxMembers: 2,
				Timeout:    30 * time.Second,
			},
			expectedError: true,
		},
		{
			name: "Zero timeout",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job",
				MinMembers: 2,
				Timeout:    0,
			},
			expectedError: true,
		},
		{
			name: "Empty resource requests",
			request: &gang.GangSchedulingRequest{
				JobID:      "test-job",
				MinMembers: 2,
				Timeout:    30 * time.Second,
				Resources:  corev1.ResourceRequirements{},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gang.ValidateGangRequest(tt.request)
			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestResourceCalculator_CalculateGangResources(t *testing.T) {
	calculator := &gang.ResourceCalculator{}

	memberResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			"amd.com/gpu":         resource.MustParse("1"),
		},
	}

	tests := []struct {
		name    string
		members int
		want    map[corev1.ResourceName]int64
	}{
		{
			name:    "Single member",
			members: 1,
			want: map[corev1.ResourceName]int64{
				corev1.ResourceCPU:    2000,       // 2 cores in millicores
				corev1.ResourceMemory: 4294967296, // 4Gi in bytes
				"amd.com/gpu":         1,
			},
		},
		{
			name:    "Multiple members",
			members: 4,
			want: map[corev1.ResourceName]int64{
				corev1.ResourceCPU:    8000,        // 8 cores in millicores
				corev1.ResourceMemory: 17179869184, // 16Gi in bytes
				"amd.com/gpu":         4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculateGangResources(tt.members, memberResources)

			for resourceName, expectedValue := range tt.want {
				if actualValue, exists := result[resourceName]; !exists {
					t.Errorf("Expected resource %s not found in result", resourceName)
				} else if actualValue.Value() != expectedValue {
					t.Errorf("Resource %s: expected %d, got %d",
						resourceName, expectedValue, actualValue.Value())
				}
			}
		})
	}
}

func TestResourceCalculator_CalculateResourceEfficiency(t *testing.T) {
	calculator := &gang.ResourceCalculator{}

	tests := []struct {
		name      string
		allocated corev1.ResourceList
		requested corev1.ResourceList
		expected  float64
	}{
		{
			name: "Perfect efficiency",
			allocated: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			requested: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			expected: 1.0,
		},
		{
			name: "50% efficiency",
			allocated: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
			requested: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
			expected: 0.5,
		},
		{
			name:      "Empty requested resources",
			allocated: corev1.ResourceList{},
			requested: corev1.ResourceList{},
			expected:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			efficiency := calculator.CalculateResourceEfficiency(tt.allocated, tt.requested)
			if efficiency != tt.expected {
				t.Errorf("Expected efficiency %f, got %f", tt.expected, efficiency)
			}
		})
	}
}

// Benchmark tests for gang scheduling performance
func BenchmarkGangScheduler_ScheduleGang(b *testing.B) {
	kubeClient := fake.NewSimpleClientset()
	priorityScheduler := &enhanced.PriorityScheduler{}
	resourceAllocator := &enhanced.ResourceAllocator{}
	loadBalancer := &enhanced.LoadBalancer{}

	gangScheduler := gang.NewGangScheduler(kubeClient, priorityScheduler, resourceAllocator, loadBalancer)

	request := &gang.GangSchedulingRequest{
		JobID:      "benchmark-job",
		JobName:    "benchmark",
		Namespace:  "default",
		MinMembers: 4,
		Timeout:    30 * time.Second,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
				"amd.com/gpu":         resource.MustParse("1"),
			},
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Update job ID for each iteration
		request.JobID = fmt.Sprintf("benchmark-job-%d", i)

		_, err := gangScheduler.ScheduleGang(ctx, request)
		if err != nil {
			b.Errorf("Gang scheduling failed: %v", err)
		}
	}
}

func BenchmarkResourceCalculator_CalculateGangResources(b *testing.B) {
	calculator := &gang.ResourceCalculator{}

	memberResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
			"amd.com/gpu":         resource.MustParse("1"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculator.CalculateGangResources(4, memberResources)
	}
}
