package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/silogen/kaiwo/apis/kaiwo/v1alpha1"
)

// AlertManager implements intelligent alerting for KaiwoJobs
type AlertManager struct {
	client  client.Client
	mu      sync.RWMutex
	alerts  map[string]*Alert
	metrics *AlertManagerMetrics
	rules   []AlertRule
}

// Alert represents an alert condition
type Alert struct {
	ID         string
	JobName    string
	Namespace  string
	Type       AlertType
	Severity   AlertSeverity
	Message    string
	Timestamp  time.Time
	Resolved   bool
	ResolvedAt *time.Time
	Metrics    map[string]interface{}
}

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeHighCPUUsage           AlertType = "HighCPUUsage"
	AlertTypeHighMemoryUsage        AlertType = "HighMemoryUsage"
	AlertTypeHighGPUUsage           AlertType = "HighGPUUsage"
	AlertTypeJobFailure             AlertType = "JobFailure"
	AlertTypePodFailure             AlertType = "PodFailure"
	AlertTypeResourceExhaustion     AlertType = "ResourceExhaustion"
	AlertTypePerformanceDegradation AlertType = "PerformanceDegradation"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "Info"
	AlertSeverityWarning  AlertSeverity = "Warning"
	AlertSeverityCritical AlertSeverity = "Critical"
)

// AlertRule defines a rule for triggering alerts
type AlertRule struct {
	Type        AlertType
	Severity    AlertSeverity
	Threshold   float64
	Duration    time.Duration
	Description string
}

// AlertManagerMetrics tracks alert manager performance metrics
type AlertManagerMetrics struct {
	TotalAlerts      int64
	ActiveAlerts     int64
	ResolvedAlerts   int64
	AverageAlertTime time.Duration
	mu               sync.RWMutex
}

// NewAlertManager creates a new alert manager instance
func NewAlertManager(client client.Client) *AlertManager {
	am := &AlertManager{
		client: client,
		alerts: make(map[string]*Alert),
		metrics: &AlertManagerMetrics{
			TotalAlerts:    0,
			ActiveAlerts:   0,
			ResolvedAlerts: 0,
		},
		rules: make([]AlertRule, 0),
	}

	// Initialize default alert rules
	am.initializeDefaultRules()

	return am
}

// initializeDefaultRules sets up default alert rules
func (am *AlertManager) initializeDefaultRules() {
	am.rules = []AlertRule{
		{
			Type:        AlertTypeHighCPUUsage,
			Severity:    AlertSeverityWarning,
			Threshold:   0.9, // 90% CPU usage
			Duration:    5 * time.Minute,
			Description: "High CPU usage detected",
		},
		{
			Type:        AlertTypeHighMemoryUsage,
			Severity:    AlertSeverityWarning,
			Threshold:   0.9, // 90% memory usage
			Duration:    5 * time.Minute,
			Description: "High memory usage detected",
		},
		{
			Type:        AlertTypeHighGPUUsage,
			Severity:    AlertSeverityWarning,
			Threshold:   0.95, // 95% GPU usage
			Duration:    5 * time.Minute,
			Description: "High GPU usage detected",
		},
		{
			Type:        AlertTypeJobFailure,
			Severity:    AlertSeverityCritical,
			Threshold:   0.0, // Any failure
			Duration:    1 * time.Minute,
			Description: "Job failure detected",
		},
		{
			Type:        AlertTypePodFailure,
			Severity:    AlertSeverityWarning,
			Threshold:   0.5, // 50% pod failures
			Duration:    2 * time.Minute,
			Description: "High pod failure rate detected",
		},
		{
			Type:        AlertTypePerformanceDegradation,
			Severity:    AlertSeverityWarning,
			Threshold:   0.5, // 50% performance drop
			Duration:    10 * time.Minute,
			Description: "Performance degradation detected",
		},
	}
}

// CheckAlerts checks for alert conditions on a job
func (am *AlertManager) CheckAlerts(ctx context.Context, job *v1alpha1.KaiwoJob, metrics map[string]interface{}) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check each alert rule
	for _, rule := range am.rules {
		if am.shouldTriggerAlert(job, rule, metrics) {
			if err := am.createAlert(ctx, job, rule, metrics); err != nil {
				return fmt.Errorf("failed to create alert: %w", err)
			}
		}
	}

	// Check for resolved alerts
	am.checkResolvedAlerts(job, metrics)

	return nil
}

// shouldTriggerAlert determines if an alert should be triggered
func (am *AlertManager) shouldTriggerAlert(job *v1alpha1.KaiwoJob, rule AlertRule, metrics map[string]interface{}) bool {
	alertKey := fmt.Sprintf("%s-%s-%s", job.Namespace, job.Name, rule.Type)

	// Check if alert already exists and is active
	if existingAlert, exists := am.alerts[alertKey]; exists && !existingAlert.Resolved {
		return false
	}

	// Check threshold based on alert type
	switch rule.Type {
	case AlertTypeHighCPUUsage:
		if cpuUsage, ok := metrics["cpu_usage"].(float64); ok {
			return cpuUsage > rule.Threshold
		}
	case AlertTypeHighMemoryUsage:
		if memUsage, ok := metrics["memory_usage"].(float64); ok {
			return memUsage > rule.Threshold
		}
	case AlertTypeHighGPUUsage:
		if gpuUsage, ok := metrics["gpu_usage"].(float64); ok {
			return gpuUsage > rule.Threshold
		}
	case AlertTypeJobFailure:
		if job.Status.Status == v1alpha1.WorkloadStatusFailed {
			return true
		}
	case AlertTypePodFailure:
		if podFailures, ok := metrics["pod_failure_rate"].(float64); ok {
			return podFailures > rule.Threshold
		}
	case AlertTypePerformanceDegradation:
		if performance, ok := metrics["performance"].(float64); ok {
			return performance < rule.Threshold
		}
	}

	return false
}

