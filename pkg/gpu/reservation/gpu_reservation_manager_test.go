package reservation

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewGPUReservationManager(t *testing.T) {
	config := ReservationManagerConfig{
		MaxReservationsPerGPU:    5,
		MaxReservationsPerUser:   3,
		DefaultReservationWindow: 2 * time.Hour,
		ConflictResolutionPolicy: "flexible",
		EnablePreemption:         true,
		MaxReservationDuration:   24 * time.Hour,
		CleanupInterval:          30 * time.Minute,
	}

	manager := NewGPUReservationManager(config)

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}

	if manager.config.MaxReservationsPerGPU != 5 {
		t.Errorf("Expected max reservations per GPU 5, got %d", manager.config.MaxReservationsPerGPU)
	}

	if manager.config.MaxReservationsPerUser != 3 {
		t.Errorf("Expected max reservations per user 3, got %d", manager.config.MaxReservationsPerUser)
	}

	if manager.config.ConflictResolutionPolicy != "flexible" {
		t.Errorf("Expected conflict resolution policy 'flexible', got %s", manager.config.ConflictResolutionPolicy)
	}
}

func TestGPUReservationManagerDefaultConfig(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	if manager.config.MaxReservationsPerGPU != 10 {
		t.Errorf("Expected default max reservations per GPU 10, got %d", manager.config.MaxReservationsPerGPU)
	}

	if manager.config.MaxReservationsPerUser != 5 {
		t.Errorf("Expected default max reservations per user 5, got %d", manager.config.MaxReservationsPerUser)
	}

	if manager.config.DefaultReservationWindow != 24*time.Hour {
		t.Errorf("Expected default reservation window 24h, got %v", manager.config.DefaultReservationWindow)
	}

	if manager.config.ConflictResolutionPolicy != "strict" {
		t.Errorf("Expected default conflict resolution policy 'strict', got %s", manager.config.ConflictResolutionPolicy)
	}

	if manager.config.MaxReservationDuration != 7*24*time.Hour {
		t.Errorf("Expected default max reservation duration 1 week, got %v", manager.config.MaxReservationDuration)
	}

	if manager.config.CleanupInterval != 1*time.Hour {
		t.Errorf("Expected default cleanup interval 1h, got %v", manager.config.CleanupInterval)
	}
}

func TestCreateReservation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	request := &ReservationRequest{
		UserID:         "user1",
		WorkloadID:     "workload1",
		GPUID:          "card0",
		Fraction:       0.5,
		MemoryRequest:  2048, // 2GB
		StartTime:      time.Now().Add(1 * time.Hour),
		Duration:       2 * time.Hour,
		Priority:       ReservationPriorityNormal,
		Annotations:    map[string]string{"test": "value"},
		IsolationType:  "time-slicing",
		SharingEnabled: true,
	}

	reservation, err := manager.CreateReservation(context.Background(), request)
	if err != nil {
		t.Fatalf("Failed to create reservation: %v", err)
	}

	if reservation == nil {
		t.Fatal("Expected non-nil reservation")
	}

	if reservation.UserID != "user1" {
		t.Errorf("Expected user ID 'user1', got %s", reservation.UserID)
	}

	if reservation.WorkloadID != "workload1" {
		t.Errorf("Expected workload ID 'workload1', got %s", reservation.WorkloadID)
	}

	if reservation.GPUID != "card0" {
		t.Errorf("Expected GPU ID 'card0', got %s", reservation.GPUID)
	}

	if reservation.Fraction != 0.5 {
		t.Errorf("Expected fraction 0.5, got %f", reservation.Fraction)
	}

	if reservation.MemoryRequest != 2048 {
		t.Errorf("Expected memory request 2048, got %d", reservation.MemoryRequest)
	}

	if reservation.Priority != ReservationPriorityNormal {
		t.Errorf("Expected priority %d, got %d", ReservationPriorityNormal, reservation.Priority)
	}

	if reservation.Status != ReservationStatusPending {
		t.Errorf("Expected status 'pending', got %s", reservation.Status)
	}

	if reservation.IsolationType != "time-slicing" {
		t.Errorf("Expected isolation type 'time-slicing', got %s", reservation.IsolationType)
	}

	if !reservation.SharingEnabled {
		t.Error("Expected sharing enabled")
	}
}

