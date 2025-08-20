package manager

import (
	"fmt"
	"math"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// MI300XPartitionMode represents the compute partitioning mode
type MI300XPartitionMode string

const (
	// MI300XPartitionModeSPX - Single Partition X-celerator: All 8 XCDs as single device
	MI300XPartitionModeSPX MI300XPartitionMode = "SPX"
	// MI300XPartitionModeCPX - Core Partitioned X-celerator: Each XCD as separate GPU
	MI300XPartitionModeCPX MI300XPartitionMode = "CPX"
	// MI300XPartitionModeTPX - Additional partitioning mode
	MI300XPartitionModeTPX MI300XPartitionMode = "TPX"
)

// MI300XMemoryMode represents the memory partitioning mode
type MI300XMemoryMode string

const (
	// MI300XMemoryModeNPS1 - Entire memory accessible to all XCDs
	MI300XMemoryModeNPS1 MI300XMemoryMode = "NPS1"
	// MI300XMemoryModeNPS4 - Memory partitioned into quadrants
	MI300XMemoryModeNPS4 MI300XMemoryMode = "NPS4"
)

// MI300XPartitionConfig represents the partitioning configuration for MI300X
type MI300XPartitionConfig struct {
	ComputeMode MI300XPartitionMode `json:"computeMode"`
	MemoryMode  MI300XMemoryMode    `json:"memoryMode"`
	XCDCount    int                 `json:"xcdCount"` // Number of XCDs (always 8 for MI300X)
}

// MI300XFractionalAllocator manages fractional GPU allocations for MI300X
type MI300XFractionalAllocator struct {
	// allocations tracks fractional allocations per GPU
	allocations map[string][]*types.GPUAllocation

	// gpuCapacity tracks the total capacity of each GPU
	gpuCapacity map[string]float64

	// gpuMemoryCapacity tracks the memory capacity of each GPU
	gpuMemoryCapacity map[string]int64

	// partitionConfig tracks the partitioning configuration for each GPU
	partitionConfig map[string]*MI300XPartitionConfig

	// xcdAllocations tracks XCD-level allocations for CPX mode
	xcdAllocations map[string]map[int]*types.GPUAllocation // deviceID -> xcdIndex -> allocation
}

// NewMI300XFractionalAllocator creates a new MI300X-aware fractional allocator
func NewMI300XFractionalAllocator() *MI300XFractionalAllocator {
	return &MI300XFractionalAllocator{
		allocations:       make(map[string][]*types.GPUAllocation),
		gpuCapacity:       make(map[string]float64),
		gpuMemoryCapacity: make(map[string]int64),
		partitionConfig:   make(map[string]*MI300XPartitionConfig),
		xcdAllocations:    make(map[string]map[int]*types.GPUAllocation),
	}
}

// RegisterMI300XGPU registers an MI300X GPU with the fractional allocator
func (f *MI300XFractionalAllocator) RegisterMI300XGPU(deviceID string, totalMemory int64, config *MI300XPartitionConfig) error {
	if config == nil {
		// Default to SPX mode if no config provided
		config = &MI300XPartitionConfig{
			ComputeMode: MI300XPartitionModeSPX,
			MemoryMode:  MI300XMemoryModeNPS1,
			XCDCount:    8,
		}
	}

	// Validate configuration
	if err := f.validatePartitionConfig(config); err != nil {
		return fmt.Errorf("invalid partition config for GPU %s: %w", deviceID, err)
	}

	f.gpuCapacity[deviceID] = 1.0 // Full GPU capacity
	f.gpuMemoryCapacity[deviceID] = totalMemory
	f.allocations[deviceID] = make([]*types.GPUAllocation, 0)
	f.partitionConfig[deviceID] = config
	f.xcdAllocations[deviceID] = make(map[int]*types.GPUAllocation)

	return nil
}

// validatePartitionConfig validates the MI300X partitioning configuration
func (f *MI300XFractionalAllocator) validatePartitionConfig(config *MI300XPartitionConfig) error {
	if config.XCDCount != 8 {
		return fmt.Errorf("MI300X must have exactly 8 XCDs, got %d", config.XCDCount)
	}

	switch config.ComputeMode {
	case MI300XPartitionModeSPX, MI300XPartitionModeCPX, MI300XPartitionModeTPX:
		// Valid compute modes
	default:
		return fmt.Errorf("invalid compute mode: %s", config.ComputeMode)
	}

	switch config.MemoryMode {
	case MI300XMemoryModeNPS1, MI300XMemoryModeNPS4:
		// Valid memory modes
	default:
		return fmt.Errorf("invalid memory mode: %s", config.MemoryMode)
	}

	// Validate mode compatibility
	if config.ComputeMode == MI300XPartitionModeSPX && config.MemoryMode == MI300XMemoryModeNPS4 {
		return fmt.Errorf("NPS4 memory mode is not compatible with SPX compute mode")
	}

	return nil
}

// GetValidFractions returns the valid fractional allocations for the given GPU
func (f *MI300XFractionalAllocator) GetValidFractions(deviceID string) []float64 {
	config, exists := f.partitionConfig[deviceID]
	if !exists {
		return []float64{1.0} // Default to full GPU if not configured
	}

	switch config.ComputeMode {
	case MI300XPartitionModeSPX:
		// SPX mode: Only full GPU allocation (1.0)
		return []float64{1.0}

	case MI300XPartitionModeCPX:
		// CPX mode: Each XCD is 1/8 of the GPU
		fractions := make([]float64, 0)
		for i := 1; i <= 8; i++ {
			fractions = append(fractions, float64(i)/8.0)
		}
		return fractions

	case MI300XPartitionModeTPX:
		// TPX mode: Custom partitioning (implementation specific)
		// For now, return common fractions
		return []float64{0.125, 0.25, 0.5, 0.75, 1.0}

	default:
		return []float64{1.0}
	}
}

// ValidateFraction validates if a fraction is valid for the given GPU
func (f *MI300XFractionalAllocator) ValidateFraction(deviceID string, fraction float64) error {
	validFractions := f.GetValidFractions(deviceID)
	
	for _, valid := range validFractions {
		if math.Abs(fraction-valid) < 0.001 { // Allow small floating point differences
			return nil
		}
	}

	return fmt.Errorf("fraction %f is not valid for GPU %s. Valid fractions: %v", 
		fraction, deviceID, validFractions)
}

// CanAllocate checks if a fractional allocation is possible for MI300X
func (f *MI300XFractionalAllocator) CanAllocate(deviceID string, request *types.GPURequest) (bool, error) {
	if request == nil {
		return false, fmt.Errorf("GPU request cannot be nil")
	}

	if err := types.ValidateGPURequest(request); err != nil {
		return false, fmt.Errorf("invalid GPU request: %v", err)
	}

	// Check if GPU is registered
	if _, exists := f.gpuCapacity[deviceID]; !exists {
		return false, fmt.Errorf("GPU %s is not registered", deviceID)
	}

	// Validate fraction for MI300X partitioning
	if err := f.ValidateFraction(deviceID, request.Fraction); err != nil {
		return false, err
	}

	config := f.partitionConfig[deviceID]

	// Check allocation based on partitioning mode
	switch config.ComputeMode {
	case MI300XPartitionModeSPX:
		return f.canAllocateSPX(deviceID, request)
	case MI300XPartitionModeCPX:
		return f.canAllocateCPX(deviceID, request)
	case MI300XPartitionModeTPX:
		return f.canAllocateTPX(deviceID, request)
	default:
		return false, fmt.Errorf("unknown compute mode: %s", config.ComputeMode)
	}
}

// canAllocateSPX checks allocation for SPX mode (single partition)
func (f *MI300XFractionalAllocator) canAllocateSPX(deviceID string, request *types.GPURequest) (bool, error) {
	// SPX mode only allows full GPU allocation
	if request.Fraction != 1.0 {
		return false, fmt.Errorf("SPX mode only supports full GPU allocation (1.0), requested %f", request.Fraction)
	}

	// Check if GPU is already allocated
	allocations := f.allocations[deviceID]
	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			return false, fmt.Errorf("GPU %s is already allocated in SPX mode", deviceID)
		}
	}

	// Check memory capacity
	if request.MemoryRequest > 0 {
		availableMemory := f.getAvailableMemory(deviceID)
		if request.MemoryRequest*1024*1024 > availableMemory {
			return false, fmt.Errorf("insufficient memory: requested %d MiB, available %d bytes",
				request.MemoryRequest, availableMemory)
		}
	}

	return true, nil
}

