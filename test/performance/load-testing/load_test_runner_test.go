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

package load_testing

import (
	"math"
	"sort"
	"testing"
	"time"
)

// TestLoadTestRunner_BaselineScenario tests the baseline performance scenario
func TestLoadTestRunner_BaselineScenario(t *testing.T) {
	scenario := PredefinedScenarios["baseline"]
	// Reduce duration for testing
	scenario.Duration = 30 * time.Second
	scenario.RampUpDuration = 5 * time.Second
	scenario.RampDownDuration = 5 * time.Second
	scenario.JobsPerClient = 10

	runner := NewLoadTestRunner(scenario)

	results, err := runner.RunLoadTest()
	if err != nil {
		t.Fatalf("Load test failed: %v", err)
	}

	// Validate results
	if results.TotalJobsSubmitted == 0 {
		t.Error("Expected jobs to be submitted")
	}

	if results.TotalJobsCompleted == 0 {
		t.Error("Expected jobs to be completed")
	}

	successRate := float64(results.TotalJobsCompleted) / float64(results.TotalJobsSubmitted)
	if successRate < 0.8 {
		t.Errorf("Success rate too low: %.2f", successRate)
	}

	t.Logf("Baseline Load Test Results:")
	t.Logf("  Jobs Submitted: %d", results.TotalJobsSubmitted)
	t.Logf("  Jobs Completed: %d", results.TotalJobsCompleted)
	t.Logf("  Success Rate: %.2f%%", successRate*100)
	t.Logf("  Average Throughput: %.2f jobs/sec", results.ThroughputMetrics.JobsPerSecond)
	t.Logf("  Average Latency: %.2fms", results.LatencyMetrics.EndToEndLatency.Mean)
}

// TestLoadTestRunner_StressScenario tests the stress testing scenario
func TestLoadTestRunner_StressScenario(t *testing.T) {
	scenario := PredefinedScenarios["stress-test"]
	// Reduce parameters for testing
	scenario.Duration = 1 * time.Minute
	scenario.ConcurrentClients = 10
	scenario.JobsPerClient = 20
	scenario.RampUpDuration = 10 * time.Second
	scenario.RampDownDuration = 10 * time.Second

	runner := NewLoadTestRunner(scenario)

	results, err := runner.RunLoadTest()
	if err != nil {
		t.Fatalf("Stress test failed: %v", err)
	}

	// Validate stress test results
	if results.TotalJobsSubmitted == 0 {
		t.Error("Expected jobs to be submitted")
	}

	// Stress test may have lower success rate due to resource constraints
	successRate := float64(results.TotalJobsCompleted) / float64(results.TotalJobsSubmitted)
	if successRate < 0.5 {
		t.Errorf("Success rate too low even for stress test: %.2f", successRate)
	}

	t.Logf("Stress Test Results:")
	t.Logf("  Jobs Submitted: %d", results.TotalJobsSubmitted)
	t.Logf("  Jobs Completed: %d", results.TotalJobsCompleted)
	t.Logf("  Jobs Failed: %d", results.TotalJobsFailed)
	t.Logf("  Success Rate: %.2f%%", successRate*100)
	t.Logf("  Peak Throughput: %.2f jobs/sec", results.ThroughputMetrics.PeakJobsPerSecond)
	t.Logf("  Error Rate: %.2f%%", results.ErrorAnalysis.ErrorRate*100)
}

