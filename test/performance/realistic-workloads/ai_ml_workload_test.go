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

package realistic_workloads

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Realistic AI/ML workload profiles based on industry use cases
type WorkloadProfile struct {
	Name                  string
	ContainerImage        string
	CPURequest            string
	CPULimit              string
	MemoryRequest         string
	MemoryLimit           string
	GPURequest            float64 // Fractional GPU allocation
	ExpectedDuration      time.Duration
	MemoryPattern         string // "constant", "ramping", "spiky"
	GPUUtilizationPattern string // "steady", "burst", "periodic"
	ScalingRequirements   bool
}

var AIMLWorkloadProfiles = []WorkloadProfile{
	// Deep Learning Training Workloads
	{
		Name:                  "llm-training-70b",
		ContainerImage:        "rocm/pytorch:latest",
		CPURequest:            "16",
		CPULimit:              "32",
		MemoryRequest:         "128Gi",
		MemoryLimit:           "256Gi",
		GPURequest:            4.0, // Full 4 GPUs for large model training
		ExpectedDuration:      8 * time.Hour,
		MemoryPattern:         "ramping",
		GPUUtilizationPattern: "steady",
		ScalingRequirements:   false,
	},
	{
		Name:                  "computer-vision-training",
		ContainerImage:        "rocm/tensorflow:latest",
		CPURequest:            "8",
		CPULimit:              "16",
		MemoryRequest:         "32Gi",
		MemoryLimit:           "64Gi",
		GPURequest:            2.0, // 2 GPUs for CV training
		ExpectedDuration:      4 * time.Hour,
		MemoryPattern:         "spiky",
		GPUUtilizationPattern: "burst",
		ScalingRequirements:   true,
	},
	{
		Name:                  "nlp-fine-tuning",
		ContainerImage:        "huggingface/transformers-pytorch-gpu",
		CPURequest:            "4",
		CPULimit:              "8",
		MemoryRequest:         "16Gi",
		MemoryLimit:           "32Gi",
		GPURequest:            1.0, // Single GPU for fine-tuning
		ExpectedDuration:      2 * time.Hour,
		MemoryPattern:         "constant",
		GPUUtilizationPattern: "steady",
		ScalingRequirements:   false,
	},
	// Inference Workloads
	{
		Name:                  "real-time-inference",
		ContainerImage:        "tritonserver:latest",
		CPURequest:            "2",
		CPULimit:              "4",
		MemoryRequest:         "8Gi",
		MemoryLimit:           "16Gi",
		GPURequest:            0.5, // Fractional GPU for inference
		ExpectedDuration:      30 * time.Minute,
		MemoryPattern:         "constant",
		GPUUtilizationPattern: "periodic",
		ScalingRequirements:   true,
	},
	{
		Name:                  "batch-inference",
		ContainerImage:        "rocm/pytorch:latest",
		CPURequest:            "8",
		CPULimit:              "16",
		MemoryRequest:         "32Gi",
		MemoryLimit:           "64Gi",
		GPURequest:            2.0, // 2 GPUs for batch processing
		ExpectedDuration:      1 * time.Hour,
		MemoryPattern:         "spiky",
		GPUUtilizationPattern: "burst",
		ScalingRequirements:   true,
	},
	// Research and Development Workloads
	{
		Name:                  "hyperparameter-tuning",
		ContainerImage:        "optuna/optuna:latest",
		CPURequest:            "4",
		CPULimit:              "8",
		MemoryRequest:         "16Gi",
		MemoryLimit:           "32Gi",
		GPURequest:            1.0, // Single GPU per trial
		ExpectedDuration:      6 * time.Hour,
		MemoryPattern:         "ramping",
		GPUUtilizationPattern: "periodic",
		ScalingRequirements:   true,
	},
	{
		Name:                  "data-preprocessing",
		ContainerImage:        "dask/dask:latest",
		CPURequest:            "16",
		CPULimit:              "32",
		MemoryRequest:         "64Gi",
		MemoryLimit:           "128Gi",
		GPURequest:            0.25, // Light GPU usage for data processing
		ExpectedDuration:      3 * time.Hour,
		MemoryPattern:         "spiky",
		GPUUtilizationPattern: "burst",
		ScalingRequirements:   true,
	},
	// Scientific Computing Workloads
	{
		Name:                  "molecular-dynamics",
		ContainerImage:        "gromacs/gromacs:latest",
		CPURequest:            "32",
		CPULimit:              "64",
		MemoryRequest:         "128Gi",
		MemoryLimit:           "256Gi",
		GPURequest:            8.0, // Multiple GPUs for simulation
		ExpectedDuration:      24 * time.Hour,
		MemoryPattern:         "constant",
		GPUUtilizationPattern: "steady",
		ScalingRequirements:   false,
	},
	{
		Name:                  "weather-simulation",
		ContainerImage:        "wrf/wrf:latest",
		CPURequest:            "64",
		CPULimit:              "128",
		MemoryRequest:         "256Gi",
		MemoryLimit:           "512Gi",
		GPURequest:            16.0, // Massive GPU requirements
		ExpectedDuration:      12 * time.Hour,
		MemoryPattern:         "ramping",
		GPUUtilizationPattern: "steady",
		ScalingRequirements:   true,
	},
	// Edge AI Workloads
	{
		Name:                  "edge-ai-inference",
		ContainerImage:        "onnxruntime/onnxruntime:latest",
		CPURequest:            "1",
		CPULimit:              "2",
		MemoryRequest:         "2Gi",
		MemoryLimit:           "4Gi",
		GPURequest:            0.1, // Very light GPU usage
		ExpectedDuration:      15 * time.Minute,
		MemoryPattern:         "constant",
		GPUUtilizationPattern: "periodic",
		ScalingRequirements:   true,
	},
}

