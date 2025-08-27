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
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// PolicyEngine provides advanced policy evaluation and enforcement
type PolicyEngine struct {
	// Policy cache
	policyCache map[string]*Policy
	cacheMutex  sync.RWMutex

	// Evaluation metrics
	metrics *PolicyMetrics

	// Configuration
	config PolicyEngineConfig
}

// PolicyEngineConfig holds configuration for the policy engine
type PolicyEngineConfig struct {
	// Cache timeout
	CacheTimeout time.Duration

	// Maximum policy complexity
	MaxComplexity int

	// Enable performance monitoring
	EnableMetrics bool
}

// Policy represents a security policy
type Policy struct {
	// Basic information
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Rules
	Rules []PolicyRule `json:"rules"`

	// Enforcement mode
	EnforcementMode string `json:"enforcementMode"` // "enforce", "warn", "audit"

	// Status
	Enabled bool      `json:"enabled"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// PolicyRule defines a specific policy rule
type PolicyRule struct {
	// Rule identification
	Name        string `json:"name"`
	Description string `json:"description"`

	// Conditions
	Conditions []PolicyCondition `json:"conditions"`

	// Action to take when rule matches
	Action PolicyAction `json:"action"`

	// Priority (higher numbers take precedence)
	Priority int `json:"priority"`

	// Enable/disable this rule
	Enabled bool `json:"enabled"`
}

// PolicyCondition defines when a rule applies
type PolicyCondition struct {
	// Field to evaluate
	Field string `json:"field"`

	// Operator
	Operator string `json:"operator"`

	// Values to compare against
	Values []string `json:"values"`

	// Logical operator with next condition
	LogicalOperator string `json:"logicalOperator,omitempty"` // "AND", "OR"
}

// PolicyAction defines what action to take
type PolicyAction struct {
	// Action type
	Type string `json:"type"` // "allow", "deny", "modify", "warn"

	// Parameters for the action
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Message to display
	Message string `json:"message,omitempty"`
}

// PolicyMetrics tracks policy evaluation performance
type PolicyMetrics struct {
	// Total evaluations
	TotalEvaluations int64 `json:"totalEvaluations"`

	// Allowed actions
	AllowedActions int64 `json:"allowedActions"`

	// Denied actions
	DeniedActions int64 `json:"deniedActions"`

	// Average evaluation time
	AvgEvaluationTime time.Duration `json:"avgEvaluationTime"`

	// Policy hit counts
	PolicyHits map[string]int64 `json:"policyHits"`

	// Last reset time
	LastReset time.Time `json:"lastReset"`

	mutex sync.RWMutex
}

// PolicyEvaluationRequest represents a request for policy evaluation
type PolicyEvaluationRequest struct {
	// Context
	Context context.Context

	// User information
	UserID   string   `json:"userID"`
	Username string   `json:"username"`
	Groups   []string `json:"groups,omitempty"`

	// Resource information
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	Namespace    string                 `json:"namespace,omitempty"`
	ResourceName string                 `json:"resourceName,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`

	// Request metadata
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestID"`
}

// PolicyEvaluationResult represents the result of policy evaluation
type PolicyEvaluationResult struct {
	// Decision
	Decision string `json:"decision"` // "allow", "deny", "warn"

	// Reason for the decision
	Reason string `json:"reason"`

	// Matched policies
	MatchedPolicies []string `json:"matchedPolicies"`

	// Warnings
	Warnings []string `json:"warnings,omitempty"`

	// Modifications (if applicable)
	Modifications map[string]interface{} `json:"modifications,omitempty"`

	// Evaluation time
	EvaluationTime time.Duration `json:"evaluationTime"`
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	config := PolicyEngineConfig{
		CacheTimeout:  time.Hour,
		MaxComplexity: 100,
		EnableMetrics: true,
	}

	return &PolicyEngine{
		policyCache: make(map[string]*Policy),
		metrics:     NewPolicyMetrics(),
		config:      config,
	}
}

// NewPolicyMetrics creates a new policy metrics instance
func NewPolicyMetrics() *PolicyMetrics {
	return &PolicyMetrics{
		PolicyHits: make(map[string]int64),
		LastReset:  time.Now(),
	}
}