func TestCreateReservationValidation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	tests := []struct {
		name    string
		request *ReservationRequest
		wantErr bool
	}{
		{
			name: "missing user ID",
			request: &ReservationRequest{
				WorkloadID: "workload1",
				GPUID:      "card0",
				Fraction:   0.5,
				StartTime:  time.Now().Add(1 * time.Hour),
				Duration:   1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "missing workload ID",
			request: &ReservationRequest{
				UserID:    "user1",
				GPUID:     "card0",
				Fraction:  0.5,
				StartTime: time.Now().Add(1 * time.Hour),
				Duration:  1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "missing GPU ID",
			request: &ReservationRequest{
				UserID:     "user1",
				WorkloadID: "workload1",
				Fraction:   0.5,
				StartTime:  time.Now().Add(1 * time.Hour),
				Duration:   1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid fraction too low",
			request: &ReservationRequest{
				UserID:     "user1",
				WorkloadID: "workload1",
				GPUID:      "card0",
				Fraction:   0.05,
				StartTime:  time.Now().Add(1 * time.Hour),
				Duration:   1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid fraction too high",
			request: &ReservationRequest{
				UserID:     "user1",
				WorkloadID: "workload1",
				GPUID:      "card0",
				Fraction:   1.5,
				StartTime:  time.Now().Add(1 * time.Hour),
				Duration:   1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "negative memory request",
			request: &ReservationRequest{
				UserID:        "user1",
				WorkloadID:    "workload1",
				GPUID:         "card0",
				Fraction:      0.5,
				MemoryRequest: -1024,
				StartTime:     time.Now().Add(1 * time.Hour),
				Duration:      1 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "zero duration",
			request: &ReservationRequest{
				UserID:     "user1",
				WorkloadID: "workload1",
				GPUID:      "card0",
				Fraction:   0.5,
				StartTime:  time.Now().Add(1 * time.Hour),
				Duration:   0,
			},
			wantErr: true,
		},
		{
			name: "past start time",
			request: &ReservationRequest{
				UserID:     "user1",
				WorkloadID: "workload1",
				GPUID:      "card0",
				Fraction:   0.5,
				StartTime:  time.Now().Add(-1 * time.Hour),
				Duration:   1 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.CreateReservation(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateReservation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetReservation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	// Create a reservation
	request := &ReservationRequest{
		UserID:     "user1",
		WorkloadID: "workload1",
		GPUID:      "card0",
		Fraction:   0.5,
		StartTime:  time.Now().Add(1 * time.Hour),
		Duration:   1 * time.Hour,
	}

	reservation, err := manager.CreateReservation(context.Background(), request)
	if err != nil {
		t.Fatalf("Failed to create reservation: %v", err)
	}

	// Get the reservation
	retrieved, exists := manager.GetReservation(reservation.ID)
	if !exists {
		t.Fatal("Expected reservation to exist")
	}

	if retrieved.ID != reservation.ID {
		t.Errorf("Expected reservation ID %s, got %s", reservation.ID, retrieved.ID)
	}

	// Test non-existent reservation
	_, exists = manager.GetReservation("non-existent")
	if exists {
		t.Error("Expected non-existent reservation to not exist")
	}
}

func TestListReservations(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	// Create multiple reservations
	requests := []*ReservationRequest{
		{
			UserID:     "user1",
			WorkloadID: "workload1",
			GPUID:      "card0",
			Fraction:   0.5,
			StartTime:  time.Now().Add(1 * time.Hour),
			Duration:   1 * time.Hour,
		},
		{
			UserID:     "user1",
			WorkloadID: "workload2",
			GPUID:      "card1",
			Fraction:   0.3,
			StartTime:  time.Now().Add(2 * time.Hour),
			Duration:   1 * time.Hour,
		},
		{
			UserID:     "user2",
			WorkloadID: "workload3",
			GPUID:      "card0",
			Fraction:   0.7,
			StartTime:  time.Now().Add(3 * time.Hour),
			Duration:   1 * time.Hour,
		},
	}

	for _, req := range requests {
		_, err := manager.CreateReservation(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to create reservation: %v", err)
		}
	}

	// Test listing all reservations
	allReservations := manager.ListReservations(nil)
	if len(allReservations) != 3 {
		t.Errorf("Expected 3 reservations, got %d", len(allReservations))
	}

	// Test filtering by user
	user1Reservations := manager.ListReservations(&ReservationFilters{UserID: "user1"})
	if len(user1Reservations) != 2 {
		t.Errorf("Expected 2 reservations for user1, got %d", len(user1Reservations))
	}

	// Test filtering by GPU
	card0Reservations := manager.ListReservations(&ReservationFilters{GPUID: "card0"})
	if len(card0Reservations) != 2 {
		t.Errorf("Expected 2 reservations for card0, got %d", len(card0Reservations))
	}

	// Test filtering by status
	pendingReservations := manager.ListReservations(&ReservationFilters{Status: ReservationStatusPending})
	if len(pendingReservations) != 3 {
		t.Errorf("Expected 3 pending reservations, got %d", len(pendingReservations))
	}
}

func TestUpdateReservation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	// Create a reservation
	request := &ReservationRequest{
		UserID:     "user1",
		WorkloadID: "workload1",
		GPUID:      "card0",
		Fraction:   0.5,
		StartTime:  time.Now().Add(1 * time.Hour),
		Duration:   1 * time.Hour,
	}

	reservation, err := manager.CreateReservation(context.Background(), request)
	if err != nil {
		t.Fatalf("Failed to create reservation: %v", err)
	}

	// Update the reservation
	updates := map[string]interface{}{
		"fraction":       0.7,
		"memory_request": int64(4096),
		"priority":       ReservationPriorityHigh,
	}

	updated, err := manager.UpdateReservation(reservation.ID, updates)
	if err != nil {
		t.Fatalf("Failed to update reservation: %v", err)
	}

	if updated.Fraction != 0.7 {
		t.Errorf("Expected updated fraction 0.7, got %f", updated.Fraction)
	}

	if updated.MemoryRequest != 4096 {
		t.Errorf("Expected updated memory request 4096, got %d", updated.MemoryRequest)
	}

	if updated.Priority != ReservationPriorityHigh {
		t.Errorf("Expected updated priority %d, got %d", ReservationPriorityHigh, updated.Priority)
	}

	// Test updating non-existent reservation
	_, err = manager.UpdateReservation("non-existent", updates)
	if err == nil {
		t.Error("Expected error when updating non-existent reservation")
	}
}

func createTestReservation(t *testing.T, manager *GPUReservationManager) *GPUReservation {
	request := &ReservationRequest{
		UserID:     "user1",
		WorkloadID: "workload1",
		GPUID:      "card0",
		Fraction:   0.5,
		StartTime:  time.Now().Add(1 * time.Hour),
		Duration:   1 * time.Hour,
	}

	reservation, err := manager.CreateReservation(context.Background(), request)
	if err != nil {
		t.Fatalf("Failed to create reservation: %v", err)
	}
	return reservation
}

func TestCancelReservation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	reservation := createTestReservation(t, manager)

	// Cancel the reservation
	err := manager.CancelReservation(reservation.ID)
	if err != nil {
		t.Fatalf("Failed to cancel reservation: %v", err)
	}

	// Verify status
	retrieved, exists := manager.GetReservation(reservation.ID)
	if !exists {
		t.Fatal("Expected reservation to still exist")
	}

	if retrieved.Status != ReservationStatusCancelled {
		t.Errorf("Expected status 'cancelled', got %s", retrieved.Status)
	}

	// Test cancelling non-existent reservation
	err = manager.CancelReservation("non-existent")
	if err == nil {
		t.Error("Expected error when cancelling non-existent reservation")
	}
}

func TestCompleteReservation(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	reservation := createTestReservation(t, manager)

	// Complete the reservation
	err := manager.CompleteReservation(reservation.ID)
	if err != nil {
		t.Fatalf("Failed to complete reservation: %v", err)
	}

	// Verify status
	retrieved, exists := manager.GetReservation(reservation.ID)
	if !exists {
		t.Fatal("Expected reservation to still exist")
	}

	if retrieved.Status != ReservationStatusCompleted {
		t.Errorf("Expected status 'completed', got %s", retrieved.Status)
	}

	// Test completing non-existent reservation
	err = manager.CompleteReservation("non-existent")
	if err == nil {
		t.Error("Expected error when completing non-existent reservation")
	}
}

func TestGetReservationConflicts(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	// Create an existing reservation
	existingRequest := &ReservationRequest{
		UserID:     "user1",
		WorkloadID: "workload1",
		GPUID:      "card0",
		Fraction:   0.5,
		StartTime:  time.Now().Add(2 * time.Hour),
		Duration:   2 * time.Hour,
	}

	_, err := manager.CreateReservation(context.Background(), existingRequest)
	if err != nil {
		t.Fatalf("Failed to create existing reservation: %v", err)
	}

	// Test conflicting request
	conflictingRequest := &ReservationRequest{
		UserID:     "user2",
		WorkloadID: "workload2",
		GPUID:      "card0",
		Fraction:   0.3,
		StartTime:  time.Now().Add(3 * time.Hour), // Overlaps with existing
		Duration:   1 * time.Hour,
	}

	conflicts := manager.GetReservationConflicts(conflictingRequest)
	if len(conflicts) == 0 {
		t.Error("Expected conflicts to be detected")
	}

	// Test non-conflicting request
	nonConflictingRequest := &ReservationRequest{
		UserID:     "user2",
		WorkloadID: "workload3",
		GPUID:      "card1", // Different GPU
		Fraction:   0.3,
		StartTime:  time.Now().Add(3 * time.Hour),
		Duration:   1 * time.Hour,
	}

	conflicts = manager.GetReservationConflicts(nonConflictingRequest)
	if len(conflicts) > 0 {
		t.Error("Expected no conflicts for different GPU")
	}
}

func TestGetReservationStats(t *testing.T) {
	manager := NewGPUReservationManager(ReservationManagerConfig{})

	// Create reservations with different statuses
	requests := []*ReservationRequest{
		{
			UserID:     "user1",
			WorkloadID: "workload1",
			GPUID:      "card0",
			Fraction:   0.5,
			StartTime:  time.Now().Add(1 * time.Hour),
			Duration:   1 * time.Hour,
		},
		{
			UserID:     "user1",
			WorkloadID: "workload2",
			GPUID:      "card1",
			Fraction:   0.3,
			StartTime:  time.Now().Add(2 * time.Hour),
			Duration:   1 * time.Hour,
		},
	}

	for _, req := range requests {
		_, err := manager.CreateReservation(context.Background(), req)
		if err != nil {
			t.Fatalf("Failed to create reservation: %v", err)
		}
	}

	// Get stats
	stats := manager.GetReservationStats()

	if stats.TotalReservations != 2 {
		t.Errorf("Expected 2 total reservations, got %d", stats.TotalReservations)
	}

	if stats.PendingReservations != 2 {
		t.Errorf("Expected 2 pending reservations, got %d", stats.PendingReservations)
	}

	if stats.ActiveReservations != 0 {
		t.Errorf("Expected 0 active reservations, got %d", stats.ActiveReservations)
	}

	if len(stats.ReservationsByGPU) != 2 {
		t.Errorf("Expected 2 GPUs with reservations, got %d", len(stats.ReservationsByGPU))
	}

	if len(stats.ReservationsByUser) != 1 {
		t.Errorf("Expected 1 user with reservations, got %d", len(stats.ReservationsByUser))
	}

	if stats.ReservationsByUser["user1"] != 2 {
		t.Errorf("Expected user1 to have 2 reservations, got %d", stats.ReservationsByUser["user1"])
	}
}

func TestUserLimits(t *testing.T) {
	config := ReservationManagerConfig{
		MaxReservationsPerUser: 2,
	}
	manager := NewGPUReservationManager(config)

	// Create maximum allowed reservations
	for i := 0; i < 2; i++ {
		request := &ReservationRequest{
			UserID:     "user1",
			WorkloadID: fmt.Sprintf("workload%d", i),
			GPUID:      fmt.Sprintf("card%d", i),
			Fraction:   0.5,
			StartTime:  time.Now().Add(time.Duration(i+1) * time.Hour),
			Duration:   1 * time.Hour,
		}

		_, err := manager.CreateReservation(context.Background(), request)
		if err != nil {
			t.Fatalf("Failed to create reservation %d: %v", i, err)
		}
	}

	// Try to create one more reservation
	request := &ReservationRequest{
		UserID:     "user1",
		WorkloadID: "workload3",
		GPUID:      "card3",
		Fraction:   0.5,
		StartTime:  time.Now().Add(4 * time.Hour),
		Duration:   1 * time.Hour,
	}

	_, err := manager.CreateReservation(context.Background(), request)
	if err == nil {
		t.Error("Expected error when exceeding user limits")
	}
}

func TestGPULimits(t *testing.T) {
	config := ReservationManagerConfig{
		MaxReservationsPerGPU: 2,
	}
	manager := NewGPUReservationManager(config)

	// Create maximum allowed reservations for the same GPU
	for i := 0; i < 2; i++ {
		request := &ReservationRequest{
			UserID:     fmt.Sprintf("user%d", i),
			WorkloadID: fmt.Sprintf("workload%d", i),
			GPUID:      "card0",
			Fraction:   0.5,
			StartTime:  time.Now().Add(time.Duration(i+1) * time.Hour),
			Duration:   1 * time.Hour,
		}

		_, err := manager.CreateReservation(context.Background(), request)
		if err != nil {
			t.Fatalf("Failed to create reservation %d: %v", i, err)
		}
	}

	// Try to create one more reservation for the same GPU
	request := &ReservationRequest{
		UserID:     "user3",
		WorkloadID: "workload3",
		GPUID:      "card0",
		Fraction:   0.5,
		StartTime:  time.Now().Add(4 * time.Hour),
		Duration:   1 * time.Hour,
	}

	_, err := manager.CreateReservation(context.Background(), request)
	if err == nil {
		t.Error("Expected error when exceeding GPU limits")
	}
}
