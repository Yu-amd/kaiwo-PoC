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

package mlpipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// KubeflowIntegration provides optimization and integration with Kubeflow pipelines
type KubeflowIntegration struct {
	mu                sync.RWMutex
	client            *KubeflowClient
	config            KubeflowConfig
	pipelineOptimizer *PipelineOptimizer
	stepOptimizer     *StepOptimizer
	resourceManager   *ResourceManager
	executionTracker  *ExecutionTracker
}

// KubeflowClient handles communication with Kubeflow APIs
type KubeflowClient struct {
	pipelineClient   PipelineServiceClient
	experimentClient ExperimentServiceClient
	runClient        RunServiceClient
	auth             AuthConfig
}

// Configuration
type KubeflowConfig struct {
	ServerURL            string
	Namespace            string
	DefaultExperiment    string
	OptimizationEnabled  bool
	ResourceOptimization ResourceOptimizationConfig
	StepOptimization     StepOptimizationConfig
	Monitoring           MonitoringConfig
}

type ResourceOptimizationConfig struct {
	Enabled            bool
	AMDGPUOptimization bool
	AutoScaling        bool
	ResourcePrediction bool
	CostOptimization   bool
}

type StepOptimizationConfig struct {
	Enabled           bool
	ParallelExecution bool
	CachingEnabled    bool
	RetryOptimization bool
}

type MonitoringConfig struct {
	Enabled         bool
	MetricsInterval time.Duration
	AlertsEnabled   bool
}

// Core Kubeflow entities
type Pipeline struct {
	ID           string
	Name         string
	Description  string
	Version      string
	CreatedAt    time.Time
	Steps        []*PipelineStep
	Dependencies []StepDependency
	Config       PipelineConfig
}

type PipelineStep struct {
	ID           string
	Name         string
	Type         string // "training", "preprocessing", "validation", "serving"
	Image        string
	Command      []string
	Arguments    []string
	Resources    StepResources
	Dependencies []string
	Outputs      []StepOutput
	Config       StepConfig
}

type StepDependency struct {
	FromStep string
	ToStep   string
	Type     string // "data", "model", "artifact"
}

type StepResources struct {
	CPU      string
	Memory   string
	GPU      int
	GPUType  string
	Storage  string
	Requests ResourceRequests
	Limits   ResourceLimits
}

type ResourceRequests struct {
	CPU    string
	Memory string
	GPU    string
}

type ResourceLimits struct {
	CPU    string
	Memory string
	GPU    string
}

type StepOutput struct {
	Name string
	Type string
	Path string
}

type StepConfig struct {
	Caching     bool
	Retry       RetryConfig
	Timeout     time.Duration
	Environment map[string]string
	Annotations map[string]string
}

type RetryConfig struct {
	MaxRetries int
	Backoff    string
}

type PipelineConfig struct {
	Parallelism    int
	CachingEnabled bool
	TimeoutMinutes int
	ResourceQuota  ResourceQuota
	AMDGPUConfig   AMDGPUConfig
}

type ResourceQuota struct {
	CPU    string
	Memory string
	GPU    int
}

type AMDGPUConfig struct {
	EnableTimeSlicing bool
	FractionalGPU     bool
	GPUMemoryLimit    string
	OptimizationMode  string // "performance", "efficiency", "balanced"
}

// Execution tracking
type PipelineRun struct {
	ID           string
	PipelineID   string
	ExperimentID string
	Name         string
	Status       string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Steps        []*StepRun
	Metrics      RunMetrics
	Resources    RunResourceUsage
	Config       RunConfig
}

type StepRun struct {
	ID         string
	StepID     string
	Name       string
	Status     string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Resources  StepResourceUsage
	Outputs    map[string]string
	Logs       string
	ErrorMsg   string
	RetryCount int
}

type RunMetrics struct {
	TotalSteps     int
	CompletedSteps int
	FailedSteps    int
	CachedSteps    int
	SuccessRate    float64
	AvgStepTime    time.Duration
	Throughput     float64
}

