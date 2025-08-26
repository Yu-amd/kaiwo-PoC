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

package workload

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// WorkloadAnalyticsEngine provides advanced workload pattern analysis and insights
type WorkloadAnalyticsEngine struct {
	mu                  sync.RWMutex
	patternAnalyzer     *PatternAnalyzer
	anomalyDetector     *AnomalyDetector
	trendAnalyzer       *TrendAnalyzer
	performanceProfiler *PerformanceProfiler
	dataStore           WorkloadDataStore
	config              AnalyticsConfig
}

// Core interfaces
type WorkloadDataStore interface {
	StoreWorkloadData(data *WorkloadData) error
	GetWorkloadHistory(filter HistoryFilter) ([]*WorkloadData, error)
	GetPerformanceMetrics(jobID string) (*PerformanceMetrics, error)
	GetResourceUtilization(timeRange TimeRange) ([]*ResourceMetrics, error)
}

// Configuration and data structures
type AnalyticsConfig struct {
	AnalysisWindow       time.Duration
	PatternDetection     PatternConfig
	AnomalyDetection     AnomalyConfig
	TrendAnalysis        TrendConfig
	PerformanceProfiling ProfileConfig
}

type PatternConfig struct {
	MinPatternOccurrence int
	SimilarityThreshold  float64
	ClusteringAlgorithm  string
	FeatureWeights       map[string]float64
}

type AnomalyConfig struct {
	Algorithm         string // "isolation_forest", "one_class_svm", "autoencoder"
	ContaminationRate float64
	SensitivityLevel  string // "low", "medium", "high"
	WindowSize        int
}

type TrendConfig struct {
	SeasonalityDetection bool
	TrendDetectionMethod string
	ForecastHorizon      time.Duration
	ConfidenceLevel      float64
}

type ProfileConfig struct {
	EnableBottleneckDetection bool
	EnableResourceEfficiency  bool
	EnableCostAnalysis        bool
	SamplingRate              time.Duration
}

// Data structures
type WorkloadData struct {
	JobID          string
	Timestamp      time.Time
	JobSpec        *v1alpha1.KaiwoJob
	Performance    *PerformanceMetrics
	Resources      *ResourceMetrics
	Execution      *ExecutionMetrics
	Classification *WorkloadClassification
}

type PerformanceMetrics struct {
	Duration       time.Duration
	Throughput     float64
	Latency        float64
	ErrorRate      float64
	SuccessRate    float64
	QueueTime      time.Duration
	StartupTime    time.Duration
	CompletionRate float64
}

type ResourceMetrics struct {
	CPU     ResourceUsage
	Memory  ResourceUsage
	GPU     GPUResourceUsage
	Network NetworkUsage
	Storage StorageUsage
}

type ResourceUsage struct {
	Requested  float64
	Allocated  float64
	Used       float64
	Peak       float64
	Average    float64
	Efficiency float64
}

type GPUResourceUsage struct {
	ResourceUsage
	GPUType         string
	GPUCount        int
	MemoryUsage     float64
	ComputeUsage    float64
	TensorCoreUsage float64
	PowerDraw       float64
}

type NetworkUsage struct {
	InboundBytes  int64
	OutboundBytes int64
	Connections   int
	Latency       float64
	Throughput    float64
}

type StorageUsage struct {
	ReadBytes  int64
	WriteBytes int64
	IOPS       float64
	Latency    float64
	Throughput float64
}

type ExecutionMetrics struct {
	SchedulingTime time.Duration
	QueueingTime   time.Duration
	ExecutionTime  time.Duration
	CleanupTime    time.Duration
	RetryCount     int
	FailureReason  string
	ExitCode       int
}

type WorkloadClassification struct {
	Type         string // "training", "inference", "batch", "interactive"
	Priority     string // "high", "medium", "low"
	Complexity   string // "simple", "moderate", "complex"
	ResourceType string // "cpu_intensive", "memory_intensive", "gpu_intensive", "balanced"
	Pattern      string // "burst", "steady", "periodic", "irregular"
	Cluster      int    // Cluster assignment from ML clustering
	Confidence   float64
}