// TestLoadTestRunner_GPUIntensiveScenario tests GPU-intensive workloads
func TestLoadTestRunner_GPUIntensiveScenario(t *testing.T) {
	scenario := PredefinedScenarios["gpu-intensive"]
	// Reduce parameters for testing
	scenario.Duration = 45 * time.Second
	scenario.ConcurrentClients = 5
	scenario.JobsPerClient = 10
	scenario.RampUpDuration = 10 * time.Second
	scenario.RampDownDuration = 10 * time.Second

	runner := NewLoadTestRunner(scenario)

	results, err := runner.RunLoadTest()
	if err != nil {
		t.Fatalf("GPU-intensive test failed: %v", err)
	}

	// GPU-intensive workloads should show different resource utilization patterns
	if results.ResourceUtilization.GPU.Average == 0 {
		t.Error("Expected GPU utilization to be recorded")
	}

	t.Logf("GPU-Intensive Test Results:")
	t.Logf("  Jobs Submitted: %d", results.TotalJobsSubmitted)
	t.Logf("  Jobs Completed: %d", results.TotalJobsCompleted)
	t.Logf("  GPU Utilization: %.2f%%", results.ResourceUtilization.GPU.Average*100)
	t.Logf("  CPU Utilization: %.2f%%", results.ResourceUtilization.CPU.Average*100)
	t.Logf("  Memory Utilization: %.2f%%", results.ResourceUtilization.Memory.Average*100)
	t.Logf("  Recommendations: %v", results.Recommendations)
}

// TestLoadTestRunner_BurstCapacityScenario tests burst capacity handling
func TestLoadTestRunner_BurstCapacityScenario(t *testing.T) {
	scenario := PredefinedScenarios["burst-capacity"]
	// Reduce parameters for testing
	scenario.Duration = 30 * time.Second
	scenario.ConcurrentClients = 20
	scenario.JobsPerClient = 5
	scenario.RampUpDuration = 5 * time.Second
	scenario.RampDownDuration = 5 * time.Second

	runner := NewLoadTestRunner(scenario)

	results, err := runner.RunLoadTest()
	if err != nil {
		t.Fatalf("Burst capacity test failed: %v", err)
	}

	// Burst scenarios should show high throughput variation
	if results.ThroughputMetrics.ThroughputVariation == 0 {
		t.Error("Expected throughput variation in burst scenario")
	}

	t.Logf("Burst Capacity Test Results:")
	t.Logf("  Jobs Submitted: %d", results.TotalJobsSubmitted)
	t.Logf("  Peak Jobs/sec: %.2f", results.ThroughputMetrics.PeakJobsPerSecond)
	t.Logf("  Sustained Jobs/sec: %.2f", results.ThroughputMetrics.SustainedThroughput)
	t.Logf("  Throughput Variation: %.2f%%", results.ThroughputMetrics.ThroughputVariation*100)
}

// BenchmarkLoadTestRunner benchmarks different load test scenarios
func BenchmarkLoadTestRunner(b *testing.B) {
	scenarios := []string{"baseline", "stress-test", "gpu-intensive", "burst-capacity"}

	for _, scenarioName := range scenarios {
		b.Run(scenarioName, func(b *testing.B) {
			scenario := PredefinedScenarios[scenarioName]
			// Reduce parameters for benchmarking
			scenario.Duration = 10 * time.Second
			scenario.ConcurrentClients = 5
			scenario.JobsPerClient = 5
			scenario.RampUpDuration = 2 * time.Second
			scenario.RampDownDuration = 2 * time.Second

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runner := NewLoadTestRunner(scenario)
				results, err := runner.RunLoadTest()
				if err != nil {
					b.Fatalf("Load test failed: %v", err)
				}

				// Report key metrics
				b.ReportMetric(float64(results.TotalJobsSubmitted), "jobs-submitted")
				b.ReportMetric(float64(results.TotalJobsCompleted), "jobs-completed")
				b.ReportMetric(results.ThroughputMetrics.JobsPerSecond, "jobs/sec")
				b.ReportMetric(results.LatencyMetrics.EndToEndLatency.Mean, "avg-latency-ms")
			}
		})
	}
}

// Additional methods for LoadTestRunner to complete the implementation

// simulateFailures simulates various failure scenarios during load testing
func (r *LoadTestRunner) simulateFailures() {
	for _, failure := range r.scenario.FailureSimulation.SchedulerFailures {
		select {
		case <-r.ctx.Done():
			return
		case <-time.After(failure.StartTime):
			// Simulate scheduler failure
			originalSuccessRate := r.scheduler.successRate
			r.scheduler.successRate *= 0.1 // Reduce success rate during failure

			time.Sleep(failure.Duration)

			// Recovery
			time.Sleep(failure.RecoveryTime)
			r.scheduler.successRate = originalSuccessRate
		}
	}
}

