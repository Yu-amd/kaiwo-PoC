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

package profiling

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start profiling
	err := profiler.StartProfiling(ctx)
	if err != nil {
		t.Fatalf("Failed to start profiling: %v", err)
	}

	// Simulate some work
	for i := 0; i < 100; i++ {
		job := createTestJob(fmt.Sprintf("test-job-%d", i))
		start := time.Now()

		// Simulate scheduling work
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

		profiler.RecordJobScheduling(job, time.Since(start), true)
	}

	// Wait for some samples
	time.Sleep(1 * time.Second)

	// Stop profiling and get report
	report := profiler.StopProfiling()
	if report == nil {
		t.Fatal("Expected performance report, got nil")
	}

	// Validate report contents
	if report.TotalOperations == 0 {
		t.Error("Expected operations to be recorded")
	}

	if report.MemoryUsage == nil {
		t.Error("Expected memory usage metrics")
	}

	if report.CPUUsage == nil {
		t.Error("Expected CPU usage metrics")
	}

	if len(report.Recommendations) == 0 {
		t.Log("No recommendations generated (this is okay for short tests)")
	}

	t.Logf("Performance Report Summary:")
	t.Logf("  Total Operations: %d", report.TotalOperations)
	t.Logf("  Operations/Second: %.2f", report.OperationsPerSecond)
	t.Logf("  Memory Allocated: %d bytes", report.MemoryUsage.AllocatedBytes)
	t.Logf("  Goroutines: %d", report.CPUUsage.GoroutineCount)
	t.Logf("  Recommendations: %v", report.Recommendations)
}

// BenchmarkSchedulingPerformanceWithProfiling benchmarks scheduling with comprehensive profiling
func BenchmarkSchedulingPerformanceWithProfiling(b *testing.B) {
	scenarios := []struct {
		name        string
		jobCount    int
		complexity  string
		concurrency int
	}{
		{"Light-Load-Sequential", 10, "simple", 1},
		{"Medium-Load-Sequential", 50, "medium", 1},
		{"Heavy-Load-Sequential", 100, "complex", 1},
		{"Light-Load-Concurrent", 10, "simple", 5},
		{"Medium-Load-Concurrent", 50, "medium", 10},
		{"Heavy-Load-Concurrent", 100, "complex", 20},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create new profiler for each iteration
				profiler := NewPerformanceProfiler(50 * time.Millisecond)

				// Start profiling for this iteration
				profiler.StartProfiling(ctx)

				jobs := createTestJobsForScenario(scenario.jobCount, scenario.complexity)

				if scenario.concurrency == 1 {
					// Sequential processing
					for _, job := range jobs {
						start := time.Now()
						_ = processJob(job) // Mock processing
						profiler.RecordJobScheduling(job, time.Since(start), true)
					}
				} else {
					// Concurrent processing
					processConcurrentJobs(jobs, scenario.concurrency, profiler)
				}

				// Stop profiling and collect metrics
				report := profiler.StopProfiling()
				if report != nil {
					b.ReportMetric(report.OperationsPerSecond, "ops/sec")
					b.ReportMetric(float64(report.MemoryUsage.AllocatedBytes)/1024/1024, "MB")
					b.ReportMetric(float64(report.CPUUsage.GoroutineCount), "goroutines")
				}
			}
		})
	}
}

// BenchmarkMemoryPressureImpact benchmarks performance under different memory pressure scenarios
func BenchmarkMemoryPressureImpact(b *testing.B) {
	memoryPressureScenarios := []struct {
		name        string
		allocSize   int
		allocCount  int
		gcFrequency time.Duration
	}{
		{"Low-Memory-Pressure", 1024, 100, 1 * time.Second},
		{"Medium-Memory-Pressure", 1024 * 1024, 100, 500 * time.Millisecond},
		{"High-Memory-Pressure", 10 * 1024 * 1024, 50, 100 * time.Millisecond},
		{"Extreme-Memory-Pressure", 50 * 1024 * 1024, 20, 50 * time.Millisecond},
	}

	for _, scenario := range memoryPressureScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			profiler := NewPerformanceProfiler(100 * time.Millisecond)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				profiler.StartProfiling(ctx)

				// Create memory pressure
				allocations := createMemoryPressure(scenario.allocSize, scenario.allocCount)

				// Schedule jobs under memory pressure
				jobs := createTestJobs(20)
				for _, job := range jobs {
					start := time.Now()
					_ = processJob(job)
					profiler.RecordJobScheduling(job, time.Since(start), true)

					// Trigger GC periodically
					if time.Since(start) > scenario.gcFrequency {
						runtime.GC()
					}
				}

				// Clean up allocations
				for i := range allocations {
					allocations[i] = nil
				}
				runtime.GC()

				report := profiler.StopProfiling()
				if report != nil {
					b.ReportMetric(report.OperationsPerSecond, "ops/sec")
					b.ReportMetric(float64(report.MemoryUsage.GCPauseTime)/float64(time.Millisecond), "gc-pause-ms")
				}
			}
		})
	}
}

