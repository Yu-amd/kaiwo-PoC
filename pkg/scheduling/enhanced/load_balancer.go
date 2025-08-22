package enhanced

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// LoadBalancer implements dynamic load balancing for KaiwoJobs
type LoadBalancer struct {
	client    client.Client
	mu        sync.RWMutex
	nodeStats map[string]*NodeStats
	metrics   *LoadBalancerMetrics
}

// NodeStats tracks resource usage statistics for a node
type NodeStats struct {
	NodeName    string
	TotalGPU    int64
	UsedGPU     int64
	TotalCPU    resource.Quantity
	UsedCPU     resource.Quantity
	TotalMemory resource.Quantity
	UsedMemory  resource.Quantity
	LoadScore   float64
	LastUpdated time.Time
}

// LoadBalancerMetrics tracks load balancing performance metrics
type LoadBalancerMetrics struct {
	TotalRebalances      int64
	SuccessfulRebalances int64
	FailedRebalances     int64
	AverageRebalanceTime time.Duration
	mu                   sync.RWMutex
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(client client.Client) *LoadBalancer {
	return &LoadBalancer{
		client:    client,
		nodeStats: make(map[string]*NodeStats),
		metrics: &LoadBalancerMetrics{
			TotalRebalances:      0,
			SuccessfulRebalances: 0,
			FailedRebalances:     0,
		},
	}
}

// UpdateNodeStats updates the resource statistics for a node
func (lb *LoadBalancer) UpdateNodeStats(ctx context.Context, nodeName string) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Get node information
	var node corev1.Node
	if err := lb.client.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// Get pods running on this node
	var pods corev1.PodList
	if err := lb.client.List(ctx, &pods, client.MatchingFields{"spec.nodeName": nodeName}); err != nil {
		return fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	// Calculate resource usage
	stats := &NodeStats{
		NodeName:    nodeName,
		LastUpdated: time.Now(),
	}

	// Get total capacity
	if cpu, ok := node.Status.Capacity[corev1.ResourceCPU]; ok {
		stats.TotalCPU = cpu
	}
	if mem, ok := node.Status.Capacity[corev1.ResourceMemory]; ok {
		stats.TotalMemory = mem
	}

	// Calculate used resources from pods
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			for _, container := range pod.Spec.Containers {
				if container.Resources.Requests != nil {
					if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
						stats.UsedCPU.Add(cpu)
					}
					if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
						stats.UsedMemory.Add(mem)
					}
				}
			}
		}
	}

	// Calculate load score (weighted average of resource utilization)
	stats.LoadScore = lb.calculateLoadScore(stats)

	// Update node stats
	lb.nodeStats[nodeName] = stats

	return nil
}

// calculateLoadScore calculates a load score for a node based on resource utilization
func (lb *LoadBalancer) calculateLoadScore(stats *NodeStats) float64 {
	if stats.TotalGPU == 0 && stats.TotalCPU.IsZero() && stats.TotalMemory.IsZero() {
		return 0.0
	}

	gpuScore := 0.0
	if stats.TotalGPU > 0 {
		gpuScore = float64(stats.UsedGPU) / float64(stats.TotalGPU)
	}

	cpuScore := 0.0
	if !stats.TotalCPU.IsZero() {
		cpuScore = float64(stats.UsedCPU.MilliValue()) / float64(stats.TotalCPU.MilliValue())
	}

	memScore := 0.0
	if !stats.TotalMemory.IsZero() {
		memScore = float64(stats.UsedMemory.Value()) / float64(stats.TotalMemory.Value())
	}

	// Weighted average: GPU (50%), CPU (30%), Memory (20%)
	return (gpuScore * 0.5) + (cpuScore * 0.3) + (memScore * 0.2)
}