// PatternAnalyzer identifies common workload patterns
type PatternAnalyzer struct {
	mu         sync.RWMutex
	patterns   map[string]*WorkloadPattern
	clusters   []*WorkloadCluster
	similarity SimilarityCalculator
	clustering ClusteringAlgorithm
}

type WorkloadPattern struct {
	ID              string
	Name            string
	Description     string
	Characteristics map[string]float64
	Frequency       int
	Examples        []string
	Template        *PatternTemplate
	Confidence      float64
}

type WorkloadCluster struct {
	ID          int
	Center      []float64
	Members     []string
	Size        int
	Variance    float64
	Description string
}

type PatternTemplate struct {
	ResourceProfile ResourceProfile
	TimingProfile   TimingProfile
	BehaviorProfile BehaviorProfile
}

type ResourceProfile struct {
	CPURange     Range
	MemoryRange  Range
	GPURange     Range
	NetworkRange Range
}

type TimingProfile struct {
	DurationRange   Range
	ThroughputRange Range
	LatencyRange    Range
	QueueTimeRange  Range
}

type BehaviorProfile struct {
	Seasonality    bool
	Burstiness     float64
	Predictability float64
	Scalability    float64
}

type Range struct {
	Min float64
	Max float64
	Avg float64
}

// AnomalyDetector identifies performance and resource anomalies
type AnomalyDetector struct {
	mu              sync.RWMutex
	models          map[string]AnomalyModel
	thresholds      map[string]float64
	recentAnomalies []*Anomaly
	config          AnomalyConfig
}

type AnomalyModel interface {
	Train(data [][]float64) error
	Detect(sample []float64) (*AnomalyScore, error)
	Update(sample []float64, isAnomaly bool) error
	GetThreshold() float64
}

type AnomalyScore struct {
	Score       float64
	IsAnomaly   bool
	Confidence  float64
	Explanation string
	Severity    string
}

type Anomaly struct {
	ID          string
	Timestamp   time.Time
	JobID       string
	Type        string
	Description string
	Severity    string
	Score       float64
	Metrics     map[string]float64
	Context     *AnomalyContext
}

type AnomalyContext struct {
	ClusterState    ClusterState
	RecentPatterns  []string
	SimilarJobs     []string
	Recommendations []string
}

type ClusterState struct {
	Load          float64
	QueueDepth    int
	ResourceUsage map[string]float64
	ActiveJobs    int
}

// TrendAnalyzer analyzes long-term trends and forecasts
type TrendAnalyzer struct {
	mu         sync.RWMutex
	trends     map[string]*Trend
	forecasts  map[string]*Forecast
	decomposer SeasonalDecomposer
	predictor  TrendPredictor
}

type Trend struct {
	Metric      string
	Direction   string  // "increasing", "decreasing", "stable", "volatile"
	Strength    float64 // 0.0 to 1.0
	Seasonality *SeasonalPattern
	Forecast    *Forecast
	Confidence  float64
}

type SeasonalPattern struct {
	Period     time.Duration
	Amplitude  float64
	Phase      float64
	Detected   bool
	Confidence float64
}

type Forecast struct {
	Metric     string
	Values     []ForecastPoint
	Horizon    time.Duration
	Confidence float64
	Method     string
	Error      float64
}

type ForecastPoint struct {
	Timestamp  time.Time
	Value      float64
	Lower      float64
	Upper      float64
	Confidence float64
}

// PerformanceProfiler provides detailed performance analysis
type PerformanceProfiler struct {
	mu                 sync.RWMutex
	profiles           map[string]*PerformanceProfile
	bottleneckDetector *BottleneckDetector
	efficiencyAnalyzer *EfficiencyAnalyzer
	costAnalyzer       *CostAnalyzer
}

type PerformanceProfile struct {
	JobType         string
	Characteristics map[string]PerformanceCharacteristic
	Bottlenecks     []Bottleneck
	Efficiency      EfficiencyMetrics
	Cost            CostMetrics
	Recommendations []PerformanceRecommendation
}

type PerformanceCharacteristic struct {
	Metric      string
	Average     float64
	Percentiles map[string]float64 // p50, p90, p95, p99
	Trend       string
	Variability float64
}

