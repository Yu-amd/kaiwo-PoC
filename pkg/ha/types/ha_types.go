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

// ComponentStatus represents the health status of a component
type ComponentStatus string

const (
	ComponentStatusHealthy    ComponentStatus = "Healthy"
	ComponentStatusUnhealthy  ComponentStatus = "Unhealthy"
	ComponentStatusFailed     ComponentStatus = "Failed"
	ComponentStatusRecovering ComponentStatus = "Recovering"
	ComponentStatusUnknown    ComponentStatus = "Unknown"
)

// ComponentHealth represents the health information of a component
type ComponentHealth struct {
	// Component identification
	Name     string `json:"name"`
	Type     string `json:"type"` // scheduler, controller, api-server, etc.
	Instance string `json:"instance"`
	Version  string `json:"version"`

	// Health status
	Status        ComponentStatus `json:"status"`
	StatusMessage string          `json:"statusMessage,omitempty"`

	// Metadata
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`

	// Health metrics
	HealthMetrics HealthMetrics `json:"healthMetrics"`

	// Timing information
	LastHealthCheck metav1.Time          `json:"lastHealthCheck"`
	HealthHistory   []HealthHistoryEntry `json:"healthHistory,omitempty"`

	// Failover configuration
	FailoverConfig FailoverConfig `json:"failoverConfig,omitempty"`

	// Dependencies
	Dependencies []string `json:"dependencies,omitempty"`
}

// HealthMetrics represents detailed health metrics
type HealthMetrics struct {
	// CPU usage (percentage)
	CPUUsage float64 `json:"cpuUsage,omitempty"`

	// Memory usage (percentage)
	MemoryUsage float64 `json:"memoryUsage,omitempty"`

	// Response time (milliseconds)
	ResponseTime float64 `json:"responseTime,omitempty"`

	// Error rate (percentage)
	ErrorRate float64 `json:"errorRate,omitempty"`

	// Throughput (requests per second)
	Throughput float64 `json:"throughput,omitempty"`

	// Connection count
	ConnectionCount int32 `json:"connectionCount,omitempty"`

	// Custom metrics
	CustomMetrics map[string]float64 `json:"customMetrics,omitempty"`
}

// HealthHistoryEntry represents a point in health history
type HealthHistoryEntry struct {
	Timestamp metav1.Time     `json:"timestamp"`
	Status    ComponentStatus `json:"status"`
	Message   string          `json:"message,omitempty"`
	Metrics   HealthMetrics   `json:"metrics,omitempty"`
}

// FailoverConfig defines failover behavior for a component
type FailoverConfig struct {
	// Enable automatic failover
	Enabled bool `json:"enabled"`

	// Failover strategy
	Strategy string `json:"strategy"` // "immediate", "graceful", "manual"

	// Failover timeout
	Timeout time.Duration `json:"timeout"`

	// Maximum failover attempts
	MaxAttempts int `json:"maxAttempts"`

	// Target instances for failover
	FailoverTargets []string `json:"failoverTargets,omitempty"`

	// Health thresholds
	HealthThresholds HealthThresholds `json:"healthThresholds"`
}

// HealthThresholds defines when to consider a component unhealthy
type HealthThresholds struct {
	// CPU threshold (percentage)
	CPUThreshold float64 `json:"cpuThreshold,omitempty"`

	// Memory threshold (percentage)
	MemoryThreshold float64 `json:"memoryThreshold,omitempty"`

	// Response time threshold (milliseconds)
	ResponseTimeThreshold float64 `json:"responseTimeThreshold,omitempty"`

	// Error rate threshold (percentage)
	ErrorRateThreshold float64 `json:"errorRateThreshold,omitempty"`

	// Consecutive failures before marking unhealthy
	FailureThreshold int `json:"failureThreshold,omitempty"`
}

// ClusterHealth represents overall cluster health
type ClusterHealth struct {
	// Overall status
	OverallStatus ComponentStatus `json:"overallStatus"`

	// Component counts
	ComponentCount int `json:"componentCount"`
	HealthyCount   int `json:"healthyCount"`
	UnhealthyCount int `json:"unhealthyCount"`
	FailedCount    int `json:"failedCount"`

	// Component details
	Components map[string]ComponentStatus `json:"components"`

	// Health summary
	HealthSummary HealthSummary `json:"healthSummary"`

	// Last update
	LastUpdate metav1.Time `json:"lastUpdate"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	// Average metrics across all components
	AvgCPUUsage     float64 `json:"avgCpuUsage"`
	AvgMemoryUsage  float64 `json:"avgMemoryUsage"`
	AvgResponseTime float64 `json:"avgResponseTime"`

	// Peak metrics
	PeakCPUUsage     float64 `json:"peakCpuUsage"`
	PeakMemoryUsage  float64 `json:"peakMemoryUsage"`
	PeakResponseTime float64 `json:"peakResponseTime"`

	// Health score (0-100)
	HealthScore float64 `json:"healthScore"`
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	// Event identification
	ID        string      `json:"id"`
	Timestamp metav1.Time `json:"timestamp"`

	// Component information
	ComponentName string `json:"componentName"`
	ComponentType string `json:"componentType"`

	// Failover details
	Reason         string `json:"reason"`
	Strategy       string `json:"strategy"`
	SourceInstance string `json:"sourceInstance"`
	TargetInstance string `json:"targetInstance"`

	// Status
	Status        string `json:"status"` // "initiated", "in_progress", "completed", "failed"
	StatusMessage string `json:"statusMessage,omitempty"`

	// Timing
	StartTime metav1.Time   `json:"startTime"`
	EndTime   *metav1.Time  `json:"endTime,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	// Impact assessment
	ImpactAssessment ImpactAssessment `json:"impactAssessment"`
}

// ImpactAssessment represents the impact of a failover
type ImpactAssessment struct {
	// Affected workloads
	AffectedWorkloads int `json:"affectedWorkloads"`

	// Downtime duration
	DowntimeDuration time.Duration `json:"downtimeDuration"`

	// Data loss assessment
	DataLoss bool `json:"dataLoss"`

	// Performance impact
	PerformanceImpact string `json:"performanceImpact"` // "none", "minimal", "moderate", "significant"

	// Recovery time
	RecoveryTime time.Duration `json:"recoveryTime"`
}

// BackupRequest represents a backup request
type BackupRequest struct {
	// Request identification
	ID        string      `json:"id"`
	Timestamp metav1.Time `json:"timestamp"`

	// Backup details
	Type       string   `json:"type"` // "full", "incremental", "differential"
	Components []string `json:"components"`
	Namespaces []string `json:"namespaces,omitempty"`

	// Options
	Incremental bool `json:"incremental"`
	Compression bool `json:"compression"`
	Encryption  bool `json:"encryption"`

	// Metadata
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	// Trigger information
	TriggeredBy   string `json:"triggeredBy"` // "manual", "scheduled", "auto"
	TriggerReason string `json:"triggerReason,omitempty"`
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	// Request reference
	RequestID string `json:"requestID"`

	// Result identification
	BackupID  string      `json:"backupID"`
	Timestamp metav1.Time `json:"timestamp"`

	// Status
	Status        string `json:"status"` // "in_progress", "completed", "failed"
	StatusMessage string `json:"statusMessage,omitempty"`

	// Timing
	StartTime metav1.Time   `json:"startTime"`
	EndTime   *metav1.Time  `json:"endTime,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	// Backup information
	BackupSize  int64  `json:"backupSize"`
	StoragePath string `json:"storagePath"`
	Checksum    string `json:"checksum"`

	// Components backed up
	ComponentsBackedUp []string `json:"componentsBackedUp"`

	// Statistics
	FilesProcessed   int64   `json:"filesProcessed"`
	DataTransferred  int64   `json:"dataTransferred"`
	CompressionRatio float64 `json:"compressionRatio,omitempty"`

	// Verification
	Verified            bool   `json:"verified"`
	VerificationDetails string `json:"verificationDetails,omitempty"`
}