// createAlert creates a new alert
func (am *AlertManager) createAlert(ctx context.Context, job *v1alpha1.KaiwoJob, rule AlertRule, metrics map[string]interface{}) error {
	alertKey := fmt.Sprintf("%s-%s-%s", job.Namespace, job.Name, rule.Type)

	alert := &Alert{
		ID:        alertKey,
		JobName:   job.Name,
		Namespace: job.Namespace,
		Type:      rule.Type,
		Severity:  rule.Severity,
		Message:   rule.Description,
		Timestamp: time.Now(),
		Resolved:  false,
		Metrics:   metrics,
	}

	am.alerts[alertKey] = alert

	// Update metrics
	am.metrics.mu.Lock()
	am.metrics.TotalAlerts++
	am.metrics.ActiveAlerts++
	am.metrics.mu.Unlock()

	// Log alert (in a real implementation, this would send notifications)
	fmt.Printf("ALERT: %s - %s - %s: %s\n", alert.Severity, alert.Type, alert.JobName, alert.Message)

	return nil
}

// checkResolvedAlerts checks if existing alerts should be resolved
func (am *AlertManager) checkResolvedAlerts(job *v1alpha1.KaiwoJob, metrics map[string]interface{}) {
	for _, alert := range am.alerts {
		if alert.JobName == job.Name && alert.Namespace == job.Namespace && !alert.Resolved {
			if am.isAlertResolved(alert, metrics) {
				am.resolveAlert(alert)
			}
		}
	}
}

// isAlertResolved determines if an alert should be resolved
func (am *AlertManager) isAlertResolved(alert *Alert, metrics map[string]interface{}) bool {
	switch alert.Type {
	case AlertTypeHighCPUUsage:
		if cpuUsage, ok := metrics["cpu_usage"].(float64); ok {
			return cpuUsage < 0.7 // Resolve when CPU usage drops below 70%
		}
	case AlertTypeHighMemoryUsage:
		if memUsage, ok := metrics["memory_usage"].(float64); ok {
			return memUsage < 0.7 // Resolve when memory usage drops below 70%
		}
	case AlertTypeHighGPUUsage:
		if gpuUsage, ok := metrics["gpu_usage"].(float64); ok {
			return gpuUsage < 0.8 // Resolve when GPU usage drops below 80%
		}
	case AlertTypeJobFailure:
		// Job failure alerts are resolved when job status changes
		return false // This would be handled by status change detection
	case AlertTypePodFailure:
		if podFailures, ok := metrics["pod_failure_rate"].(float64); ok {
			return podFailures < 0.2 // Resolve when failure rate drops below 20%
		}
	case AlertTypePerformanceDegradation:
		if performance, ok := metrics["performance"].(float64); ok {
			return performance > 0.8 // Resolve when performance improves above 80%
		}
	}

	return false
}

// resolveAlert marks an alert as resolved
func (am *AlertManager) resolveAlert(alert *Alert) {
	alert.Resolved = true
	now := time.Now()
	alert.ResolvedAt = &now

	// Update metrics
	am.metrics.mu.Lock()
	am.metrics.ActiveAlerts--
	am.metrics.ResolvedAlerts++
	am.metrics.mu.Unlock()

	// Log resolution
	fmt.Printf("RESOLVED: %s - %s - %s\n", alert.Severity, alert.Type, alert.JobName)
}

// GetAlerts returns all alerts for a job
func (am *AlertManager) GetAlerts(jobName, namespace string) ([]*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var jobAlerts []*Alert
	for _, alert := range am.alerts {
		if alert.JobName == jobName && alert.Namespace == namespace {
			jobAlerts = append(jobAlerts, alert)
		}
	}

	return jobAlerts, nil
}

// GetActiveAlerts returns all active (unresolved) alerts
func (am *AlertManager) GetActiveAlerts() ([]*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts, nil
}

// GetAllAlerts returns all alerts
func (am *AlertManager) GetAllAlerts() map[string]*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Return a copy to avoid race conditions
	allAlerts := make(map[string]*Alert)
	for k, v := range am.alerts {
		allAlerts[k] = v
	}

	return allAlerts
}

// AddAlertRule adds a custom alert rule
func (am *AlertManager) AddAlertRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.rules = append(am.rules, rule)
}

// RemoveAlertRule removes an alert rule
func (am *AlertManager) RemoveAlertRule(alertType AlertType) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, rule := range am.rules {
		if rule.Type == alertType {
			am.rules = append(am.rules[:i], am.rules[i+1:]...)
			break
		}
	}
}

// GetAlertRules returns all alert rules
func (am *AlertManager) GetAlertRules() []AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Return a copy to avoid race conditions
	rules := make([]AlertRule, len(am.rules))
	copy(rules, am.rules)

	return rules
}

// GetMetrics returns current alert manager metrics
func (am *AlertManager) GetMetrics() AlertManagerMetrics {
	am.metrics.mu.RLock()
	defer am.metrics.mu.RUnlock()

	return *am.metrics
}

// ClearResolvedAlerts removes all resolved alerts older than the specified duration
func (am *AlertManager) ClearResolvedAlerts(olderThan time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoffTime := time.Now().Add(-olderThan)
	for alertKey, alert := range am.alerts {
		if alert.Resolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoffTime) {
			delete(am.alerts, alertKey)
		}
	}
}