// EvaluatePolicy evaluates a request against all policies
func (pe *PolicyEngine) EvaluatePolicy(req *PolicyEvaluationRequest) *PolicyEvaluationResult {
	startTime := time.Now()

	result := &PolicyEvaluationResult{
		Decision:        "allow", // Default allow
		MatchedPolicies: make([]string, 0),
		Warnings:        make([]string, 0),
		Modifications:   make(map[string]interface{}),
	}

	pe.cacheMutex.RLock()
	policies := make([]*Policy, 0, len(pe.policyCache))
	for _, policy := range pe.policyCache {
		if policy.Enabled {
			policies = append(policies, policy)
		}
	}
	pe.cacheMutex.RUnlock()

	// Evaluate each policy
	for _, policy := range policies {
		policyResult := pe.evaluatePolicy(req, policy)

		if policyResult.Matched {
			result.MatchedPolicies = append(result.MatchedPolicies, policy.Name)

			// Update metrics
			if pe.config.EnableMetrics {
				pe.metrics.RecordPolicyHit(policy.Name)
			}

			// Apply policy action
			switch policyResult.Action.Type {
			case "deny":
				result.Decision = "deny"
				result.Reason = policyResult.Action.Message
				// Early exit on deny
				result.EvaluationTime = time.Since(startTime)
				pe.metrics.RecordEvaluation(result.Decision, result.EvaluationTime)
				return result

			case "warn":
				result.Warnings = append(result.Warnings, policyResult.Action.Message)

			case "modify":
				// Apply modifications
				for key, value := range policyResult.Action.Parameters {
					result.Modifications[key] = value
				}
			}
		}
	}

	result.EvaluationTime = time.Since(startTime)

	// Update metrics
	if pe.config.EnableMetrics {
		pe.metrics.RecordEvaluation(result.Decision, result.EvaluationTime)
	}

	return result
}

// PolicyEvaluationResult represents the result of evaluating a single policy
type policyEvaluationResult struct {
	Matched bool
	Action  PolicyAction
}

// evaluatePolicy evaluates a request against a specific policy
func (pe *PolicyEngine) evaluatePolicy(req *PolicyEvaluationRequest, policy *Policy) *policyEvaluationResult {
	// Sort rules by priority (highest first)
	rules := make([]PolicyRule, len(policy.Rules))
	copy(rules, policy.Rules)

	// Simple sort by priority
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	// Evaluate each rule
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if pe.evaluateRule(req, &rule) {
			return &policyEvaluationResult{
				Matched: true,
				Action:  rule.Action,
			}
		}
	}

	return &policyEvaluationResult{Matched: false}
}

// evaluateRule evaluates a request against a specific rule
func (pe *PolicyEngine) evaluateRule(req *PolicyEvaluationRequest, rule *PolicyRule) bool {
	if len(rule.Conditions) == 0 {
		return true // No conditions means always match
	}

	// Evaluate conditions
	result := true
	logicalOp := "AND" // Default logical operator

	for _, condition := range rule.Conditions {
		conditionResult := pe.evaluateCondition(req, &condition)

		// Apply logical operator
		if logicalOp == "AND" {
			result = result && conditionResult
		} else if logicalOp == "OR" {
			result = result || conditionResult
		}

		// Set next logical operator
		if condition.LogicalOperator != "" {
			logicalOp = condition.LogicalOperator
		}
	}

	return result
}

// evaluateCondition evaluates a single condition
func (pe *PolicyEngine) evaluateCondition(req *PolicyEvaluationRequest, condition *PolicyCondition) bool {
	// Get field value from request
	value := pe.getFieldValue(req, condition.Field)
	if value == nil {
		return false
	}

	strValue := fmt.Sprintf("%v", value)

	// Evaluate based on operator
	switch condition.Operator {
	case "equals":
		for _, condValue := range condition.Values {
			if strValue == condValue {
				return true
			}
		}
		return false

	case "not_equals":
		for _, condValue := range condition.Values {
			if strValue == condValue {
				return false
			}
		}
		return true

	case "contains":
		for _, condValue := range condition.Values {
			if len(strValue) >= len(condValue) {
				for i := 0; i <= len(strValue)-len(condValue); i++ {
					if strValue[i:i+len(condValue)] == condValue {
						return true
					}
				}
			}
		}
		return false

	case "in":
		for _, condValue := range condition.Values {
			if strValue == condValue {
				return true
			}
		}
		return false

	case "not_in":
		for _, condValue := range condition.Values {
			if strValue == condValue {
				return false
			}
		}
		return true

	case "starts_with":
		for _, condValue := range condition.Values {
			if len(strValue) >= len(condValue) && strValue[:len(condValue)] == condValue {
				return true
			}
		}
		return false

	case "ends_with":
		for _, condValue := range condition.Values {
			if len(strValue) >= len(condValue) && strValue[len(strValue)-len(condValue):] == condValue {
				return true
			}
		}
		return false

	case "regex":
		// TODO: Implement regex matching
		return false

	default:
		klog.Warningf("Unknown condition operator: %s", condition.Operator)
		return false
	}
}