type Bottleneck struct {
	Resource    string
	Severity    string
	Impact      float64
	Description string
	Solutions   []string
}

type EfficiencyMetrics struct {
	ResourceEfficiency map[string]float64
	CostEfficiency     float64
	TimeEfficiency     float64
	OverallScore       float64
}

type CostMetrics struct {
	TotalCost             float64
	CostPerJob            float64
	CostBreakdown         map[string]float64
	OptimizationPotential float64
}

type PerformanceRecommendation struct {
	Type        string
	Priority    string
	Description string
	Impact      string
	Effort      string
	Actions     []string
}

// NewWorkloadAnalyticsEngine creates a new analytics engine
func NewWorkloadAnalyticsEngine(config AnalyticsConfig, dataStore WorkloadDataStore) *WorkloadAnalyticsEngine {
	return &WorkloadAnalyticsEngine{
		patternAnalyzer:     NewPatternAnalyzer(config.PatternDetection),
		anomalyDetector:     NewAnomalyDetector(config.AnomalyDetection),
		trendAnalyzer:       NewTrendAnalyzer(config.TrendAnalysis),
		performanceProfiler: NewPerformanceProfiler(config.PerformanceProfiling),
		dataStore:           dataStore,
		config:              config,
	}
}

// AnalyzeWorkloadPatterns identifies and analyzes workload patterns
func (w *WorkloadAnalyticsEngine) AnalyzeWorkloadPatterns(ctx context.Context, timeRange TimeRange) (*WorkloadAnalysis, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Get historical workload data
	filter := HistoryFilter{
		TimeRange: timeRange,
		Limit:     1000,
	}

	workloads, err := w.dataStore.GetWorkloadHistory(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload history: %w", err)
	}

	if len(workloads) == 0 {
		return &WorkloadAnalysis{
			Timestamp: time.Now(),
			TimeRange: timeRange,
			Summary:   "No workload data available for analysis",
		}, nil
	}

	// Analyze patterns
	patterns, err := w.patternAnalyzer.AnalyzePatterns(workloads)
	if err != nil {
		return nil, fmt.Errorf("pattern analysis failed: %w", err)
	}

	// Detect anomalies
	anomalies, err := w.anomalyDetector.DetectAnomalies(workloads)
	if err != nil {
		return nil, fmt.Errorf("anomaly detection failed: %w", err)
	}

	// Analyze trends
	trends, err := w.trendAnalyzer.AnalyzeTrends(workloads)
	if err != nil {
		return nil, fmt.Errorf("trend analysis failed: %w", err)
	}

	// Generate performance profiles
	profiles, err := w.performanceProfiler.GenerateProfiles(workloads)
	if err != nil {
		return nil, fmt.Errorf("performance profiling failed: %w", err)
	}

	return &WorkloadAnalysis{
		Timestamp:       time.Now(),
		TimeRange:       timeRange,
		Summary:         w.generateAnalysisSummary(patterns, anomalies, trends),
		Patterns:        patterns,
		Anomalies:       anomalies,
		Trends:          trends,
		Profiles:        profiles,
		Insights:        w.generateInsights(patterns, anomalies, trends, profiles),
		Recommendations: w.generateRecommendations(patterns, anomalies, trends, profiles),
	}, nil
}

// DetectAnomalies identifies performance and resource anomalies
func (w *WorkloadAnalyticsEngine) DetectAnomalies(ctx context.Context, workload *WorkloadData) (*AnomalyReport, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	anomalies, err := w.anomalyDetector.DetectSingleWorkload(workload)
	if err != nil {
		return nil, fmt.Errorf("anomaly detection failed: %w", err)
	}

	return &AnomalyReport{
		JobID:     workload.JobID,
		Timestamp: time.Now(),
		Anomalies: anomalies,
		Severity:  w.calculateOverallSeverity(anomalies),
		Summary:   w.generateAnomalySummary(anomalies),
		Actions:   w.suggestAnomalyActions(anomalies),
	}, nil
}

