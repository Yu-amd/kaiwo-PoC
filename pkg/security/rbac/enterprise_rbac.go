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

// EnterpriseRBAC provides advanced role-based access control
type EnterpriseRBAC struct {
	// Role definitions
	roles     map[string]*Role
	roleMutex sync.RWMutex

	// User bindings
	userBindings map[string]*UserBinding
	userMutex    sync.RWMutex

	// Policy engine
	policyEngine *PolicyEngine

	// Audit logging
	auditLogger *AuditLogger

	// Session management
	sessionManager *SessionManager

	// Configuration
	config RBACConfig
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	// Session timeout
	SessionTimeout time.Duration

	// Password policy
	PasswordPolicy PasswordPolicy

	// Multi-factor authentication
	MFARequired bool

	// Audit configuration
	AuditConfig AuditConfig

	// Integration settings
	LDAPConfig *LDAPConfig
	OIDCConfig *OIDCConfig
}

// Role represents an enterprise role
type Role struct {
	// Basic information
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Permissions
	Permissions []Permission `json:"permissions"`

	// Resource constraints
	ResourceConstraints ResourceConstraints `json:"resourceConstraints"`

	// Time-based access
	TimeRestrictions []TimeRestriction `json:"timeRestrictions,omitempty"`

	// Inheritance
	InheritsFrom []string `json:"inheritsFrom,omitempty"`

	// Status
	Enabled bool      `json:"enabled"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// Permission defines what actions are allowed
type Permission struct {
	// Resource type (e.g., "kaiwojobs", "clusters", "policies")
	Resource string `json:"resource"`

	// Allowed verbs (e.g., "get", "create", "update", "delete")
	Verbs []string `json:"verbs"`

	// Resource names (specific instances, optional)
	ResourceNames []string `json:"resourceNames,omitempty"`

	// Namespace restrictions
	Namespaces []string `json:"namespaces,omitempty"`

	// Attribute-based conditions
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents an attribute-based access control condition
type Condition struct {
	// Field to check
	Field string `json:"field"`

	// Operator (equals, not_equals, in, not_in, greater_than, less_than)
	Operator string `json:"operator"`

	// Values to compare against
	Values []string `json:"values"`
}

// ResourceConstraints defines resource usage limits for a role
type ResourceConstraints struct {
	// Maximum CPU per workload
	MaxCPU string `json:"maxCPU,omitempty"`

	// Maximum memory per workload
	MaxMemory string `json:"maxMemory,omitempty"`

	// Maximum GPU per workload
	MaxGPU int32 `json:"maxGPU,omitempty"`

	// Maximum storage per workload
	MaxStorage string `json:"maxStorage,omitempty"`

	// Maximum concurrent workloads
	MaxWorkloads int32 `json:"maxWorkloads,omitempty"`

	// Allowed clusters
	AllowedClusters []string `json:"allowedClusters,omitempty"`

	// Allowed regions
	AllowedRegions []string `json:"allowedRegions,omitempty"`

	// Cost limits
	CostLimits CostLimits `json:"costLimits,omitempty"`
}

// CostLimits defines cost-based constraints
type CostLimits struct {
	// Daily cost limit (in USD)
	DailyLimit float64 `json:"dailyLimit,omitempty"`

	// Monthly cost limit (in USD)
	MonthlyLimit float64 `json:"monthlyLimit,omitempty"`

	// Per-workload cost limit (in USD)
	PerWorkloadLimit float64 `json:"perWorkloadLimit,omitempty"`
}

// TimeRestriction defines time-based access control
type TimeRestriction struct {
	// Days of week (0=Sunday, 6=Saturday)
	DaysOfWeek []int `json:"daysOfWeek"`

	// Start time (24-hour format, e.g., "09:00")
	StartTime string `json:"startTime"`

	// End time (24-hour format, e.g., "17:00")
	EndTime string `json:"endTime"`

	// Timezone
	Timezone string `json:"timezone"`
}

// UserBinding represents user-to-role binding
type UserBinding struct {
	// User information
	UserID     string            `json:"userID"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Groups     []string          `json:"groups,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`

	// Role assignments
	Roles []RoleAssignment `json:"roles"`

	// Status
	Enabled bool      `json:"enabled"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`

	// Security settings
	MFAEnabled      bool      `json:"mfaEnabled"`
	LastLogin       time.Time `json:"lastLogin,omitempty"`
	PasswordChanged time.Time `json:"passwordChanged,omitempty"`
}