// canAllocateCPX checks allocation for CPX mode (8 separate XCDs)
func (f *MI300XFractionalAllocator) canAllocateCPX(deviceID string, request *types.GPURequest) (bool, error) {
	// Calculate how many XCDs are needed
	xcdsNeeded := int(math.Ceil(request.Fraction * 8.0))
	
	// Check if enough XCDs are available
	availableXCDs := f.getAvailableXCDs(deviceID)
	if xcdsNeeded > availableXCDs {
		return false, fmt.Errorf("insufficient XCDs: requested %d XCDs, available %d XCDs",
			xcdsNeeded, availableXCDs)
	}

	// Check memory capacity
	if request.MemoryRequest > 0 {
		availableMemory := f.getAvailableMemory(deviceID)
		if request.MemoryRequest*1024*1024 > availableMemory {
			return false, fmt.Errorf("insufficient memory: requested %d MiB, available %d bytes",
				request.MemoryRequest, availableMemory)
		}
	}

	return true, nil
}

// canAllocateTPX checks allocation for TPX mode (custom partitioning)
func (f *MI300XFractionalAllocator) canAllocateTPX(deviceID string, request *types.GPURequest) (bool, error) {
	// TPX mode allows more flexible partitioning
	// For now, use similar logic to CPX but with more flexibility
	availableFraction := f.getAvailableFraction(deviceID)
	if request.Fraction > availableFraction {
		return false, fmt.Errorf("insufficient fractional capacity: requested %f, available %f",
			request.Fraction, availableFraction)
	}

	// Check memory capacity
	if request.MemoryRequest > 0 {
		availableMemory := f.getAvailableMemory(deviceID)
		if request.MemoryRequest*1024*1024 > availableMemory {
			return false, fmt.Errorf("insufficient memory: requested %d MiB, available %d bytes",
				request.MemoryRequest, availableMemory)
		}
	}

	return true, nil
}