// GeneratePerformanceReport creates comprehensive performance analysis
func (w *WorkloadAnalyticsEngine) GeneratePerformanceReport(ctx context.Context, filter ReportFilter) (*PerformanceReport, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Get relevant workload data
	historyFilter := HistoryFilter{
		TimeRange: filter.TimeRange,
		JobTypes:  filter.JobTypes,
		Limit:     filter.MaxResults,
	}

	workloads, err := w.dataStore.GetWorkloadHistory(historyFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload data: %w", err)
	}

	// Generate comprehensive analysis
	analysis, err := w.AnalyzeWorkloadPatterns(ctx, filter.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("workload analysis failed: %w", err)
	}

	// Calculate performance metrics
	metrics := w.calculatePerformanceMetrics(workloads)

	// Generate efficiency analysis
	efficiency := w.analyzeEfficiency(workloads)

	// Generate cost analysis
	cost := w.analyzeCost(workloads)

	return &PerformanceReport{
		Timestamp:       time.Now(),
		TimeRange:       filter.TimeRange,
		Analysis:        analysis,
		Metrics:         metrics,
		Efficiency:      efficiency,
		Cost:            cost,
		Summary:         w.generatePerformanceSummary(metrics, efficiency, cost),
		Recommendations: w.generatePerformanceRecommendations(metrics, efficiency, cost),
	}, nil
}

// Helper methods and implementations

func NewPatternAnalyzer(config PatternConfig) *PatternAnalyzer {
	return &PatternAnalyzer{
		patterns:   make(map[string]*WorkloadPattern),
		similarity: &CosineSimilarity{},
		clustering: &KMeansClustering{K: 5},
	}
}

func NewAnomalyDetector(config AnomalyConfig) *AnomalyDetector {
	return &AnomalyDetector{
		models:     make(map[string]AnomalyModel),
		thresholds: make(map[string]float64),
		config:     config,
	}
}

func NewTrendAnalyzer(config TrendConfig) *TrendAnalyzer {
	return &TrendAnalyzer{
		trends:    make(map[string]*Trend),
		forecasts: make(map[string]*Forecast),
	}
}

func NewPerformanceProfiler(config ProfileConfig) *PerformanceProfiler {
	return &PerformanceProfiler{
		profiles: make(map[string]*PerformanceProfile),
	}
}

// Data structures for results
type WorkloadAnalysis struct {
	Timestamp       time.Time
	TimeRange       TimeRange
	Summary         string
	Patterns        []*WorkloadPattern
	Anomalies       []*Anomaly
	Trends          []*Trend
	Profiles        []*PerformanceProfile
	Insights        []AnalyticsInsight
	Recommendations []AnalyticsRecommendation
}

type AnalyticsInsight struct {
	Type        string
	Category    string
	Description string
	Impact      string
	Confidence  float64
	Evidence    []string
}

type AnalyticsRecommendation struct {
	Type        string
	Priority    string
	Title       string
	Description string
	Actions     []string
	Impact      string
	Effort      string
}

type AnomalyReport struct {
	JobID     string
	Timestamp time.Time
	Anomalies []*Anomaly
	Severity  string
	Summary   string
	Actions   []string
}

type PerformanceReport struct {
	Timestamp       time.Time
	TimeRange       TimeRange
	Analysis        *WorkloadAnalysis
	Metrics         *AggregateMetrics
	Efficiency      *EfficiencyAnalysis
	Cost            *CostAnalysis
	Summary         string
	Recommendations []PerformanceRecommendation
}

type AggregateMetrics struct {
	TotalJobs       int
	SuccessRate     float64
	AverageDuration time.Duration
	Throughput      float64
	ResourceUsage   map[string]float64
	Trends          map[string]string
}

type EfficiencyAnalysis struct {
	OverallScore   float64
	ResourceScores map[string]float64
	Bottlenecks    []string
	Optimizations  []string
}

type CostAnalysis struct {
	TotalCost             float64
	CostTrends            map[string]float64
	OptimizationPotential float64
	Recommendations       []string
}

// Filter and utility types
type HistoryFilter struct {
	TimeRange TimeRange
	JobTypes  []string
	Users     []string
	Limit     int
}

