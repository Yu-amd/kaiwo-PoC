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

package prediction

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// PerformancePredictor provides ML-based performance prediction capabilities
type PerformancePredictor struct {
	mu            sync.RWMutex
	models        map[string]PredictionModel
	featureStore  FeatureStore
	modelRegistry ModelRegistry
	config        PredictorConfig
}

// PredictionModel defines the interface for ML prediction models
type PredictionModel interface {
	Predict(features FeatureVector) (*Prediction, error)
	Train(trainingData []TrainingExample) error
	Validate(validationData []TrainingExample) (*ValidationResult, error)
	GetAccuracy() float64
	GetModelInfo() *ModelInfo
}

// FeatureStore manages feature engineering and storage
type FeatureStore interface {
	ExtractFeatures(job *v1alpha1.KaiwoJob, context *JobContext) (*FeatureVector, error)
	StoreFeatures(jobID string, features *FeatureVector) error
	GetHistoricalFeatures(filter FeatureFilter) ([]FeatureVector, error)
	CreateFeaturePipeline(jobType string) (*FeaturePipeline, error)
}

// ModelRegistry manages ML model lifecycle
type ModelRegistry interface {
	RegisterModel(model PredictionModel, metadata *ModelMetadata) error
	GetModel(modelID string) (PredictionModel, error)
	UpdateModel(modelID string, model PredictionModel) error
	GetBestModel(modelType string, metric string) (PredictionModel, error)
	ListModels(filter ModelFilter) ([]*ModelInfo, error)
}

// Core data structures
type PredictorConfig struct {
	EnabledModels     []string
	TrainingSchedule  TrainingSchedule
	AccuracyThreshold float64
	MaxPredictionAge  time.Duration
	FeatureWindow     time.Duration
	ModelUpdatePolicy string
}

type TrainingSchedule struct {
	Interval            time.Duration
	MinTrainingData     int
	ValidationSplit     float64
	EarlyStoppingRounds int
}

type FeatureVector struct {
	JobID     string
	Timestamp time.Time
	Features  map[string]float64
	Labels    map[string]float64
	Metadata  map[string]interface{}
}

type Prediction struct {
	Value       float64
	Confidence  float64
	Interval    ConfidenceInterval
	Model       string
	Timestamp   time.Time
	Features    *FeatureVector
	Explanation *PredictionExplanation
}

type ConfidenceInterval struct {
	Lower      float64
	Upper      float64
	Confidence float64
}

type PredictionExplanation struct {
	TopFeatures   []FeatureImportance
	ModelDecision string
	Alternatives  []AlternativePrediction
}

type FeatureImportance struct {
	Feature    string
	Importance float64
	Impact     string
}

type AlternativePrediction struct {
	Model      string
	Value      float64
	Confidence float64
}

type TrainingExample struct {
	Features  *FeatureVector
	Target    float64
	Weight    float64
	Timestamp time.Time
}

type ValidationResult struct {
	Accuracy      float64
	MAE           float64 // Mean Absolute Error
	RMSE          float64 // Root Mean Square Error
	R2Score       float64 // R-squared
	Predictions   []float64
	ActualValues  []float64
	CrossValScore float64
}

type ModelInfo struct {
	ID           string
	Type         string
	Version      string
	Accuracy     float64
	TrainingTime time.Time
	LastUpdated  time.Time
	TrainingData int
	Features     []string
	Hyperparams  map[string]interface{}
}

type ModelMetadata struct {
	Name         string
	Description  string
	Version      string
	Author       string
	TrainingData DatasetInfo
	Hyperparams  map[string]interface{}
	Metrics      map[string]float64
}

