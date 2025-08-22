# Kaiwo Phase 1 Command-Line Demo Guide

This guide provides step-by-step commands to demonstrate the complete Phase 1 implementation of the Kaiwo project.

## Prerequisites

Ensure you're on the AMD GPU node and have the following tools available:
```bash
# Verify environment
hostname
rocm-smi
go version
kubectl version --client
```

## Demo Overview

This demo will showcase:
1. **Advanced GPU Management** - Fractional allocation, memory-based requests, AMD time-slicing
2. **Enhanced Queue Management** - Hierarchical queues, quota management, DRF policies
3. **Plugin Architecture** - Extensible plugin system with interfaces
4. **Enhanced Scheduling** - Priority-based, resource-aware, load-balanced scheduling
5. **Resource Optimization** - Dynamic resource allocation and optimization
6. **Enhanced Monitoring** - Real-time metrics collection and intelligent alerting

---

## Demo 1: Advanced GPU Management

### Step 1.1: Verify AMD GPU Environment
```bash
# Check AMD GPU status
rocm-smi

# Verify GPU discovery
ls -la pkg/gpu/manager/
```

### Step 1.2: Demonstrate GPU Resource Manager
```bash
# Show GPU manager implementation
cat pkg/gpu/manager/fractional_allocator.go | head -30

# Show AMD-specific GPU sharing
cat pkg/gpu/manager/amd_gpu_sharing.go | head -40

# Show MI300X fractional allocator
cat pkg/gpu/manager/mi300x_fractional_allocator.go | head -30
```

### Step 1.3: Test GPU Allocation Features
```bash
# Run GPU manager tests
go test -v ./pkg/gpu/manager/ -run TestFractionalAllocation

# Test AMD GPU sharing
go test -v ./pkg/gpu/manager/ -run TestAMDGPUSharing

# Performance benchmark
go test -bench=. -benchmem ./pkg/gpu/manager/
```

### Step 1.4: Show GPU Types and Annotations
```bash
# Display supported GPU annotations
cat pkg/gpu/types/gpu_types.go | grep -A 10 "GPUIsolationType"

# Show annotation parsing
cat pkg/gpu/types/gpu_types.go | grep -A 20 "ParseGPUAnnotations"
```

---

## Demo 2: Enhanced Queue Management

### Step 2.1: Verify Queue Controller Implementation
```bash
# Show queue controller
cat internal/controller/kaiwoqueueconfig_controller.go | head -40

# List queue-related files
find internal/controller/ -name "*queue*" -type f
```

### Step 2.2: Demonstrate Queue Hierarchy
```bash
# Show queue configuration types
cat apis/kaiwo/v1alpha1/kaiwoqueueconfig_types.go | head -50

# Display queue management features
grep -r "ClusterQueue\|LocalQueue" internal/controller/
```

### Step 2.3: Test Queue Management
```bash
# Run queue controller tests
go test -v ./internal/controller/ -run TestQueue

# Performance test
go test -bench=. -benchmem ./internal/controller/
```

---

## Demo 3: Plugin Architecture

### Step 3.1: Show Plugin Interface Design
```bash
# Display plugin interfaces
cat pkg/workloads/common/interfaces.go | head -50

# Show plugin registry
find pkg/ -name "*plugin*" -type f
```

### Step 3.2: Demonstrate Plugin System
```bash
# List plugin implementations
ls -la pkg/gpu/manager/
ls -la pkg/scheduling/enhanced/
ls -la pkg/monitoring/
```

### Step 3.3: Test Plugin Architecture
```bash
# Run plugin tests
go test -v ./pkg/workloads/common/ -run TestPlugin

# Performance benchmark
go test -bench=. -benchmem ./pkg/workloads/common/
```

---

## Demo 4: Enhanced Scheduling

### Step 4.1: Show Scheduling Components
```bash
# Display priority scheduler
cat pkg/scheduling/enhanced/priority_scheduler.go | head -40

# Show resource allocator
cat pkg/scheduling/enhanced/resource_allocator.go | head -40

# Show load balancer
cat pkg/scheduling/enhanced/load_balancer.go | head -40
```

### Step 4.2: Test Scheduling Performance
```bash
# Run scheduling tests
go test -v ./pkg/scheduling/enhanced/ -run TestPriorityScheduler
go test -v ./pkg/scheduling/enhanced/ -run TestResourceAllocator
go test -v ./pkg/scheduling/enhanced/ -run TestLoadBalancer

# Performance benchmarks
go test -bench=. -benchmem ./pkg/scheduling/enhanced/
```

### Step 4.3: Demonstrate AMD GPU Integration
```bash
# Show AMD GPU resource handling
grep -r "amd.com/gpu" pkg/scheduling/enhanced/

# Display resource calculation
grep -A 10 "calculateRequiredGPU" pkg/scheduling/enhanced/priority_scheduler.go
```

---

## Demo 5: Resource Optimization

### Step 5.1: Show Dynamic Allocator
```bash
# Display dynamic allocator implementation
cat pkg/optimization/dynamic_allocator.go | head -50

# Show optimization strategies
grep -r "Optimize\|Allocate\|Release" pkg/optimization/
```

### Step 5.2: Test Resource Optimization
```bash
# Run optimization tests
go test -v ./pkg/optimization/ -run TestDynamicAllocator

# Performance benchmark
go test -bench=. -benchmem ./pkg/optimization/
```

---

## Demo 6: Enhanced Monitoring

