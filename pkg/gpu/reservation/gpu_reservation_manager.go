package reservation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// ReservationStatus represents the status of a GPU reservation
type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusCompleted ReservationStatus = "completed"
	ReservationStatusCancelled ReservationStatus = "cancelled"
	ReservationStatusExpired   ReservationStatus = "expired"
)

const (
	ConflictResolutionPolicyStrict   = "strict"
	ConflictResolutionPolicyFlexible = "flexible"
	ConflictResolutionPolicyOverlap  = "overlap"
)

// ReservationPriority represents the priority of a reservation
type ReservationPriority int

const (
	ReservationPriorityLow    ReservationPriority = 1
	ReservationPriorityNormal ReservationPriority = 5
	ReservationPriorityHigh   ReservationPriority = 10
	ReservationPriorityUrgent ReservationPriority = 15
)

// GPUReservation represents a GPU reservation
type GPUReservation struct {
	ID             string
	UserID         string
	WorkloadID     string
	GPUID          string
	Fraction       float64
	MemoryRequest  int64 // in MiB
	StartTime      time.Time
	EndTime        time.Time
	Priority       ReservationPriority
	Status         ReservationStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Annotations    map[string]string
	IsolationType  string // "time-slicing", "none"
	SharingEnabled bool
}

// ReservationRequest represents a request to create a GPU reservation
type ReservationRequest struct {
	UserID         string
	WorkloadID     string
	GPUID          string
	Fraction       float64
	MemoryRequest  int64 // in MiB
	StartTime      time.Time
	Duration       time.Duration
	Priority       ReservationPriority
	Annotations    map[string]string
	IsolationType  string
	SharingEnabled bool
}

// ReservationConflict represents a conflict between reservations
type ReservationConflict struct {
	ReservationID           string
	ConflictType            string
	Message                 string
	ConflictingReservations []string
}

// GPUReservationManager manages GPU reservations
type GPUReservationManager struct {
	reservations map[string]*GPUReservation
	config       ReservationManagerConfig
	mu           sync.RWMutex
}

// ReservationManagerConfig contains configuration for the reservation manager
type ReservationManagerConfig struct {
	MaxReservationsPerGPU    int
	MaxReservationsPerUser   int
	DefaultReservationWindow time.Duration
	ConflictResolutionPolicy string // "strict", "flexible", "overlap"
	EnablePreemption         bool
	MaxReservationDuration   time.Duration
	CleanupInterval          time.Duration
}

// NewGPUReservationManager creates a new GPU reservation manager
func NewGPUReservationManager(config ReservationManagerConfig) *GPUReservationManager {
	if config.MaxReservationsPerGPU == 0 {
		config.MaxReservationsPerGPU = 10
	}
	if config.MaxReservationsPerUser == 0 {
		config.MaxReservationsPerUser = 5
	}
	if config.DefaultReservationWindow == 0 {
		config.DefaultReservationWindow = 24 * time.Hour
	}
	if config.ConflictResolutionPolicy == "" {
		config.ConflictResolutionPolicy = ConflictResolutionPolicyStrict
	}
	if config.MaxReservationDuration == 0 {
		config.MaxReservationDuration = 7 * 24 * time.Hour // 1 week
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}

	manager := &GPUReservationManager{
		reservations: make(map[string]*GPUReservation),
		config:       config,
	}

	// Start cleanup goroutine
	go manager.cleanupExpiredReservations()

	return manager
}

// CreateReservation creates a new GPU reservation
func (r *GPUReservationManager) CreateReservation(ctx context.Context, request *ReservationRequest) (*GPUReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate request
	if err := r.validateReservationRequest(request); err != nil {
		return nil, fmt.Errorf("invalid reservation request: %w", err)
	}

	// Check for conflicts
	conflicts := r.checkConflicts(request)
	if len(conflicts) > 0 && r.config.ConflictResolutionPolicy == ConflictResolutionPolicyStrict {
		return nil, fmt.Errorf("reservation conflicts detected: %v", conflicts)
	}

	// Check user limits
	if err := r.checkUserLimits(request.UserID); err != nil {
		return nil, fmt.Errorf("user limits exceeded: %w", err)
	}

	// Check GPU limits
	if err := r.checkGPULimits(request.GPUID); err != nil {
		return nil, fmt.Errorf("GPU limits exceeded: %w", err)
	}

	// Calculate end time
	endTime := request.StartTime.Add(request.Duration)

	// Create reservation
	reservation := &GPUReservation{
		ID:             r.generateReservationID(request),
		UserID:         request.UserID,
		WorkloadID:     request.WorkloadID,
		GPUID:          request.GPUID,
		Fraction:       request.Fraction,
		MemoryRequest:  request.MemoryRequest,
		StartTime:      request.StartTime,
		EndTime:        endTime,
		Priority:       request.Priority,
		Status:         ReservationStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Annotations:    request.Annotations,
		IsolationType:  request.IsolationType,
		SharingEnabled: request.SharingEnabled,
	}

	// Handle conflicts based on policy
	if len(conflicts) > 0 {
		if err := r.resolveConflicts(reservation, conflicts); err != nil {
			return nil, fmt.Errorf("failed to resolve conflicts: %w", err)
		}
	}

	// Add reservation
	r.reservations[reservation.ID] = reservation

	// Update status if reservation starts immediately
	if time.Now().After(request.StartTime) || time.Now().Equal(request.StartTime) {
		reservation.Status = ReservationStatusActive
	}

	return reservation, nil
}