type DatasetInfo struct {
	Size         int
	Features     []string
	TimeRange    TimeRange
	Distribution map[string]interface{}
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

// JobDurationPredictor predicts job completion times
type JobDurationPredictor struct {
	baseModel PredictionModel
	features  []string
	scaler    *FeatureScaler
	encoder   *CategoryEncoder
}

// ResourceRequirementPredictor predicts optimal resource allocation
type ResourceRequirementPredictor struct {
	cpuModel    PredictionModel
	memoryModel PredictionModel
	gpuModel    PredictionModel
	multiTarget bool
}

// SchedulingOptimizer provides ML-driven scheduling recommendations
type SchedulingOptimizer struct {
	placementModel PredictionModel
	scoreModel     PredictionModel
	constraints    []SchedulingConstraint
}

type SchedulingConstraint struct {
	Type      string
	Condition string
	Weight    float64
}

// RandomForestModel implements a Random Forest prediction model
type RandomForestModel struct {
	mu          sync.RWMutex
	trees       []*DecisionTree
	features    []string
	numTrees    int
	maxDepth    int
	minSamples  int
	randomState int
	accuracy    float64
	trained     bool
}

type DecisionTree struct {
	Root       *TreeNode
	MaxDepth   int
	MinSamples int
	Features   []string
}

type TreeNode struct {
	Feature   string
	Threshold float64
	Left      *TreeNode
	Right     *TreeNode
	Value     float64
	Samples   int
	IsLeaf    bool
}

// FeatureScaler normalizes features for ML models
type FeatureScaler struct {
	Method string // "standard", "minmax", "robust"
	Params map[string]float64
}

// CategoryEncoder encodes categorical features
type CategoryEncoder struct {
	Method   string // "onehot", "label", "target"
	Mappings map[string]map[string]int
}

// NewPerformancePredictor creates a new performance predictor
func NewPerformancePredictor(config PredictorConfig, featureStore FeatureStore, modelRegistry ModelRegistry) *PerformancePredictor {
	return &PerformancePredictor{
		models:        make(map[string]PredictionModel),
		featureStore:  featureStore,
		modelRegistry: modelRegistry,
		config:        config,
	}
}

// PredictJobDuration predicts how long a job will take to complete
func (p *PerformancePredictor) PredictJobDuration(ctx context.Context, job *v1alpha1.KaiwoJob) (*Prediction, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Extract features from the job
	jobContext := &JobContext{
		ClusterState:   p.getCurrentClusterState(),
		HistoricalData: p.getHistoricalData(job),
	}

	features, err := p.featureStore.ExtractFeatures(job, jobContext)
	if err != nil {
		return nil, fmt.Errorf("failed to extract features: %w", err)
	}

	// Get the best duration prediction model
	model, exists := p.models["duration"]
	if !exists {
		model, err = p.modelRegistry.GetBestModel("duration", "accuracy")
		if err != nil {
			return nil, fmt.Errorf("failed to get duration model: %w", err)
		}
		p.models["duration"] = model
	}

	// Make prediction
	prediction, err := model.Predict(*features)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}

	// Add explanation
	prediction.Explanation = p.explainPrediction(prediction, features, "duration")

	return prediction, nil
}

// PredictResourceRequirements predicts optimal CPU, memory, and GPU allocation
func (p *PerformancePredictor) PredictResourceRequirements(ctx context.Context, workload *WorkloadSpec) (*ResourcePrediction, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	features, err := p.extractWorkloadFeatures(workload)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workload features: %w", err)
	}

	// Get resource prediction models
	cpuModel := p.models["cpu_requirements"]
	memModel := p.models["memory_requirements"]
	gpuModel := p.models["gpu_requirements"]

	if cpuModel == nil || memModel == nil || gpuModel == nil {
		return nil, fmt.Errorf("resource prediction models not available")
	}

	// Make predictions for each resource type
	cpuPred, err := cpuModel.Predict(*features)
	if err != nil {
		return nil, fmt.Errorf("CPU prediction failed: %w", err)
	}

	memPred, err := memModel.Predict(*features)
	if err != nil {
		return nil, fmt.Errorf("memory prediction failed: %w", err)
	}

	gpuPred, err := gpuModel.Predict(*features)
	if err != nil {
		return nil, fmt.Errorf("GPU prediction failed: %w", err)
	}

	return &ResourcePrediction{
		CPU: ResourceAllocation{
			Requested:  cpuPred.Value,
			Confidence: cpuPred.Confidence,
			Interval:   cpuPred.Interval,
		},
		Memory: ResourceAllocation{
			Requested:  memPred.Value,
			Confidence: memPred.Confidence,
			Interval:   memPred.Interval,
		},
		GPU: ResourceAllocation{
			Requested:  gpuPred.Value,
			Confidence: gpuPred.Confidence,
			Interval:   gpuPred.Interval,
		},
		Timestamp:  time.Now(),
		Confidence: (cpuPred.Confidence + memPred.Confidence + gpuPred.Confidence) / 3.0,
	}, nil
}