// Allocate performs a fractional allocation for MI300X
func (f *MI300XFractionalAllocator) Allocate(deviceID string, request *types.AllocationRequest) (*types.GPUAllocation, error) {
	canAllocate, err := f.CanAllocate(deviceID, request.GPURequest)
	if err != nil {
		return nil, err
	}

	if !canAllocate {
		return nil, fmt.Errorf("cannot allocate on GPU %s", deviceID)
	}

	// Create allocation
	allocation := &types.GPUAllocation{
		ID:            request.ID,
		DeviceID:      deviceID,
		Fraction:      request.GPURequest.Fraction,
		MemoryRequest: request.GPURequest.MemoryRequest,
		IsolationType: request.GPURequest.IsolationType,
		PodName:       request.PodName,
		Namespace:     request.Namespace,
		ContainerName: request.ContainerName,
		Status:        types.GPUAllocationStatusActive,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     0, // No expiration by default
	}

	// Set expiration if specified
	if request.ExpiresAt != nil {
		allocation.ExpiresAt = request.ExpiresAt.Unix()
	}

	// Add allocation to the GPU
	f.allocations[deviceID] = append(f.allocations[deviceID], allocation)

	// Handle XCD allocation for CPX mode
	config := f.partitionConfig[deviceID]
	if config.ComputeMode == MI300XPartitionModeCPX {
		f.allocateXCDs(deviceID, allocation)
	}

	return allocation, nil
}

// allocateXCDs allocates XCDs for CPX mode
func (f *MI300XFractionalAllocator) allocateXCDs(deviceID string, allocation *types.GPUAllocation) {
	xcdsNeeded := int(math.Ceil(allocation.Fraction * 8.0))
	allocatedXCDs := 0

	for xcdIndex := 0; xcdIndex < 8 && allocatedXCDs < xcdsNeeded; xcdIndex++ {
		if f.xcdAllocations[deviceID][xcdIndex] == nil {
			f.xcdAllocations[deviceID][xcdIndex] = allocation
			allocatedXCDs++
		}
	}
}

// getAvailableXCDs returns the number of available XCDs for CPX mode
func (f *MI300XFractionalAllocator) getAvailableXCDs(deviceID string) int {
	allocatedXCDs := 0
	for xcdIndex := 0; xcdIndex < 8; xcdIndex++ {
		if f.xcdAllocations[deviceID][xcdIndex] != nil {
			allocatedXCDs++
		}
	}
	return 8 - allocatedXCDs
}

// Release releases a fractional allocation for MI300X
func (f *MI300XFractionalAllocator) Release(allocationID string) error {
	for deviceID, allocations := range f.allocations {
		for i, allocation := range allocations {
			if allocation.ID == allocationID {
				// Remove allocation from slice
				f.allocations[deviceID] = append(allocations[:i], allocations[i+1:]...)

				// Release XCDs for CPX mode
				config := f.partitionConfig[deviceID]
				if config.ComputeMode == MI300XPartitionModeCPX {
					f.releaseXCDs(deviceID, allocation)
				}

				return nil
			}
		}
	}

	return fmt.Errorf("allocation %s not found", allocationID)
}

// releaseXCDs releases XCDs for CPX mode
func (f *MI300XFractionalAllocator) releaseXCDs(deviceID string, allocation *types.GPUAllocation) {
	for xcdIndex := 0; xcdIndex < 8; xcdIndex++ {
		if f.xcdAllocations[deviceID][xcdIndex] == nil {
			continue
		}
		if f.xcdAllocations[deviceID][xcdIndex].ID == allocation.ID {
			delete(f.xcdAllocations[deviceID], xcdIndex)
		}
	}
}

