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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// SessionManager manages user sessions with enterprise features
type SessionManager struct {
	// Active sessions
	sessions     map[string]*Session
	sessionMutex sync.RWMutex

	// Session timeout
	timeout time.Duration

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCh        chan struct{}

	// Configuration
	config SessionConfig
}

// SessionConfig holds session configuration
type SessionConfig struct {
	// Session timeout
	Timeout time.Duration

	// Maximum concurrent sessions per user
	MaxSessionsPerUser int

	// Session refresh interval
	RefreshInterval time.Duration

	// Enable session monitoring
	EnableMonitoring bool

	// Session cookie settings
	CookieSettings CookieSettings
}

// CookieSettings holds cookie configuration
type CookieSettings struct {
	// Cookie name
	Name string

	// Secure flag
	Secure bool

	// HTTP only flag
	HTTPOnly bool

	// Same site policy
	SameSite string

	// Domain
	Domain string

	// Path
	Path string
}

// Session represents a user session
type Session struct {
	// Session identification
	ID       string `json:"id"`
	UserID   string `json:"userID"`
	Username string `json:"username"`

	// Session lifecycle
	CreatedAt  time.Time `json:"createdAt"`
	LastAccess time.Time `json:"lastAccess"`
	ExpiresAt  time.Time `json:"expiresAt"`

	// Session metadata
	SourceIP   string            `json:"sourceIP"`
	UserAgent  string            `json:"userAgent"`
	Attributes map[string]string `json:"attributes,omitempty"`

	// Security features
	MFAVerified bool     `json:"mfaVerified"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`

	// Session monitoring
	ActivityLog []SessionActivity `json:"activityLog,omitempty"`
	RiskScore   float64           `json:"riskScore"`

	// Status
	Active bool `json:"active"`
}

// SessionActivity represents activity within a session
type SessionActivity struct {
	// Activity details
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource,omitempty"`
	Result    string    `json:"result"`

	// Context
	SourceIP  string `json:"sourceIP,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`

	// Risk assessment
	RiskFactors []string `json:"riskFactors,omitempty"`
}

// SessionInfo represents session information for API responses
type SessionInfo struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userID"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"createdAt"`
	LastAccess time.Time `json:"lastAccess"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Active     bool      `json:"active"`
	RiskScore  float64   `json:"riskScore"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(timeout time.Duration) *SessionManager {
	config := SessionConfig{
		Timeout:            timeout,
		MaxSessionsPerUser: 5,
		RefreshInterval:    time.Hour,
		EnableMonitoring:   true,
		CookieSettings: CookieSettings{
			Name:     "kaiwo-session",
			Secure:   true,
			HTTPOnly: true,
			SameSite: "Strict",
			Path:     "/",
		},
	}

	sm := &SessionManager{
		sessions: make(map[string]*Session),
		timeout:  timeout,
		config:   config,
		stopCh:   make(chan struct{}),
	}

	// Start cleanup routine
	sm.cleanupTicker = time.NewTicker(time.Minute * 15) // Cleanup every 15 minutes
	go sm.runCleanup()

	return sm
}

