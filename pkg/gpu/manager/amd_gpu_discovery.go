// Copyright 2025 Advanced Micro Devices, Inc.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/silogen/kaiwo/pkg/gpu/types"
)

// AMDGPUDiscovery handles real AMD GPU discovery using ROCm tools
type AMDGPUDiscovery struct {
	// rocmSMIPath is the path to rocm-smi executable
	rocmSMIPath string

	// sysClassDRMPath is the path to /sys/class/drm
	sysClassDRMPath string

	// timeout for commands
	timeout time.Duration
}

// NewAMDGPUDiscovery creates a new AMD GPU discovery instance
func NewAMDGPUDiscovery() *AMDGPUDiscovery {
	return &AMDGPUDiscovery{
		rocmSMIPath:     findROCmSMI(),
		sysClassDRMPath: "/sys/class/drm",
		timeout:         30 * time.Second,
	}
}

// DiscoverGPUs discovers AMD GPUs using multiple methods
func (d *AMDGPUDiscovery) DiscoverGPUs(ctx context.Context) ([]*types.GPUInfo, error) {
	// Try ROCm SMI first (most comprehensive)
	if d.rocmSMIPath != "" {
		gpus, err := d.discoverWithROCmSMI(ctx)
		if err == nil && len(gpus) > 0 {
			return gpus, nil
		}
		fmt.Printf("ROCm SMI discovery failed: %v, falling back to sysfs\n", err)
	}

	// Fall back to sysfs discovery
	gpus, err := d.discoverWithSysfs(ctx)
	if err != nil {
		return nil, fmt.Errorf("all GPU discovery methods failed: %v", err)
	}

	return gpus, nil
}