// BenchmarkConcurrentSchedulingWithContention benchmarks concurrent scheduling under contention
func BenchmarkConcurrentSchedulingWithContention(b *testing.B) {
	contentionScenarios := []struct {
		name           string
		goroutines     int
		jobsPerRoutine int
		sharedResource bool
	}{
		{"Low-Concurrency", 5, 10, false},
		{"Medium-Concurrency", 20, 10, false},
		{"High-Concurrency", 50, 10, false},
		{"Low-Concurrency-Contention", 5, 10, true},
		{"Medium-Concurrency-Contention", 20, 10, true},
		{"High-Concurrency-Contention", 50, 10, true},
	}

	for _, scenario := range contentionScenarios {
		b.Run(scenario.name, func(b *testing.B) {
			profiler := NewPerformanceProfiler(50 * time.Millisecond)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				profiler.StartProfiling(ctx)

				done := make(chan struct{})
				sharedCounter := 0

				for g := 0; g < scenario.goroutines; g++ {
					go func(routineID int) {
						defer func() { done <- struct{}{} }()

						jobs := createTestJobs(scenario.jobsPerRoutine)
						for _, job := range jobs {
							start := time.Now()

							if scenario.sharedResource {
								// Simulate contention for shared resource
								sharedCounter++
								time.Sleep(time.Duration(rand.Intn(5)) * time.Microsecond)
							}

							_ = processJob(job)
							profiler.RecordJobScheduling(job, time.Since(start), true)
						}
					}(g)
				}

				// Wait for all goroutines to complete
				for g := 0; g < scenario.goroutines; g++ {
					<-done
				}

				report := profiler.StopProfiling()
				if report != nil {
					b.ReportMetric(report.OperationsPerSecond, "ops/sec")
					b.ReportMetric(float64(report.CPUUsage.GoroutineCount), "peak-goroutines")
				}
			}
		})
	}
}

// BenchmarkResourceUtilizationPatterns benchmarks different resource utilization patterns
func BenchmarkResourceUtilizationPatterns(b *testing.B) {
	utilizationPatterns := []struct {
		name       string
		cpuPattern []float64
		memPattern []float64
		gpuPattern []float64
		duration   time.Duration
	}{
		{
			name:       "Steady-State",
			cpuPattern: []float64{0.5, 0.5, 0.5, 0.5, 0.5},
			memPattern: []float64{0.6, 0.6, 0.6, 0.6, 0.6},
			gpuPattern: []float64{0.7, 0.7, 0.7, 0.7, 0.7},
			duration:   100 * time.Millisecond,
		},
		{
			name:       "Gradual-Ramp",
			cpuPattern: []float64{0.1, 0.3, 0.5, 0.7, 0.9},
			memPattern: []float64{0.2, 0.4, 0.6, 0.8, 0.9},
			gpuPattern: []float64{0.1, 0.4, 0.6, 0.8, 0.95},
			duration:   200 * time.Millisecond,
		},
		{
			name:       "Spiky-Load",
			cpuPattern: []float64{0.1, 0.9, 0.2, 0.8, 0.1},
			memPattern: []float64{0.2, 0.8, 0.3, 0.9, 0.2},
			gpuPattern: []float64{0.1, 0.95, 0.2, 0.9, 0.1},
			duration:   150 * time.Millisecond,
		},
	}

	for _, pattern := range utilizationPatterns {
		b.Run(pattern.name, func(b *testing.B) {
			profiler := NewPerformanceProfiler(50 * time.Millisecond)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				profiler.StartProfiling(ctx)

				for phase, cpu := range pattern.cpuPattern {
					mem := pattern.memPattern[phase]
					gpu := pattern.gpuPattern[phase]

					// Simulate resource utilization
					profiler.RecordResourceUtilization(cpu, mem, gpu, "test-node")

					// Schedule jobs during this utilization phase
					jobs := createTestJobs(5)
					for _, job := range jobs {
						start := time.Now()
						_ = processJobWithUtilization(job, cpu, mem, gpu)
						profiler.RecordJobScheduling(job, time.Since(start), true)
					}

					time.Sleep(pattern.duration / time.Duration(len(pattern.cpuPattern)))
				}

				report := profiler.StopProfiling()
				if report != nil {
					b.ReportMetric(report.OperationsPerSecond, "ops/sec")
					if len(report.ResourceMetrics.GPUUtilization) > 0 {
						avgGPU := 0.0
						for _, util := range report.ResourceMetrics.GPUUtilization {
							avgGPU += util
						}
						avgGPU /= float64(len(report.ResourceMetrics.GPUUtilization))
						b.ReportMetric(avgGPU*100, "avg-gpu-util-%")
					}
				}
			}
		})
	}
}