// GetOptimalPlacement recommends the best node placement for a job
func (p *PerformancePredictor) GetOptimalPlacement(ctx context.Context, job *v1alpha1.KaiwoJob) (*PlacementRecommendation, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get available nodes and their current state
	nodes := p.getAvailableNodes()
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes for placement")
	}

	// Score each node for this job
	var placements []NodePlacement
	for _, node := range nodes {
		score, err := p.scoreNodeForJob(job, node)
		if err != nil {
			continue // Skip nodes that can't be scored
		}

		placements = append(placements, NodePlacement{
			NodeID:     node.ID,
			Score:      score.Value,
			Confidence: score.Confidence,
			Reasons:    score.Explanation.TopFeatures,
		})
	}

	if len(placements) == 0 {
		return nil, fmt.Errorf("no suitable nodes found for job placement")
	}

	// Sort by score (highest first)
	sort.Slice(placements, func(i, j int) bool {
		return placements[i].Score > placements[j].Score
	})

	return &PlacementRecommendation{
		BestPlacement: placements[0],
		Alternatives:  placements[1:],
		Timestamp:     time.Now(),
		JobID:         job.Name,
	}, nil
}

// TrainModels trains or retrains prediction models with recent data
func (p *PerformancePredictor) TrainModels(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get training data
	trainingData, err := p.getTrainingData()
	if err != nil {
		return fmt.Errorf("failed to get training data: %w", err)
	}

	if len(trainingData) < p.config.TrainingSchedule.MinTrainingData {
		return fmt.Errorf("insufficient training data: need %d, have %d",
			p.config.TrainingSchedule.MinTrainingData, len(trainingData))
	}

	// Train each enabled model
	for _, modelType := range p.config.EnabledModels {
		if err := p.trainModel(modelType, trainingData); err != nil {
			return fmt.Errorf("failed to train %s model: %w", modelType, err)
		}
	}

	return nil
}

// Helper methods and data structures

type JobContext struct {
	ClusterState   *ClusterState
	HistoricalData []HistoricalJobData
}

type ClusterState struct {
	AvailableNodes []NodeInfo
	QueueDepth     int
	ResourceUsage  ResourceUsageStats
}

type NodeInfo struct {
	ID              string
	AvailableCPU    float64
	AvailableMemory float64
	AvailableGPU    float64
	GPUType         string
	Location        string
	Load            float64
}

type ResourceUsageStats struct {
	CPUUtilization    float64
	MemoryUtilization float64
	GPUUtilization    float64
}

type HistoricalJobData struct {
	JobSpec     *v1alpha1.KaiwoJob
	Duration    time.Duration
	Resources   ResourceUsage
	Performance PerformanceMetrics
}

type ResourceUsage struct {
	CPU    float64
	Memory float64
	GPU    float64
}

type PerformanceMetrics struct {
	Throughput float64
	Latency    float64
	Efficiency float64
	ErrorRate  float64
}

type WorkloadSpec struct {
	Type           string
	ContainerImage string
	Command        []string
	Arguments      []string
	Environment    map[string]string
	Annotations    map[string]string
}

type ResourcePrediction struct {
	CPU        ResourceAllocation
	Memory     ResourceAllocation
	GPU        ResourceAllocation
	Timestamp  time.Time
	Confidence float64
}

type ResourceAllocation struct {
	Requested  float64
	Confidence float64
	Interval   ConfidenceInterval
}

type PlacementRecommendation struct {
	BestPlacement NodePlacement
	Alternatives  []NodePlacement
	Timestamp     time.Time
	JobID         string
}

type NodePlacement struct {
	NodeID     string
	Score      float64
	Confidence float64
	Reasons    []FeatureImportance
}

// Implementation helpers (simplified for this initial version)