// GetAvailableFraction returns the available fractional capacity for a GPU
func (f *MI300XFractionalAllocator) getAvailableFraction(deviceID string) float64 {
	config := f.partitionConfig[deviceID]
	if config == nil {
		return 0.0
	}

	switch config.ComputeMode {
	case MI300XPartitionModeSPX:
		// SPX mode: Either full GPU (1.0) or nothing (0.0)
		allocations := f.allocations[deviceID]
		for _, allocation := range allocations {
			if allocation.Status == types.GPUAllocationStatusActive {
				return 0.0 // GPU is allocated
			}
		}
		return 1.0 // GPU is available

	case MI300XPartitionModeCPX:
		// CPX mode: Available XCDs / 8
		availableXCDs := f.getAvailableXCDs(deviceID)
		return float64(availableXCDs) / 8.0

	case MI300XPartitionModeTPX:
		// TPX mode: More flexible calculation
		totalCapacity := f.gpuCapacity[deviceID]
		usedCapacity := f.getUsedFraction(deviceID)
		available := totalCapacity - usedCapacity
		if available < 0 {
			available = 0
		}
		return available

	default:
		return 0.0
	}
}

// GetAvailableMemory returns the available memory for a GPU
func (f *MI300XFractionalAllocator) getAvailableMemory(deviceID string) int64 {
	totalMemory := f.gpuMemoryCapacity[deviceID]
	usedMemory := f.getUsedMemory(deviceID)

	available := totalMemory - usedMemory
	if available < 0 {
		available = 0
	}

	return available
}

// GetUsedFraction returns the used fractional capacity for a GPU
func (f *MI300XFractionalAllocator) getUsedFraction(deviceID string) float64 {
	allocations := f.allocations[deviceID]
	var used float64

	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			used += allocation.Fraction
		}
	}

	return used
}

// GetUsedMemory returns the used memory for a GPU
func (f *MI300XFractionalAllocator) getUsedMemory(deviceID string) int64 {
	allocations := f.allocations[deviceID]
	var used int64

	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			used += allocation.MemoryRequest * 1024 * 1024 // Convert MiB to bytes
		}
	}

	return used
}

// GetGPUUtilization returns the utilization statistics for a GPU
func (f *MI300XFractionalAllocator) GetGPUUtilization(deviceID string) *GPUUtilizationStats {
	allocations := f.allocations[deviceID]

	stats := &GPUUtilizationStats{
		DeviceID:              deviceID,
		TotalCapacity:         f.gpuCapacity[deviceID],
		TotalMemory:           f.gpuMemoryCapacity[deviceID],
		UsedFraction:          f.getUsedFraction(deviceID),
		UsedMemory:            f.getUsedMemory(deviceID),
		ActiveAllocations:     0,
		UtilizationRate:       0.0,
		MemoryUtilizationRate: 0.0,
	}

	// Count active allocations
	for _, allocation := range allocations {
		if allocation.Status == types.GPUAllocationStatusActive {
			stats.ActiveAllocations++
		}
	}

	// Calculate utilization rates
	if stats.TotalCapacity > 0 {
		stats.UtilizationRate = stats.UsedFraction / stats.TotalCapacity
	}

	if stats.TotalMemory > 0 {
		stats.MemoryUtilizationRate = float64(stats.UsedMemory) / float64(stats.TotalMemory)
	}

	return stats
}

// GetPartitionConfig returns the partitioning configuration for a GPU
func (f *MI300XFractionalAllocator) GetPartitionConfig(deviceID string) *MI300XPartitionConfig {
	return f.partitionConfig[deviceID]
}

// GetXCDAllocations returns the XCD allocations for CPX mode
func (f *MI300XFractionalAllocator) GetXCDAllocations(deviceID string) map[int]*types.GPUAllocation {
	xcdAllocs := make(map[int]*types.GPUAllocation)
	for xcdIndex, allocation := range f.xcdAllocations[deviceID] {
		xcdAllocs[xcdIndex] = allocation
	}
	return xcdAllocs
}

// CleanupExpiredAllocations removes expired allocations
func (f *MI300XFractionalAllocator) CleanupExpiredAllocations() {
	now := time.Now().Unix()

	for deviceID, allocations := range f.allocations {
		var validAllocations []*types.GPUAllocation

		for _, allocation := range allocations {
			if allocation.ExpiresAt > 0 && allocation.ExpiresAt <= now {
				// Mark as expired
				allocation.Status = types.GPUAllocationStatusExpired
				
				// Release XCDs for CPX mode
				config := f.partitionConfig[deviceID]
				if config != nil && config.ComputeMode == MI300XPartitionModeCPX {
					f.releaseXCDs(deviceID, allocation)
				}
			} else {
				validAllocations = append(validAllocations, allocation)
			}
		}

		f.allocations[deviceID] = validAllocations
	}
}
