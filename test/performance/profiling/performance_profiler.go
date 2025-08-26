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
	"runtime"
	"sync"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// PerformanceProfiler provides comprehensive performance profiling capabilities
type PerformanceProfiler struct {
	metrics      map[string]*MetricCollector
	mu           sync.RWMutex
	samplingRate time.Duration
	isRunning    bool
	stopChan     chan struct{}
}

// MetricCollector collects and analyzes specific performance metrics
type MetricCollector struct {
	Name       string
	Values     []float64
	Timestamps []time.Time
	mu         sync.RWMutex
	aggregates *MetricAggregates
}

// MetricAggregates holds statistical aggregates of collected metrics
type MetricAggregates struct {
	Count  int64
	Sum    float64
	Min    float64
	Max    float64
	Mean   float64
	P50    float64
	P90    float64
	P95    float64
	P99    float64
	StdDev float64
}

// PerformanceReport contains comprehensive performance analysis results
type PerformanceReport struct {
	Timestamp           time.Time
	TestDuration        time.Duration
	TotalOperations     int64
	OperationsPerSecond float64
	MemoryUsage         *MemoryMetrics
	CPUUsage            *CPUMetrics
	SchedulingMetrics   *SchedulingMetrics
	ResourceMetrics     *ResourceMetrics
	ErrorMetrics        *ErrorMetrics
	LatencyDistribution *LatencyMetrics
	ThroughputMetrics   *ThroughputMetrics
	Recommendations     []string
}

type MemoryMetrics struct {
	AllocatedBytes    uint64
	TotalAllocations  uint64
	GCPauseTime       time.Duration
	HeapSize          uint64
	StackSize         uint64
	MemoryUtilization float64
}

type CPUMetrics struct {
	CPUUtilization  float64
	GoroutineCount  int
	CGOCalls        int64
	ContextSwitches int64
	CPUTime         time.Duration
}

type SchedulingMetrics struct {
	JobsScheduled         int64
	SchedulingSuccessRate float64
	AverageSchedulingTime time.Duration
	QueueDepth            int64
	ResourceUtilization   float64
	NodeUtilization       map[string]float64
}

type ResourceMetrics struct {
	GPUUtilization          map[string]float64
	MemoryFragmentation     float64
	NetworkBandwidth        float64
	DiskIOPS                float64
	ResourceWastePercentage float64
}

type ErrorMetrics struct {
	TotalErrors        int64
	ErrorRate          float64
	SchedulingFailures int64
	ResourceExhaustion int64
	TimeoutErrors      int64
	ErrorsByType       map[string]int64
}

type LatencyMetrics struct {
	SchedulingLatency    *MetricAggregates
	JobStartLatency      *MetricAggregates
	JobCompletionLatency *MetricAggregates
	EndToEndLatency      *MetricAggregates
}

type ThroughputMetrics struct {
	JobsPerSecond      float64
	ResourceThroughput map[string]float64
	DataThroughput     float64
	NetworkThroughput  float64
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler(samplingRate time.Duration) *PerformanceProfiler {
	return &PerformanceProfiler{
		metrics:      make(map[string]*MetricCollector),
		samplingRate: samplingRate,
		stopChan:     make(chan struct{}),
	}
}

// StartProfiling begins performance data collection
func (p *PerformanceProfiler) StartProfiling(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("profiler is already running")
	}

	p.isRunning = true

	// Initialize metric collectors
	p.initializeMetricCollectors()

	// Start background collection goroutines
	go p.collectSystemMetrics(ctx)
	go p.collectMemoryMetrics(ctx)
	go p.collectCPUMetrics(ctx)

	return nil
}

// StopProfiling stops performance data collection and returns a report
func (p *PerformanceProfiler) StopProfiling() *PerformanceReport {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	p.isRunning = false

	// Close stop channel safely
	select {
	case <-p.stopChan:
		// Already closed
	default:
		close(p.stopChan)
	}

	// Generate performance report
	return p.generateReport()
}

// RecordJobScheduling records job scheduling performance metrics
func (p *PerformanceProfiler) RecordJobScheduling(job *v1alpha1.KaiwoJob, schedulingTime time.Duration, success bool) {
	p.recordMetric("scheduling_time", schedulingTime.Seconds())
	p.recordMetric("scheduling_success", boolToFloat(success))

	// Record job-specific metrics based on common metadata
	if job.Spec.Gpus > 0 {
		p.recordMetric("requested_gpus", float64(job.Spec.Gpus))
	}

	// Record user and vendor information
	if job.Spec.User != "" {
		p.recordMetric("user_jobs", 1.0)
	}
	if job.Spec.GpuVendor == "amd" {
		p.recordMetric("amd_gpu_jobs", 1.0)
	}
}