### Step 6.1: Show Monitoring Components
```bash
# Display metrics collector
cat pkg/monitoring/realtime/metrics_collector.go | head -40

# Show alert manager
cat pkg/monitoring/alerting/alert_manager.go | head -40
```

### Step 6.2: Test Monitoring Performance
```bash
# Run monitoring tests
go test -v ./pkg/monitoring/realtime/ -run TestMetricsCollector
go test -v ./pkg/monitoring/alerting/ -run TestAlertManager

# Performance benchmarks
go test -bench=. -benchmem ./pkg/monitoring/
```

### Step 6.3: Demonstrate Real-time Metrics
```bash
# Show metrics collection
grep -r "CollectMetrics\|UpdateMetrics" pkg/monitoring/

# Display alerting rules
grep -r "AddAlert\|CheckAlerts" pkg/monitoring/alerting/
```

---

## Demo 7: Integration Testing

### Step 7.1: Run Comprehensive Tests
```bash
# Run all Phase 1 tests
./scripts/run-comprehensive-tests.sh --only phase1-enhanced-scheduling
./scripts/run-comprehensive-tests.sh --only phase1-resource-optimization
./scripts/run-comprehensive-tests.sh --only phase1-enhanced-monitoring
```

### Step 7.2: Performance Validation
```bash
# Run all performance benchmarks
go test -bench=. -benchmem ./pkg/scheduling/enhanced/
go test -bench=. -benchmem ./pkg/optimization/
go test -bench=. -benchmem ./pkg/monitoring/
go test -bench=. -benchmem ./pkg/gpu/manager/
```

### Step 7.3: Code Quality Check
```bash
# Run linter
go vet ./pkg/...
go vet ./internal/...

# Check for unused imports
go mod tidy
go mod verify
```

---

## Demo 8: AMD GPU Specific Features

### Step 8.1: Show MI300X Optimization
```bash
# Display MI300X specific features
cat pkg/gpu/manager/mi300x_fractional_allocator.go | grep -A 10 "SPX\|CPX"

# Show chiplet optimization
grep -r "SPX\|CPX\|NPS1\|NPS4" pkg/gpu/manager/
```

### Step 8.2: Demonstrate AMD GPU Sharing
```bash
# Show time-slicing implementation
grep -A 20 "time-slicing" pkg/gpu/manager/amd_gpu_sharing.go

# Display GPU reservation system
ls -la pkg/gpu/reservation/
```

### Step 8.3: Test AMD GPU Features
```bash
# Test MI300X allocator
go test -v ./pkg/gpu/manager/ -run TestMI300X

# Test AMD GPU sharing
go test -v ./pkg/gpu/manager/ -run TestAMDGPU
```

---

## Demo 9: API Integration

### Step 9.1: Show KaiwoJob API
```bash
# Display KaiwoJob structure
cat apis/kaiwo/v1alpha1/kaiwojob_types.go | head -50

# Show common types
cat apis/kaiwo/v1alpha1/common_types.go | head -40
```

### Step 9.2: Demonstrate Kueue Integration
```bash
# Show Kueue resource management
grep -r "ResourceFlavor\|ClusterQueue\|LocalQueue" internal/controller/

# Display workload priority classes
grep -r "WorkloadPriorityClass" apis/kaiwo/v1alpha1/
```

---

## Demo 10: Summary and Validation

### Step 10.1: Show Implementation Summary
```bash
# Display Phase 1 summary
cat PHASE1-IMPLEMENTATION-SUMMARY.md | head -50

# Show performance results
grep -A 5 "Performance" PHASE1-IMPLEMENTATION-SUMMARY.md
```

### Step 10.2: Verify All Components
```bash
# List all implemented components
find pkg/ -name "*.go" | grep -E "(scheduling|optimization|monitoring|gpu)" | wc -l

# Show component structure
tree pkg/ -I "*.md|*.txt" | head -20
```

### Step 10.3: Final Validation
```bash
# Run final comprehensive test
./scripts/run-comprehensive-tests.sh

# Check all tests pass
echo "Phase 1 Implementation Complete!"
```

---

## Quick Demo Commands

For a quick overview, run these key commands:

```bash
# 1. Environment check
rocm-smi && echo "AMD GPU detected"

# 2. Show GPU management
ls -la pkg/gpu/manager/ && echo "GPU management components ready"

# 3. Show scheduling
ls -la pkg/scheduling/enhanced/ && echo "Enhanced scheduling ready"

# 4. Show monitoring
ls -la pkg/monitoring/ && echo "Enhanced monitoring ready"

# 5. Run performance tests
go test -bench=. -benchmem ./pkg/scheduling/enhanced/ | head -10

# 6. Show implementation summary
echo "Phase 1 Implementation Status:" && grep -A 3 "Implementation Overview" PHASE1-IMPLEMENTATION-SUMMARY.md
```

---

## Troubleshooting

If any commands fail:

1. **Check Go environment**: `go version && go env GOPATH`
2. **Verify dependencies**: `go mod tidy && go mod verify`
3. **Check file permissions**: `ls -la scripts/`
4. **Verify AMD GPU**: `rocm-smi`
5. **Check Kubernetes**: `kubectl version --client`

---

## Next Steps

After completing this demo:

1. **Unit Tests**: Create comprehensive unit tests for all components
2. **Integration Tests**: Set up proper integration testing with Kubernetes
3. **Performance Optimization**: Optimize components to meet <10ms targets
4. **Documentation**: Expand API documentation and user guides
5. **Phase 2 Planning**: Begin planning for Phase 2 implementation

**Ready to demonstrate the complete Phase 1: Foundation Enhancement implementation!**