// BenchmarkRealisticAIMLWorkloads benchmarks performance with realistic AI/ML workloads
func BenchmarkRealisticAIMLWorkloads(b *testing.B) {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	// Create workload mix representative of real production environments
	workloadMix := createRealisticWorkloadMix(100)

	b.ResetTimer()
	b.Run("Sequential-Scheduling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scheduler := NewKaiwoScheduler()
			for _, job := range workloadMix {
				result := scheduler.ScheduleJob(ctx, job)
				_ = result
			}
		}
	})

	b.Run("Concurrent-Scheduling", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scheduler := NewKaiwoScheduler()

			// Simulate concurrent job submissions
			jobChan := make(chan *v1alpha1.KaiwoJob, len(workloadMix))
			resultChan := make(chan SchedulingResult, len(workloadMix))

			// Start workers
			for w := 0; w < 10; w++ {
				go func() {
					for job := range jobChan {
						result := scheduler.ScheduleJob(ctx, job)
						resultChan <- result
					}
				}()
			}

			// Submit jobs
			for _, job := range workloadMix {
				jobChan <- job
			}
			close(jobChan)

			// Collect results
			for range workloadMix {
				<-resultChan
			}
		}
	})
}

// BenchmarkGangSchedulingRealisticWorkloads benchmarks gang scheduling with realistic distributed workloads
func BenchmarkGangSchedulingRealisticWorkloads(b *testing.B) {
	ctx := context.Background()

	// Create gang jobs that represent distributed training scenarios
	gangJobs := createDistributedTrainingGangs(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gangScheduler := NewGangScheduler()

		for _, gang := range gangJobs {
			result := gangScheduler.ScheduleGang(ctx, gang)
			_ = result
		}
	}
}

// BenchmarkElasticScalingAIWorkloads benchmarks elastic scaling with AI workloads
func BenchmarkElasticScalingAIWorkloads(b *testing.B) {
	ctx := context.Background()

	// Create workloads that require dynamic scaling
	scalableWorkloads := createScalableAIWorkloads(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elasticController := NewElasticController()

		for _, workload := range scalableWorkloads {
			decision := elasticController.EvaluateScaling(ctx, workload)
			if decision.ShouldScale {
				result := elasticController.ExecuteScaling(ctx, workload, decision)
				_ = result
			}
		}
	}
}

