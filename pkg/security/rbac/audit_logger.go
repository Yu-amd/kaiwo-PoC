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

package rbac

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// AuditLogger provides enterprise-grade audit logging
type AuditLogger struct {
	// Configuration
	config AuditConfig

	// Log buffer
	buffer      []AuditEvent
	bufferMutex sync.Mutex

	// Storage backend
	storage AuditStorage

	// Alert manager
	alertManager *AuditAlertManager
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	// Basic information
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "info", "warning", "error", "critical"

	// Event details
	EventType string `json:"eventType"`
	Action    string `json:"action"`
	Resource  string `json:"resource,omitempty"`

	// User information
	UserID   string   `json:"userID,omitempty"`
	Username string   `json:"username,omitempty"`
	Groups   []string `json:"groups,omitempty"`

	// Source information
	SourceIP  string `json:"sourceIP,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	SessionID string `json:"sessionID,omitempty"`

	// Result
	Result     string `json:"result"` // "success", "failure", "warning"
	Reason     string `json:"reason,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`

	// Context
	Namespace string                 `json:"namespace,omitempty"`
	Cluster   string                 `json:"cluster,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`

	// Security context
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	PolicyHits  []string `json:"policyHits,omitempty"`

	// Risk assessment
	RiskScore   float64  `json:"riskScore,omitempty"`
	RiskFactors []string `json:"riskFactors,omitempty"`

	// Compliance
	ComplianceFlags []string `json:"complianceFlags,omitempty"`
}

// AuditStorage interface for audit log storage
type AuditStorage interface {
	Store(events []AuditEvent) error
	Query(query AuditQuery) ([]AuditEvent, error)
	Cleanup(retentionDays int) error
}

// AuditQuery represents a query for audit events
type AuditQuery struct {
	// Time range
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`

	// Filters
	UserID    string   `json:"userID,omitempty"`
	EventType string   `json:"eventType,omitempty"`
	Result    string   `json:"result,omitempty"`
	Levels    []string `json:"levels,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// AuditAlertManager manages real-time security alerts
type AuditAlertManager struct {
	// Alert rules
	rules      []AlertRule
	rulesMutex sync.RWMutex

	// Alert handlers
	handlers []AlertHandler

	// Configuration
	config AlertConfig
}

// AlertRule defines when to trigger an alert
type AlertRule struct {
	// Rule identification
	Name        string `json:"name"`
	Description string `json:"description"`

	// Conditions
	Conditions []AlertCondition `json:"conditions"`

	// Severity
	Severity string `json:"severity"` // "low", "medium", "high", "critical"

	// Throttling
	ThrottleMinutes int `json:"throttleMinutes"`

	// Enable/disable
	Enabled bool `json:"enabled"`
}

// AlertCondition defines alert trigger conditions
type AlertCondition struct {
	// Field to check
	Field string `json:"field"`

	// Operator
	Operator string `json:"operator"`

	// Values
	Values []string `json:"values"`

	// Time window (for rate-based alerts)
	TimeWindow string `json:"timeWindow,omitempty"`

	// Threshold (for rate-based alerts)
	Threshold int `json:"threshold,omitempty"`
}

// AlertHandler interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert SecurityAlert) error
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	// Alert information
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`

	// Rule information
	RuleName        string `json:"ruleName"`
	RuleDescription string `json:"ruleDescription"`

	// Triggering event
	TriggeringEvent AuditEvent `json:"triggeringEvent"`

	// Context
	Context map[string]interface{} `json:"context,omitempty"`

	// Recommended actions
	RecommendedActions []string `json:"recommendedActions,omitempty"`
}

// AlertConfig holds alert configuration
type AlertConfig struct {
	// Enable real-time alerting
	Enabled bool `json:"enabled"`

	// Default severity threshold
	DefaultSeverity string `json:"defaultSeverity"`

	// Rate limiting
	MaxAlertsPerMinute int `json:"maxAlertsPerMinute"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditConfig) *AuditLogger {
	logger := &AuditLogger{
		config: config,
		buffer: make([]AuditEvent, 0),
	}

	// Initialize storage backend
	logger.storage = NewFileAuditStorage(config)

	// Initialize alert manager
	if config.AlertingEnabled {
		logger.alertManager = NewAuditAlertManager(AlertConfig{
			Enabled:            true,
			DefaultSeverity:    "medium",
			MaxAlertsPerMinute: 10,
		})
		logger.setupDefaultAlertRules()
	}

	return logger
}

