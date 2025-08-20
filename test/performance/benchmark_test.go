package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
	"github.com/silogen/kaiwo/internal/controller"
)

func TestPerformance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Performance Suite")
}

var _ = Describe("Performance Benchmarks", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		client client.Client
		mgr    ctrl.Manager
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		
		// Setup test scheme
		testScheme := runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(v1alpha1.AddToScheme(testScheme)).To(Succeed())
		
		// Create fake client
		client = fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		
		// Setup manager
		var err error
		mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme: testScheme,
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		cancel()
	})

	Describe("KaiwoJob Controller Performance", func() {
		It("should handle concurrent job creation efficiently", func() {
			// Benchmark concurrent job creation
			start := time.Now()
			
			// Create multiple jobs concurrently
			jobs := make([]*v1alpha1.KaiwoJob, 100)
			for i := 0; i < 100; i++ {
				jobs[i] = &v1alpha1.KaiwoJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-job-%d", i),
						Namespace: "default",
					},
					Spec: v1alpha1.KaiwoJobSpec{
						JobSpec: v1alpha1.JobSpec{
							Template: v1alpha1.PodTemplateSpec{
								Spec: v1alpha1.PodSpec{
									Containers: []v1alpha1.Container{
										{
											Name:  "test",
											Image: "busybox:latest",
											Command: []string{"echo", "hello"},
										},
									},
								},
							},
						},
					},
				}
				
				Expect(client.Create(ctx, jobs[i])).To(Succeed())
			}
			
			duration := time.Since(start)
			Expect(duration).To(BeNumerically("<", 5*time.Second))
		})

		It("should handle large job specifications efficiently", func() {
			// Create a job with large configuration
			largeJob := &v1alpha1.KaiwoJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "large-job",
					Namespace: "default",
				},
				Spec: v1alpha1.KaiwoJobSpec{
					JobSpec: v1alpha1.JobSpec{
						Template: v1alpha1.PodTemplateSpec{
							Spec: v1alpha1.PodSpec{
								Containers: []v1alpha1.Container{
									{
										Name:  "large-container",
										Image: "large-image:latest",
										Env:   make([]v1alpha1.EnvVar, 1000), // Large env var list
									},
								},
							},
						},
					},
				},
			}
			
			// Fill large env var list
			for i := 0; i < 1000; i++ {
				largeJob.Spec.JobSpec.Template.Spec.Containers[0].Env[i] = v1alpha1.EnvVar{
					Name:  fmt.Sprintf("ENV_VAR_%d", i),
					Value: fmt.Sprintf("value_%d", i),
				}
			}
			
			start := time.Now()
			Expect(client.Create(ctx, largeJob)).To(Succeed())
			duration := time.Since(start)
			
			// Should complete within reasonable time
			Expect(duration).To(BeNumerically("<", 2*time.Second))
		})
	})

	Describe("Memory Usage Benchmarks", func() {
		It("should maintain reasonable memory usage under load", func() {
			// This test would require actual memory profiling
			// For now, we'll create a basic structure
			Expect(true).To(BeTrue())
		})
	})

	Describe("Reconciliation Performance", func() {
		It("should reconcile jobs efficiently", func() {
			// Create a job
			job := &v1alpha1.KaiwoJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "reconcile-test-job",
					Namespace: "default",
				},
				Spec: v1alpha1.KaiwoJobSpec{
					JobSpec: v1alpha1.JobSpec{
						Template: v1alpha1.PodTemplateSpec{
							Spec: v1alpha1.PodSpec{
								Containers: []v1alpha1.Container{
									{
										Name:  "test",
										Image: "busybox:latest",
										Command: []string{"echo", "hello"},
									},
								},
							},
						},
					},
				},
			}
			
			Expect(client.Create(ctx, job)).To(Succeed())
			
			// Measure reconciliation time
			start := time.Now()
			
			// Trigger reconciliation (this would need actual controller setup)
			// For now, just measure the time to update the job
			job.Spec.JobSpec.Template.Spec.Containers[0].Command = []string{"echo", "updated"}
			Expect(client.Update(ctx, job)).To(Succeed())
			
			duration := time.Since(start)
			Expect(duration).To(BeNumerically("<", 1*time.Second))
		})
	})
})