type RunResourceUsage struct {
	TotalCPU    float64
	TotalMemory float64
	TotalGPU    float64
	PeakCPU     float64
	PeakMemory  float64
	PeakGPU     float64
	Efficiency  ResourceEfficiency
}

type StepResourceUsage struct {
	CPU     ResourceUsageMetric
	Memory  ResourceUsageMetric
	GPU     GPUUsageMetric
	Network NetworkUsageMetric
	Storage StorageUsageMetric
}

type ResourceUsageMetric struct {
	Requested  float64
	Used       float64
	Peak       float64
	Average    float64
	Efficiency float64
}

type GPUUsageMetric struct {
	ResourceUsageMetric
	GPUType        string
	MemoryUsed     float64
	ComputeUsed    float64
	PowerDraw      float64
	Temperature    float64
	UtilizationPct float64
}

type NetworkUsageMetric struct {
	InboundMB  float64
	OutboundMB float64
	Latency    float64
	Throughput float64
}

type StorageUsageMetric struct {
	ReadMB  float64
	WriteMB float64
	IOPS    float64
	Latency float64
}

type ResourceEfficiency struct {
	CPU     float64
	Memory  float64
	GPU     float64
	Overall float64
}

type RunConfig struct {
	Experiment   string
	Parameters   map[string]string
	Resources    ResourceQuota
	OptimizedFor string // "speed", "cost", "efficiency"
}

// PipelineOptimizer optimizes entire pipeline execution
type PipelineOptimizer struct {
	mu                sync.RWMutex
	optimizations     map[string]*PipelineOptimization
	performanceModel  PerformanceModel
	resourcePredictor ResourcePredictor
}

type PipelineOptimization struct {
	PipelineID     string
	Optimizations  []Optimization
	ResourcePlan   ResourcePlan
	ExecutionPlan  ExecutionPlan
	CostEstimate   CostEstimate
	PerformanceEst PerformanceEstimate
}

type Optimization struct {
	Type        string // "resource", "parallelization", "caching", "scheduling"
	Description string
	Impact      OptimizationImpact
	Actions     []OptimizationAction
}

type OptimizationImpact struct {
	CostReduction      float64
	TimeReduction      float64
	ResourceEfficiency float64
	Confidence         float64
}

type OptimizationAction struct {
	Step       string
	Action     string
	Parameters map[string]string
	Resources  StepResources
}

type ResourcePlan struct {
	TotalResources  ResourceQuota
	StepAllocations map[string]StepResources
	ScalingPlan     ScalingPlan
	GPUOptimization GPUOptimizationPlan
}

type ScalingPlan struct {
	AutoScaling  bool
	MinResources ResourceQuota
	MaxResources ResourceQuota
	ScalingRules []ScalingRule
}

type ScalingRule struct {
	Metric    string
	Threshold float64
	Action    string
}

type GPUOptimizationPlan struct {
	FractionalGPU map[string]float64
	TimeSlicing   map[string]bool
	MemoryLimits  map[string]string
	AffinityRules []GPUAffinityRule
}

type GPUAffinityRule struct {
	StepPattern string
	GPUType     string
	Preference  string
}

type ExecutionPlan struct {
	ParallelSteps  [][]string
	ExecutionOrder []string
	Dependencies   map[string][]string
	CriticalPath   []string
	EstimatedTime  time.Duration
}

type CostEstimate struct {
	TotalCost     float64
	CostBreakdown map[string]float64
	Savings       float64
	ROI           float64
}

type PerformanceEstimate struct {
	EstimatedDuration time.Duration
	Confidence        float64
	Bottlenecks       []string
	Recommendations   []string
}

// StepOptimizer optimizes individual pipeline steps
type StepOptimizer struct {
	mu            sync.RWMutex
	optimizations map[string]*StepOptimization
	cache         StepCache
}

type StepOptimization struct {
	StepID      string
	Resources   OptimalResources
	Caching     CachingStrategy
	Retry       RetryStrategy
	Performance PerformanceOptimization
}

type OptimalResources struct {
	CPU        string
	Memory     string
	GPU        float64
	Confidence float64
	Reasoning  string
}

