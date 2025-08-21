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

package resource_optimization

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/api/core/v1"
	batchv1 "k8s.io/api/batch/v1"
)

// BenchmarkDynamicAllocationAdjustment benchmarks dynamic allocation adjustment performance
func BenchmarkDynamicAllocationAdjustment(b *testing.B) {
	// Setup test data with varying utilization patterns
	jobs := make([]*v1alpha1.KaiwoJob, 50)
	for i := 0; i < 50; i++ {
		_ = float64(i%100) / 100.0 // 0-99% utilization (unused in mock)
		jobs[i] = &v1alpha1.KaiwoJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-job-%d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.KaiwoJobSpec{
				CommonMetaSpec: v1alpha1.CommonMetaSpec{
					User:      "test@amd.com",
					GpuVendor: "amd",
				},
				EntryPoint: "sleep 1",
				Job: &batchv1.Job{
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "busybox:latest",
										Resources: v1.ResourceRequirements{
											Requests: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("4"),
												v1.ResourceMemory: resource.MustParse("8Gi"),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("4"),
												v1.ResourceMemory: resource.MustParse("8Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate dynamic allocation adjustment
		ctx := context.Background()
		optimizer := NewDynamicAllocationOptimizer()
		
		for _, job := range jobs {
			adjustment := optimizer.AdjustAllocation(ctx, job, 0.5) // 50% utilization
			_ = adjustment // Use adjustment to avoid compiler optimization
		}
	}
}

// BenchmarkMemoryOptimization benchmarks memory optimization performance
func BenchmarkMemoryOptimization(b *testing.B) {
	// Setup test data with memory fragmentation scenarios
	memoryPatterns := []struct {
		name     string
		request  string
		limit    string
		fragmentation float64
	}{
		{"small-fragmented", "1Gi", "2Gi", 0.8},
		{"medium-fragmented", "4Gi", "8Gi", 0.6},
		{"large-fragmented", "16Gi", "32Gi", 0.4},
		{"well-allocated", "8Gi", "8Gi", 0.1},
	}

	jobs := make([]*v1alpha1.KaiwoJob, 100)
	for i := 0; i < 100; i++ {
		pattern := memoryPatterns[i%len(memoryPatterns)]
		jobs[i] = &v1alpha1.KaiwoJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-job-%d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.KaiwoJobSpec{
				CommonMetaSpec: v1alpha1.CommonMetaSpec{
					User:      "test@amd.com",
					GpuVendor: "amd",
				},
				EntryPoint: "sleep 1",
				Job: &batchv1.Job{
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "busybox:latest",
										Resources: v1.ResourceRequirements{
											Requests: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2"),
												v1.ResourceMemory: resource.MustParse(pattern.request),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2"),
												v1.ResourceMemory: resource.MustParse(pattern.limit),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate memory optimization
		ctx := context.Background()
		optimizer := NewMemoryOptimizer()
		
		for _, job := range jobs {
			optimization := optimizer.OptimizeMemory(ctx, job, 0.5)
			_ = optimization // Use optimization to avoid compiler optimization
		}
	}
}

// BenchmarkPerformanceDrivenScheduling benchmarks performance-driven scheduling
func BenchmarkPerformanceDrivenScheduling(b *testing.B) {
	// Setup test data with performance metrics
	performanceProfiles := []struct {
		name           string
		expectedThroughput float64
		latencySensitivity bool
		memoryEfficiency   float64
	}{
		{"high-throughput", 1000.0, false, 0.8},
		{"low-latency", 500.0, true, 0.6},
		{"memory-efficient", 750.0, false, 0.9},
		{"balanced", 600.0, false, 0.7},
	}

	jobs := make([]*v1alpha1.KaiwoJob, 100)
	for i := 0; i < 100; i++ {
		_ = performanceProfiles[i%len(performanceProfiles)] // profile unused in mock
		jobs[i] = &v1alpha1.KaiwoJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-job-%d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.KaiwoJobSpec{
				CommonMetaSpec: v1alpha1.CommonMetaSpec{
					User:      "test@amd.com",
					GpuVendor: "amd",
				},
				EntryPoint: "sleep 1",
				Job: &batchv1.Job{
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "busybox:latest",
										Resources: v1.ResourceRequirements{
											Requests: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("4"),
												v1.ResourceMemory: resource.MustParse("8Gi"),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("4"),
												v1.ResourceMemory: resource.MustParse("8Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate performance-driven scheduling
		ctx := context.Background()
		scheduler := NewPerformanceDrivenScheduler()
		
		for _, job := range jobs {
			decision := scheduler.ScheduleForPerformance(ctx, job)
			_ = decision // Use decision to avoid compiler optimization
		}
	}
}

// BenchmarkResourceRebalancing benchmarks resource rebalancing performance
func BenchmarkResourceRebalancing(b *testing.B) {
	// Setup cluster state with imbalanced resources
	clusterState := &ClusterState{
		Nodes: []NodeState{
			{Name: "node-1", CPUUtilization: 0.9, MemoryUtilization: 0.8, GPUUtilization: 0.95},
			{Name: "node-2", CPUUtilization: 0.3, MemoryUtilization: 0.2, GPUUtilization: 0.1},
			{Name: "node-3", CPUUtilization: 0.7, MemoryUtilization: 0.6, GPUUtilization: 0.8},
			{Name: "node-4", CPUUtilization: 0.1, MemoryUtilization: 0.1, GPUUtilization: 0.05},
		},
	}

	jobs := make([]*v1alpha1.KaiwoJob, 50)
	for i := 0; i < 50; i++ {
		jobs[i] = &v1alpha1.KaiwoJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-job-%d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.KaiwoJobSpec{
				CommonMetaSpec: v1alpha1.CommonMetaSpec{
					User:      "test@amd.com",
					GpuVendor: "amd",
				},
				EntryPoint: "sleep 1",
				Job: &batchv1.Job{
					Spec: batchv1.JobSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "busybox:latest",
										Resources: v1.ResourceRequirements{
											Requests: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2"),
												v1.ResourceMemory: resource.MustParse("4Gi"),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("2"),
												v1.ResourceMemory: resource.MustParse("4Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate resource rebalancing
		ctx := context.Background()
		rebalancer := NewResourceRebalancer(clusterState)
		
		for _, job := range jobs {
			rebalancing := rebalancer.RebalanceResources(ctx, job)
			_ = rebalancing // Use rebalancing to avoid compiler optimization
		}
	}
}

// Mock types for benchmarking
type DynamicAllocationOptimizer struct{}

func NewDynamicAllocationOptimizer() *DynamicAllocationOptimizer {
	return &DynamicAllocationOptimizer{}
}

func (o *DynamicAllocationOptimizer) AdjustAllocation(ctx context.Context, job *v1alpha1.KaiwoJob, utilization float64) *AllocationAdjustment {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	return &AllocationAdjustment{}
}

type MemoryOptimizer struct{}

func NewMemoryOptimizer() *MemoryOptimizer {
	return &MemoryOptimizer{}
}

func (o *MemoryOptimizer) OptimizeMemory(ctx context.Context, job *v1alpha1.KaiwoJob, fragmentation float64) *MemoryOptimization {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	return &MemoryOptimization{}
}

type PerformanceDrivenScheduler struct{}

func NewPerformanceDrivenScheduler() *PerformanceDrivenScheduler {
	return &PerformanceDrivenScheduler{}
}

func (s *PerformanceDrivenScheduler) ScheduleForPerformance(ctx context.Context, job *v1alpha1.KaiwoJob) *PerformanceDecision {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	return &PerformanceDecision{}
}

type ResourceRebalancer struct {
	clusterState *ClusterState
}

func NewResourceRebalancer(clusterState *ClusterState) *ResourceRebalancer {
	return &ResourceRebalancer{clusterState: clusterState}
}

func (r *ResourceRebalancer) RebalanceResources(ctx context.Context, job *v1alpha1.KaiwoJob) *RebalancingAction {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	return &RebalancingAction{}
}

// Mock types
type AllocationAdjustment struct{}
type MemoryOptimization struct{}
type PerformanceDecision struct{}
type RebalancingAction struct{}

type ClusterState struct {
	Nodes []NodeState
}

type NodeState struct {
	Name              string
	CPUUtilization    float64
	MemoryUtilization float64
	GPUUtilization    float64
}