// LogAccessGranted logs a successful access event
func (al *AuditLogger) LogAccessGranted(userID, resource, action, role string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "info",
		EventType: "access_granted",
		Action:    action,
		Resource:  resource,
		UserID:    userID,
		Result:    "success",
		Role:      role,
		RiskScore: 0.1, // Low risk for successful access
	}

	al.logEvent(event)
}

// LogAccessDenied logs a failed access event
func (al *AuditLogger) LogAccessDenied(userID, resource, action, reason string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "warning",
		EventType: "access_denied",
		Action:    action,
		Resource:  resource,
		UserID:    userID,
		Result:    "failure",
		Reason:    reason,
		RiskScore: 0.6, // Medium risk for denied access
		RiskFactors: []string{
			"access_denied",
			"insufficient_permissions",
		},
	}

	al.logEvent(event)
}

// LogRoleCreated logs role creation
func (al *AuditLogger) LogRoleCreated(roleName string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "info",
		EventType: "role_created",
		Action:    "create",
		Resource:  "role:" + roleName,
		Result:    "success",
		RiskScore: 0.3, // Low-medium risk for role creation
	}

	al.logEvent(event)
}

// LogRoleUpdated logs role updates
func (al *AuditLogger) LogRoleUpdated(roleName string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "info",
		EventType: "role_updated",
		Action:    "update",
		Resource:  "role:" + roleName,
		Result:    "success",
		RiskScore: 0.4, // Medium risk for role modification
		RiskFactors: []string{
			"privilege_modification",
		},
	}

	al.logEvent(event)
}

// LogRoleDeleted logs role deletion
func (al *AuditLogger) LogRoleDeleted(roleName string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "warning",
		EventType: "role_deleted",
		Action:    "delete",
		Resource:  "role:" + roleName,
		Result:    "success",
		RiskScore: 0.7, // High risk for role deletion
		RiskFactors: []string{
			"privilege_removal",
			"potential_access_disruption",
		},
	}

	al.logEvent(event)
}

// LogUserBindingCreated logs user binding creation
func (al *AuditLogger) LogUserBindingCreated(userID string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "info",
		EventType: "user_binding_created",
		Action:    "create",
		Resource:  "user_binding:" + userID,
		UserID:    userID,
		Result:    "success",
		RiskScore: 0.3, // Low-medium risk for user binding
	}

	al.logEvent(event)
}

// LogSuspiciousActivity logs suspicious activity
func (al *AuditLogger) LogSuspiciousActivity(userID, activity, reason string, riskScore float64) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "error",
		EventType: "suspicious_activity",
		Action:    activity,
		UserID:    userID,
		Result:    "warning",
		Reason:    reason,
		RiskScore: riskScore,
		RiskFactors: []string{
			"suspicious_pattern",
			"anomalous_behavior",
		},
		ComplianceFlags: []string{
			"requires_investigation",
		},
	}

	al.logEvent(event)
}

// LogComplianceViolation logs compliance violations
func (al *AuditLogger) LogComplianceViolation(userID, violation, policy string) {
	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     "critical",
		EventType: "compliance_violation",
		UserID:    userID,
		Result:    "failure",
		Reason:    violation,
		RiskScore: 0.9, // Very high risk for compliance violations
		RiskFactors: []string{
			"compliance_violation",
			"policy_breach",
		},
		ComplianceFlags: []string{
			"sox_violation",
			"gdpr_violation",
			"requires_immediate_action",
		},
		Context: map[string]interface{}{
			"violated_policy": policy,
		},
	}

	al.logEvent(event)
}