// RestoreRequest represents a restore request
type RestoreRequest struct {
	// Request identification
	ID        string      `json:"id"`
	Timestamp metav1.Time `json:"timestamp"`

	// Source backup
	BackupID    string `json:"backupID"`
	StoragePath string `json:"storagePath"`

	// Restore options
	Components        []string     `json:"components,omitempty"`
	Namespaces        []string     `json:"namespaces,omitempty"`
	PointInTime       *metav1.Time `json:"pointInTime,omitempty"`
	OverwriteExisting bool         `json:"overwriteExisting"`

	// Validation
	ValidateBeforeRestore bool `json:"validateBeforeRestore"`
	DryRun                bool `json:"dryRun"`

	// Metadata
	Description string `json:"description,omitempty"`
	RequestedBy string `json:"requestedBy"`
}

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	// Request reference
	RequestID string `json:"requestID"`

	// Result identification
	RestoreID string      `json:"restoreID"`
	Timestamp metav1.Time `json:"timestamp"`

	// Status
	Status        string `json:"status"` // "in_progress", "completed", "failed", "partially_completed"
	StatusMessage string `json:"statusMessage,omitempty"`

	// Timing
	StartTime metav1.Time   `json:"startTime"`
	EndTime   *metav1.Time  `json:"endTime,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	// Restore information
	ComponentsRestored []string `json:"componentsRestored"`
	ComponentsFailed   []string `json:"componentsFailed,omitempty"`

	// Statistics
	FilesRestored     int64 `json:"filesRestored"`
	DataRestored      int64 `json:"dataRestored"`
	ErrorsEncountered int   `json:"errorsEncountered"`

	// Validation results
	ValidationResults []ValidationResult `json:"validationResults,omitempty"`
}

// ValidationResult represents validation result for a component
type ValidationResult struct {
	Component string                 `json:"component"`
	Valid     bool                   `json:"valid"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// RecoveryRequest represents a recovery request