// monitorResources monitors resource utilization during the test
func (r *LoadTestRunner) monitorResources() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Simulate resource monitoring
			currentLoad := float64(r.scheduler.currentLoad)
			maxLoad := float64(r.scenario.ResourceConstraints.TotalCPUs)

			sample := ResourceSample{
				Timestamp: time.Now(),
				CPU:       currentLoad / maxLoad,
				Memory:    (currentLoad * 2) / maxLoad, // Simulate memory usage
				GPU:       (currentLoad * 0.5) / float64(r.scenario.ResourceConstraints.TotalGPUs),
				NodeID:    "cluster",
			}

			r.metrics.mu.Lock()
			r.metrics.resourceUtilization = append(r.metrics.resourceUtilization, sample)
			r.metrics.mu.Unlock()
		}
	}
}

// monitorThroughput monitors throughput during the test
func (r *LoadTestRunner) monitorThroughput() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastSubmitted := int64(0)
	lastTime := time.Now()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			currentSubmitted := r.metrics.jobsSubmitted
			currentTime := time.Now()

			duration := currentTime.Sub(lastTime).Seconds()
			jobsInPeriod := currentSubmitted - lastSubmitted

			if duration > 0 {
				throughput := float64(jobsInPeriod) / duration

				sample := ThroughputSample{
					Timestamp:     currentTime,
					JobsPerSecond: throughput,
					QueueDepth:    r.metrics.activeJobs,
				}

				r.metrics.mu.Lock()
				r.metrics.throughputSamples = append(r.metrics.throughputSamples, sample)
				r.metrics.mu.Unlock()
			}

			lastSubmitted = currentSubmitted
			lastTime = currentTime
		}
	}
}

// generateResults compiles comprehensive load test results
func (r *LoadTestRunner) generateResults(startTime, endTime time.Time) *LoadTestResults {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	duration := endTime.Sub(startTime)

	results := &LoadTestResults{
		Scenario:            r.scenario,
		StartTime:           startTime,
		EndTime:             endTime,
		TotalDuration:       duration,
		TotalJobsSubmitted:  r.metrics.jobsSubmitted,
		TotalJobsCompleted:  r.metrics.jobsCompleted,
		TotalJobsFailed:     r.metrics.jobsFailed,
		ThroughputMetrics:   r.calculateThroughputMetrics(duration),
		LatencyMetrics:      r.calculateLatencyMetrics(),
		ResourceUtilization: r.calculateResourceUtilization(),
		ErrorAnalysis:       r.calculateErrorAnalysis(),
		PerformanceProfile:  r.calculatePerformanceProfile(),
		Recommendations:     r.generateRecommendations(),
	}

	return results
}

// calculateThroughputMetrics calculates throughput-related metrics
func (r *LoadTestRunner) calculateThroughputMetrics(duration time.Duration) ThroughputMetrics {
	avgThroughput := float64(r.metrics.jobsCompleted) / duration.Seconds()

	// Calculate peak and sustained throughput from samples
	peakThroughput := 0.0
	throughputSum := 0.0

	for _, sample := range r.metrics.throughputSamples {
		if sample.JobsPerSecond > peakThroughput {
			peakThroughput = sample.JobsPerSecond
		}
		throughputSum += sample.JobsPerSecond
	}

	sustainedThroughput := avgThroughput
	if len(r.metrics.throughputSamples) > 0 {
		sustainedThroughput = throughputSum / float64(len(r.metrics.throughputSamples))
	}

	// Calculate throughput variation
	variation := 0.0
	if len(r.metrics.throughputSamples) > 1 {
		mean := sustainedThroughput
		sumSquaredDiff := 0.0
		for _, sample := range r.metrics.throughputSamples {
			diff := sample.JobsPerSecond - mean
			sumSquaredDiff += diff * diff
		}
		variance := sumSquaredDiff / float64(len(r.metrics.throughputSamples))
		stdDev := math.Sqrt(variance)
		variation = stdDev / mean
	}

	return ThroughputMetrics{
		JobsPerSecond:       avgThroughput,
		PeakJobsPerSecond:   peakThroughput,
		SustainedThroughput: sustainedThroughput,
		ThroughputVariation: variation,
		QueueThroughput:     avgThroughput, // Simplified
	}
}