// getFieldValue extracts a field value from the request
func (pe *PolicyEngine) getFieldValue(req *PolicyEvaluationRequest, field string) interface{} {
	switch field {
	case "user.id":
		return req.UserID
	case "user.name":
		return req.Username
	case "user.groups":
		return req.Groups
	case "resource":
		return req.Resource
	case "action":
		return req.Action
	case "namespace":
		return req.Namespace
	case "resource.name":
		return req.ResourceName
	case "timestamp":
		return req.Timestamp
	default:
		// Check attributes
		if req.Attributes != nil {
			if value, exists := req.Attributes[field]; exists {
				return value
			}
		}
		return nil
	}
}

// AddPolicy adds a new policy to the engine
func (pe *PolicyEngine) AddPolicy(policy *Policy) error {
	pe.cacheMutex.Lock()
	defer pe.cacheMutex.Unlock()

	if policy.Name == "" {
		return fmt.Errorf("policy name cannot be empty")
	}

	// Validate policy
	if err := pe.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %v", err)
	}

	// Set timestamps
	now := time.Now()
	policy.Created = now
	policy.Updated = now
	policy.Enabled = true

	// Store policy
	pe.policyCache[policy.Name] = policy

	klog.Infof("Added policy %s to engine", policy.Name)
	return nil
}

// RemovePolicy removes a policy from the engine
func (pe *PolicyEngine) RemovePolicy(policyName string) error {
	pe.cacheMutex.Lock()
	defer pe.cacheMutex.Unlock()

	if _, exists := pe.policyCache[policyName]; !exists {
		return fmt.Errorf("policy %s not found", policyName)
	}

	delete(pe.policyCache, policyName)

	klog.Infof("Removed policy %s from engine", policyName)
	return nil
}

// GetMetrics returns current policy metrics
func (pe *PolicyEngine) GetMetrics() *PolicyMetrics {
	pe.metrics.mutex.RLock()
	defer pe.metrics.mutex.RUnlock()

	// Return a copy
	metricsCopy := *pe.metrics
	metricsCopy.PolicyHits = make(map[string]int64)
	for k, v := range pe.metrics.PolicyHits {
		metricsCopy.PolicyHits[k] = v
	}

	return &metricsCopy
}

// ResetMetrics resets all policy metrics
func (pe *PolicyEngine) ResetMetrics() {
	pe.metrics.mutex.Lock()
	defer pe.metrics.mutex.Unlock()

	pe.metrics.TotalEvaluations = 0
	pe.metrics.AllowedActions = 0
	pe.metrics.DeniedActions = 0
	pe.metrics.AvgEvaluationTime = 0
	pe.metrics.PolicyHits = make(map[string]int64)
	pe.metrics.LastReset = time.Now()
}

// validatePolicy validates a policy
func (pe *PolicyEngine) validatePolicy(policy *Policy) error {
	if len(policy.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}

	// Check complexity
	complexity := len(policy.Rules)
	for _, rule := range policy.Rules {
		complexity += len(rule.Conditions)
	}

	if complexity > pe.config.MaxComplexity {
		return fmt.Errorf("policy complexity (%d) exceeds maximum (%d)", complexity, pe.config.MaxComplexity)
	}

	// Validate rules
	for _, rule := range policy.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule name cannot be empty")
		}

		if rule.Action.Type == "" {
			return fmt.Errorf("rule action type cannot be empty")
		}

		// Validate action type
		validActions := []string{"allow", "deny", "modify", "warn"}
		validAction := false
		for _, valid := range validActions {
			if rule.Action.Type == valid {
				validAction = true
				break
			}
		}
		if !validAction {
			return fmt.Errorf("invalid action type: %s", rule.Action.Type)
		}
	}

	return nil
}

// RecordEvaluation records an evaluation in metrics
func (pm *PolicyMetrics) RecordEvaluation(decision string, evaluationTime time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.TotalEvaluations++

	switch decision {
	case "allow":
		pm.AllowedActions++
	case "deny":
		pm.DeniedActions++
	}

	// Update average evaluation time
	if pm.TotalEvaluations == 1 {
		pm.AvgEvaluationTime = evaluationTime
	} else {
		total := time.Duration(pm.TotalEvaluations-1) * pm.AvgEvaluationTime
		pm.AvgEvaluationTime = (total + evaluationTime) / time.Duration(pm.TotalEvaluations)
	}
}

// RecordPolicyHit records a policy hit in metrics
func (pm *PolicyMetrics) RecordPolicyHit(policyName string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.PolicyHits[policyName]++
}
