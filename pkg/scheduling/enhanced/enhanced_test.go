package enhanced

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// Test Priority Scheduler
func TestPriorityScheduler_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	scheduler := NewPriorityScheduler(client)

	if scheduler == nil {
		t.Fatal("Expected non-nil scheduler")
	}

	if scheduler.client != client {
		t.Error("Expected client to be set")
	}

	if scheduler.jobQueue == nil {
		t.Error("Expected job queue to be initialized")
	}

	if scheduler.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestPriorityScheduler_GetJobPriority(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	scheduler := NewPriorityScheduler(client)

	// Test basic job
	job := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job",
			CreationTimestamp: metav1.Now(),
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Gpus:                  2,
				WorkloadPriorityClass: "high",
			},
		},
	}

	priority := scheduler.getJobPriority(job)
	if priority <= 0 {
		t.Errorf("Expected positive priority, got %d", priority)
	}
}

func TestPriorityScheduler_GetMetrics(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	scheduler := NewPriorityScheduler(client)

	metrics := scheduler.GetMetrics()
	// Check that metrics is a valid struct
	if metrics.TotalJobsScheduled < 0 {
		t.Error("Expected non-negative total jobs scheduled")
	}
}

// Test Resource Allocator
func TestResourceAllocator_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	allocator := NewResourceAllocator(client)

	if allocator == nil {
		t.Fatal("Expected non-nil allocator")
	}

	if allocator.client != client {
		t.Error("Expected client to be set")
	}

	if allocator.allocations == nil {
		t.Error("Expected allocations map to be initialized")
	}

	if allocator.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestResourceAllocator_CalculateRequiredGPU(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	allocator := NewResourceAllocator(client)

	job := &v1alpha1.KaiwoJob{
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Gpus: 4,
			},
		},
	}

	result := allocator.calculateRequiredGPU(job)
	if result != 4 {
		t.Errorf("Expected 4 GPUs, got %d", result)
	}
}

func TestResourceAllocator_CalculateRequiredCPU(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	allocator := NewResourceAllocator(client)

	job := &v1alpha1.KaiwoJob{
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("8"),
					},
				},
			},
		},
	}

	result := allocator.calculateRequiredCPU(job)
	expected := resource.MustParse("8")
	if !result.Equal(expected) {
		t.Errorf("Expected %s CPU, got %s", expected.String(), result.String())
	}
}

func TestResourceAllocator_GetMetrics(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	allocator := NewResourceAllocator(client)

	metrics := allocator.GetMetrics()
	// metrics is a value type, not a pointer
	if metrics.TotalAllocations < 0 {
		t.Error("Expected non-negative total allocations")
	}
}

// Test Load Balancer
func TestLoadBalancer_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	lb := NewLoadBalancer(client)

	if lb == nil {
		t.Fatal("Expected non-nil load balancer")
	}

	if lb.client != client {
		t.Error("Expected client to be set")
	}

	if lb.nodeStats == nil {
		t.Error("Expected nodeStats map to be initialized")
	}

	if lb.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

func TestLoadBalancer_CalculateLoadScore(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	lb := NewLoadBalancer(client)

	// Test empty node
	stats := &NodeStats{
		TotalCPU:    resource.MustParse("16"),
		UsedCPU:     resource.MustParse("0"),
		TotalMemory: resource.MustParse("64Gi"),
		UsedMemory:  resource.MustParse("0Gi"),
		TotalGPU:    4,
		UsedGPU:     0,
	}

	score := lb.calculateLoadScore(stats)
	if score != 0.0 {
		t.Errorf("Expected load score 0.0 for empty node, got %f", score)
	}

	// Test half-loaded node
	stats.UsedCPU = resource.MustParse("8")
	stats.UsedMemory = resource.MustParse("32Gi")
	stats.UsedGPU = 2

	score = lb.calculateLoadScore(stats)
	if score < 0.4 || score > 0.6 {
		t.Errorf("Expected load score around 0.5, got %f", score)
	}
}

func TestLoadBalancer_CalculateRequiredGPU(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	lb := NewLoadBalancer(client)

	job := &v1alpha1.KaiwoJob{
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Gpus: 3,
			},
		},
	}

	result := lb.calculateRequiredGPU(job)
	if result != 3 {
		t.Errorf("Expected 3 GPUs, got %d", result)
	}
}

func TestLoadBalancer_GetMetrics(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	lb := NewLoadBalancer(client)

	metrics := lb.GetMetrics()
	// metrics is a value type, not a pointer
	if metrics.TotalRebalances < 0 {
		t.Error("Expected non-negative total rebalances")
	}
}

// Benchmark tests
func BenchmarkPriorityScheduler_GetJobPriority(b *testing.B) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	scheduler := NewPriorityScheduler(client)

	job := &v1alpha1.KaiwoJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "benchmark-job",
			CreationTimestamp: metav1.Now(),
		},
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Gpus:                  2,
				WorkloadPriorityClass: "high",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scheduler.getJobPriority(job)
	}
}

func BenchmarkResourceAllocator_CalculateRequiredGPU(b *testing.B) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	allocator := NewResourceAllocator(client)

	job := &v1alpha1.KaiwoJob{
		Spec: v1alpha1.KaiwoJobSpec{
			CommonMetaSpec: v1alpha1.CommonMetaSpec{
				Gpus: 4,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		allocator.calculateRequiredGPU(job)
	}
}

func BenchmarkLoadBalancer_CalculateLoadScore(b *testing.B) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	lb := NewLoadBalancer(client)

	stats := &NodeStats{
		TotalCPU:    resource.MustParse("16"),
		UsedCPU:     resource.MustParse("8"),
		TotalMemory: resource.MustParse("64Gi"),
		UsedMemory:  resource.MustParse("32Gi"),
		TotalGPU:    4,
		UsedGPU:     2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.calculateLoadScore(stats)
	}
}