// calculateLatencyMetrics calculates latency-related metrics
func (r *LoadTestRunner) calculateLatencyMetrics() LatencyMetrics {
	if len(r.metrics.latencies) == 0 {
		return LatencyMetrics{}
	}

	// Convert durations to milliseconds for easier interpretation
	latenciesMs := make([]float64, len(r.metrics.latencies))
	for i, latency := range r.metrics.latencies {
		latenciesMs[i] = float64(latency.Nanoseconds()) / 1e6
	}

	endToEndStats := calculateStatisticalMetrics(latenciesMs)

	return LatencyMetrics{
		SubmissionToScheduling: endToEndStats, // Simplified - would need more detailed tracking
		SchedulingToStart:      endToEndStats,
		StartToCompletion:      endToEndStats,
		EndToEndLatency:        endToEndStats,
		QueueWaitTime:          endToEndStats,
	}
}

// calculateResourceUtilization calculates resource utilization metrics
func (r *LoadTestRunner) calculateResourceUtilization() ResourceUtilizationMetrics {
	if len(r.metrics.resourceUtilization) == 0 {
		return ResourceUtilizationMetrics{}
	}

	cpuValues := make([]float64, len(r.metrics.resourceUtilization))
	memoryValues := make([]float64, len(r.metrics.resourceUtilization))
	gpuValues := make([]float64, len(r.metrics.resourceUtilization))

	for i, sample := range r.metrics.resourceUtilization {
		cpuValues[i] = sample.CPU
		memoryValues[i] = sample.Memory
		gpuValues[i] = sample.GPU
	}

	return ResourceUtilizationMetrics{
		CPU:    calculateResourceStats(cpuValues),
		Memory: calculateResourceStats(memoryValues),
		GPU:    calculateResourceStats(gpuValues),
		Node:   make(map[string]ResourceUtilizationStats),
	}
}

// calculateResourceStats calculates resource utilization statistics
func calculateResourceStats(values []float64) ResourceUtilizationStats {
	if len(values) == 0 {
		return ResourceUtilizationStats{}
	}

	sort.Float64s(values)

	sum := 0.0
	min := values[0]
	max := values[len(values)-1]

	for _, v := range values {
		sum += v
	}

	average := sum / float64(len(values))
	efficiency := average // Simplified efficiency calculation
	waste := 1.0 - efficiency

	return ResourceUtilizationStats{
		Average:    average,
		Peak:       max,
		Valley:     min,
		Efficiency: efficiency,
		Waste:      waste,
	}
}

// calculateErrorAnalysis analyzes errors that occurred during the test
func (r *LoadTestRunner) calculateErrorAnalysis() ErrorAnalysis {
	totalErrors := int64(len(r.metrics.errors))
	totalOps := r.metrics.jobsSubmitted

	errorRate := 0.0
	if totalOps > 0 {
		errorRate = float64(totalErrors) / float64(totalOps)
	}

	errorsByType := make(map[string]int64)
	for _, err := range r.metrics.errors {
		errorsByType[err.Type]++
	}

	return ErrorAnalysis{
		ErrorRate:          errorRate,
		ErrorsByType:       errorsByType,
		ErrorsByTimeWindow: make(map[string]int64), // Would implement time-based analysis
		RecoveryTimes:      StatisticalMetrics{},   // Would track recovery times
		CascadingFailures:  0,                      // Would detect cascading failures
	}
}

