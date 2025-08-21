package manager

import (
	"testing"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

func TestNewMI300XFractionalAllocator(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	if allocator == nil {
		t.Fatal("Expected non-nil allocator")
	}

	if len(allocator.allocations) != 0 {
		t.Errorf("Expected empty allocations map, got %d", len(allocator.allocations))
	}

	if len(allocator.gpuCapacity) != 0 {
		t.Errorf("Expected empty GPU capacity map, got %d", len(allocator.gpuCapacity))
	}

	if len(allocator.partitionConfig) != 0 {
		t.Errorf("Expected empty partition config map, got %d", len(allocator.partitionConfig))
	}
}

func TestRegisterMI300XGPU(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	// Test default SPX mode registration
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, nil) // 8GB
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	config := allocator.GetPartitionConfig("card0")
	if config == nil {
		t.Fatal("Expected partition config to be set")
	}

	if config.ComputeMode != MI300XPartitionModeSPX {
		t.Errorf("Expected SPX mode, got %s", config.ComputeMode)
	}

	if config.MemoryMode != MI300XMemoryModeNPS1 {
		t.Errorf("Expected NPS1 mode, got %s", config.MemoryMode)
	}

	if config.XCDCount != 8 {
		t.Errorf("Expected 8 XCDs, got %d", config.XCDCount)
	}
}

func TestRegisterMI300XGPUWithConfig(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	config := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}

	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, config)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	retrievedConfig := allocator.GetPartitionConfig("card0")
	if retrievedConfig.ComputeMode != MI300XPartitionModeCPX {
		t.Errorf("Expected CPX mode, got %s", retrievedConfig.ComputeMode)
	}

	if retrievedConfig.MemoryMode != MI300XMemoryModeNPS4 {
		t.Errorf("Expected NPS4 mode, got %s", retrievedConfig.MemoryMode)
	}
}

