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

package enhanced_scheduling

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

// BenchmarkPriorityScheduling benchmarks priority-based scheduling performance
func BenchmarkPriorityScheduling(b *testing.B) {
	// Setup test data
	jobs := make([]*v1alpha1.KaiwoJob, 100)
	for i := 0; i < 100; i++ {
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
												v1.ResourceCPU:    resource.MustParse("1"),
												v1.ResourceMemory: resource.MustParse("1Gi"),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("1"),
												v1.ResourceMemory: resource.MustParse("1Gi"),
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
		// Simulate priority scheduling
		ctx := context.Background()
		scheduler := NewEnhancedScheduler()
		
		for _, job := range jobs {
			scheduler.ScheduleJob(ctx, job)
		}
		
		// Get scheduling decisions
		decisions := scheduler.GetSchedulingDecisions()
		_ = decisions // Use decisions to avoid compiler optimization
	}
}

// BenchmarkResourceAwareAllocation benchmarks resource-aware allocation performance
func BenchmarkResourceAwareAllocation(b *testing.B) {
	// Setup test data with different resource requirements
	jobTypes := []struct {
		name     string
		cpu      string
		memory   string
		gpuCount int
	}{
		{"memory-intensive", "2", "8Gi", 1},
		{"compute-intensive", "8", "2Gi", 1},
		{"balanced", "4", "4Gi", 1},
		{"gpu-intensive", "2", "4Gi", 4},
	}

	jobs := make([]*v1alpha1.KaiwoJob, 50)
	for i := 0; i < 50; i++ {
		jobType := jobTypes[i%len(jobTypes)]
		jobs[i] = &v1alpha1.KaiwoJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-job-%d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.KaiwoJobSpec{
				CommonMetaSpec: v1alpha1.CommonMetaSpec{
					User:      "test@amd.com",
					GpuVendor: "amd",
					Gpus:      jobType.gpuCount,
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
												v1.ResourceCPU:    resource.MustParse(jobType.cpu),
												v1.ResourceMemory: resource.MustParse(jobType.memory),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse(jobType.cpu),
												v1.ResourceMemory: resource.MustParse(jobType.memory),
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
		// Simulate resource-aware allocation
		ctx := context.Background()
		allocator := NewResourceAwareAllocator()
		
		for _, job := range jobs {
			allocation := allocator.AllocateResources(ctx, job)
			_ = allocation // Use allocation to avoid compiler optimization
		}
	}
}

// BenchmarkDynamicLoadBalancing benchmarks dynamic load balancing performance
func BenchmarkDynamicLoadBalancing(b *testing.B) {
	// Setup cluster state with multiple nodes
	nodes := []ClusterNode{
		{Name: "node-1", AvailableCPU: 8, AvailableMemory: "16Gi", AvailableGPU: 4},
		{Name: "node-2", AvailableCPU: 4, AvailableMemory: "8Gi", AvailableGPU: 2},
		{Name: "node-3", AvailableCPU: 16, AvailableMemory: "32Gi", AvailableGPU: 8},
	}

	jobs := make([]*v1alpha1.KaiwoJob, 100)
	for i := 0; i < 100; i++ {
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
												v1.ResourceCPU:    resource.MustParse("1"),
												v1.ResourceMemory: resource.MustParse("1Gi"),
											},
											Limits: v1.ResourceList{
												v1.ResourceCPU:    resource.MustParse("1"),
												v1.ResourceMemory: resource.MustParse("1Gi"),
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
		// Simulate dynamic load balancing
		ctx := context.Background()
		balancer := NewDynamicLoadBalancer(nodes)
		
		for _, job := range jobs {
			node := balancer.SelectOptimalNode(ctx, job)
			_ = node // Use node to avoid compiler optimization
		}
	}
}

// Mock types for benchmarking (these would be replaced with actual implementations)
type EnhancedScheduler struct{}

func NewEnhancedScheduler() *EnhancedScheduler {
	return &EnhancedScheduler{}
}

func (s *EnhancedScheduler) ScheduleJob(ctx context.Context, job *v1alpha1.KaiwoJob) {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
}

func (s *EnhancedScheduler) GetSchedulingDecisions() []SchedulingDecision {
	return []SchedulingDecision{}
}

type ResourceAwareAllocator struct{}

func NewResourceAwareAllocator() *ResourceAwareAllocator {
	return &ResourceAwareAllocator{}
}

func (a *ResourceAwareAllocator) AllocateResources(ctx context.Context, job *v1alpha1.KaiwoJob) *ResourceAllocation {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	return &ResourceAllocation{}
}

type DynamicLoadBalancer struct {
	nodes []ClusterNode
}

func NewDynamicLoadBalancer(nodes []ClusterNode) *DynamicLoadBalancer {
	return &DynamicLoadBalancer{nodes: nodes}
}

func (b *DynamicLoadBalancer) SelectOptimalNode(ctx context.Context, job *v1alpha1.KaiwoJob) *ClusterNode {
	// Mock implementation
	time.Sleep(1 * time.Millisecond)
	if len(b.nodes) > 0 {
		return &b.nodes[0]
	}
	return nil
}

// Mock types
type SchedulingDecision struct{}
type ResourceAllocation struct{}
type ClusterNode struct {
	Name           string
	AvailableCPU   int
	AvailableMemory string
	AvailableGPU   int
}