// BenchmarkResourceUtilizationPatterns benchmarks performance under different utilization patterns
func BenchmarkResourceUtilizationPatterns(b *testing.B) {
	utilizationPatterns := []struct {
		name          string
		cpuPattern    []float64
		memoryPattern []float64
		gpuPattern    []float64
	}{
		{
			name:          "constant-high-utilization",
			cpuPattern:    []float64{0.9, 0.9, 0.9, 0.9, 0.9},
			memoryPattern: []float64{0.85, 0.85, 0.85, 0.85, 0.85},
			gpuPattern:    []float64{0.95, 0.95, 0.95, 0.95, 0.95},
		},
		{
			name:          "bursty-utilization",
			cpuPattern:    []float64{0.2, 0.8, 0.3, 0.9, 0.1},
			memoryPattern: []float64{0.3, 0.7, 0.4, 0.8, 0.2},
			gpuPattern:    []float64{0.1, 0.9, 0.2, 0.95, 0.05},
		},
		{
			name:          "gradual-ramp-up",
			cpuPattern:    []float64{0.1, 0.3, 0.5, 0.7, 0.9},
			memoryPattern: []float64{0.2, 0.4, 0.6, 0.8, 0.9},
			gpuPattern:    []float64{0.05, 0.25, 0.5, 0.75, 0.95},
		},
	}

	for _, pattern := range utilizationPatterns {
		b.Run(pattern.name, func(b *testing.B) {
			ctx := context.Background()
			workloads := createWorkloadsWithUtilizationPattern(pattern, 30)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				optimizer := NewResourceOptimizer()

				for j, workload := range workloads {
					utilizationIndex := j % len(pattern.cpuPattern)
					optimization := optimizer.OptimizeForUtilization(ctx, workload,
						pattern.cpuPattern[utilizationIndex],
						pattern.memoryPattern[utilizationIndex],
						pattern.gpuPattern[utilizationIndex])
					_ = optimization
				}
			}
		})
	}
}

// BenchmarkMemoryIntensiveWorkloads benchmarks memory-intensive AI workloads
func BenchmarkMemoryIntensiveWorkloads(b *testing.B) {
	memoryProfiles := []struct {
		name               string
		baseMemory         string
		peakMemory         string
		allocationRate     float64 // MB/sec
		fragmentationLevel float64
	}{
		{"large-model-loading", "64Gi", "256Gi", 1000.0, 0.1},
		{"data-streaming", "16Gi", "128Gi", 500.0, 0.3},
		{"memory-fragmented", "32Gi", "64Gi", 200.0, 0.7},
		{"constant-high-memory", "128Gi", "128Gi", 50.0, 0.05},
	}

	for _, profile := range memoryProfiles {
		b.Run(profile.name, func(b *testing.B) {
			ctx := context.Background()
			workloads := createMemoryIntensiveWorkloads(profile, 25)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				memoryManager := NewMemoryManager()

				for _, workload := range workloads {
					allocation := memoryManager.AllocateMemory(ctx, workload)
					_ = allocation
				}
			}
		})
	}
}

// Helper functions to create realistic workload scenarios

func createRealisticWorkloadMix(count int) []*v1alpha1.KaiwoJob {
	jobs := make([]*v1alpha1.KaiwoJob, count)

	for i := 0; i < count; i++ {
		profile := AIMLWorkloadProfiles[rand.Intn(len(AIMLWorkloadProfiles))]
		jobs[i] = createJobFromProfile(profile, i)
	}

	return jobs
}

func createJobFromProfile(profile WorkloadProfile, index int) *v1alpha1.KaiwoJob {
	annotations := map[string]string{
		"kaiwo.ai/workload-type":     profile.Name,
		"kaiwo.ai/gpu-fraction":      fmt.Sprintf("%.2f", profile.GPURequest),
		"kaiwo.ai/memory-pattern":    profile.MemoryPattern,
		"kaiwo.ai/gpu-utilization":   profile.GPUUtilizationPattern,
		"kaiwo.ai/expected-duration": profile.ExpectedDuration.String(),
	}

	if profile.ScalingRequirements {
		annotations["kaiwo.ai/auto-scaling"] = "enabled"
	}

	job := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%d", profile.Name, index),
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				User:      "ai-researcher@amd.com",
				GpuVendor: "amd",
			},
			EntryPoint: generateRealisticEntrypoint(profile),
		},
	}

	// Add gang scheduling configuration for distributed workloads
	if profile.GPURequest >= 2.0 {
		job.Spec.GangScheduling = &v1alpha1.GangSchedulingSpec{
			Enabled:    true,
			MinMembers: int(profile.GPURequest),
			Timeout:    &metav1.Duration{Duration: 10 * time.Minute},
			Policy:     "strict",
		}
	}

	// Add elastic scaling configuration for scalable workloads
	if profile.ScalingRequirements {
		job.Spec.ElasticScaling = &v1alpha1.ElasticScalingSpec{
			Enabled:     true,
			MinReplicas: 1,
			MaxReplicas: 10,
			ScalingPolicy: &v1alpha1.ScalingPolicySpec{
				ScaleUpRate:   2,
				ScaleDownRate: 1,
				Cooldown:      &metav1.Duration{Duration: 5 * time.Minute},
			},
			Metrics: []v1alpha1.ScalingMetricSpec{
				{
					Type:      "cpu",
					Threshold: 70.0,
				},
				{
					Type:      "gpu",
					Threshold: 80.0,
				},
			},
		}
	}

	return job
}