// RoleAssignment represents a role assigned to a user
type RoleAssignment struct {
	// Role name
	Role string `json:"role"`

	// Scope (e.g., namespace, cluster)
	Scope string `json:"scope,omitempty"`

	// Conditions for this assignment
	Conditions []Condition `json:"conditions,omitempty"`

	// Expiration time
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// PasswordPolicy defines password requirements
type PasswordPolicy struct {
	// Minimum password length
	MinLength int `json:"minLength"`

	// Require uppercase characters
	RequireUppercase bool `json:"requireUppercase"`

	// Require lowercase characters
	RequireLowercase bool `json:"requireLowercase"`

	// Require numbers
	RequireNumbers bool `json:"requireNumbers"`

	// Require special characters
	RequireSpecialChars bool `json:"requireSpecialChars"`

	// Password expiration (in days)
	ExpirationDays int `json:"expirationDays"`

	// Password history (prevent reuse)
	HistoryCount int `json:"historyCount"`
}

// AuditConfig defines audit logging configuration
type AuditConfig struct {
	// Enable audit logging
	Enabled bool `json:"enabled"`

	// Audit levels
	Level string `json:"level"` // "minimal", "standard", "verbose"

	// Retention period (in days)
	RetentionDays int `json:"retentionDays"`

	// Storage backend
	StorageBackend string `json:"storageBackend"`

	// Real-time alerts
	AlertingEnabled bool `json:"alertingEnabled"`
}

// LDAPConfig defines LDAP integration
type LDAPConfig struct {
	// LDAP server URL
	ServerURL string `json:"serverURL"`

	// Bind DN
	BindDN string `json:"bindDN"`

	// Bind password
	BindPassword string `json:"bindPassword"`

	// User search base
	UserSearchBase string `json:"userSearchBase"`

	// Group search base
	GroupSearchBase string `json:"groupSearchBase"`

	// Attribute mappings
	Attributes LDAPAttributes `json:"attributes"`
}

// LDAPAttributes defines LDAP attribute mappings
type LDAPAttributes struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Groups   string `json:"groups"`
}

// OIDCConfig defines OpenID Connect integration
type OIDCConfig struct {
	// Issuer URL
	IssuerURL string `json:"issuerURL"`

	// Client ID
	ClientID string `json:"clientID"`

	// Client secret
	ClientSecret string `json:"clientSecret"`

	// Redirect URL
	RedirectURL string `json:"redirectURL"`

	// Scopes
	Scopes []string `json:"scopes"`
}

// NewEnterpriseRBAC creates a new enterprise RBAC manager
func NewEnterpriseRBAC(config RBACConfig) *EnterpriseRBAC {
	rbac := &EnterpriseRBAC{
		roles:        make(map[string]*Role),
		userBindings: make(map[string]*UserBinding),
		config:       config,
	}

	// Initialize components
	rbac.policyEngine = NewPolicyEngine()
	rbac.auditLogger = NewAuditLogger(config.AuditConfig)
	rbac.sessionManager = NewSessionManager(config.SessionTimeout)

	// Create default roles
	rbac.createDefaultRoles()

	return rbac
}

// CreateRole creates a new role
func (rbac *EnterpriseRBAC) CreateRole(role *Role) error {
	rbac.roleMutex.Lock()
	defer rbac.roleMutex.Unlock()

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	if _, exists := rbac.roles[role.Name]; exists {
		return fmt.Errorf("role %s already exists", role.Name)
	}

	// Validate role
	if err := rbac.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %v", err)
	}

	// Set timestamps
	now := time.Now()
	role.Created = now
	role.Updated = now
	role.Enabled = true

	// Store role
	rbac.roles[role.Name] = role

	// Audit log
	rbac.auditLogger.LogRoleCreated(role.Name)

	klog.Infof("Created role %s", role.Name)
	return nil
}

// UpdateRole updates an existing role
func (rbac *EnterpriseRBAC) UpdateRole(role *Role) error {
	rbac.roleMutex.Lock()
	defer rbac.roleMutex.Unlock()

	existing, exists := rbac.roles[role.Name]
	if !exists {
		return fmt.Errorf("role %s not found", role.Name)
	}

	// Validate updated role
	if err := rbac.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %v", err)
	}

	// Preserve creation time
	role.Created = existing.Created
	role.Updated = time.Now()

	// Update role
	rbac.roles[role.Name] = role

	// Audit log
	rbac.auditLogger.LogRoleUpdated(role.Name)

	klog.Infof("Updated role %s", role.Name)
	return nil
}