// RecordGangScheduling records gang scheduling performance metrics
func (p *PerformanceProfiler) RecordGangScheduling(gangSize int, schedulingTime time.Duration, success bool) {
	p.recordMetric("gang_size", float64(gangSize))
	p.recordMetric("gang_scheduling_time", schedulingTime.Seconds())
	p.recordMetric("gang_success", boolToFloat(success))
}

// RecordElasticScaling records elastic scaling performance metrics
func (p *PerformanceProfiler) RecordElasticScaling(currentReplicas, targetReplicas int32, scalingTime time.Duration) {
	p.recordMetric("current_replicas", float64(currentReplicas))
	p.recordMetric("target_replicas", float64(targetReplicas))
	p.recordMetric("scaling_time", scalingTime.Seconds())
	p.recordMetric("scaling_factor", float64(targetReplicas)/float64(currentReplicas))
}

// RecordResourceUtilization records resource utilization metrics
func (p *PerformanceProfiler) RecordResourceUtilization(cpu, memory, gpu float64, nodeName string) {
	p.recordMetric(fmt.Sprintf("cpu_utilization_%s", nodeName), cpu)
	p.recordMetric(fmt.Sprintf("memory_utilization_%s", nodeName), memory)
	p.recordMetric(fmt.Sprintf("gpu_utilization_%s", nodeName), gpu)
	p.recordMetric("cluster_cpu_utilization", cpu)
	p.recordMetric("cluster_memory_utilization", memory)
	p.recordMetric("cluster_gpu_utilization", gpu)
}

// RecordError records error metrics
func (p *PerformanceProfiler) RecordError(errorType string, errorMessage string) {
	p.recordMetric(fmt.Sprintf("error_%s", errorType), 1.0)
	p.recordMetric("total_errors", 1.0)
}

// Private methods

func (p *PerformanceProfiler) initializeMetricCollectors() {
	metricNames := []string{
		"scheduling_time", "scheduling_success", "gang_scheduling_time", "gang_success",
		"scaling_time", "scaling_factor", "cpu_utilization", "memory_utilization",
		"gpu_utilization", "total_errors", "memory_allocated", "gc_pause_time",
		"goroutine_count", "jobs_per_second", "queue_depth", "resource_waste",
	}

	for _, name := range metricNames {
		p.metrics[name] = &MetricCollector{
			Name:       name,
			Values:     make([]float64, 0),
			Timestamps: make([]time.Time, 0),
			aggregates: &MetricAggregates{},
		}
	}
}

func (p *PerformanceProfiler) recordMetric(name string, value float64) {
	p.mu.RLock()
	collector, exists := p.metrics[name]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		collector = &MetricCollector{
			Name:       name,
			Values:     make([]float64, 0),
			Timestamps: make([]time.Time, 0),
			aggregates: &MetricAggregates{},
		}
		p.metrics[name] = collector
		p.mu.Unlock()
	}

	collector.mu.Lock()
	collector.Values = append(collector.Values, value)
	collector.Timestamps = append(collector.Timestamps, time.Now())
	collector.mu.Unlock()
}

func (p *PerformanceProfiler) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(p.samplingRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			// Collect system-level metrics
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			p.recordMetric("memory_allocated", float64(m.Alloc))
			p.recordMetric("gc_pause_time", float64(m.PauseTotalNs)/1e9)
			p.recordMetric("goroutine_count", float64(runtime.NumGoroutine()))
			p.recordMetric("heap_size", float64(m.HeapSys))
			p.recordMetric("stack_size", float64(m.StackSys))
		}
	}
}

func (p *PerformanceProfiler) collectMemoryMetrics(ctx context.Context) {
	ticker := time.NewTicker(p.samplingRate * 2) // Collect less frequently
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Calculate memory utilization
			memUtilization := float64(m.Alloc) / float64(m.Sys) * 100
			p.recordMetric("memory_utilization_percent", memUtilization)

			// Calculate memory fragmentation
			fragmentation := 1.0 - (float64(m.HeapAlloc) / float64(m.HeapSys))
			p.recordMetric("memory_fragmentation", fragmentation)
		}
	}
}

func (p *PerformanceProfiler) collectCPUMetrics(ctx context.Context) {
	ticker := time.NewTicker(p.samplingRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			// Record CPU-related metrics
			p.recordMetric("cpu_cores", float64(runtime.NumCPU()))
			p.recordMetric("cgo_calls", float64(runtime.NumCgoCall()))
		}
	}
}