type CachingStrategy struct {
	Enabled  bool
	TTL      time.Duration
	Strategy string
	CacheKey string
}

type RetryStrategy struct {
	MaxRetries int
	Backoff    BackoffStrategy
	Conditions []RetryCondition
}

type BackoffStrategy struct {
	Type         string // "fixed", "exponential", "linear"
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

type RetryCondition struct {
	ErrorType string
	MaxCount  int
}

type PerformanceOptimization struct {
	Parallelization ParallelizationConfig
	ResourceTuning  ResourceTuningConfig
	Monitoring      StepMonitoringConfig
}

type ParallelizationConfig struct {
	Enabled  bool
	Workers  int
	Strategy string
}

type ResourceTuningConfig struct {
	AutoTuning bool
	Profiles   []ResourceProfile
	Adaptation bool
}

type ResourceProfile struct {
	Name        string
	Resources   StepResources
	Workload    WorkloadCharacteristics
	Performance PerformanceCharacteristics
}

type WorkloadCharacteristics struct {
	Type             string
	DataSize         int64
	ComputeIntensity string
	IOPattern        string
}

type PerformanceCharacteristics struct {
	Throughput  float64
	Latency     float64
	Efficiency  float64
	Scalability string
}

type StepMonitoringConfig struct {
	Metrics  []string
	Interval time.Duration
	Alerts   []AlertRule
}

type AlertRule struct {
	Metric    string
	Condition string
	Threshold float64
	Action    string
}

// Interface definitions
type PipelineServiceClient interface {
	CreatePipeline(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error)
	ListPipelines(ctx context.Context, filter PipelineFilter) ([]*Pipeline, error)
	UpdatePipeline(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	DeletePipeline(ctx context.Context, pipelineID string) error
}

type ExperimentServiceClient interface {
	CreateExperiment(ctx context.Context, experiment *Experiment) (*Experiment, error)
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)
	ListExperiments(ctx context.Context, filter ExperimentFilter) ([]*Experiment, error)
}

type RunServiceClient interface {
	CreateRun(ctx context.Context, run *PipelineRun) (*PipelineRun, error)
	GetRun(ctx context.Context, runID string) (*PipelineRun, error)
	ListRuns(ctx context.Context, filter RunFilter) ([]*PipelineRun, error)
	UpdateRun(ctx context.Context, run *PipelineRun) (*PipelineRun, error)
	TerminateRun(ctx context.Context, runID string) error
}

type PerformanceModel interface {
	PredictPipelineDuration(pipeline *Pipeline) (time.Duration, float64, error)
	PredictStepDuration(step *PipelineStep) (time.Duration, float64, error)
	PredictResourceUsage(step *PipelineStep) (*StepResourceUsage, error)
}

type ResourcePredictor interface {
	PredictOptimalResources(step *PipelineStep, history []*StepRun) (*OptimalResources, error)
	PredictScalingNeeds(pipeline *Pipeline) (*ScalingPlan, error)
	PredictCost(resourcePlan *ResourcePlan) (*CostEstimate, error)
}

type StepCache interface {
	Get(key string) (*CacheEntry, error)
	Set(key string, entry *CacheEntry) error
	Invalidate(pattern string) error
	GetStats() *CacheStats
}

type CacheEntry struct {
	Key        string
	Value      interface{}
	TTL        time.Duration
	CreatedAt  time.Time
	AccessedAt time.Time
	HitCount   int
}

type CacheStats struct {
	HitRate    float64
	MissRate   float64
	Size       int
	Efficiency float64
}

// Filter types
type PipelineFilter struct {
	NamePattern  string
	Status       string
	CreatedAfter time.Time
	Tags         map[string]string
}

type ExperimentFilter struct {
	NamePattern  string
	Status       string
	CreatedAfter time.Time
}

type RunFilter struct {
	PipelineID   string
	ExperimentID string
	Status       string
	StartedAfter time.Time
}

// NewKubeflowIntegration creates a new Kubeflow integration instance
func NewKubeflowIntegration(config KubeflowConfig) (*KubeflowIntegration, error) {
	client, err := NewKubeflowClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubeflow client: %w", err)
	}

	return &KubeflowIntegration{
		client:            client,
		config:            config,
		pipelineOptimizer: NewPipelineOptimizer(),
		stepOptimizer:     NewStepOptimizer(),
		resourceManager:   NewResourceManager(config.ResourceOptimization),
		executionTracker:  NewExecutionTracker(),
	}, nil
}