// DeleteRole deletes a role
func (rbac *EnterpriseRBAC) DeleteRole(roleName string) error {
	rbac.roleMutex.Lock()
	defer rbac.roleMutex.Unlock()

	if _, exists := rbac.roles[roleName]; !exists {
		return fmt.Errorf("role %s not found", roleName)
	}

	// Check if role is in use
	if rbac.isRoleInUse(roleName) {
		return fmt.Errorf("role %s is currently in use", roleName)
	}

	// Delete role
	delete(rbac.roles, roleName)

	// Audit log
	rbac.auditLogger.LogRoleDeleted(roleName)

	klog.Infof("Deleted role %s", roleName)
	return nil
}

// GetRole returns a specific role
func (rbac *EnterpriseRBAC) GetRole(roleName string) (*Role, error) {
	rbac.roleMutex.RLock()
	defer rbac.roleMutex.RUnlock()

	role, exists := rbac.roles[roleName]
	if !exists {
		return nil, fmt.Errorf("role %s not found", roleName)
	}

	// Return a copy
	roleCopy := *role
	return &roleCopy, nil
}

// CreateUserBinding creates a user-to-role binding
func (rbac *EnterpriseRBAC) CreateUserBinding(binding *UserBinding) error {
	rbac.userMutex.Lock()
	defer rbac.userMutex.Unlock()

	if binding.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if _, exists := rbac.userBindings[binding.UserID]; exists {
		return fmt.Errorf("user binding for %s already exists", binding.UserID)
	}

	// Validate role assignments
	for _, assignment := range binding.Roles {
		if _, exists := rbac.roles[assignment.Role]; !exists {
			return fmt.Errorf("role %s not found", assignment.Role)
		}
	}

	// Set timestamps
	now := time.Now()
	binding.Created = now
	binding.Updated = now
	binding.Enabled = true

	// Store binding
	rbac.userBindings[binding.UserID] = binding

	// Audit log
	rbac.auditLogger.LogUserBindingCreated(binding.UserID)

	klog.Infof("Created user binding for %s", binding.UserID)
	return nil
}

// CheckPermission checks if a user has permission for a specific action
func (rbac *EnterpriseRBAC) CheckPermission(ctx context.Context, userID, resource, verb string, attributes map[string]interface{}) (bool, error) {
	rbac.userMutex.RLock()
	defer rbac.userMutex.RUnlock()

	// Get user binding
	binding, exists := rbac.userBindings[userID]
	if !exists || !binding.Enabled {
		rbac.auditLogger.LogAccessDenied(userID, resource, verb, "user not found or disabled")
		return false, nil
	}

	// Check each role assignment
	for _, assignment := range binding.Roles {
		// Check if assignment is expired
		if assignment.ExpiresAt != nil && time.Now().After(*assignment.ExpiresAt) {
			continue
		}

		// Get role
		role, exists := rbac.roles[assignment.Role]
		if !exists || !role.Enabled {
			continue
		}

		// Check time restrictions
		if !rbac.checkTimeRestrictions(role.TimeRestrictions) {
			continue
		}

		// Check permissions
		if rbac.checkRolePermission(role, resource, verb, attributes) {
			rbac.auditLogger.LogAccessGranted(userID, resource, verb, assignment.Role)
			return true, nil
		}
	}

	rbac.auditLogger.LogAccessDenied(userID, resource, verb, "insufficient permissions")
	return false, nil
}

// Private helper methods