// BenchmarkGangSchedulingPerformance benchmarks gang scheduling performance with profiling
func BenchmarkGangSchedulingPerformance(b *testing.B) {
	gangSizes := []int{2, 4, 8, 16, 32}

	for _, gangSize := range gangSizes {
		b.Run(fmt.Sprintf("GangSize-%d", gangSize), func(b *testing.B) {
			profiler := NewPerformanceProfiler(100 * time.Millisecond)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				profiler.StartProfiling(ctx)

				// Create gang scheduling workloads
				gangs := createGangWorkloads(5, gangSize) // 5 gangs of specified size

				for _, gang := range gangs {
					start := time.Now()
					success := processGang(gang)
					profiler.RecordGangScheduling(len(gang), time.Since(start), success)
				}

				report := profiler.StopProfiling()
				if report != nil {
					b.ReportMetric(report.OperationsPerSecond, "gangs/sec")
					b.ReportMetric(float64(gangSize), "gang-size")
				}
			}
		})
	}
}

// Helper functions for benchmarks

func createTestJob(name string) *v1alpha1.KaiwoJob {
	return &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				User:      "test@amd.com",
				GpuVendor: "amd",
				Gpus:      1,
			},
			EntryPoint: "echo hello",
		},
	}
}

func createTestJobs(count int) []*v1alpha1.KaiwoJob {
	jobs := make([]*v1alpha1.KaiwoJob, count)
	for i := 0; i < count; i++ {
		jobs[i] = createTestJob(fmt.Sprintf("test-job-%d", i))
	}
	return jobs
}

func createTestJobsForScenario(count int, complexity string) []*v1alpha1.KaiwoJob {
	jobs := make([]*v1alpha1.KaiwoJob, count)

	for i := 0; i < count; i++ {
		job := createTestJob(fmt.Sprintf("scenario-job-%d", i))

		// Adjust complexity based on scenario
		switch complexity {
		case "simple":
			// Keep default settings
		case "medium":
			job.Spec.Gpus = 2
		case "complex":
			job.Spec.Gpus = 4

			// Add gang scheduling for complex jobs
			job.Spec.GangScheduling = &v1alpha1.GangSchedulingSpec{
				Enabled:    true,
				MinMembers: 2,
				Timeout:    &metav1.Duration{Duration: 5 * time.Minute},
				Policy:     "strict",
			}
		}

		jobs[i] = job
	}

	return jobs
}

func processConcurrentJobs(jobs []*v1alpha1.KaiwoJob, concurrency int, profiler *PerformanceProfiler) {
	jobChan := make(chan *v1alpha1.KaiwoJob, len(jobs))
	done := make(chan struct{})

	// Start workers
	for w := 0; w < concurrency; w++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for job := range jobChan {
				start := time.Now()
				_ = processJob(job)
				profiler.RecordJobScheduling(job, time.Since(start), true)
			}
		}()
	}

	// Send jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to complete
	for w := 0; w < concurrency; w++ {
		<-done
	}
}

func createMemoryPressure(allocSize, allocCount int) [][]byte {
	allocations := make([][]byte, allocCount)
	for i := 0; i < allocCount; i++ {
		allocations[i] = make([]byte, allocSize)
		// Write to memory to ensure it's actually allocated
		for j := 0; j < len(allocations[i]); j += 4096 {
			allocations[i][j] = byte(i % 256)
		}
	}
	return allocations
}

func createGangWorkloads(gangCount, gangSize int) [][]*v1alpha1.KaiwoJob {
	gangs := make([][]*v1alpha1.KaiwoJob, gangCount)

	for g := 0; g < gangCount; g++ {
		gang := make([]*v1alpha1.KaiwoJob, gangSize)
		for m := 0; m < gangSize; m++ {
			job := createTestJob(fmt.Sprintf("gang-%d-member-%d", g, m))
			job.Spec.GangScheduling = &v1alpha1.GangSchedulingSpec{
				Enabled:    true,
				MinMembers: gangSize,
				Timeout:    &metav1.Duration{Duration: 2 * time.Minute},
				Policy:     "strict",
			}
			gang[m] = job
		}
		gangs[g] = gang
	}

	return gangs
}

// Mock processing functions
func processJob(job *v1alpha1.KaiwoJob) bool {
	// Simulate job processing time based on resource requirements
	baseTime := 1 * time.Millisecond

	// Adjust time based on GPU count
	if job.Spec.Gpus > 0 {
		baseTime += time.Duration(job.Spec.Gpus) * time.Millisecond
	}

	time.Sleep(baseTime + time.Duration(rand.Intn(5))*time.Millisecond)
	return true
}

func processJobWithUtilization(job *v1alpha1.KaiwoJob, cpu, mem, gpu float64) bool {
	// Simulate job processing time adjusted for resource utilization
	baseTime := 1 * time.Millisecond

	// Higher utilization means longer processing time
	utilizationFactor := (cpu + mem + gpu) / 3.0
	adjustedTime := time.Duration(float64(baseTime) * (1.0 + utilizationFactor))

	time.Sleep(adjustedTime)
	return true
}

func processGang(gang []*v1alpha1.KaiwoJob) bool {
	// Gang processing is all-or-nothing and takes longer
	gangProcessingTime := time.Duration(len(gang)) * 2 * time.Millisecond
	time.Sleep(gangProcessingTime)

	// 95% success rate for gang scheduling
	return rand.Float64() > 0.05
}