// discoverWithROCmSMI uses rocm-smi to discover GPUs
func (d *AMDGPUDiscovery) discoverWithROCmSMI(ctx context.Context) ([]*types.GPUInfo, error) {
	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// Execute rocm-smi with JSON output
	cmd := exec.CommandContext(cmdCtx, d.rocmSMIPath, "--showallinfo", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute rocm-smi: %v", err)
	}

	// Parse JSON output as a generic map to handle the actual structure
	var rocmOutput map[string]interface{}
	if err := json.Unmarshal(output, &rocmOutput); err != nil {
		return nil, fmt.Errorf("failed to parse rocm-smi JSON output: %v", err)
	}

	// Convert to GPUInfo
	var gpus []*types.GPUInfo
	for cardID, cardData := range rocmOutput {
		// Skip system info
		if cardID == "system" {
			continue
		}

		// Convert card data to map
		cardMap, ok := cardData.(map[string]interface{})
		if !ok {
			continue
		}

		gpu, err := d.convertROCmSMIToGPUInfo(cardID, cardMap)
		if err != nil {
			fmt.Printf("Failed to convert ROCm SMI data for card %s: %v\n", cardID, err)
			continue
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// convertROCmSMIToGPUInfo converts ROCm SMI data to GPUInfo
func (d *AMDGPUDiscovery) convertROCmSMIToGPUInfo(cardID string, cardMap map[string]interface{}) (*types.GPUInfo, error) {
	// Extract values from the map
	temperature := d.getFloatValue(cardMap, "Temperature (Sensor edge) (C)", 0.0)
	utilization := d.getFloatValue(cardMap, "GPU use (%)", 0.0)
	power := d.getFloatValue(cardMap, "Current Socket Graphics Package Power (W)", 0.0)
	cardSeries := d.getStringValue(cardMap, "Card Series", "AMD GPU")
	cardModel := d.getStringValue(cardMap, "Card Model", "Unknown")
	memoryAllocated := d.getFloatValue(cardMap, "GPU Memory Allocated (VRAM%)", 0.0)

	// Calculate memory (estimate based on allocation percentage)
	// For AMD Instinct GPUs, we'll use typical memory sizes
	var totalMemory int64
	switch {
	case strings.Contains(strings.ToLower(cardSeries), "instinct"):
		totalMemory = 32 * 1024 * 1024 * 1024 // 32GB for Instinct
	case strings.Contains(strings.ToLower(cardSeries), "radeon"):
		totalMemory = 8 * 1024 * 1024 * 1024 // 8GB for Radeon
	default:
		totalMemory = 16 * 1024 * 1024 * 1024 // 16GB default
	}

	usedMemory := int64(float64(totalMemory) * memoryAllocated / 100.0)
	availableMemory := totalMemory - usedMemory

	// Get node name
	nodeName, _ := os.Hostname()

	return &types.GPUInfo{
		DeviceID:          cardID,
		Type:              types.GPUTypeAMD,
		Model:             fmt.Sprintf("%s %s", cardSeries, cardModel),
		TotalMemory:       totalMemory,
		AvailableMemory:   availableMemory,
		Utilization:       utilization,
		Temperature:       temperature,
		Power:             power,
		NodeName:          nodeName,
		IsAvailable:       d.isGPUHealthy(temperature, utilization),
		IsolationType:     types.GPUIsolationNone,
		ActiveAllocations: 0,
	}, nil
}

// discoverWithSysfs uses /sys/class/drm to discover GPUs
func (d *AMDGPUDiscovery) discoverWithSysfs(ctx context.Context) ([]*types.GPUInfo, error) {
	if _, err := os.Stat(d.sysClassDRMPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("sysfs DRM path not found: %s", d.sysClassDRMPath)
	}

	// Find AMD GPU cards
	cards, err := d.findAMDCards()
	if err != nil {
		return nil, fmt.Errorf("failed to find AMD cards: %v", err)
	}

	var gpus []*types.GPUInfo
	for _, cardPath := range cards {
		gpu, err := d.parseCardFromSysfs(cardPath)
		if err != nil {
			fmt.Printf("Failed to parse card %s: %v\n", cardPath, err)
			continue
		}
		gpus = append(gpus, gpu)
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("no AMD GPUs found in sysfs")
	}

	return gpus, nil
}

// findAMDCards finds AMD GPU cards in /sys/class/drm
func (d *AMDGPUDiscovery) findAMDCards() ([]string, error) {
	entries, err := os.ReadDir(d.sysClassDRMPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DRM directory: %v", err)
	}

	var amdCards []string
	cardRegex := regexp.MustCompile(`^card\d+$`)

	for _, entry := range entries {
		if !cardRegex.MatchString(entry.Name()) {
			continue
		}

		cardPath := filepath.Join(d.sysClassDRMPath, entry.Name())

		// Check if it's an AMD GPU
		if d.isAMDCard(cardPath) {
			amdCards = append(amdCards, cardPath)
		}
	}

	return amdCards, nil
}

// isAMDCard checks if a card is an AMD GPU
func (d *AMDGPUDiscovery) isAMDCard(cardPath string) bool {
	// Check vendor ID
	vendorPath := filepath.Join(cardPath, "device", "vendor")
	vendor, err := os.ReadFile(vendorPath)
	if err != nil {
		return false
	}

	// AMD vendor ID is 0x1002
	vendorStr := strings.TrimSpace(string(vendor))
	return vendorStr == "0x1002"
}

// parseCardFromSysfs parses GPU information from sysfs
func (d *AMDGPUDiscovery) parseCardFromSysfs(cardPath string) (*types.GPUInfo, error) {
	deviceID := filepath.Base(cardPath)

	// Read device info
	devicePath := filepath.Join(cardPath, "device")

	// Get device name/model
	model := d.readSysfsFile(filepath.Join(devicePath, "device"))
	if model == "" {
		model = "AMD GPU"
	}

	// Get memory info (if available)
	var totalMemory int64 = 8 * 1024 * 1024 * 1024 // Default 8GB if not readable
	memInfoPath := filepath.Join(devicePath, "mem_info_vram_total")
	if memStr := d.readSysfsFile(memInfoPath); memStr != "" {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			totalMemory = mem
		}
	}

	// Get memory usage (if available)
	var usedMemory int64
	memUsedPath := filepath.Join(devicePath, "mem_info_vram_used")
	if memStr := d.readSysfsFile(memUsedPath); memStr != "" {
		if mem, err := strconv.ParseInt(memStr, 10, 64); err == nil {
			usedMemory = mem
		}
	}

	availableMemory := totalMemory - usedMemory

	// Get temperature (if available)
	var temperature float64
	tempPaths := []string{
		filepath.Join(devicePath, "hwmon", "hwmon*", "temp1_input"),
		filepath.Join(devicePath, "hwmon", "hwmon*", "temp2_input"),
	}

	for _, tempPattern := range tempPaths {
		if matches, _ := filepath.Glob(tempPattern); len(matches) > 0 {
			if tempStr := d.readSysfsFile(matches[0]); tempStr != "" {
				if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
					temperature = temp / 1000.0 // Convert millidegrees to degrees
					break
				}
			}
		}
	}

	// Get utilization (if available)
	var utilization float64
	utilizationPath := filepath.Join(devicePath, "gpu_busy_percent")
	if utilStr := d.readSysfsFile(utilizationPath); utilStr != "" {
		if util, err := strconv.ParseFloat(utilStr, 64); err == nil {
			utilization = util
		}
	}

	// Get power (if available)
	var power float64
	powerPaths := []string{
		filepath.Join(devicePath, "hwmon", "hwmon*", "power1_average"),
		filepath.Join(devicePath, "hwmon", "hwmon*", "power1_input"),
	}

	for _, powerPattern := range powerPaths {
		if matches, _ := filepath.Glob(powerPattern); len(matches) > 0 {
			if powerStr := d.readSysfsFile(matches[0]); powerStr != "" {
				if pow, err := strconv.ParseFloat(powerStr, 64); err == nil {
					power = pow / 1000000.0 // Convert microwatts to watts
					break
				}
			}
		}
	}

	// Get node name
	nodeName, _ := os.Hostname()

	return &types.GPUInfo{
		DeviceID:          deviceID,
		Type:              types.GPUTypeAMD,
		Model:             model,
		TotalMemory:       totalMemory,
		AvailableMemory:   availableMemory,
		Utilization:       utilization,
		Temperature:       temperature,
		Power:             power,
		NodeName:          nodeName,
		IsAvailable:       d.isGPUHealthy(temperature, utilization),
		IsolationType:     types.GPUIsolationNone,
		ActiveAllocations: 0,
	}, nil
}

// readSysfsFile safely reads a sysfs file
func (d *AMDGPUDiscovery) readSysfsFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// isGPUHealthy determines if a GPU is healthy based on temperature and utilization
func (d *AMDGPUDiscovery) isGPUHealthy(temperature, utilization float64) bool {
	// Check temperature threshold (< 90°C)
	return temperature <= 90.0
}

// findROCmSMI finds the rocm-smi executable
func findROCmSMI() string {
	// Common paths for rocm-smi
	commonPaths := []string{
		"/opt/rocm/bin/rocm-smi",
		"/usr/bin/rocm-smi",
		"/usr/local/bin/rocm-smi",
	}

	// Check PATH first
	if path, err := exec.LookPath("rocm-smi"); err == nil {
		return path
	}

	// Check common paths
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// parseFloat safely parses a float from string
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// getStringValue safely extracts a string value from a map
func (d *AMDGPUDiscovery) getStringValue(m map[string]interface{}, key, defaultValue string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return defaultValue
}

// getFloatValue safely extracts a float value from a map
func (d *AMDGPUDiscovery) getFloatValue(m map[string]interface{}, key string, defaultValue float64) float64 {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			if f, err := parseFloat(str); err == nil {
				return f
			}
		}
	}
	return defaultValue
}