func (rbac *EnterpriseRBAC) createDefaultRoles() {
	// Cluster Administrator
	adminRole := &Role{
		Name:        "cluster-admin",
		Description: "Full cluster administration access",
		Permissions: []Permission{
			{
				Resource: "*",
				Verbs:    []string{"*"},
			},
		},
		Enabled: true,
	}
	rbac.roles[adminRole.Name] = adminRole

	// ML Engineer
	mlEngineerRole := &Role{
		Name:        "ml-engineer",
		Description: "ML workload management and analytics access",
		Permissions: []Permission{
			{
				Resource: "kaiwojobs",
				Verbs:    []string{"get", "create", "update", "delete", "list"},
			},
			{
				Resource: "analytics",
				Verbs:    []string{"get", "list"},
			},
			{
				Resource: "predictions",
				Verbs:    []string{"get", "create", "list"},
			},
		},
		ResourceConstraints: ResourceConstraints{
			MaxGPU:       8,
			MaxWorkloads: 50,
			CostLimits: CostLimits{
				DailyLimit:       1000.0,
				MonthlyLimit:     30000.0,
				PerWorkloadLimit: 500.0,
			},
		},
		Enabled: true,
	}
	rbac.roles[mlEngineerRole.Name] = mlEngineerRole

	// Data Scientist
	dataScientistRole := &Role{
		Name:        "data-scientist",
		Description: "Data science workload access with limited resources",
		Permissions: []Permission{
			{
				Resource: "kaiwojobs",
				Verbs:    []string{"get", "create", "update", "list"},
				Conditions: []Condition{
					{
						Field:    "spec.resources.gpu",
						Operator: "less_than",
						Values:   []string{"4"},
					},
				},
			},
			{
				Resource: "analytics",
				Verbs:    []string{"get", "list"},
			},
		},
		ResourceConstraints: ResourceConstraints{
			MaxGPU:       2,
			MaxWorkloads: 10,
			CostLimits: CostLimits{
				DailyLimit:       200.0,
				MonthlyLimit:     5000.0,
				PerWorkloadLimit: 100.0,
			},
		},
		TimeRestrictions: []TimeRestriction{
			{
				DaysOfWeek: []int{1, 2, 3, 4, 5}, // Monday to Friday
				StartTime:  "08:00",
				EndTime:    "18:00",
				Timezone:   "UTC",
			},
		},
		Enabled: true,
	}
	rbac.roles[dataScientistRole.Name] = dataScientistRole

	// Viewer
	viewerRole := &Role{
		Name:        "viewer",
		Description: "Read-only access to resources",
		Permissions: []Permission{
			{
				Resource: "*",
				Verbs:    []string{"get", "list"},
			},
		},
		Enabled: true,
	}
	rbac.roles[viewerRole.Name] = viewerRole
}

func (rbac *EnterpriseRBAC) validateRole(role *Role) error {
	if len(role.Permissions) == 0 {
		return fmt.Errorf("role must have at least one permission")
	}

	// Validate permissions
	for _, permission := range role.Permissions {
		if permission.Resource == "" {
			return fmt.Errorf("permission resource cannot be empty")
		}
		if len(permission.Verbs) == 0 {
			return fmt.Errorf("permission must have at least one verb")
		}
	}

	// Validate inheritance
	for _, parentRole := range role.InheritsFrom {
		if _, exists := rbac.roles[parentRole]; !exists {
			return fmt.Errorf("parent role %s not found", parentRole)
		}
	}

	return nil
}

func (rbac *EnterpriseRBAC) isRoleInUse(roleName string) bool {
	rbac.userMutex.RLock()
	defer rbac.userMutex.RUnlock()

	for _, binding := range rbac.userBindings {
		for _, assignment := range binding.Roles {
			if assignment.Role == roleName {
				return true
			}
		}
	}

	return false
}

func (rbac *EnterpriseRBAC) checkTimeRestrictions(restrictions []TimeRestriction) bool {
	if len(restrictions) == 0 {
		return true // No restrictions
	}

	now := time.Now()

	for _, restriction := range restrictions {
		// Check day of week
		weekday := int(now.Weekday())
		dayAllowed := false
		for _, allowedDay := range restriction.DaysOfWeek {
			if weekday == allowedDay {
				dayAllowed = true
				break
			}
		}

		if !dayAllowed {
			continue
		}

		// Check time range
		currentTime := now.Format("15:04")
		if currentTime >= restriction.StartTime && currentTime <= restriction.EndTime {
			return true
		}
	}

	return false
}

func (rbac *EnterpriseRBAC) checkRolePermission(role *Role, resource, verb string, attributes map[string]interface{}) bool {
	for _, permission := range role.Permissions {
		// Check resource match
		if permission.Resource != "*" && permission.Resource != resource {
			continue
		}

		// Check verb match
		verbMatched := false
		for _, allowedVerb := range permission.Verbs {
			if allowedVerb == "*" || allowedVerb == verb {
				verbMatched = true
				break
			}
		}

		if !verbMatched {
			continue
		}

		// Check conditions
		if rbac.checkConditions(permission.Conditions, attributes) {
			return true
		}
	}

	return false
}

func (rbac *EnterpriseRBAC) checkConditions(conditions []Condition, attributes map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true // No conditions to check
	}

	for _, condition := range conditions {
		if !rbac.evaluateCondition(condition, attributes) {
			return false
		}
	}

	return true
}

func (rbac *EnterpriseRBAC) evaluateCondition(condition Condition, attributes map[string]interface{}) bool {
	// Get attribute value
	value, exists := attributes[condition.Field]
	if !exists {
		return false
	}

	// Convert to string for comparison
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

	// TODO: Implement numeric comparisons for greater_than, less_than

	default:
		return false
	}
}