type RecoveryRequest struct {
	// Request identification
	ID        string      `json:"id"`
	Timestamp metav1.Time `json:"timestamp"`

	// Component to recover
	ComponentName string `json:"componentName"`
	ComponentType string `json:"componentType"`

	// Recovery strategy
	Strategy string `json:"strategy"` // "restart", "restore", "recreate", "failover"

	// Options
	Options RecoveryOptions `json:"options"`

	// Trigger information
	TriggeredBy string `json:"triggeredBy"` // "manual", "auto", "failover"
	Reason      string `json:"reason"`
}

// RecoveryOptions defines recovery options
type RecoveryOptions struct {
	// Force recovery even if component appears healthy
	Force bool `json:"force"`

	// Maximum retry attempts
	MaxRetries int `json:"maxRetries"`

	// Timeout for recovery operation
	Timeout time.Duration `json:"timeout"`

	// Backup to restore from
	RestoreFromBackup string `json:"restoreFromBackup,omitempty"`

	// Preserve data during recovery
	PreserveData bool `json:"preserveData"`

	// Validation before marking recovery complete
	ValidateAfterRecovery bool `json:"validateAfterRecovery"`
}

// RecoveryResult represents the result of a recovery operation
type RecoveryResult struct {
	// Request reference
	RequestID string `json:"requestID"`

	// Result identification
	RecoveryID string      `json:"recoveryID"`
	Timestamp  metav1.Time `json:"timestamp"`

	// Status
	Status        string `json:"status"` // "in_progress", "completed", "failed"
	StatusMessage string `json:"statusMessage,omitempty"`

	// Timing
	StartTime metav1.Time   `json:"startTime"`
	EndTime   *metav1.Time  `json:"endTime,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`

	// Recovery steps
	StepsCompleted []RecoveryStep `json:"stepsCompleted"`
	CurrentStep    *RecoveryStep  `json:"currentStep,omitempty"`

	// Retry information
	RetryCount int `json:"retryCount"`

	// Final component status
	FinalComponentStatus ComponentStatus `json:"finalComponentStatus"`

	// Recovery assessment
	RecoveryAssessment RecoveryAssessment `json:"recoveryAssessment"`
}

// RecoveryStep represents a step in the recovery process
type RecoveryStep struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      string        `json:"status"` // "pending", "in_progress", "completed", "failed", "skipped"
	StartTime   metav1.Time   `json:"startTime"`
	EndTime     *metav1.Time  `json:"endTime,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Message     string        `json:"message,omitempty"`
}

// RecoveryAssessment represents the assessment of recovery success
type RecoveryAssessment struct {
	// Overall success
	Successful bool `json:"successful"`

	// Recovery completeness (0-100)
	Completeness float64 `json:"completeness"`

	// Data integrity
	DataIntegrityCheck  bool   `json:"dataIntegrityCheck"`
	DataIntegrityStatus string `json:"dataIntegrityStatus"`

	// Performance impact
	PerformanceImpact string `json:"performanceImpact"`

	// Recommendations
	Recommendations []string `json:"recommendations,omitempty"`

	// Follow-up actions required
	FollowUpRequired bool     `json:"followUpRequired"`
	FollowUpActions  []string `json:"followUpActions,omitempty"`
}