// MonitorGPUs continuously monitors GPU metrics
func (d *AMDGPUDiscovery) MonitorGPUs(ctx context.Context, gpus map[string]*types.GPUInfo, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.updateGPUMetrics(ctx, gpus)
		}
	}
}

// updateGPUMetrics updates metrics for existing GPUs
func (d *AMDGPUDiscovery) updateGPUMetrics(ctx context.Context, gpus map[string]*types.GPUInfo) {
	// If ROCm SMI is available, use it for detailed metrics
	if d.rocmSMIPath != "" {
		d.updateMetricsWithROCmSMI(ctx, gpus)
	} else {
		d.updateMetricsWithSysfs(ctx, gpus)
	}
}

// updateMetricsWithROCmSMI updates metrics using ROCm SMI
func (d *AMDGPUDiscovery) updateMetricsWithROCmSMI(ctx context.Context, gpus map[string]*types.GPUInfo) {
	discoveredGPUs, err := d.discoverWithROCmSMI(ctx)
	if err != nil {
		fmt.Printf("Failed to update metrics with ROCm SMI: %v\n", err)
		return
	}

	for _, discoveredGPU := range discoveredGPUs {
		if existingGPU, exists := gpus[discoveredGPU.DeviceID]; exists {
			// Update metrics while preserving allocation info
			existingGPU.Utilization = discoveredGPU.Utilization
			existingGPU.Temperature = discoveredGPU.Temperature
			existingGPU.Power = discoveredGPU.Power
			existingGPU.AvailableMemory = discoveredGPU.AvailableMemory
			existingGPU.IsAvailable = d.isGPUHealthy(existingGPU.Temperature, existingGPU.Utilization) &&
				existingGPU.ActiveAllocations < 10 // Allocation limit
		}
	}
}

// updateMetricsWithSysfs updates metrics using sysfs
func (d *AMDGPUDiscovery) updateMetricsWithSysfs(ctx context.Context, gpus map[string]*types.GPUInfo) {
	for deviceID, gpu := range gpus {
		cardPath := filepath.Join(d.sysClassDRMPath, deviceID)
		devicePath := filepath.Join(cardPath, "device")

		// Update utilization
		if utilStr := d.readSysfsFile(filepath.Join(devicePath, "gpu_busy_percent")); utilStr != "" {
			if util, err := strconv.ParseFloat(utilStr, 64); err == nil {
				gpu.Utilization = util
			}
		}

		// Update temperature
		tempPaths := []string{
			filepath.Join(devicePath, "hwmon", "hwmon*", "temp1_input"),
		}

		for _, tempPattern := range tempPaths {
			if matches, _ := filepath.Glob(tempPattern); len(matches) > 0 {
				if tempStr := d.readSysfsFile(matches[0]); tempStr != "" {
					if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
						gpu.Temperature = temp / 1000.0
						break
					}
				}
			}
		}

		// Update power
		powerPaths := []string{
			filepath.Join(devicePath, "hwmon", "hwmon*", "power1_average"),
		}

		for _, powerPattern := range powerPaths {
			if matches, _ := filepath.Glob(powerPattern); len(matches) > 0 {
				if powerStr := d.readSysfsFile(matches[0]); powerStr != "" {
					if pow, err := strconv.ParseFloat(powerStr, 64); err == nil {
						gpu.Power = pow / 1000000.0
						break
					}
				}
			}
		}

		// Update availability
		gpu.IsAvailable = d.isGPUHealthy(gpu.Temperature, gpu.Utilization) &&
			gpu.ActiveAllocations < 10
	}
}