func (p *PerformanceProfiler) generateReport() *PerformanceReport {
	now := time.Now()

	// Calculate aggregates for all metrics
	for _, collector := range p.metrics {
		collector.aggregates = p.calculateAggregates(collector)
	}

	report := &PerformanceReport{
		Timestamp:           now,
		MemoryUsage:         p.getMemoryMetrics(),
		CPUUsage:            p.getCPUMetrics(),
		SchedulingMetrics:   p.getSchedulingMetrics(),
		ResourceMetrics:     p.getResourceMetrics(),
		ErrorMetrics:        p.getErrorMetrics(),
		LatencyDistribution: p.getLatencyMetrics(),
		ThroughputMetrics:   p.getThroughputMetrics(),
		Recommendations:     p.generateRecommendations(),
	}

	// Calculate overall performance metrics
	if schedulingTime := p.metrics["scheduling_time"]; schedulingTime != nil && len(schedulingTime.Values) > 0 {
		totalOps := int64(len(schedulingTime.Values))
		report.TotalOperations = totalOps

		if report.TestDuration > 0 {
			report.OperationsPerSecond = float64(totalOps) / report.TestDuration.Seconds()
		}
	}

	return report
}

func (p *PerformanceProfiler) calculateAggregates(collector *MetricCollector) *MetricAggregates {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	if len(collector.Values) == 0 {
		return &MetricAggregates{}
	}

	// Sort values for percentile calculations
	sortedValues := make([]float64, len(collector.Values))
	copy(sortedValues, collector.Values)

	// Simple sort (for production, use sort.Float64s)
	for i := 0; i < len(sortedValues); i++ {
		for j := i + 1; j < len(sortedValues); j++ {
			if sortedValues[i] > sortedValues[j] {
				sortedValues[i], sortedValues[j] = sortedValues[j], sortedValues[i]
			}
		}
	}

	count := int64(len(sortedValues))
	sum := 0.0
	min := sortedValues[0]
	max := sortedValues[count-1]

	for _, v := range sortedValues {
		sum += v
	}

	mean := sum / float64(count)

	// Calculate standard deviation
	variance := 0.0
	for _, v := range sortedValues {
		variance += (v - mean) * (v - mean)
	}
	stdDev := variance / float64(count)

	return &MetricAggregates{
		Count:  count,
		Sum:    sum,
		Min:    min,
		Max:    max,
		Mean:   mean,
		P50:    sortedValues[count/2],
		P90:    sortedValues[int(float64(count)*0.9)],
		P95:    sortedValues[int(float64(count)*0.95)],
		P99:    sortedValues[int(float64(count)*0.99)],
		StdDev: stdDev,
	}
}

func (p *PerformanceProfiler) getMemoryMetrics() *MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memUtil := 0.0
	if memMetric := p.metrics["memory_utilization_percent"]; memMetric != nil && memMetric.aggregates != nil {
		memUtil = memMetric.aggregates.Mean
	}

	return &MemoryMetrics{
		AllocatedBytes:    m.Alloc,
		TotalAllocations:  m.Mallocs,
		GCPauseTime:       time.Duration(m.PauseTotalNs),
		HeapSize:          m.HeapSys,
		StackSize:         m.StackSys,
		MemoryUtilization: memUtil,
	}
}

func (p *PerformanceProfiler) getCPUMetrics() *CPUMetrics {
	cpuUtil := 0.0
	if cpuMetric := p.metrics["cluster_cpu_utilization"]; cpuMetric != nil && cpuMetric.aggregates != nil {
		cpuUtil = cpuMetric.aggregates.Mean
	}

	return &CPUMetrics{
		CPUUtilization:  cpuUtil,
		GoroutineCount:  runtime.NumGoroutine(),
		CGOCalls:        runtime.NumCgoCall(),
		ContextSwitches: 0, // Would need OS-specific implementation
		CPUTime:         0, // Would need OS-specific implementation
	}
}

func (p *PerformanceProfiler) getSchedulingMetrics() *SchedulingMetrics {
	schedulingTime := 0.0
	successRate := 0.0
	queueDepth := 0.0

	if metric := p.metrics["scheduling_time"]; metric != nil && metric.aggregates != nil {
		schedulingTime = metric.aggregates.Mean
	}
	if metric := p.metrics["scheduling_success"]; metric != nil && metric.aggregates != nil {
		successRate = metric.aggregates.Mean * 100 // Convert to percentage
	}
	if metric := p.metrics["queue_depth"]; metric != nil && metric.aggregates != nil {
		queueDepth = metric.aggregates.Mean
	}

	jobsScheduled := int64(0)
	if metric := p.metrics["scheduling_time"]; metric != nil && metric.aggregates != nil {
		jobsScheduled = metric.aggregates.Count
	}

	return &SchedulingMetrics{
		JobsScheduled:         jobsScheduled,
		SchedulingSuccessRate: successRate,
		AverageSchedulingTime: time.Duration(schedulingTime * float64(time.Second)),
		QueueDepth:            int64(queueDepth),
		ResourceUtilization:   0.0, // Calculate from resource metrics
		NodeUtilization:       make(map[string]float64),
	}
}