func generateRealisticEntrypoint(profile WorkloadProfile) string {
	switch profile.Name {
	case "llm-training-70b":
		return "python train_llm.py --model-size 70b --batch-size 32 --learning-rate 1e-4"
	case "computer-vision-training":
		return "python train_cv.py --dataset imagenet --epochs 100 --batch-size 128"
	case "nlp-fine-tuning":
		return "python fine_tune.py --model bert-large --task classification --epochs 10"
	case "real-time-inference":
		return "tritonserver --model-repository=/models --log-verbose=1"
	case "batch-inference":
		return "python batch_inference.py --input-dir /data --output-dir /results --batch-size 64"
	case "hyperparameter-tuning":
		return "python optimize.py --n-trials 100 --sampler tpe --pruner median"
	case "data-preprocessing":
		return "python preprocess.py --input /raw-data --output /processed-data --workers 16"
	case "molecular-dynamics":
		return "gmx mdrun -s topol.tpr -deffnm md -cpi md.cpt -append -gpu_id 0123"
	case "weather-simulation":
		return "mpirun -np 64 wrf.exe"
	case "edge-ai-inference":
		return "python edge_inference.py --model mobilenet --optimization-level 3"
	default:
		return "python main.py"
	}
}

func generateRealisticEnvVars(profile WorkloadProfile) []v1.EnvVar {
	baseEnvs := []v1.EnvVar{
		{Name: "ROCM_VERSION", Value: "5.7"},
		{Name: "WORKLOAD_TYPE", Value: profile.Name},
		{Name: "GPU_MEMORY_FRACTION", Value: fmt.Sprintf("%.2f", profile.GPURequest)},
		{Name: "EXPECTED_DURATION", Value: profile.ExpectedDuration.String()},
	}

	// Add workload-specific environment variables
	switch profile.Name {
	case "llm-training-70b":
		baseEnvs = append(baseEnvs, []v1.EnvVar{
			{Name: "NCCL_DEBUG", Value: "INFO"},
			{Name: "NCCL_IB_DISABLE", Value: "1"},
			{Name: "PYTORCH_CUDA_ALLOC_CONF", Value: "max_split_size_mb:128"},
		}...)
	case "computer-vision-training":
		baseEnvs = append(baseEnvs, []v1.EnvVar{
			{Name: "OPENCV_VERSION", Value: "4.8"},
			{Name: "DATASET_PATH", Value: "/data/imagenet"},
		}...)
	case "real-time-inference":
		baseEnvs = append(baseEnvs, []v1.EnvVar{
			{Name: "TRITON_MAX_BATCH_SIZE", Value: "32"},
			{Name: "TRITON_RESPONSE_CACHE_BYTE_SIZE", Value: "1048576"},
		}...)
	}

	return baseEnvs
}

func createDistributedTrainingGangs(count int) []*Gang {
	gangs := make([]*Gang, count)

	for i := 0; i < count; i++ {
		gangs[i] = &Gang{
			ID:         fmt.Sprintf("distributed-training-gang-%d", i),
			MinMembers: 4,                                   // Minimum 4 workers for distributed training
			MaxMembers: 8,                                   // Maximum 8 workers
			Jobs:       createGangMembers(4 + rand.Intn(5)), // 4-8 workers
			Timeout:    10 * time.Minute,
		}
	}

	return gangs
}

func createGangMembers(memberCount int) []*v1alpha1.KaiwoJob {
	members := make([]*v1alpha1.KaiwoJob, memberCount)

	for i := 0; i < memberCount; i++ {
		// Use distributed training profile
		profile := WorkloadProfile{
			Name:                  "distributed-training-worker",
			ContainerImage:        "rocm/pytorch:latest",
			CPURequest:            "8",
			CPULimit:              "16",
			MemoryRequest:         "32Gi",
			MemoryLimit:           "64Gi",
			GPURequest:            2.0, // 2 GPUs per worker
			ExpectedDuration:      4 * time.Hour,
			MemoryPattern:         "constant",
			GPUUtilizationPattern: "steady",
			ScalingRequirements:   false,
		}

		job := createJobFromProfile(profile, i)
		job.ObjectMeta.Name = fmt.Sprintf("gang-worker-%d", i)

		// Add gang-specific annotations
		if job.ObjectMeta.Annotations == nil {
			job.ObjectMeta.Annotations = make(map[string]string)
		}
		job.ObjectMeta.Annotations["kaiwo.ai/gang-role"] = fmt.Sprintf("worker-%d", i)
		if i == 0 {
			job.ObjectMeta.Annotations["kaiwo.ai/gang-role"] = "master"
		}

		members[i] = job
	}

	return members
}