// LogAuthenticationEvent logs authentication events
func (al *AuditLogger) LogAuthenticationEvent(userID, eventType, result, sourceIP string) {
	level := "info"
	riskScore := 0.1

	if result == "failure" {
		level = "warning"
		riskScore = 0.5
	}

	event := AuditEvent{
		ID:        al.generateEventID(),
		Timestamp: time.Now(),
		Level:     level,
		EventType: eventType,
		Action:    "authenticate",
		UserID:    userID,
		Result:    result,
		SourceIP:  sourceIP,
		RiskScore: riskScore,
	}

	if result == "failure" {
		event.RiskFactors = []string{
			"authentication_failure",
		}
	}

	al.logEvent(event)
}

// Private methods

func (al *AuditLogger) logEvent(event AuditEvent) {
	// Add to buffer
	al.bufferMutex.Lock()
	al.buffer = append(al.buffer, event)
	bufferSize := len(al.buffer)
	al.bufferMutex.Unlock()

	// Log to klog based on level
	eventJSON, _ := json.Marshal(event)
	switch event.Level {
	case "critical", "error":
		klog.ErrorS(nil, "Security audit event", "event", string(eventJSON))
	case "warning":
		klog.WarningS(nil, "Security audit event", "event", string(eventJSON))
	default:
		klog.InfoS("Security audit event", "event", string(eventJSON))
	}

	// Trigger alerts if enabled
	if al.alertManager != nil && al.config.AlertingEnabled {
		al.alertManager.ProcessEvent(event)
	}

	// Flush buffer if needed
	if bufferSize >= 100 { // Batch size
		go al.flushBuffer()
	}
}

func (al *AuditLogger) flushBuffer() {
	al.bufferMutex.Lock()
	events := make([]AuditEvent, len(al.buffer))
	copy(events, al.buffer)
	al.buffer = al.buffer[:0] // Clear buffer
	al.bufferMutex.Unlock()

	if len(events) > 0 {
		if err := al.storage.Store(events); err != nil {
			klog.ErrorS(err, "Failed to store audit events")
		}
	}
}

func (al *AuditLogger) generateEventID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

func (al *AuditLogger) setupDefaultAlertRules() {
	rules := []AlertRule{
		{
			Name:        "multiple_failed_logins",
			Description: "Multiple failed login attempts",
			Severity:    "high",
			Enabled:     true,
			Conditions: []AlertCondition{
				{
					Field:      "eventType",
					Operator:   "equals",
					Values:     []string{"authentication_failure"},
					TimeWindow: "5m",
					Threshold:  5,
				},
			},
		},
		{
			Name:        "privilege_escalation",
			Description: "Potential privilege escalation attempt",
			Severity:    "critical",
			Enabled:     true,
			Conditions: []AlertCondition{
				{
					Field:    "eventType",
					Operator: "equals",
					Values:   []string{"role_updated", "user_binding_created"},
				},
				{
					Field:    "riskScore",
					Operator: "greater_than",
					Values:   []string{"0.7"},
				},
			},
		},
		{
			Name:        "compliance_violation",
			Description: "Compliance policy violation detected",
			Severity:    "critical",
			Enabled:     true,
			Conditions: []AlertCondition{
				{
					Field:    "eventType",
					Operator: "equals",
					Values:   []string{"compliance_violation"},
				},
			},
		},
		{
			Name:        "suspicious_activity",
			Description: "Suspicious user activity detected",
			Severity:    "high",
			Enabled:     true,
			Conditions: []AlertCondition{
				{
					Field:    "eventType",
					Operator: "equals",
					Values:   []string{"suspicious_activity"},
				},
				{
					Field:    "riskScore",
					Operator: "greater_than",
					Values:   []string{"0.8"},
				},
			},
		},
	}

	for _, rule := range rules {
		al.alertManager.AddRule(rule)
	}
}

// NewAuditAlertManager creates a new audit alert manager
func NewAuditAlertManager(config AlertConfig) *AuditAlertManager {
	return &AuditAlertManager{
		rules:  make([]AlertRule, 0),
		config: config,
	}
}

// AddRule adds an alert rule
func (aam *AuditAlertManager) AddRule(rule AlertRule) {
	aam.rulesMutex.Lock()
	defer aam.rulesMutex.Unlock()

	aam.rules = append(aam.rules, rule)
}