// GetReservation returns a reservation by ID
func (r *GPUReservationManager) GetReservation(id string) (*GPUReservation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reservation, exists := r.reservations[id]
	return reservation, exists
}

// ListReservations returns all reservations with optional filters
func (r *GPUReservationManager) ListReservations(filters *ReservationFilters) []*GPUReservation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var reservations []*GPUReservation

	for _, reservation := range r.reservations {
		if r.matchesFilters(reservation, filters) {
			reservations = append(reservations, reservation)
		}
	}

	return reservations
}

// UpdateReservation updates an existing reservation
func (r *GPUReservationManager) UpdateReservation(id string, updates map[string]interface{}) (*GPUReservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	reservation, exists := r.reservations[id]
	if !exists {
		return nil, fmt.Errorf("reservation %s not found", id)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "fraction":
			if fraction, ok := value.(float64); ok {
				reservation.Fraction = fraction
			}
		case "memory_request":
			if memory, ok := value.(int64); ok {
				reservation.MemoryRequest = memory
			}
		case "start_time":
			if startTime, ok := value.(time.Time); ok {
				reservation.StartTime = startTime
			}
		case "end_time":
			if endTime, ok := value.(time.Time); ok {
				reservation.EndTime = endTime
			}
		case "priority":
			if priority, ok := value.(ReservationPriority); ok {
				reservation.Priority = priority
			}
		case "status":
			if status, ok := value.(ReservationStatus); ok {
				reservation.Status = status
			}
		case "annotations":
			if annotations, ok := value.(map[string]string); ok {
				reservation.Annotations = annotations
			}
		}
	}

	reservation.UpdatedAt = time.Now()
	return reservation, nil
}

// CancelReservation cancels a reservation
func (r *GPUReservationManager) CancelReservation(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	reservation, exists := r.reservations[id]
	if !exists {
		return fmt.Errorf("reservation %s not found", id)
	}

	if reservation.Status == ReservationStatusCompleted || reservation.Status == ReservationStatusCancelled {
		return fmt.Errorf("cannot cancel reservation in status %s", reservation.Status)
	}

	reservation.Status = ReservationStatusCancelled
	reservation.UpdatedAt = time.Now()

	return nil
}

// CompleteReservation marks a reservation as completed
func (r *GPUReservationManager) CompleteReservation(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	reservation, exists := r.reservations[id]
	if !exists {
		return fmt.Errorf("reservation %s not found", id)
	}

	reservation.Status = ReservationStatusCompleted
	reservation.UpdatedAt = time.Now()

	return nil
}

// GetReservationConflicts returns conflicts for a reservation request
func (r *GPUReservationManager) GetReservationConflicts(request *ReservationRequest) []*ReservationConflict {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.checkConflicts(request)
}

// GetReservationStats returns statistics about reservations
func (r *GPUReservationManager) GetReservationStats() *types.ReservationStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &types.ReservationStats{
		TotalReservations:     len(r.reservations),
		PendingReservations:   0,
		ActiveReservations:    0,
		CompletedReservations: 0,
		CancelledReservations: 0,
		ExpiredReservations:   0,
		ReservationsByGPU:     make(map[string]int),
		ReservationsByUser:    make(map[string]int),
		ReservationsByStatus:  make(map[string]int),
	}

	for _, reservation := range r.reservations {
		// Count by status
		statusStr := string(reservation.Status)
		stats.ReservationsByStatus[statusStr]++

		switch reservation.Status {
		case ReservationStatusPending:
			stats.PendingReservations++
		case ReservationStatusActive:
			stats.ActiveReservations++
		case ReservationStatusCompleted:
			stats.CompletedReservations++
		case ReservationStatusCancelled:
			stats.CancelledReservations++
		case ReservationStatusExpired:
			stats.ExpiredReservations++
		}

		// Count by GPU
		stats.ReservationsByGPU[reservation.GPUID]++

		// Count by user
		stats.ReservationsByUser[reservation.UserID]++
	}

	return stats
}

// validateReservationRequest validates a reservation request
func (r *GPUReservationManager) validateReservationRequest(request *ReservationRequest) error {
	if request.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if request.WorkloadID == "" {
		return fmt.Errorf("workload ID is required")
	}

	if request.GPUID == "" {
		return fmt.Errorf("GPU ID is required")
	}

	if request.Fraction < 0.1 || request.Fraction > 1.0 {
		return fmt.Errorf("GPU fraction must be between 0.1 and 1.0, got %f", request.Fraction)
	}

	if request.MemoryRequest < 0 {
		return fmt.Errorf("memory request must be non-negative, got %d", request.MemoryRequest)
	}

	if request.Duration <= 0 {
		return fmt.Errorf("duration must be positive, got %v", request.Duration)
	}

	if request.Duration > r.config.MaxReservationDuration {
		return fmt.Errorf("duration exceeds maximum allowed duration of %v", r.config.MaxReservationDuration)
	}

	if request.StartTime.Before(time.Now()) {
		return fmt.Errorf("start time cannot be in the past")
	}

	return nil
}