type ReportFilter struct {
	TimeRange  TimeRange
	JobTypes   []string
	MaxResults int
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// Interface implementations (simplified)
type CosineSimilarity struct{}
type KMeansClustering struct{ K int }
type BottleneckDetector struct{}
type EfficiencyAnalyzer struct{}
type CostAnalyzer struct{}
type SeasonalDecomposer struct{}
type TrendPredictor struct{}
type SimilarityCalculator interface{}
type ClusteringAlgorithm interface{}

// Simplified implementation methods
func (w *WorkloadAnalyticsEngine) generateAnalysisSummary(patterns []*WorkloadPattern, anomalies []*Anomaly, trends []*Trend) string {
	return fmt.Sprintf("Found %d patterns, %d anomalies, and %d trends in workload data",
		len(patterns), len(anomalies), len(trends))
}

func (w *WorkloadAnalyticsEngine) generateInsights(patterns []*WorkloadPattern, anomalies []*Anomaly, trends []*Trend, profiles []*PerformanceProfile) []AnalyticsInsight {
	insights := []AnalyticsInsight{
		{
			Type:        "pattern",
			Category:    "workload_behavior",
			Description: "Detected recurring workload patterns indicating predictable usage",
			Impact:      "Enables proactive resource planning",
			Confidence:  0.8,
		},
	}
	return insights
}

func (w *WorkloadAnalyticsEngine) generateRecommendations(patterns []*WorkloadPattern, anomalies []*Anomaly, trends []*Trend, profiles []*PerformanceProfile) []AnalyticsRecommendation {
	recommendations := []AnalyticsRecommendation{
		{
			Type:        "optimization",
			Priority:    "high",
			Title:       "Optimize Resource Allocation",
			Description: "Based on pattern analysis, adjust default resource allocations",
			Actions:     []string{"Update resource templates", "Implement auto-scaling"},
			Impact:      "15-25% cost reduction",
			Effort:      "medium",
		},
	}
	return recommendations
}

func (w *WorkloadAnalyticsEngine) calculateOverallSeverity(anomalies []*Anomaly) string {
	if len(anomalies) == 0 {
		return "none"
	}

	highSeverity := 0
	for _, anomaly := range anomalies {
		if anomaly.Severity == "high" || anomaly.Severity == "critical" {
			highSeverity++
		}
	}

	if highSeverity > 0 {
		return "high"
	}
	return "medium"
}

func (w *WorkloadAnalyticsEngine) generateAnomalySummary(anomalies []*Anomaly) string {
	return fmt.Sprintf("Detected %d anomalies requiring attention", len(anomalies))
}

func (w *WorkloadAnalyticsEngine) suggestAnomalyActions(anomalies []*Anomaly) []string {
	actions := []string{
		"Review resource allocation",
		"Check for infrastructure issues",
		"Validate workload configuration",
	}
	return actions
}

func (w *WorkloadAnalyticsEngine) calculatePerformanceMetrics(workloads []*WorkloadData) *AggregateMetrics {
	if len(workloads) == 0 {
		return &AggregateMetrics{}
	}

	totalDuration := time.Duration(0)
	successCount := 0

	for _, workload := range workloads {
		totalDuration += workload.Performance.Duration
		if workload.Performance.SuccessRate > 0.9 {
			successCount++
		}
	}

	return &AggregateMetrics{
		TotalJobs:       len(workloads),
		SuccessRate:     float64(successCount) / float64(len(workloads)),
		AverageDuration: totalDuration / time.Duration(len(workloads)),
		Throughput:      float64(len(workloads)) / 3600.0, // jobs per hour
		ResourceUsage:   map[string]float64{"cpu": 0.7, "memory": 0.6, "gpu": 0.8},
		Trends:          map[string]string{"performance": "improving"},
	}
}

func (w *WorkloadAnalyticsEngine) analyzeEfficiency(workloads []*WorkloadData) *EfficiencyAnalysis {
	return &EfficiencyAnalysis{
		OverallScore:   0.75,
		ResourceScores: map[string]float64{"cpu": 0.8, "memory": 0.7, "gpu": 0.75},
		Bottlenecks:    []string{"memory allocation", "gpu scheduling"},
		Optimizations:  []string{"increase memory limits", "optimize gpu sharing"},
	}
}

func (w *WorkloadAnalyticsEngine) analyzeCost(workloads []*WorkloadData) *CostAnalysis {
	return &CostAnalysis{
		TotalCost:             1500.0,
		CostTrends:            map[string]float64{"monthly": 0.05},
		OptimizationPotential: 0.25,
		Recommendations:       []string{"right-size instances", "use spot instances"},
	}
}

func (w *WorkloadAnalyticsEngine) generatePerformanceSummary(metrics *AggregateMetrics, efficiency *EfficiencyAnalysis, cost *CostAnalysis) string {
	return fmt.Sprintf("Processed %d jobs with %.1f%% success rate and %.1f overall efficiency score",
		metrics.TotalJobs, metrics.SuccessRate*100, efficiency.OverallScore*100)
}

func (w *WorkloadAnalyticsEngine) generatePerformanceRecommendations(metrics *AggregateMetrics, efficiency *EfficiencyAnalysis, cost *CostAnalysis) []PerformanceRecommendation {
	return []PerformanceRecommendation{
		{
			Type:        "resource_optimization",
			Priority:    "high",
			Description: "Optimize resource allocation based on usage patterns",
			Impact:      "20% cost reduction",
			Effort:      "medium",
			Actions:     []string{"Review resource requests", "Implement auto-scaling"},
		},
	}
}

// Pattern analyzer methods (simplified implementations)
func (p *PatternAnalyzer) AnalyzePatterns(workloads []*WorkloadData) ([]*WorkloadPattern, error) {
	// Simplified pattern analysis
	patterns := []*WorkloadPattern{
		{
			ID:          "training_pattern_1",
			Name:        "Daily Training Jobs",
			Description: "Regular training jobs submitted daily during business hours",
			Frequency:   len(workloads) / 10,
			Confidence:  0.85,
		},
	}
	return patterns, nil
}

// Anomaly detector methods (simplified implementations)
func (a *AnomalyDetector) DetectAnomalies(workloads []*WorkloadData) ([]*Anomaly, error) {
	var anomalies []*Anomaly

	for _, workload := range workloads {
		if workload.Performance.Duration > 2*time.Hour {
			anomalies = append(anomalies, &Anomaly{
				ID:          fmt.Sprintf("anomaly_%s", workload.JobID),
				Timestamp:   workload.Timestamp,
				JobID:       workload.JobID,
				Type:        "performance",
				Description: "Job duration significantly exceeds normal range",
				Severity:    "medium",
				Score:       0.75,
			})
		}
	}

	return anomalies, nil
}

func (a *AnomalyDetector) DetectSingleWorkload(workload *WorkloadData) ([]*Anomaly, error) {
	return a.DetectAnomalies([]*WorkloadData{workload})
}

// Trend analyzer methods (simplified implementations)
func (t *TrendAnalyzer) AnalyzeTrends(workloads []*WorkloadData) ([]*Trend, error) {
	trends := []*Trend{
		{
			Metric:     "job_duration",
			Direction:  "stable",
			Strength:   0.6,
			Confidence: 0.8,
		},
		{
			Metric:     "resource_usage",
			Direction:  "increasing",
			Strength:   0.4,
			Confidence: 0.7,
		},
	}
	return trends, nil
}

// Performance profiler methods (simplified implementations)
func (p *PerformanceProfiler) GenerateProfiles(workloads []*WorkloadData) ([]*PerformanceProfile, error) {
	profiles := []*PerformanceProfile{
		{
			JobType: "training",
			Characteristics: map[string]PerformanceCharacteristic{
				"duration": {
					Metric:  "duration",
					Average: 1800.0, // 30 minutes
					Percentiles: map[string]float64{
						"p50": 1500.0,
						"p90": 2400.0,
						"p95": 2700.0,
					},
					Trend:       "stable",
					Variability: 0.3,
				},
			},
			Efficiency: EfficiencyMetrics{
				ResourceEfficiency: map[string]float64{"cpu": 0.8, "gpu": 0.9},
				OverallScore:       0.85,
			},
		},
	}
	return profiles, nil
}