func createScalableAIWorkloads(count int) []*ElasticWorkload {
	workloads := make([]*ElasticWorkload, count)

	scalableProfiles := []WorkloadProfile{
		AIMLWorkloadProfiles[1], // computer-vision-training
		AIMLWorkloadProfiles[3], // real-time-inference
		AIMLWorkloadProfiles[4], // batch-inference
		AIMLWorkloadProfiles[5], // hyperparameter-tuning
		AIMLWorkloadProfiles[6], // data-preprocessing
	}

	for i := 0; i < count; i++ {
		profile := scalableProfiles[rand.Intn(len(scalableProfiles))]
		job := createJobFromProfile(profile, i)

		workloads[i] = &ElasticWorkload{
			Job:             job,
			CurrentReplicas: 1,
			DesiredReplicas: 1,
			MinReplicas:     1,
			MaxReplicas:     10,
			CurrentMetrics:  generateRealisticMetrics(),
		}
	}

	return workloads
}

func createWorkloadsWithUtilizationPattern(pattern struct {
	name          string
	cpuPattern    []float64
	memoryPattern []float64
	gpuPattern    []float64
}, count int) []*v1alpha1.KaiwoJob {

	jobs := make([]*v1alpha1.KaiwoJob, count)

	for i := 0; i < count; i++ {
		// Select a representative profile
		profile := AIMLWorkloadProfiles[rand.Intn(len(AIMLWorkloadProfiles))]
		job := createJobFromProfile(profile, i)

		// Add utilization pattern annotations
		patternIndex := i % len(pattern.cpuPattern)
		job.ObjectMeta.Annotations["kaiwo.ai/target-cpu-utilization"] = fmt.Sprintf("%.2f", pattern.cpuPattern[patternIndex])
		job.ObjectMeta.Annotations["kaiwo.ai/target-memory-utilization"] = fmt.Sprintf("%.2f", pattern.memoryPattern[patternIndex])
		job.ObjectMeta.Annotations["kaiwo.ai/target-gpu-utilization"] = fmt.Sprintf("%.2f", pattern.gpuPattern[patternIndex])

		jobs[i] = job
	}

	return jobs
}

func createMemoryIntensiveWorkloads(profile struct {
	name               string
	baseMemory         string
	peakMemory         string
	allocationRate     float64
	fragmentationLevel float64
}, count int) []*v1alpha1.KaiwoJob {

	jobs := make([]*v1alpha1.KaiwoJob, count)

	workloadProfile := WorkloadProfile{
		Name:                  profile.name,
		ContainerImage:        "rocm/pytorch:latest",
		CPURequest:            "8",
		CPULimit:              "16",
		MemoryRequest:         profile.baseMemory,
		MemoryLimit:           profile.peakMemory,
		GPURequest:            1.0,
		ExpectedDuration:      2 * time.Hour,
		MemoryPattern:         "ramping",
		GPUUtilizationPattern: "steady",
		ScalingRequirements:   false,
	}

	for i := 0; i < count; i++ {
		job := createJobFromProfile(workloadProfile, i)

		// Add memory-specific annotations
		job.ObjectMeta.Annotations["kaiwo.ai/memory-allocation-rate"] = fmt.Sprintf("%.2f", profile.allocationRate)
		job.ObjectMeta.Annotations["kaiwo.ai/memory-fragmentation-level"] = fmt.Sprintf("%.2f", profile.fragmentationLevel)

		jobs[i] = job
	}

	return jobs
}

func generateRealisticMetrics() map[string]float64 {
	return map[string]float64{
		"cpu_utilization":    0.3 + rand.Float64()*0.6, // 30-90%
		"memory_utilization": 0.2 + rand.Float64()*0.7, // 20-90%
		"gpu_utilization":    0.1 + rand.Float64()*0.8, // 10-90%
		"network_io":         rand.Float64() * 1000,    // 0-1000 MB/s
		"disk_io":            rand.Float64() * 500,     // 0-500 MB/s
		"queue_length":       rand.Float64() * 100,     // 0-100 jobs
		"throughput":         rand.Float64() * 1000,    // 0-1000 ops/sec
		"latency_p95":        10 + rand.Float64()*90,   // 10-100ms
	}
}