func (p *PerformanceProfiler) getResourceMetrics() *ResourceMetrics {
	gpuUtil := make(map[string]float64)
	memFrag := 0.0
	resourceWaste := 0.0

	if metric := p.metrics["cluster_gpu_utilization"]; metric != nil && metric.aggregates != nil {
		gpuUtil["cluster"] = metric.aggregates.Mean
	}
	if metric := p.metrics["memory_fragmentation"]; metric != nil && metric.aggregates != nil {
		memFrag = metric.aggregates.Mean
	}
	if metric := p.metrics["resource_waste"]; metric != nil && metric.aggregates != nil {
		resourceWaste = metric.aggregates.Mean
	}

	return &ResourceMetrics{
		GPUUtilization:          gpuUtil,
		MemoryFragmentation:     memFrag,
		NetworkBandwidth:        0.0, // Would need network monitoring
		DiskIOPS:                0.0, // Would need disk monitoring
		ResourceWastePercentage: resourceWaste,
	}
}

func (p *PerformanceProfiler) getErrorMetrics() *ErrorMetrics {
	totalErrors := int64(0)
	if metric := p.metrics["total_errors"]; metric != nil && metric.aggregates != nil {
		totalErrors = int64(metric.aggregates.Sum)
	}

	errorRate := 0.0
	if totalOps := p.metrics["scheduling_time"]; totalOps != nil && totalOps.aggregates != nil && totalOps.aggregates.Count > 0 {
		errorRate = float64(totalErrors) / float64(totalOps.aggregates.Count) * 100
	}

	return &ErrorMetrics{
		TotalErrors:        totalErrors,
		ErrorRate:          errorRate,
		SchedulingFailures: 0, // Would track specific error types
		ResourceExhaustion: 0,
		TimeoutErrors:      0,
		ErrorsByType:       make(map[string]int64),
	}
}

func (p *PerformanceProfiler) getLatencyMetrics() *LatencyMetrics {
	return &LatencyMetrics{
		SchedulingLatency:    p.metrics["scheduling_time"].aggregates,
		JobStartLatency:      p.metrics["scheduling_time"].aggregates, // Placeholder
		JobCompletionLatency: p.metrics["scaling_time"].aggregates,    // Placeholder
		EndToEndLatency:      p.metrics["scheduling_time"].aggregates, // Placeholder
	}
}

func (p *PerformanceProfiler) getThroughputMetrics() *ThroughputMetrics {
	jobsPerSec := 0.0
	if metric := p.metrics["jobs_per_second"]; metric != nil && metric.aggregates != nil {
		jobsPerSec = metric.aggregates.Mean
	}

	return &ThroughputMetrics{
		JobsPerSecond:      jobsPerSec,
		ResourceThroughput: make(map[string]float64),
		DataThroughput:     0.0,
		NetworkThroughput:  0.0,
	}
}

func (p *PerformanceProfiler) generateRecommendations() []string {
	recommendations := []string{}

	// Analyze memory metrics
	if memMetric := p.metrics["memory_utilization_percent"]; memMetric != nil && memMetric.aggregates != nil {
		if memMetric.aggregates.Mean > 80 {
			recommendations = append(recommendations, "High memory utilization detected. Consider increasing memory limits or optimizing memory usage.")
		}
	}

	// Analyze scheduling performance
	if schedMetric := p.metrics["scheduling_time"]; schedMetric != nil && schedMetric.aggregates != nil {
		if schedMetric.aggregates.Mean > 5.0 { // 5 seconds
			recommendations = append(recommendations, "Slow scheduling detected. Consider optimizing scheduling algorithms or increasing scheduler resources.")
		}
	}

	// Analyze error rates
	if errorMetric := p.metrics["total_errors"]; errorMetric != nil && errorMetric.aggregates != nil {
		if errorMetric.aggregates.Sum > 0 {
			recommendations = append(recommendations, "Errors detected during testing. Review error logs and consider improving error handling.")
		}
	}

	// Analyze resource utilization
	if cpuMetric := p.metrics["cluster_cpu_utilization"]; cpuMetric != nil && cpuMetric.aggregates != nil {
		if cpuMetric.aggregates.Mean < 30 {
			recommendations = append(recommendations, "Low CPU utilization. Consider consolidating workloads or reducing resource allocations.")
		}
		if cpuMetric.aggregates.Mean > 85 {
			recommendations = append(recommendations, "High CPU utilization. Consider scaling up or load balancing.")
		}
	}

	return recommendations
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