// FindOptimalNode finds the optimal node for a job based on load balancing
func (lb *LoadBalancer) FindOptimalNode(ctx context.Context, job *v1alpha1.KaiwoJob) (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Update stats for all nodes if needed
	if err := lb.updateAllNodeStats(ctx); err != nil {
		return "", fmt.Errorf("failed to update node stats: %w", err)
	}

	// Calculate job requirements
	requiredGPU := lb.calculateRequiredGPU(job)
	requiredCPU := lb.calculateRequiredCPU(job)
	requiredMem := lb.calculateRequiredMemory(job)

	// Find nodes that can accommodate the job
	var candidateNodes []string
	for nodeName, stats := range lb.nodeStats {
		// Check if node has sufficient resources
		availableGPU := stats.TotalGPU - stats.UsedGPU
		availableCPU := stats.TotalCPU.DeepCopy()
		availableCPU.Sub(stats.UsedCPU)
		availableMem := stats.TotalMemory.DeepCopy()
		availableMem.Sub(stats.UsedMemory)

		if availableGPU >= requiredGPU &&
			availableCPU.Cmp(requiredCPU) >= 0 &&
			availableMem.Cmp(requiredMem) >= 0 {
			candidateNodes = append(candidateNodes, nodeName)
		}
	}

	if len(candidateNodes) == 0 {
		return "", fmt.Errorf("no nodes available with sufficient resources for job %s", job.Name)
	}

	// Find the node with the lowest load score
	var optimalNode string
	lowestLoadScore := 1.0

	for _, nodeName := range candidateNodes {
		stats := lb.nodeStats[nodeName]
		if stats.LoadScore < lowestLoadScore {
			lowestLoadScore = stats.LoadScore
			optimalNode = nodeName
		}
	}

	return optimalNode, nil
}

// RebalanceCluster performs load balancing across the cluster
func (lb *LoadBalancer) RebalanceCluster(ctx context.Context) error {
	startTime := time.Now()

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Update metrics
	lb.metrics.mu.Lock()
	lb.metrics.TotalRebalances++
	lb.metrics.mu.Unlock()

	// Update all node stats
	if err := lb.updateAllNodeStats(ctx); err != nil {
		lb.updateFailedMetrics(time.Since(startTime))
		return fmt.Errorf("failed to update node stats: %w", err)
	}

	// Find overloaded nodes (load score > 0.8)
	var overloadedNodes []string
	var underloadedNodes []string

	for nodeName, stats := range lb.nodeStats {
		if stats.LoadScore > 0.8 {
			overloadedNodes = append(overloadedNodes, nodeName)
		} else if stats.LoadScore < 0.3 {
			underloadedNodes = append(underloadedNodes, nodeName)
		}
	}

	// Attempt to move jobs from overloaded to underloaded nodes
	rebalanceCount := 0
	for _, overloadedNode := range overloadedNodes {
		for _, underloadedNode := range underloadedNodes {
			if rebalanceCount >= 5 { // Limit rebalancing to prevent thrashing
				break
			}

			if err := lb.moveJobFromNode(ctx, overloadedNode, underloadedNode); err == nil {
				rebalanceCount++
			}
		}
	}

	// Update successful metrics
	lb.updateSuccessfulMetrics(time.Since(startTime))

	return nil
}

// moveJobFromNode attempts to move a job from one node to another
func (lb *LoadBalancer) moveJobFromNode(ctx context.Context, fromNode, toNode string) error {
	// Get pods on the overloaded node
	var pods corev1.PodList
	if err := lb.client.List(ctx, &pods, client.MatchingFields{"spec.nodeName": fromNode}); err != nil {
		return fmt.Errorf("failed to list pods on node %s: %w", fromNode, err)
	}

	// Find a suitable job to move
	for _, pod := range pods.Items {
		// Check if this is a KaiwoJob pod
		if pod.Labels["kaiwo.ai/job-name"] != "" {
			// Check if the target node can accommodate this pod
			if lb.canNodeAccommodatePod(ctx, toNode, &pod) {
				// Evict the pod to trigger rescheduling
				if err := lb.client.Delete(ctx, &pod); err != nil {
					return fmt.Errorf("failed to evict pod %s: %w", pod.Name, err)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable jobs found to move from %s to %s", fromNode, toNode)
}

// canNodeAccommodatePod checks if a node can accommodate a pod
func (lb *LoadBalancer) canNodeAccommodatePod(ctx context.Context, nodeName string, pod *corev1.Pod) bool {
	stats, exists := lb.nodeStats[nodeName]
	if !exists {
		return false
	}

	// Calculate pod requirements
	requiredGPU := int64(0)
	requiredCPU := resource.Quantity{}
	requiredMem := resource.Quantity{}

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				requiredCPU.Add(cpu)
			}
			if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				requiredMem.Add(mem)
			}
		}
	}

	// Check if node has sufficient available resources
	availableGPU := stats.TotalGPU - stats.UsedGPU
	availableCPU := stats.TotalCPU.DeepCopy()
	availableCPU.Sub(stats.UsedCPU)
	availableMem := stats.TotalMemory.DeepCopy()
	availableMem.Sub(stats.UsedMemory)

	return availableGPU >= requiredGPU &&
		availableCPU.Cmp(requiredCPU) >= 0 &&
		availableMem.Cmp(requiredMem) >= 0
}