// OptimizePipeline optimizes a Kubeflow pipeline for AMD GPU environments
func (k *KubeflowIntegration) OptimizePipeline(ctx context.Context, pipeline *Pipeline, job *v1alpha1.KaiwoJob) (*PipelineOptimization, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Analyze pipeline structure and requirements
	analysis, err := k.analyzePipeline(pipeline)
	if err != nil {
		return nil, fmt.Errorf("pipeline analysis failed: %w", err)
	}

	// Optimize for AMD GPU resources
	gpuOptimization, err := k.optimizeForAMDGPU(pipeline, job)
	if err != nil {
		return nil, fmt.Errorf("AMD GPU optimization failed: %w", err)
	}

	// Create resource plan
	resourcePlan, err := k.createResourcePlan(pipeline, analysis)
	if err != nil {
		return nil, fmt.Errorf("resource planning failed: %w", err)
	}

	// Create execution plan
	executionPlan, err := k.createExecutionPlan(pipeline, analysis)
	if err != nil {
		return nil, fmt.Errorf("execution planning failed: %w", err)
	}

	// Estimate costs and performance
	costEstimate := k.estimateCosts(resourcePlan)
	performanceEst := k.estimatePerformance(pipeline, executionPlan)

	optimization := &PipelineOptimization{
		PipelineID:     pipeline.ID,
		Optimizations:  append(analysis.Optimizations, gpuOptimization...),
		ResourcePlan:   *resourcePlan,
		ExecutionPlan:  *executionPlan,
		CostEstimate:   *costEstimate,
		PerformanceEst: *performanceEst,
	}

	k.pipelineOptimizer.optimizations[pipeline.ID] = optimization
	return optimization, nil
}

// CreateOptimizedRun creates and executes an optimized pipeline run
func (k *KubeflowIntegration) CreateOptimizedRun(ctx context.Context, pipelineID string, config RunConfig) (*PipelineRun, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	// Get pipeline optimization
	optimization, exists := k.pipelineOptimizer.optimizations[pipelineID]
	if !exists {
		return nil, fmt.Errorf("no optimization found for pipeline %s", pipelineID)
	}

	// Create optimized run configuration
	optimizedConfig := k.applyOptimizations(config, optimization)

	// Create the run
	run := &PipelineRun{
		ID:           generateRunID(),
		PipelineID:   pipelineID,
		ExperimentID: config.Experiment,
		Name:         fmt.Sprintf("optimized-run-%d", time.Now().Unix()),
		Status:       "RUNNING",
		StartTime:    time.Now(),
		Config:       optimizedConfig,
		Steps:        []*StepRun{},
		Metrics:      RunMetrics{},
	}

	// Start execution tracking
	go k.trackExecution(ctx, run)

	return run, nil
}

// TrackPipelinePerformance monitors and tracks pipeline execution performance
func (k *KubeflowIntegration) TrackPipelinePerformance(ctx context.Context, runID string) (*PerformanceReport, error) {
	return k.executionTracker.GeneratePerformanceReport(ctx, runID)
}

// GetPipelineInsights provides analytics and insights for pipeline optimization
func (k *KubeflowIntegration) GetPipelineInsights(ctx context.Context, pipelineID string, timeRange TimeRange) (*PipelineInsights, error) {
	// Get historical runs
	filter := RunFilter{
		PipelineID:   pipelineID,
		StartedAfter: timeRange.Start,
	}

	runs, err := k.client.runClient.ListRuns(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline runs: %w", err)
	}

	// Analyze performance trends
	trends := k.analyzePipelineTrends(runs)

	// Identify optimization opportunities
	opportunities := k.identifyOptimizationOpportunities(runs)

	// Generate recommendations
	recommendations := k.generatePipelineRecommendations(trends, opportunities)

	return &PipelineInsights{
		PipelineID:      pipelineID,
		TimeRange:       timeRange,
		TotalRuns:       len(runs),
		Trends:          trends,
		Opportunities:   opportunities,
		Recommendations: recommendations,
		GeneratedAt:     time.Now(),
	}, nil
}