// checkConflicts checks for conflicts with existing reservations
func (r *GPUReservationManager) checkConflicts(request *ReservationRequest) []*ReservationConflict {
	var conflicts []*ReservationConflict

	for _, reservation := range r.reservations {
		// Skip completed and cancelled reservations
		if reservation.Status == ReservationStatusCompleted || reservation.Status == ReservationStatusCancelled {
			continue
		}

		// Check if reservations overlap in time
		if r.timeOverlaps(request, reservation) {
			// Check if they use the same GPU
			if request.GPUID == reservation.GPUID {
				conflict := &ReservationConflict{
					ReservationID:           reservation.ID,
					ConflictType:            "time_overlap",
					Message:                 fmt.Sprintf("Time overlap with reservation %s", reservation.ID),
					ConflictingReservations: []string{reservation.ID},
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// timeOverlaps checks if two reservations overlap in time
func (r *GPUReservationManager) timeOverlaps(request *ReservationRequest, reservation *GPUReservation) bool {
	requestEnd := request.StartTime.Add(request.Duration)
	reservationEnd := reservation.EndTime

	// Check for overlap
	return !(requestEnd.Before(reservation.StartTime) || request.StartTime.After(reservationEnd))
}

// resolveConflicts resolves conflicts based on the configured policy
func (r *GPUReservationManager) resolveConflicts(newReservation *GPUReservation, conflicts []*ReservationConflict) error {
	switch r.config.ConflictResolutionPolicy {
	case "flexible":
		// Allow overlapping reservations if GPU sharing is enabled
		if newReservation.SharingEnabled {
			return nil
		}
		return fmt.Errorf("conflicts cannot be resolved with flexible policy")

	case "overlap":
		// Allow overlapping reservations
		return nil

	case "strict":
		// No conflicts allowed
		return fmt.Errorf("conflicts not allowed with strict policy")

	default:
		return fmt.Errorf("unknown conflict resolution policy: %s", r.config.ConflictResolutionPolicy)
	}
}

// checkUserLimits checks if user has exceeded reservation limits
func (r *GPUReservationManager) checkUserLimits(userID string) error {
	count := 0
	for _, reservation := range r.reservations {
		if reservation.UserID == userID &&
			(reservation.Status == ReservationStatusPending || reservation.Status == ReservationStatusActive) {
			count++
		}
	}

	if count >= r.config.MaxReservationsPerUser {
		return fmt.Errorf("user %s has exceeded maximum reservations limit of %d", userID, r.config.MaxReservationsPerUser)
	}

	return nil
}

// checkGPULimits checks if GPU has exceeded reservation limits
func (r *GPUReservationManager) checkGPULimits(gpuID string) error {
	count := 0
	for _, reservation := range r.reservations {
		if reservation.GPUID == gpuID &&
			(reservation.Status == ReservationStatusPending || reservation.Status == ReservationStatusActive) {
			count++
		}
	}

	if count >= r.config.MaxReservationsPerGPU {
		return fmt.Errorf("GPU %s has exceeded maximum reservations limit of %d", gpuID, r.config.MaxReservationsPerGPU)
	}

	return nil
}

// generateReservationID generates a unique reservation ID
func (r *GPUReservationManager) generateReservationID(request *ReservationRequest) string {
	return fmt.Sprintf("res-%s-%s-%d", request.UserID, request.GPUID, time.Now().Unix())
}

// cleanupExpiredReservations periodically cleans up expired reservations
func (r *GPUReservationManager) cleanupExpiredReservations() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for _, reservation := range r.reservations {
			if reservation.EndTime.Before(now) && reservation.Status == ReservationStatusActive {
				reservation.Status = ReservationStatusExpired
				reservation.UpdatedAt = now
			}
		}
		r.mu.Unlock()
	}
}

// ReservationFilters contains filters for listing reservations
type ReservationFilters struct {
	UserID    string
	GPUID     string
	Status    ReservationStatus
	StartTime time.Time
	EndTime   time.Time
}

// matchesFilters checks if a reservation matches the given filters
func (r *GPUReservationManager) matchesFilters(reservation *GPUReservation, filters *ReservationFilters) bool {
	if filters == nil {
		return true
	}

	if filters.UserID != "" && reservation.UserID != filters.UserID {
		return false
	}

	if filters.GPUID != "" && reservation.GPUID != filters.GPUID {
		return false
	}

	if filters.Status != "" && reservation.Status != filters.Status {
		return false
	}

	if !filters.StartTime.IsZero() && reservation.StartTime.Before(filters.StartTime) {
		return false
	}

	if !filters.EndTime.IsZero() && reservation.EndTime.After(filters.EndTime) {
		return false
	}

	return true
}