// updateAllNodeStats updates statistics for all nodes
func (lb *LoadBalancer) updateAllNodeStats(ctx context.Context) error {
	var nodes corev1.NodeList
	if err := lb.client.List(ctx, &nodes); err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes.Items {
		if err := lb.UpdateNodeStats(ctx, node.Name); err != nil {
			return fmt.Errorf("failed to update stats for node %s: %w", node.Name, err)
		}
	}

	return nil
}

// calculateRequiredGPU calculates the total GPU requirements for a job
func (lb *LoadBalancer) calculateRequiredGPU(job *v1alpha1.KaiwoJob) int64 {
	// Use the Gpus field from the job spec
	return int64(job.Spec.Gpus)
}

// calculateRequiredCPU calculates the total CPU requirements for a job
func (lb *LoadBalancer) calculateRequiredCPU(job *v1alpha1.KaiwoJob) resource.Quantity {
	// Use default CPU requirements from job spec resources
	if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
		if cpu, ok := job.Spec.Resources.Requests[corev1.ResourceCPU]; ok {
			return cpu
		}
	}
	// Default CPU requirement
	return resource.MustParse("1")
}

// calculateRequiredMemory calculates the total memory requirements for a job
func (lb *LoadBalancer) calculateRequiredMemory(job *v1alpha1.KaiwoJob) resource.Quantity {
	// Use default memory requirements from job spec resources
	if job.Spec.Resources != nil && job.Spec.Resources.Requests != nil {
		if mem, ok := job.Spec.Resources.Requests[corev1.ResourceMemory]; ok {
			return mem
		}
	}
	// Default memory requirement
	return resource.MustParse("4Gi")
}

// updateSuccessfulMetrics updates metrics for successful rebalancing
func (lb *LoadBalancer) updateSuccessfulMetrics(rebalanceTime time.Duration) {
	lb.metrics.mu.Lock()
	defer lb.metrics.mu.Unlock()

	lb.metrics.SuccessfulRebalances++

	// Update average rebalance time
	if lb.metrics.SuccessfulRebalances > 0 {
		totalTime := lb.metrics.AverageRebalanceTime * time.Duration(lb.metrics.SuccessfulRebalances-1)
		lb.metrics.AverageRebalanceTime = (totalTime + rebalanceTime) / time.Duration(lb.metrics.SuccessfulRebalances)
	} else {
		lb.metrics.AverageRebalanceTime = rebalanceTime
	}
}

// updateFailedMetrics updates metrics for failed rebalancing
func (lb *LoadBalancer) updateFailedMetrics(rebalanceTime time.Duration) {
	lb.metrics.mu.Lock()
	defer lb.metrics.mu.Unlock()

	lb.metrics.FailedRebalances++
}

// GetMetrics returns current load balancer metrics
func (lb *LoadBalancer) GetMetrics() LoadBalancerMetrics {
	lb.metrics.mu.RLock()
	defer lb.metrics.mu.RUnlock()

	return *lb.metrics
}

// GetNodeStats returns current node statistics
func (lb *LoadBalancer) GetNodeStats() map[string]*NodeStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Return a copy to avoid race conditions
	nodeStats := make(map[string]*NodeStats)
	for k, v := range lb.nodeStats {
		nodeStats[k] = v
	}

	return nodeStats
}