// CreateSession creates a new user session
func (sm *SessionManager) CreateSession(userID, username, sourceIP, userAgent string) (*Session, error) {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	// Check concurrent session limit
	userSessions := sm.getUserSessions(userID)
	if len(userSessions) >= sm.config.MaxSessionsPerUser {
		// Remove oldest session
		sm.removeOldestUserSession(userID)
	}

	// Generate session ID
	sessionID, err := sm.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Create session
	now := time.Now()
	session := &Session{
		ID:          sessionID,
		UserID:      userID,
		Username:    username,
		CreatedAt:   now,
		LastAccess:  now,
		ExpiresAt:   now.Add(sm.timeout),
		SourceIP:    sourceIP,
		UserAgent:   userAgent,
		Attributes:  make(map[string]string),
		MFAVerified: false,
		Roles:       make([]string, 0),
		Permissions: make([]string, 0),
		ActivityLog: make([]SessionActivity, 0),
		RiskScore:   sm.calculateInitialRiskScore(sourceIP, userAgent),
		Active:      true,
	}

	// Store session
	sm.sessions[sessionID] = session

	klog.Infof("Created session %s for user %s", sessionID, userID)
	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if !session.Active {
		return nil, fmt.Errorf("session is inactive")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Return a copy to prevent external modifications
	sessionCopy := *session
	return &sessionCopy, nil
}

// RefreshSession refreshes a session's expiration time
func (sm *SessionManager) RefreshSession(sessionID string) error {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if !session.Active {
		return fmt.Errorf("session is inactive")
	}

	// Update session
	now := time.Now()
	session.LastAccess = now
	session.ExpiresAt = now.Add(sm.timeout)

	// Log activity
	if sm.config.EnableMonitoring {
		activity := SessionActivity{
			Timestamp: now,
			Action:    "session_refresh",
			Result:    "success",
		}
		session.ActivityLog = append(session.ActivityLog, activity)

		// Keep activity log manageable
		if len(session.ActivityLog) > 100 {
			session.ActivityLog = session.ActivityLog[len(session.ActivityLog)-100:]
		}
	}

	klog.V(4).Infof("Refreshed session %s", sessionID)
	return nil
}

// InvalidateSession invalidates a session
func (sm *SessionManager) InvalidateSession(sessionID string) error {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Mark as inactive
	session.Active = false

	// Log activity
	if sm.config.EnableMonitoring {
		activity := SessionActivity{
			Timestamp: time.Now(),
			Action:    "session_invalidated",
			Result:    "success",
		}
		session.ActivityLog = append(session.ActivityLog, activity)
	}

	klog.Infof("Invalidated session %s", sessionID)
	return nil
}

// InvalidateUserSessions invalidates all sessions for a user
func (sm *SessionManager) InvalidateUserSessions(userID string) error {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	count := 0
	for _, session := range sm.sessions {
		if session.UserID == userID && session.Active {
			session.Active = false

			// Log activity
			if sm.config.EnableMonitoring {
				activity := SessionActivity{
					Timestamp: time.Now(),
					Action:    "session_invalidated_by_admin",
					Result:    "success",
				}
				session.ActivityLog = append(session.ActivityLog, activity)
			}

			count++
		}
	}

	klog.Infof("Invalidated %d sessions for user %s", count, userID)
	return nil
}

// UpdateSessionActivity updates session activity
func (sm *SessionManager) UpdateSessionActivity(sessionID, action, resource, result string) error {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if !session.Active {
		return fmt.Errorf("session is inactive")
	}

	// Update last access
	session.LastAccess = time.Now()

	// Log activity
	if sm.config.EnableMonitoring {
		activity := SessionActivity{
			Timestamp: time.Now(),
			Action:    action,
			Resource:  resource,
			Result:    result,
		}

		// Add risk factors based on activity
		activity.RiskFactors = sm.assessActivityRisk(action, resource, result)

		session.ActivityLog = append(session.ActivityLog, activity)

		// Update session risk score
		session.RiskScore = sm.updateRiskScore(session, activity)

		// Keep activity log manageable
		if len(session.ActivityLog) > 100 {
			session.ActivityLog = session.ActivityLog[len(session.ActivityLog)-100:]
		}
	}

	return nil
}

// GetUserSessions returns all active sessions for a user
func (sm *SessionManager) GetUserSessions(userID string) []SessionInfo {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	var userSessions []SessionInfo
	for _, session := range sm.sessions {
		if session.UserID == userID && session.Active && time.Now().Before(session.ExpiresAt) {
			sessionInfo := SessionInfo{
				ID:         session.ID,
				UserID:     session.UserID,
				Username:   session.Username,
				CreatedAt:  session.CreatedAt,
				LastAccess: session.LastAccess,
				ExpiresAt:  session.ExpiresAt,
				Active:     session.Active,
				RiskScore:  session.RiskScore,
			}
			userSessions = append(userSessions, sessionInfo)
		}
	}

	return userSessions
}

// GetAllSessions returns all active sessions (admin function)
func (sm *SessionManager) GetAllSessions() []SessionInfo {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	var allSessions []SessionInfo
	for _, session := range sm.sessions {
		if session.Active && time.Now().Before(session.ExpiresAt) {
			sessionInfo := SessionInfo{
				ID:         session.ID,
				UserID:     session.UserID,
				Username:   session.Username,
				CreatedAt:  session.CreatedAt,
				LastAccess: session.LastAccess,
				ExpiresAt:  session.ExpiresAt,
				Active:     session.Active,
				RiskScore:  session.RiskScore,
			}
			allSessions = append(allSessions, sessionInfo)
		}
	}

	return allSessions
}

// GetSessionMetrics returns session metrics
func (sm *SessionManager) GetSessionMetrics() map[string]interface{} {
	sm.sessionMutex.RLock()
	defer sm.sessionMutex.RUnlock()

	now := time.Now()
	totalSessions := 0
	activeSessions := 0
	expiredSessions := 0
	highRiskSessions := 0

	for _, session := range sm.sessions {
		totalSessions++

		if session.Active {
			if now.Before(session.ExpiresAt) {
				activeSessions++

				if session.RiskScore > 0.7 {
					highRiskSessions++
				}
			} else {
				expiredSessions++
			}
		}
	}

	return map[string]interface{}{
		"total_sessions":     totalSessions,
		"active_sessions":    activeSessions,
		"expired_sessions":   expiredSessions,
		"high_risk_sessions": highRiskSessions,
	}
}

// Stop stops the session manager
func (sm *SessionManager) Stop() {
	close(sm.stopCh)
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
}

// Private helper methods

func (sm *SessionManager) generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (sm *SessionManager) getUserSessions(userID string) []*Session {
	var userSessions []*Session
	for _, session := range sm.sessions {
		if session.UserID == userID && session.Active {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

func (sm *SessionManager) removeOldestUserSession(userID string) {
	var oldestSession *Session
	var oldestSessionID string

	for id, session := range sm.sessions {
		if session.UserID == userID && session.Active {
			if oldestSession == nil || session.CreatedAt.Before(oldestSession.CreatedAt) {
				oldestSession = session
				oldestSessionID = id
			}
		}
	}

	if oldestSession != nil {
		oldestSession.Active = false
		klog.Infof("Removed oldest session %s for user %s due to limit", oldestSessionID, userID)
	}
}

func (sm *SessionManager) calculateInitialRiskScore(sourceIP, userAgent string) float64 {
	riskScore := 0.0

	// Basic risk assessment
	if sourceIP == "" {
		riskScore += 0.2
	}

	if userAgent == "" {
		riskScore += 0.1
	}

	// TODO: Add more sophisticated risk assessment
	// - Geolocation analysis
	// - Known malicious IP ranges
	// - Browser fingerprinting
	// - Historical patterns

	return riskScore
}

func (sm *SessionManager) assessActivityRisk(action, resource, result string) []string {
	var riskFactors []string

	// Failed actions increase risk
	if result == "failure" || result == "error" {
		riskFactors = append(riskFactors, "failed_action")
	}

	// Sensitive actions increase risk
	sensitiveActions := []string{"delete", "update", "create_role", "delete_role"}
	for _, sensitiveAction := range sensitiveActions {
		if action == sensitiveAction {
			riskFactors = append(riskFactors, "sensitive_action")
			break
		}
	}

	// Administrative resources increase risk
	adminResources := []string{"users", "roles", "policies", "clusters"}
	for _, adminResource := range adminResources {
		if resource == adminResource {
			riskFactors = append(riskFactors, "administrative_resource")
			break
		}
	}

	return riskFactors
}

func (sm *SessionManager) updateRiskScore(session *Session, activity SessionActivity) float64 {
	currentScore := session.RiskScore

	// Increase risk based on activity
	if len(activity.RiskFactors) > 0 {
		riskIncrease := float64(len(activity.RiskFactors)) * 0.1
		currentScore += riskIncrease
	}

	// Decay risk over time for good behavior
	if activity.Result == "success" && len(activity.RiskFactors) == 0 {
		currentScore *= 0.95 // 5% reduction for good activity
	}

	// Ensure score stays within bounds
	if currentScore > 1.0 {
		currentScore = 1.0
	}
	if currentScore < 0.0 {
		currentScore = 0.0
	}

	return currentScore
}

func (sm *SessionManager) runCleanup() {
	for {
		select {
		case <-sm.stopCh:
			return
		case <-sm.cleanupTicker.C:
			sm.cleanupExpiredSessions()
		}
	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.sessionMutex.Lock()
	defer sm.sessionMutex.Unlock()

	now := time.Now()
	expiredCount := 0

	// Mark expired sessions as inactive
	for _, session := range sm.sessions {
		if session.Active && now.After(session.ExpiresAt) {
			session.Active = false
			expiredCount++
		}
	}

	// Remove old inactive sessions (older than 24 hours)
	cutoff := now.Add(-24 * time.Hour)
	removedCount := 0

	for id, session := range sm.sessions {
		if !session.Active && session.ExpiresAt.Before(cutoff) {
			delete(sm.sessions, id)
			removedCount++
		}
	}

	if expiredCount > 0 || removedCount > 0 {
		klog.V(4).Infof("Session cleanup: expired %d, removed %d old sessions", expiredCount, removedCount)
	}
}