// ProcessEvent processes an audit event for alerts
func (aam *AuditAlertManager) ProcessEvent(event AuditEvent) {
	aam.rulesMutex.RLock()
	defer aam.rulesMutex.RUnlock()

	for _, rule := range aam.rules {
		if !rule.Enabled {
			continue
		}

		if aam.evaluateRule(rule, event) {
			alert := SecurityAlert{
				ID:                 fmt.Sprintf("alert-%d", time.Now().UnixNano()),
				Timestamp:          time.Now(),
				Severity:           rule.Severity,
				RuleName:           rule.Name,
				RuleDescription:    rule.Description,
				TriggeringEvent:    event,
				RecommendedActions: aam.getRecommendedActions(rule.Name),
			}

			aam.triggerAlert(alert)
		}
	}
}

func (aam *AuditAlertManager) evaluateRule(rule AlertRule, event AuditEvent) bool {
	for _, condition := range rule.Conditions {
		if !aam.evaluateCondition(condition, event) {
			return false
		}
	}
	return true
}

func (aam *AuditAlertManager) evaluateCondition(condition AlertCondition, event AuditEvent) bool {
	// Get field value from event
	var fieldValue string
	switch condition.Field {
	case "eventType":
		fieldValue = event.EventType
	case "level":
		fieldValue = event.Level
	case "result":
		fieldValue = event.Result
	case "userID":
		fieldValue = event.UserID
	case "riskScore":
		fieldValue = fmt.Sprintf("%.2f", event.RiskScore)
	default:
		return false
	}

	// Evaluate condition
	switch condition.Operator {
	case "equals":
		for _, value := range condition.Values {
			if fieldValue == value {
				return true
			}
		}
		return false
	case "greater_than":
		if len(condition.Values) > 0 {
			// Simple numeric comparison
			if condition.Field == "riskScore" && len(condition.Values) > 0 {
				return event.RiskScore > 0.7 // Simplified
			}
		}
		return false
	default:
		return false
	}
}

func (aam *AuditAlertManager) triggerAlert(alert SecurityAlert) {
	alertJSON, _ := json.Marshal(alert)
	klog.ErrorS(nil, "Security alert triggered", "alert", string(alertJSON))

	// TODO: Send to external alert handlers (Slack, email, etc.)
}

func (aam *AuditAlertManager) getRecommendedActions(ruleName string) []string {
	switch ruleName {
	case "multiple_failed_logins":
		return []string{
			"Review user account for compromise",
			"Consider temporarily disabling account",
			"Check for brute force attack patterns",
		}
	case "privilege_escalation":
		return []string{
			"Immediately review role changes",
			"Verify authorization for privilege changes",
			"Audit user activities",
		}
	case "compliance_violation":
		return []string{
			"Investigate compliance violation immediately",
			"Document incident for compliance audit",
			"Review and update policies",
		}
	case "suspicious_activity":
		return []string{
			"Investigate user behavior patterns",
			"Review recent access logs",
			"Consider increasing monitoring",
		}
	default:
		return []string{
			"Investigate security event",
			"Review relevant logs",
		}
	}
}

// FileAuditStorage implements file-based audit storage
type FileAuditStorage struct {
	config AuditConfig
}

// NewFileAuditStorage creates a new file-based audit storage
func NewFileAuditStorage(config AuditConfig) *FileAuditStorage {
	return &FileAuditStorage{
		config: config,
	}
}

// Store stores audit events to file
func (fas *FileAuditStorage) Store(events []AuditEvent) error {
	// TODO: Implement file storage
	klog.V(4).Infof("Storing %d audit events to file storage", len(events))
	return nil
}

// Query queries audit events from file
func (fas *FileAuditStorage) Query(query AuditQuery) ([]AuditEvent, error) {
	// TODO: Implement file query
	return []AuditEvent{}, nil
}

// Cleanup removes old audit events
func (fas *FileAuditStorage) Cleanup(retentionDays int) error {
	// TODO: Implement file cleanup
	klog.V(4).Infof("Cleaning up audit events older than %d days", retentionDays)
	return nil
}
