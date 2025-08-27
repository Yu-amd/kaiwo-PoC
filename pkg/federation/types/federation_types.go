/*
Copyright 2025 Kaiwo Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterInfo represents information about a federated cluster
type ClusterInfo struct {
	// Name is the unique identifier for the cluster
	Name string `json:"name"`

	// Endpoint is the API server endpoint for the cluster
	Endpoint string `json:"endpoint"`

	// Region specifies the geographic region
	Region string `json:"region"`

	// Zone specifies the availability zone
	Zone string `json:"zone,omitempty"`

	// Provider specifies the cloud provider (AWS, Azure, GCP, On-premises)
	Provider string `json:"provider"`

	// Capabilities lists the cluster's capabilities
	Capabilities ClusterCapabilities `json:"capabilities"`

	// Status represents the current cluster status
	Status ClusterStatus `json:"status"`

	// LastHeartbeat is the timestamp of the last heartbeat
	LastHeartbeat metav1.Time `json:"lastHeartbeat"`

	// Metadata contains additional cluster metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ClusterCapabilities represents the capabilities of a federated cluster
type ClusterCapabilities struct {
	// GPU configuration
	GPU GPUCapabilities `json:"gpu"`

	// Compute resources
	Compute ComputeCapabilities `json:"compute"`

	// Storage capabilities
	Storage StorageCapabilities `json:"storage"`

	// Network capabilities
	Network NetworkCapabilities `json:"network"`

	// Special features
	Features []string `json:"features,omitempty"`
}

// GPUCapabilities represents GPU-specific capabilities
type GPUCapabilities struct {
	// Total GPU count
	TotalGPUs int32 `json:"totalGPUs"`

	// Available GPU count
	AvailableGPUs int32 `json:"availableGPUs"`

	// GPU types available (e.g., MI300X, MI250X)
	GPUTypes []string `json:"gpuTypes"`

	// Fractional allocation support
	FractionalSupport bool `json:"fractionalSupport"`

	// Time-slicing support
	TimeslicingSupport bool `json:"timeslicingSupport"`

	// Memory per GPU (in MB)
	MemoryPerGPU map[string]int64 `json:"memoryPerGPU"`
}

// ComputeCapabilities represents compute resource capabilities
type ComputeCapabilities struct {
	// Total CPU cores
	TotalCPU int64 `json:"totalCPU"`

	// Available CPU cores
	AvailableCPU int64 `json:"availableCPU"`

	// Total memory (in MB)
	TotalMemory int64 `json:"totalMemory"`

	// Available memory (in MB)
	AvailableMemory int64 `json:"availableMemory"`

	// Node count
	NodeCount int32 `json:"nodeCount"`

	// Architecture (e.g., amd64, arm64)
	Architecture []string `json:"architecture"`
}

// StorageCapabilities represents storage capabilities
type StorageCapabilities struct {
	// Total storage (in GB)
	TotalStorage int64 `json:"totalStorage"`

	// Available storage (in GB)
	AvailableStorage int64 `json:"availableStorage"`

	// Storage classes available
	StorageClasses []string `json:"storageClasses"`

	// High-performance storage support
	HighPerformanceStorage bool `json:"highPerformanceStorage"`
}

// NetworkCapabilities represents network capabilities
type NetworkCapabilities struct {
	// High-bandwidth networking
	HighBandwidth bool `json:"highBandwidth"`

	// Low-latency networking
	LowLatency bool `json:"lowLatency"`

	// Service mesh support
	ServiceMesh bool `json:"serviceMesh"`

	// Load balancer types
	LoadBalancerTypes []string `json:"loadBalancerTypes"`
}

// ClusterStatus represents the current status of a cluster
type ClusterStatus string

const (
	ClusterStatusActive      ClusterStatus = "Active"
	ClusterStatusUnavailable ClusterStatus = "Unavailable"
	ClusterStatusMaintenance ClusterStatus = "Maintenance"
	ClusterStatusDegraded    ClusterStatus = "Degraded"
)

// FederationPolicy defines workload placement and scheduling policies
type FederationPolicy struct {
	// Name is the policy identifier
	Name string `json:"name"`

	// Placement strategy
	Placement PlacementStrategy `json:"placement"`

	// Resource allocation strategy
	ResourceAllocation ResourceAllocationStrategy `json:"resourceAllocation"`

	// Failover configuration
	Failover FailoverConfig `json:"failover"`

	// Cost optimization settings
	CostOptimization CostOptimizationConfig `json:"costOptimization"`

	// Security policies
	Security SecurityPolicy `json:"security"`
}

// PlacementStrategy defines how workloads are placed across clusters
type PlacementStrategy struct {
	// Strategy type (Balanced, HighPerformance, CostOptimized, Custom)
	Type string `json:"type"`

	// Cluster preferences (ordered list)
	ClusterPreferences []string `json:"clusterPreferences,omitempty"`

	// Region preferences
	RegionPreferences []string `json:"regionPreferences,omitempty"`

	// Anti-affinity rules
	AntiAffinity []AntiAffinityRule `json:"antiAffinity,omitempty"`

	// Custom placement rules
	CustomRules []PlacementRule `json:"customRules,omitempty"`
}

// AntiAffinityRule defines workload anti-affinity
type AntiAffinityRule struct {
	// Scope (cluster, region, zone)
	Scope string `json:"scope"`

	// Workload selectors
	WorkloadSelectors map[string]string `json:"workloadSelectors"`
}

// PlacementRule defines custom placement logic
type PlacementRule struct {
	// Name of the rule
	Name string `json:"name"`

	// Condition for rule application
	Condition string `json:"condition"`

	// Action to take
	Action string `json:"action"`

	// Weight for scoring
	Weight int32 `json:"weight,omitempty"`
}

// ResourceAllocationStrategy defines resource allocation across clusters
type ResourceAllocationStrategy struct {
	// Strategy type
	Type string `json:"type"`

	// Resource quotas per cluster
	ClusterQuotas map[string]ResourceQuota `json:"clusterQuotas,omitempty"`

	// Dynamic allocation settings
	DynamicAllocation DynamicAllocationConfig `json:"dynamicAllocation"`
}

// ResourceQuota defines resource limits for a cluster
type ResourceQuota struct {
	// CPU quota
	CPU string `json:"cpu"`

	// Memory quota
	Memory string `json:"memory"`

	// GPU quota
	GPU int32 `json:"gpu"`

	// Storage quota
	Storage string `json:"storage"`
}

// DynamicAllocationConfig defines dynamic resource allocation
type DynamicAllocationConfig struct {
	// Enable dynamic allocation
	Enabled bool `json:"enabled"`

	// Load balancing threshold
	LoadThreshold float64 `json:"loadThreshold"`

	// Resource utilization threshold
	UtilizationThreshold float64 `json:"utilizationThreshold"`

	// Rebalancing interval
	RebalancingInterval time.Duration `json:"rebalancingInterval"`
}

// FailoverConfig defines failover behavior
type FailoverConfig struct {
	// Enable automatic failover
	AutoFailover bool `json:"autoFailover"`

	// Failover timeout
	FailoverTimeout time.Duration `json:"failoverTimeout"`

	// Maximum failover attempts
	MaxAttempts int32 `json:"maxAttempts"`

	// Backup clusters (ordered by priority)
	BackupClusters []string `json:"backupClusters"`
}

// CostOptimizationConfig defines cost optimization settings
type CostOptimizationConfig struct {
	// Enable cost optimization
	Enabled bool `json:"enabled"`

	// Spot instance preference
	PreferSpotInstances bool `json:"preferSpotInstances"`

	// Cost threshold
	CostThreshold float64 `json:"costThreshold"`

	// Preemptible workload tolerance
	PreemptibleTolerance bool `json:"preemptibleTolerance"`
}

// SecurityPolicy defines security requirements
type SecurityPolicy struct {
	// Data residency requirements
	DataResidency []string `json:"dataResidency,omitempty"`

	// Compliance requirements
	ComplianceRequirements []string `json:"complianceRequirements,omitempty"`

	// Network isolation requirements
	NetworkIsolation bool `json:"networkIsolation"`

	// Encryption requirements
	EncryptionRequired bool `json:"encryptionRequired"`
}

// FederatedWorkload represents a workload distributed across clusters
type FederatedWorkload struct {
	// Workload specification
	Spec FederatedWorkloadSpec `json:"spec"`

	// Current status
	Status FederatedWorkloadStatus `json:"status"`

	// Metadata
	Metadata metav1.ObjectMeta `json:"metadata"`
}

// FederatedWorkloadSpec defines the desired state
type FederatedWorkloadSpec struct {
	// Template for the workload
	Template interface{} `json:"template"`

	// Cluster placement requirements
	Placement WorkloadPlacement `json:"placement"`

	// Resource requirements
	Resources WorkloadResources `json:"resources"`

	// Policy reference
	PolicyRef string `json:"policyRef,omitempty"`
}

// WorkloadPlacement defines where the workload should be placed
type WorkloadPlacement struct {
	// Required clusters
	RequiredClusters []string `json:"requiredClusters,omitempty"`

	// Preferred clusters
	PreferredClusters []string `json:"preferredClusters,omitempty"`

	// Excluded clusters
	ExcludedClusters []string `json:"excludedClusters,omitempty"`

	// Minimum replicas per cluster
	MinReplicasPerCluster int32 `json:"minReplicasPerCluster,omitempty"`

	// Maximum replicas per cluster
	MaxReplicasPerCluster int32 `json:"maxReplicasPerCluster,omitempty"`
}

// WorkloadResources defines resource requirements for federated workloads
type WorkloadResources struct {
	// CPU requirements
	CPU string `json:"cpu"`

	// Memory requirements
	Memory string `json:"memory"`

	// GPU requirements
	GPU int32 `json:"gpu,omitempty"`

	// Storage requirements
	Storage string `json:"storage,omitempty"`

	// Special requirements
	SpecialRequirements map[string]string `json:"specialRequirements,omitempty"`
}

// FederatedWorkloadStatus represents the current status
type FederatedWorkloadStatus struct {
	// Overall phase
	Phase string `json:"phase"`

	// Per-cluster status
	ClusterStatus map[string]ClusterWorkloadStatus `json:"clusterStatus"`

	// Last update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ClusterWorkloadStatus represents workload status in a specific cluster
type ClusterWorkloadStatus struct {
	// Phase in this cluster
	Phase string `json:"phase"`

	// Replica count
	Replicas int32 `json:"replicas"`

	// Ready replicas
	ReadyReplicas int32 `json:"readyReplicas"`

	// Resource allocation
	AllocatedResources WorkloadResources `json:"allocatedResources"`

	// Last update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
}