// calculatePerformanceProfile calculates overall performance characteristics
func (r *LoadTestRunner) calculatePerformanceProfile() PerformanceProfile {
	// Simplified performance profile calculation
	memoryPressure := 0.0
	cpuPressure := 0.0

	if len(r.metrics.resourceUtilization) > 0 {
		totalCPU := 0.0
		totalMemory := 0.0

		for _, sample := range r.metrics.resourceUtilization {
			totalCPU += sample.CPU
			totalMemory += sample.Memory
		}

		cpuPressure = totalCPU / float64(len(r.metrics.resourceUtilization))
		memoryPressure = totalMemory / float64(len(r.metrics.resourceUtilization))
	}

	return PerformanceProfile{
		MemoryPressure:    memoryPressure,
		CPUPressure:       cpuPressure,
		GCImpact:          0.0, // Would measure GC impact
		ConcurrencyImpact: 0.0, // Would measure concurrency impact
		ScalabilityFactor: 1.0, // Would calculate scalability
	}
}

// generateRecommendations generates actionable recommendations based on test results
func (r *LoadTestRunner) generateRecommendations() []Recommendation {
	recommendations := []Recommendation{}

	// Analyze throughput
	if len(r.metrics.throughputSamples) > 0 {
		avgThroughput := 0.0
		for _, sample := range r.metrics.throughputSamples {
			avgThroughput += sample.JobsPerSecond
		}
		avgThroughput /= float64(len(r.metrics.throughputSamples))

		if avgThroughput < 10.0 {
			recommendations = append(recommendations, Recommendation{
				Category:    "performance",
				Priority:    "high",
				Description: "Low throughput detected. Consider optimizing scheduler performance or increasing parallelism.",
				Impact:      "May significantly impact system scalability",
			})
		}
	}

	// Analyze error rate
	errorRate := float64(len(r.metrics.errors)) / float64(r.metrics.jobsSubmitted)
	if errorRate > 0.1 {
		recommendations = append(recommendations, Recommendation{
			Category:    "reliability",
			Priority:    "high",
			Description: "High error rate detected. Review error logs and improve error handling.",
			Impact:      "Affects system reliability and user experience",
		})
	}

	// Analyze resource utilization
	if len(r.metrics.resourceUtilization) > 0 {
		avgCPU := 0.0
		for _, sample := range r.metrics.resourceUtilization {
			avgCPU += sample.CPU
		}
		avgCPU /= float64(len(r.metrics.resourceUtilization))

		if avgCPU > 0.8 {
			recommendations = append(recommendations, Recommendation{
				Category:    "resource",
				Priority:    "medium",
				Description: "High CPU utilization. Consider adding more resources or load balancing.",
				Impact:      "May cause performance degradation under sustained load",
			})
		}

		if avgCPU < 0.3 {
			recommendations = append(recommendations, Recommendation{
				Category:    "resource",
				Priority:    "low",
				Description: "Low CPU utilization. Consider consolidating workloads or reducing allocated resources.",
				Impact:      "Opportunity for cost optimization",
			})
		}
	}

	return recommendations
}

// calculateStatisticalMetrics calculates comprehensive statistics for a dataset
func calculateStatisticalMetrics(values []float64) StatisticalMetrics {
	if len(values) == 0 {
		return StatisticalMetrics{}
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate basic statistics
	sum := 0.0
	min := sorted[0]
	max := sorted[len(sorted)-1]

	for _, v := range sorted {
		sum += v
	}

	mean := sum / float64(len(sorted))
	median := sorted[len(sorted)/2]

	// Calculate percentiles
	p90 := sorted[int(float64(len(sorted))*0.9)]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	p99 := sorted[int(float64(len(sorted))*0.99)]

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range sorted {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(sorted)))

	return StatisticalMetrics{
		Mean:   mean,
		Median: median,
		P90:    p90,
		P95:    p95,
		P99:    p99,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
	}
}