func (p *PerformancePredictor) getCurrentClusterState() *ClusterState {
	// Simplified implementation - would integrate with actual cluster state
	return &ClusterState{
		AvailableNodes: []NodeInfo{
			{ID: "node-1", AvailableCPU: 8, AvailableMemory: 32, AvailableGPU: 2, GPUType: "MI300X"},
			{ID: "node-2", AvailableCPU: 16, AvailableMemory: 64, AvailableGPU: 4, GPUType: "MI300X"},
		},
		QueueDepth: 5,
		ResourceUsage: ResourceUsageStats{
			CPUUtilization:    0.7,
			MemoryUtilization: 0.6,
			GPUUtilization:    0.8,
		},
	}
}

func (p *PerformancePredictor) getHistoricalData(job *v1alpha1.KaiwoJob) []HistoricalJobData {
	// Simplified implementation - would query actual historical data
	return []HistoricalJobData{}
}

func (p *PerformancePredictor) explainPrediction(prediction *Prediction, features *FeatureVector, modelType string) *PredictionExplanation {
	// Simplified implementation - would use actual model explanation techniques
	return &PredictionExplanation{
		TopFeatures: []FeatureImportance{
			{Feature: "cpu_request", Importance: 0.3, Impact: "primary driver"},
			{Feature: "memory_request", Importance: 0.25, Impact: "secondary factor"},
			{Feature: "gpu_count", Importance: 0.2, Impact: "significant factor"},
		},
		ModelDecision: fmt.Sprintf("Predicted %.2f based on %s model", prediction.Value, modelType),
	}
}

func (p *PerformancePredictor) extractWorkloadFeatures(workload *WorkloadSpec) (*FeatureVector, error) {
	features := map[string]float64{
		"workload_type":    encodeWorkloadType(workload.Type),
		"has_gpu":          boolToFloat(containsGPURequirement(workload.Annotations)),
		"complexity_score": calculateComplexityScore(workload),
	}

	return &FeatureVector{
		Features:  features,
		Timestamp: time.Now(),
	}, nil
}

func (p *PerformancePredictor) getAvailableNodes() []NodeInfo {
	return p.getCurrentClusterState().AvailableNodes
}

func (p *PerformancePredictor) scoreNodeForJob(job *v1alpha1.KaiwoJob, node NodeInfo) (*Prediction, error) {
	// Simplified scoring - would use actual ML model
	score := 0.5 + (node.AvailableCPU/16.0)*0.3 + (node.AvailableGPU/8.0)*0.2

	return &Prediction{
		Value:      score,
		Confidence: 0.8,
		Interval:   ConfidenceInterval{Lower: score * 0.9, Upper: score * 1.1, Confidence: 0.8},
		Explanation: &PredictionExplanation{
			TopFeatures: []FeatureImportance{
				{Feature: "available_cpu", Importance: 0.3, Impact: "resource availability"},
				{Feature: "available_gpu", Importance: 0.2, Impact: "gpu resources"},
			},
		},
	}, nil
}

func (p *PerformancePredictor) getTrainingData() ([]TrainingExample, error) {
	// Simplified implementation - would get actual training data
	return []TrainingExample{}, nil
}

func (p *PerformancePredictor) trainModel(modelType string, data []TrainingExample) error {
	// Simplified implementation - would train actual models
	return nil
}

// Utility functions
func encodeWorkloadType(workloadType string) float64 {
	encodings := map[string]float64{
		"training":    1.0,
		"inference":   2.0,
		"batch":       3.0,
		"interactive": 4.0,
	}
	if val, exists := encodings[workloadType]; exists {
		return val
	}
	return 0.0
}

func containsGPURequirement(annotations map[string]string) bool {
	for key := range annotations {
		if key == "kaiwo.ai/gpu-fraction" || key == "amd.com/gpu" {
			return true
		}
	}
	return false
}

func calculateComplexityScore(workload *WorkloadSpec) float64 {
	score := 1.0
	if len(workload.Arguments) > 5 {
		score += 0.5
	}
	if len(workload.Environment) > 10 {
		score += 0.3
	}
	return math.Min(score, 5.0)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// Model filter and feature filter types
type ModelFilter struct {
	Type        string
	MinAccuracy float64
	MaxAge      time.Duration
}

type FeatureFilter struct {
	TimeRange TimeRange
	JobType   string
	Limit     int
}

type FeaturePipeline struct {
	Steps []FeatureStep
}

type FeatureStep struct {
	Name      string
	Transform func(input interface{}) (interface{}, error)
}