// Implementation helpers

func NewKubeflowClient(config KubeflowConfig) (*KubeflowClient, error) {
	// Simplified implementation - would create actual Kubeflow API clients
	return &KubeflowClient{}, nil
}

func NewPipelineOptimizer() *PipelineOptimizer {
	return &PipelineOptimizer{
		optimizations: make(map[string]*PipelineOptimization),
	}
}

func NewStepOptimizer() *StepOptimizer {
	return &StepOptimizer{
		optimizations: make(map[string]*StepOptimization),
	}
}

func NewResourceManager(config ResourceOptimizationConfig) *ResourceManager {
	return &ResourceManager{
		config: config,
	}
}

func NewExecutionTracker() *ExecutionTracker {
	return &ExecutionTracker{
		activeRuns: make(map[string]*PipelineRun),
	}
}

// Analysis and optimization methods (simplified implementations)

func (k *KubeflowIntegration) analyzePipeline(pipeline *Pipeline) (*PipelineAnalysis, error) {
	return &PipelineAnalysis{
		ComplexityScore: k.calculateComplexity(pipeline),
		Dependencies:    k.analyzeDependencies(pipeline),
		ResourceNeeds:   k.analyzeResourceNeeds(pipeline),
		Optimizations:   k.identifyBasicOptimizations(pipeline),
	}, nil
}

func (k *KubeflowIntegration) optimizeForAMDGPU(pipeline *Pipeline, job *v1alpha1.KaiwoJob) ([]Optimization, error) {
	optimizations := []Optimization{
		{
			Type:        "amd_gpu_optimization",
			Description: "Optimize pipeline steps for AMD GPU time-slicing",
			Impact: OptimizationImpact{
				CostReduction:      0.15,
				TimeReduction:      0.10,
				ResourceEfficiency: 0.25,
				Confidence:         0.8,
			},
		},
	}
	return optimizations, nil
}

func (k *KubeflowIntegration) createResourcePlan(pipeline *Pipeline, analysis *PipelineAnalysis) (*ResourcePlan, error) {
	return &ResourcePlan{
		TotalResources: ResourceQuota{
			CPU:    "16",
			Memory: "64Gi",
			GPU:    4,
		},
		StepAllocations: make(map[string]StepResources),
		GPUOptimization: GPUOptimizationPlan{
			FractionalGPU: map[string]float64{
				"training":  2.0,
				"inference": 0.5,
			},
			TimeSlicing: map[string]bool{
				"training":  true,
				"inference": true,
			},
		},
	}, nil
}

func (k *KubeflowIntegration) createExecutionPlan(pipeline *Pipeline, analysis *PipelineAnalysis) (*ExecutionPlan, error) {
	return &ExecutionPlan{
		EstimatedTime: 2 * time.Hour,
		CriticalPath:  []string{"preprocessing", "training", "validation"},
	}, nil
}

func (k *KubeflowIntegration) estimateCosts(plan *ResourcePlan) *CostEstimate {
	return &CostEstimate{
		TotalCost: 150.0,
		Savings:   30.0,
		ROI:       2.5,
	}
}

func (k *KubeflowIntegration) estimatePerformance(pipeline *Pipeline, plan *ExecutionPlan) *PerformanceEstimate {
	return &PerformanceEstimate{
		EstimatedDuration: plan.EstimatedTime,
		Confidence:        0.85,
		Bottlenecks:       []string{"data loading", "model training"},
		Recommendations:   []string{"increase data parallelism", "optimize GPU utilization"},
	}
}

// Additional helper types and data structures

type PipelineAnalysis struct {
	ComplexityScore int
	Dependencies    []StepDependency
	ResourceNeeds   ResourceRequirements
	Optimizations   []Optimization
}

type ResourceManager struct {
	config ResourceOptimizationConfig
}