// Mock types for realistic workload testing
type KaiwoScheduler struct{}
type GangScheduler struct{}
type ElasticController struct{}
type ResourceOptimizer struct{}
type MemoryManager struct{}

type SchedulingResult struct {
	Success   bool
	Node      string
	Duration  time.Duration
	Resources map[string]interface{}
}

type Gang struct {
	ID         string
	MinMembers int
	MaxMembers int
	Jobs       []*v1alpha1.KaiwoJob
	Timeout    time.Duration
}

type ElasticWorkload struct {
	Job             *v1alpha1.KaiwoJob
	CurrentReplicas int32
	DesiredReplicas int32
	MinReplicas     int32
	MaxReplicas     int32
	CurrentMetrics  map[string]float64
}

type ScalingDecision struct {
	ShouldScale bool
	Direction   string // "up" or "down"
	Factor      float64
	Reason      string
}

// Mock implementations
func NewKaiwoScheduler() *KaiwoScheduler       { return &KaiwoScheduler{} }
func NewGangScheduler() *GangScheduler         { return &GangScheduler{} }
func NewElasticController() *ElasticController { return &ElasticController{} }
func NewResourceOptimizer() *ResourceOptimizer { return &ResourceOptimizer{} }
func NewMemoryManager() *MemoryManager         { return &MemoryManager{} }

func (s *KaiwoScheduler) ScheduleJob(ctx context.Context, job *v1alpha1.KaiwoJob) SchedulingResult {
	// Simulate realistic scheduling time based on job complexity
	baseDelay := 1 * time.Millisecond
	if job.Spec.GangScheduling != nil {
		baseDelay += 5 * time.Millisecond // Gang scheduling takes longer
	}
	if job.Spec.ElasticScaling != nil {
		baseDelay += 2 * time.Millisecond // Elastic scaling analysis takes time
	}

	time.Sleep(baseDelay)

	return SchedulingResult{
		Success:  true,
		Node:     fmt.Sprintf("node-%d", rand.Intn(10)),
		Duration: baseDelay,
		Resources: map[string]interface{}{
			"cpu":    job.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String(),
			"memory": job.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String(),
		},
	}
}

func (g *GangScheduler) ScheduleGang(ctx context.Context, gang *Gang) SchedulingResult {
	// Gang scheduling is more complex - simulate resource coordination time
	time.Sleep(time.Duration(len(gang.Jobs)) * 2 * time.Millisecond)

	return SchedulingResult{
		Success:  rand.Float64() > 0.1, // 90% success rate for gangs
		Duration: time.Duration(len(gang.Jobs)) * 2 * time.Millisecond,
	}
}

func (e *ElasticController) EvaluateScaling(ctx context.Context, workload *ElasticWorkload) ScalingDecision {
	time.Sleep(1 * time.Millisecond)

	// Simple scaling logic based on CPU utilization
	cpuUtil := workload.CurrentMetrics["cpu_utilization"]

	if cpuUtil > 0.8 && workload.CurrentReplicas < workload.MaxReplicas {
		return ScalingDecision{
			ShouldScale: true,
			Direction:   "up",
			Factor:      1.5,
			Reason:      "High CPU utilization",
		}
	} else if cpuUtil < 0.3 && workload.CurrentReplicas > workload.MinReplicas {
		return ScalingDecision{
			ShouldScale: true,
			Direction:   "down",
			Factor:      0.5,
			Reason:      "Low CPU utilization",
		}
	}

	return ScalingDecision{ShouldScale: false}
}

func (e *ElasticController) ExecuteScaling(ctx context.Context, workload *ElasticWorkload, decision ScalingDecision) SchedulingResult {
	time.Sleep(3 * time.Millisecond) // Scaling operations take longer

	return SchedulingResult{
		Success:  true,
		Duration: 3 * time.Millisecond,
	}
}

func (r *ResourceOptimizer) OptimizeForUtilization(ctx context.Context, job *v1alpha1.KaiwoJob, cpuUtil, memUtil, gpuUtil float64) interface{} {
	// Simulate optimization computation time based on complexity
	time.Sleep(2 * time.Millisecond)
	return struct{}{}
}

func (m *MemoryManager) AllocateMemory(ctx context.Context, job *v1alpha1.KaiwoJob) interface{} {
	// Simulate memory allocation time
	time.Sleep(1 * time.Millisecond)
	return struct{}{}
}