func TestValidatePartitionConfig(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	tests := []struct {
		name    string
		config  *MI300XPartitionConfig
		wantErr bool
	}{
		{
			name: "valid SPX config",
			config: &MI300XPartitionConfig{
				ComputeMode: MI300XPartitionModeSPX,
				MemoryMode:  MI300XMemoryModeNPS1,
				XCDCount:    8,
			},
			wantErr: false,
		},
		{
			name: "valid CPX config",
			config: &MI300XPartitionConfig{
				ComputeMode: MI300XPartitionModeCPX,
				MemoryMode:  MI300XMemoryModeNPS4,
				XCDCount:    8,
			},
			wantErr: false,
		},
		{
			name: "invalid XCD count",
			config: &MI300XPartitionConfig{
				ComputeMode: MI300XPartitionModeSPX,
				MemoryMode:  MI300XMemoryModeNPS1,
				XCDCount:    4, // Should be 8
			},
			wantErr: true,
		},
		{
			name: "invalid compute mode",
			config: &MI300XPartitionConfig{
				ComputeMode: "INVALID",
				MemoryMode:  MI300XMemoryModeNPS1,
				XCDCount:    8,
			},
			wantErr: true,
		},
		{
			name: "invalid memory mode",
			config: &MI300XPartitionConfig{
				ComputeMode: MI300XPartitionModeSPX,
				MemoryMode:  "INVALID",
				XCDCount:    8,
			},
			wantErr: true,
		},
		{
			name: "incompatible SPX with NPS4",
			config: &MI300XPartitionConfig{
				ComputeMode: MI300XPartitionModeSPX,
				MemoryMode:  MI300XMemoryModeNPS4, // Not compatible
				XCDCount:    8,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := allocator.validatePartitionConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePartitionConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetValidFractions(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	// Test SPX mode
	spxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeSPX,
		MemoryMode:  MI300XMemoryModeNPS1,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, spxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	spxFractions := allocator.GetValidFractions("card0")
	expectedSPX := []float64{1.0}
	if len(spxFractions) != len(expectedSPX) {
		t.Errorf("Expected %d SPX fractions, got %d", len(expectedSPX), len(spxFractions))
	}
	if spxFractions[0] != expectedSPX[0] {
		t.Errorf("Expected SPX fraction %f, got %f", expectedSPX[0], spxFractions[0])
	}

	// Test CPX mode
	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err = allocator.RegisterMI300XGPU("card1", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	cpxFractions := allocator.GetValidFractions("card1")
	expectedCPX := []float64{0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1.0}
	if len(cpxFractions) != len(expectedCPX) {
		t.Errorf("Expected %d CPX fractions, got %d", len(expectedCPX), len(cpxFractions))
	}
	for i, expected := range expectedCPX {
		if cpxFractions[i] != expected {
			t.Errorf("Expected CPX fraction %f, got %f", expected, cpxFractions[i])
		}
	}

}

func TestValidateFraction(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	// Register SPX GPU
	spxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeSPX,
		MemoryMode:  MI300XMemoryModeNPS1,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, spxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Test valid SPX fraction
	err = allocator.ValidateFraction("card0", 1.0)
	if err != nil {
		t.Errorf("Expected valid fraction 1.0, got error: %v", err)
	}

	// Test invalid SPX fraction
	err = allocator.ValidateFraction("card0", 0.5)
	if err == nil {
		t.Error("Expected error for invalid fraction 0.5 in SPX mode")
	}

	// Register CPX GPU
	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err = allocator.RegisterMI300XGPU("card1", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Test valid CPX fractions
	validCPXFractions := []float64{0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1.0}
	for _, fraction := range validCPXFractions {
		err := allocator.ValidateFraction("card1", fraction)
		if err != nil {
			t.Errorf("Expected valid fraction %f, got error: %v", fraction, err)
		}
	}

	// Test invalid CPX fraction
	err = allocator.ValidateFraction("card1", 0.3)
	if err == nil {
		t.Error("Expected error for invalid fraction 0.3 in CPX mode")
	}
}

func TestCanAllocateSPX(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	spxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeSPX,
		MemoryMode:  MI300XMemoryModeNPS1,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, spxConfig) // 8GB
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Test valid SPX allocation
	request := &types.GPURequest{
		Fraction:      1.0,
		MemoryRequest: 4096, // 4GB
		Priority:      5,
	}

	canAllocate, err := allocator.CanAllocate("card0", request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !canAllocate {
		t.Error("Expected allocation to be possible")
	}

	// Test invalid fraction for SPX
	invalidRequest := &types.GPURequest{
		Fraction:      0.5,
		MemoryRequest: 2048,
		Priority:      5,
	}

	_, err = allocator.CanAllocate("card0", invalidRequest)
	if err == nil {
		t.Error("Expected error for invalid fraction in SPX mode")
	}
}

func TestCanAllocateCPX(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, cpxConfig) // 8GB
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Test valid CPX allocation (1 XCD = 0.125)
	request := &types.GPURequest{
		Fraction:      0.125,
		MemoryRequest: 1024, // 1GB
		Priority:      5,
	}

	canAllocate, err := allocator.CanAllocate("card0", request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !canAllocate {
		t.Error("Expected allocation to be possible")
	}

	// Test allocation requiring multiple XCDs
	multiXCDRequest := &types.GPURequest{
		Fraction:      0.5,  // 4 XCDs
		MemoryRequest: 4096, // 4GB
		Priority:      5,
	}

	canAllocate, err = allocator.CanAllocate("card0", multiXCDRequest)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !canAllocate {
		t.Error("Expected allocation to be possible")
	}

	// Test allocation exceeding available XCDs
	largeRequest := &types.GPURequest{
		Fraction:      1.0,  // 8 XCDs
		MemoryRequest: 8192, // 8GB
		Priority:      5,
	}

	canAllocate, err = allocator.CanAllocate("card0", largeRequest)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !canAllocate {
		t.Error("Expected allocation to be possible")
	}
}

func TestAllocateAndRelease(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Create allocation request
	request := &types.AllocationRequest{
		ID: "test-allocation",
		GPURequest: &types.GPURequest{
			Fraction:      0.25, // 2 XCDs
			MemoryRequest: 2048, // 2GB
			Priority:      5,
		},
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
	}

	// Allocate
	allocation, err := allocator.Allocate("card0", request)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}

	if allocation.ID != "test-allocation" {
		t.Errorf("Expected allocation ID 'test-allocation', got %s", allocation.ID)
	}

	if allocation.Fraction != 0.25 {
		t.Errorf("Expected fraction 0.25, got %f", allocation.Fraction)
	}

	// Check XCD allocations
	xcdAllocs := allocator.GetXCDAllocations("card0")
	if len(xcdAllocs) != 2 {
		t.Errorf("Expected 2 XCD allocations, got %d", len(xcdAllocs))
	}

	// Check available XCDs
	availableXCDs := allocator.getAvailableXCDs("card0")
	if availableXCDs != 6 {
		t.Errorf("Expected 6 available XCDs, got %d", availableXCDs)
	}

	// Release allocation
	err = allocator.Release("test-allocation")
	if err != nil {
		t.Fatalf("Failed to release allocation: %v", err)
	}

	// Check that XCDs are released
	xcdAllocs = allocator.GetXCDAllocations("card0")
	if len(xcdAllocs) != 0 {
		t.Errorf("Expected 0 XCD allocations after release, got %d", len(xcdAllocs))
	}

	availableXCDs = allocator.getAvailableXCDs("card0")
	if availableXCDs != 8 {
		t.Errorf("Expected 8 available XCDs after release, got %d", availableXCDs)
	}
}

func TestMultipleAllocations(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Allocate first workload (2 XCDs)
	request1 := &types.AllocationRequest{
		ID: "allocation-1",
		GPURequest: &types.GPURequest{
			Fraction:      0.25, // 2 XCDs
			MemoryRequest: 2048,
			Priority:      5,
		},
		PodName:       "pod-1",
		Namespace:     "default",
		ContainerName: "container-1",
	}

	_, err = allocator.Allocate("card0", request1)
	if err != nil {
		t.Fatalf("Failed to allocate first workload: %v", err)
	}

	// Allocate second workload (3 XCDs)
	request2 := &types.AllocationRequest{
		ID: "allocation-2",
		GPURequest: &types.GPURequest{
			Fraction:      0.375, // 3 XCDs
			MemoryRequest: 3072,
			Priority:      5,
		},
		PodName:       "pod-2",
		Namespace:     "default",
		ContainerName: "container-2",
	}

	_, err = allocator.Allocate("card0", request2)
	if err != nil {
		t.Fatalf("Failed to allocate second workload: %v", err)
	}

	// Check total allocations
	availableXCDs := allocator.getAvailableXCDs("card0")
	if availableXCDs != 3 {
		t.Errorf("Expected 3 available XCDs, got %d", availableXCDs)
	}

	// Try to allocate more than available XCDs
	request3 := &types.AllocationRequest{
		ID: "allocation-3",
		GPURequest: &types.GPURequest{
			Fraction:      0.5, // 4 XCDs (only 3 available)
			MemoryRequest: 4096,
			Priority:      5,
		},
		PodName:       "pod-3",
		Namespace:     "default",
		ContainerName: "container-3",
	}

	_, err = allocator.Allocate("card0", request3)
	if err == nil {
		t.Error("Expected error when trying to allocate more XCDs than available")
	}

	// Release first allocation
	err = allocator.Release("allocation-1")
	if err != nil {
		t.Fatalf("Failed to release first allocation: %v", err)
	}

	// Now should be able to allocate the third workload
	allocation3, err := allocator.Allocate("card0", request3)
	if err != nil {
		t.Fatalf("Failed to allocate third workload after release: %v", err)
	}

	if allocation3.ID != "allocation-3" {
		t.Errorf("Expected allocation ID 'allocation-3', got %s", allocation3.ID)
	}
}

func TestGetGPUUtilization(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Allocate workload
	request := &types.AllocationRequest{
		ID: "test-allocation",
		GPURequest: &types.GPURequest{
			Fraction:      0.5,  // 4 XCDs
			MemoryRequest: 4096, // 4GB
			Priority:      5,
		},
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
	}

	_, err = allocator.Allocate("card0", request)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}

	// Get utilization stats
	stats := allocator.GetGPUUtilization("card0")

	if stats.DeviceID != "card0" {
		t.Errorf("Expected device ID 'card0', got %s", stats.DeviceID)
	}

	if stats.TotalCapacity != 1.0 {
		t.Errorf("Expected total capacity 1.0, got %f", stats.TotalCapacity)
	}

	if stats.UsedFraction != 0.5 {
		t.Errorf("Expected used fraction 0.5, got %f", stats.UsedFraction)
	}

	if stats.UtilizationRate != 0.5 {
		t.Errorf("Expected utilization rate 0.5, got %f", stats.UtilizationRate)
	}

	if stats.ActiveAllocations != 1 {
		t.Errorf("Expected 1 active allocation, got %d", stats.ActiveAllocations)
	}

	if stats.MemoryUtilizationRate != 0.5 {
		t.Errorf("Expected memory utilization rate 0.5, got %f", stats.MemoryUtilizationRate)
	}
}

func TestCleanupExpiredAllocations(t *testing.T) {
	allocator := NewMI300XFractionalAllocator()

	cpxConfig := &MI300XPartitionConfig{
		ComputeMode: MI300XPartitionModeCPX,
		MemoryMode:  MI300XMemoryModeNPS4,
		XCDCount:    8,
	}
	err := allocator.RegisterMI300XGPU("card0", 8*1024*1024*1024, cpxConfig)
	if err != nil {
		t.Fatalf("Failed to register GPU: %v", err)
	}

	// Create allocation with expiration
	expiration := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	request := &types.AllocationRequest{
		ID: "expired-allocation",
		GPURequest: &types.GPURequest{
			Fraction:      0.25,
			MemoryRequest: 2048,
			Priority:      5,
		},
		PodName:       "test-pod",
		Namespace:     "default",
		ContainerName: "test-container",
		ExpiresAt:     &expiration,
	}

	_, err = allocator.Allocate("card0", request)
	if err != nil {
		t.Fatalf("Failed to allocate: %v", err)
	}

	// Check that allocation is active initially
	stats := allocator.GetGPUUtilization("card0")
	if stats.ActiveAllocations != 1 {
		t.Errorf("Expected 1 active allocation initially, got %d", stats.ActiveAllocations)
	}

	// Run cleanup
	allocator.CleanupExpiredAllocations()

	// Check that allocation is now expired
	stats = allocator.GetGPUUtilization("card0")
	if stats.ActiveAllocations != 0 {
		t.Errorf("Expected 0 active allocations after cleanup, got %d", stats.ActiveAllocations)
	}

	// Check that XCDs are released
	availableXCDs := allocator.getAvailableXCDs("card0")
	if availableXCDs != 8 {
		t.Errorf("Expected 8 available XCDs after cleanup, got %d", availableXCDs)
	}
}