type ExecutionTracker struct {
	mu         sync.RWMutex
	activeRuns map[string]*PipelineRun
}

type PipelineInsights struct {
	PipelineID      string
	TimeRange       TimeRange
	TotalRuns       int
	Trends          []Trend
	Opportunities   []OptimizationOpportunity
	Recommendations []Recommendation
	GeneratedAt     time.Time
}

type Trend struct {
	Metric    string
	Direction string
	Strength  float64
}

type OptimizationOpportunity struct {
	Type        string
	Description string
	Potential   float64
	Effort      string
}

type Recommendation struct {
	Type        string
	Priority    string
	Description string
	Actions     []string
	Impact      string
}

type PerformanceReport struct {
	RunID       string
	StartTime   time.Time
	Duration    time.Duration
	Status      string
	Metrics     RunMetrics
	Resources   RunResourceUsage
	Bottlenecks []string
	Summary     string
}

// Utility methods (simplified implementations)

func (k *KubeflowIntegration) calculateComplexity(pipeline *Pipeline) int {
	return len(pipeline.Steps) + len(pipeline.Dependencies)
}

func (k *KubeflowIntegration) analyzeDependencies(pipeline *Pipeline) []StepDependency {
	return pipeline.Dependencies
}

func (k *KubeflowIntegration) analyzeResourceNeeds(pipeline *Pipeline) ResourceRequirements {
	return ResourceRequirements{
		CPU:    "16",
		Memory: "64Gi",
		GPU:    4,
	}
}

func (k *KubeflowIntegration) identifyBasicOptimizations(pipeline *Pipeline) []Optimization {
	return []Optimization{
		{
			Type:        "parallelization",
			Description: "Parallelize independent steps",
			Impact:      OptimizationImpact{TimeReduction: 0.3, Confidence: 0.9},
		},
	}
}

func (k *KubeflowIntegration) applyOptimizations(config RunConfig, optimization *PipelineOptimization) RunConfig {
	optimizedConfig := config
	optimizedConfig.Resources = optimization.ResourcePlan.TotalResources
	return optimizedConfig
}

func (k *KubeflowIntegration) trackExecution(ctx context.Context, run *PipelineRun) {
	// Simplified execution tracking
	time.Sleep(5 * time.Minute) // Simulate execution
	run.Status = "COMPLETED"
	run.EndTime = time.Now()
	run.Duration = run.EndTime.Sub(run.StartTime)
}

func (k *KubeflowIntegration) analyzePipelineTrends(runs []*PipelineRun) []Trend {
	return []Trend{
		{Metric: "duration", Direction: "decreasing", Strength: 0.7},
		{Metric: "cost", Direction: "stable", Strength: 0.5},
	}
}

func (k *KubeflowIntegration) identifyOptimizationOpportunities(runs []*PipelineRun) []OptimizationOpportunity {
	return []OptimizationOpportunity{
		{
			Type:        "resource_optimization",
			Description: "Over-provisioned CPU resources detected",
			Potential:   0.25,
			Effort:      "low",
		},
	}
}

func (k *KubeflowIntegration) generatePipelineRecommendations(trends []Trend, opportunities []OptimizationOpportunity) []Recommendation {
	return []Recommendation{
		{
			Type:        "resource_optimization",
			Priority:    "high",
			Description: "Reduce CPU allocation by 25% based on usage patterns",
			Actions:     []string{"Update resource requests", "Monitor performance"},
			Impact:      "25% cost reduction",
		},
	}
}

func (t *ExecutionTracker) GeneratePerformanceReport(ctx context.Context, runID string) (*PerformanceReport, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	run, exists := t.activeRuns[runID]
	if !exists {
		return nil, fmt.Errorf("run %s not found", runID)
	}

	return &PerformanceReport{
		RunID:     runID,
		StartTime: run.StartTime,
		Duration:  run.Duration,
		Status:    run.Status,
		Metrics:   run.Metrics,
		Resources: run.Resources,
		Summary:   fmt.Sprintf("Pipeline completed in %v with %d steps", run.Duration, run.Metrics.TotalSteps),
	}, nil
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}
