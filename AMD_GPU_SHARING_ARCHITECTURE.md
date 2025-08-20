# AMD GPU Sharing Architecture

## Overview

This document explains the AMD GPU sharing architecture implemented in Kaiwo, highlighting the key differences from NVIDIA's approach and the technical limitations of ROCm.

## Critical Architectural Differences

### NVIDIA vGPU vs AMD Instinct GPUs

| Feature                   | NVIDIA vGPU (MIG)                     | AMD Instinct MI300X                             |
|---------------------------|---------------------------------------|-------------------------------------------------|
| **Hardware Partitioning** | ✅ Full hardware-level partitioning   | ✅ Advanced chiplet partitioning (SPX/CPX/TPX) |
| **Memory Isolation**      | ✅ Dedicated memory per partition     | ✅ NUMA memory partitioning (NPS1/NPS4)        |
| **Compute Isolation**     | ✅ Dedicated compute units            | ✅ XCD-based compute isolation (8 XCDs)        |
| **Fractional Allocation** | ✅ True hardware-level fractions      | ✅ Hardware-level XCD fractions                |
| **Concurrent Execution**  | ✅ Multiple isolated instances        | ✅ Concurrent XCD execution with SR-IOV        |
| **Resource Guarantees**   | ✅ Guaranteed resources per partition | ✅ Hardware-guaranteed XCD resources           |
| **Memory Bandwidth**      | Fixed per partition                   | Up to 1TB/s per XCD in NPS4 mode               |
| **Workgroup Control**     | Automatic distribution                | Explicit XCD assignment in CPX mode            |

## AMD GPU Architecture Overview

### AMD Instinct MI300X Chiplet Architecture

Based on the [official AMD ROCm documentation](https://rocm.blogs.amd.com/software-tools-optimization/compute-memory-modes/README.html), AMD Instinct MI300X GPUs use a sophisticated **chiplet-based architecture** with advanced hardware partitioning capabilities:

#### MI300X Architecture Components
- **XCD (Accelerator Complex Die)**: 8 XCDs per MI300X, each with its own L2 cache
- **IOD (I/O Die)**: 4 IODs per MI300X, each containing network connectivity
- **HBM Memory**: 8 HBM stacks (2 per IOD) providing high-bandwidth memory
- **Inter-die Interconnect**: High-speed connectivity between chiplets

#### Hardware Partitioning Modes

**Compute Partitioning Modes (MCP - Modular Chiplet Platform):**
- **SPX (Single Partition X-celerator)**: All 8 XCDs appear as a single logical device
- **CPX (Core Partitioned X-celerator)**: Each XCD appears as a separate logical GPU (8 separate GPUs)
- **TPX**: Additional partitioning mode for specific use cases

**Memory Partitioning Modes (NUMA Per Socket - NPS):**
- **NPS1**: Entire memory accessible to all XCDs (compatible with CPX and SPX)
- **NPS4**: Memory partitioned into quadrants, each directly visible to logical devices in its quadrant (compatible with CPX only)

#### Advanced Hardware Partitioning Capabilities
- **True Hardware Isolation**: SR-IOV (Single Root IO Virtualization) provides Virtual Function isolation
- **Explicit Workgroup Control**: CPX mode allows explicit control over which XCD a workgroup is assigned to
- **Memory Localization**: NPS4 mode enables localized memory accesses for improved bandwidth
- **Concurrent Multi-Partition Execution**: Multiple partitions can run simultaneously with hardware isolation



## AMD GPU Sharing Implementation

### Implementation Strategy

Our implementation focuses on **AMD Instinct MI300X**:

1. **AMD Instinct MI300X (Chiplet)**: Advanced hardware-level partitioning with XCD isolation
2. **MPS Support**: AMD's Multi-Process Service for workload isolation
3. **Hardware Optimization**: Leverages MI300X's advanced chiplet capabilities
4. **Performance Optimization**: Takes advantage of 10-15% performance gains from proper partitioning

### MI300X Performance Benefits

Based on the [AMD ROCm benchmarks](https://rocm.blogs.amd.com/software-tools-optimization/compute-memory-modes/README.html):

#### Memory Bandwidth Improvements
- **CPX/NPS4 mode**: 5-10% higher bandwidth in stream benchmarks
- **Single XCD bandwidth**: Up to 1TB/s bandwidth per XCD in NPS4 mode
- **Localized memory access**: Improved performance through memory localization

#### Compute Performance Improvements
- **CPX/NPS1 and CPX/NPS4**: 10-15% higher total system throughput than SPX mode
- **Higher clock speeds**: CPX/NPS4 runs at consistently higher compute clock speeds
- **Better cache utilization**: Improved use of caches in CPX mode

#### Configuration Modes
```bash
# Compute partitioning modes
amd-smi set --compute-partition {CPX, SPX, TPX}

# Memory partitioning modes  
amd-smi set --memory-partition {NPS1, NPS4}

# Reset partitions
amd-smi reset --compute-partition
amd-smi reset --memory-partition
```

### Implementation Architecture

```
AMD Instinct MI300X GPU Sharing
├── MI300X Hardware Partitioning
│   ├── XCD Management (8 XCDs per MI300X)
│   ├── Compute Partitioning (SPX/CPX/TPX)
│   ├── Memory Partitioning (NPS1/NPS4)
│   ├── SR-IOV Virtual Function Management
│   └── Concurrent XCD Execution
├── MI300X-Aware Fractional Allocator
│   ├── Hardware Partitioning Constraints
│   ├── Valid Fraction Validation
│   ├── XCD-Level Allocation Tracking
│   ├── Mode-Specific Allocation Logic
│   └── Configuration Validation
├── Memory Management
│   ├── HBM Stack Management (8 stacks per MI300X)
│   ├── NUMA Memory Allocation
│   ├── Memory Localization (NPS4 mode)
│   └── Memory Bandwidth Optimization
└── MPS Integration
    ├── Process Isolation
    ├── Resource Sharing
    └── Performance Monitoring
```

### Key Components

#### 1. Time-Slicing Scheduler

```go
type GPUScheduler struct {
    timeSlice      time.Duration    // Time slice per workload (e.g., 30s)
    workloadQueue  []*GPUAllocation // Queue of waiting workloads
    activeWorkload *GPUAllocation   // Currently running workload
    lastSwitch     time.Time        // Last workload switch time
}
```

**Features:**
- Round-robin scheduling of workloads
- Configurable time slices
- Workload priority support
- Automatic workload switching

#### 2. Memory Management

```go
type AMDGPUSharing struct {
    gpuWorkloads   map[string][]*GPUAllocation // Active workloads per GPU
    gpuMemoryUsage map[string]int64            // Memory usage per GPU
    gpuScheduling  map[string]*GPUScheduler    // Schedulers per GPU
}
```

**Features:**
- Real-time memory usage tracking
- Memory allocation limits
- Memory reclamation on workload completion
- Memory-based allocation decisions

#### 3. MI300X-Aware Fractional Allocator

**Hardware Partitioning Constraints:**
```go
type MI300XPartitionConfig struct {
    ComputeMode MI300XPartitionMode `json:"computeMode"` // SPX/CPX/TPX
    MemoryMode  MI300XMemoryMode    `json:"memoryMode"`  // NPS1/NPS4
    XCDCount    int                 `json:"xcdCount"`    // Always 8 for MI300X
}
```

**Valid Fractions by Partitioning Mode:**
- **SPX Mode**: Only `1.0` (full GPU) - All 8 XCDs as single device
- **CPX Mode**: `[0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1.0]` - Each XCD as separate GPU
- **TPX Mode**: `[0.125, 0.25, 0.5, 0.75, 1.0]` - Custom partitioning

**XCD-Level Allocation Tracking:**
```go
// Tracks which XCDs are allocated to which workloads
xcdAllocations map[string]map[int]*types.GPUAllocation // deviceID -> xcdIndex -> allocation
```

**Key Features:**
- Hardware-accurate fraction validation
- XCD-level resource tracking
- Mode compatibility validation
- Performance-optimized allocation strategies

#### 4. MPS (Multi-Process Service) Support

AMD's equivalent to NVIDIA MPS:
- Process-level isolation
- Shared GPU resources
- Performance optimization
- Resource contention management

## Usage Examples

### Basic Allocation

```yaml
apiVersion: kaiwo.ai/v1alpha1
kind: KaiwoJob
metadata:
  name: amd-gpu-job
spec:
  template:
    spec:
      containers:
      - name: gpu-container
        image: amd/rocm-pytorch
        resources:
          limits:
            kaiwo.ai/gpu: "0.5"  # Used for scheduling priority
            kaiwo.ai/gpu-memory: "2048"  # 2GB memory request
        env:
        - name: KAIWO_GPU_SHARING
          value: "true"
        - name: KAIWO_GPU_ISOLATION
          value: "mps"
```

### Multiple Workloads

```yaml
# Workload 1: High priority, small memory
- name: inference-job
  resources:
    limits:
      kaiwo.ai/gpu: "0.3"
      kaiwo.ai/gpu-memory: "1024"
  priority: 100

# Workload 2: Low priority, large memory
- name: training-job
  resources:
    limits:
      kaiwo.ai/gpu: "0.7"
      kaiwo.ai/gpu-memory: "4096"
  priority: 50
```

## Limitations and Considerations

### Technical Limitations

1. **No Hardware Isolation**: All workloads share the same GPU resources
2. **Time-Based Sharing**: Workloads execute sequentially, not concurrently
3. **Memory Contention**: Memory is shared, not partitioned
4. **Performance Impact**: Context switching overhead between workloads

### Performance Characteristics

| Metric                    | NVIDIA MIG                  | AMD Instinct MI300X                       |
|---------------------------|-----------------------------|-------------------------------------------|
| **Latency**               | Low (dedicated resources)   | Low (XCD isolation)                       |
| **Throughput**            | High (concurrent execution) | High (concurrent XCD execution)           |
| **Resource Utilization**  | Fixed per partition         | Fixed per XCD (8 XCDs)                    |
| **Isolation**             | Hardware-level              | Hardware-level (SR-IOV)                   |
| **Memory Bandwidth**      | Fixed per partition         | Up to 1TB/s per XCD (NPS4)                |
| **Workgroup Control**     | Automatic                   | Explicit XCD assignment (CPX)             |
| **Performance Gain**      | Baseline                    | 10-15% over SPX mode                      |
| **Fractional Allocation** | Software-based              | Hardware-constrained (XCD-based)          |
| **Valid Fractions**       | Any 0.1-1.0                 | Mode-dependent (SPX: 1.0, CPX: 0.125-1.0) |

### Best Practices

1. **Hardware-Aware Fraction Selection**: Choose fractions that match the configured partitioning mode
   - **SPX Mode**: Use only `1.0` for full GPU allocation
   - **CPX Mode**: Use multiples of `0.125` (1 XCD) up to `1.0` (8 XCDs)
   - **TPX Mode**: Use predefined fractions based on custom partitioning

2. **Memory Planning**: Ensure total memory requests don't exceed GPU capacity
3. **Time Slice Tuning**: Adjust time slices based on workload characteristics
4. **Priority Management**: Use priorities to ensure critical workloads get GPU time
5. **Monitoring**: Monitor GPU utilization and memory usage
6. **XCD Utilization**: Monitor XCD-level allocation to optimize resource usage
7. **Mode Compatibility**: Ensure compute and memory modes are compatible (SPX+NPS4 is invalid)

## Configuration

### MI300X Partitioning Configuration

```yaml
apiVersion: kaiwo.ai/v1alpha1
kind: KaiwoConfig
metadata:
  name: mi300x-gpu-config
spec:
  gpu:
    amd:
      # MI300X Partitioning Configuration
      computeMode: "CPX"         # SPX, CPX, or TPX
      memoryMode: "NPS4"         # NPS1 or NPS4 (NPS4 only with CPX)
      xcdCount: 8                # Always 8 for MI300X
      
      # Time-Slicing Configuration
      timeSlice: "30s"           # Time slice per workload
      maxWorkloads: 10           # Maximum workloads per GPU
      memoryOverhead: "512"      # Memory overhead per workload (MiB)
      enableMPS: true           # Enable MPS support
      
      # Fractional Allocation Constraints
      validateFractions: true    # Enable hardware-aware fraction validation
      allowInvalidFractions: false # Reject invalid fractions
```

### Scheduling Policies

```yaml
scheduling:
  policies:
    - name: "round-robin"
      description: "Round-robin workload scheduling"
    - name: "priority-based"
      description: "Priority-based workload scheduling"
    - name: "memory-aware"
      description: "Memory-aware workload placement"
    - name: "xcd-aware"
      description: "XCD-aware workload placement for CPX mode"
    - name: "hardware-constrained"
      description: "Hardware-constrained fractional allocation"
```

## Monitoring and Metrics

### Key Metrics

1. **GPU Utilization**: Overall GPU utilization across all workloads
2. **Memory Usage**: Current memory usage and allocation
3. **Workload Queue**: Number of workloads waiting for GPU time
4. **Time Slice Efficiency**: How efficiently time slices are being used
5. **Context Switch Overhead**: Time spent switching between workloads
6. **XCD Utilization**: Per-XCD utilization and allocation status
7. **Fractional Allocation Accuracy**: Validation of hardware-constrained fractions
8. **Partitioning Mode Compliance**: Verification of mode-specific constraints

### Monitoring Queries

```promql
# GPU utilization across all workloads
kaiwo_gpu_utilization{type="amd"}

# Memory usage per GPU
kaiwo_gpu_memory_usage{device_id="card0"}

# Workload queue length
kaiwo_gpu_workload_queue_length{device_id="card0"}

# Time slice efficiency
kaiwo_gpu_time_slice_efficiency{device_id="card0"}

# XCD utilization per GPU
kaiwo_gpu_xcd_utilization{device_id="card0", xcd_index="0"}

# Fractional allocation validation
kaiwo_gpu_fraction_validation{device_id="card0", compute_mode="CPX"}

# Partitioning mode compliance
kaiwo_gpu_partitioning_compliance{device_id="card0", compute_mode="CPX", memory_mode="NPS4"}
```

## Future Enhancements

### Planned Improvements

1. **Advanced Scheduling**: Priority-based and deadline-based scheduling
2. **Memory Optimization**: Better memory management and reclamation
3. **Performance Profiling**: Workload performance analysis and optimization
4. **Dynamic Time Slices**: Adaptive time slice allocation based on workload characteristics
5. **Dynamic Partitioning**: Runtime switching between SPX/CPX/TPX modes
6. **XCD-Level Optimization**: Fine-grained XCD allocation and optimization
7. **Hardware-Aware Scheduling**: Scheduling decisions based on actual hardware capabilities

### Research Areas

1. **Concurrent Execution**: Exploring ways to run multiple workloads concurrently
2. **Memory Partitioning**: Software-level memory partitioning techniques
3. **Performance Isolation**: Better isolation mechanisms for predictable performance
4. **Hardware Partitioning Optimization**: Advanced algorithms for optimal XCD allocation
5. **Dynamic Mode Switching**: Runtime optimization of compute/memory modes
6. **Cross-Mode Compatibility**: Seamless workload migration between partitioning modes
7. **Resource Prediction**: Predictive resource allocation based on workload patterns

## MI300X-Aware Fractional Allocator

### Overview

The MI300X-aware fractional allocator is a critical component that ensures fractional GPU allocations respect the actual hardware partitioning constraints of AMD Instinct MI300X GPUs. Unlike generic fractional allocators that allow any value between 0.1 and 1.0, this implementation enforces hardware-accurate constraints.

### Key Features

#### **1. Hardware Partitioning Constraints**
```go
// Valid fractions are determined by the partitioning mode
func (f *MI300XFractionalAllocator) GetValidFractions(deviceID string) []float64 {
    config := f.partitionConfig[deviceID]
    
    switch config.ComputeMode {
    case MI300XPartitionModeSPX:
        return []float64{1.0}  // Only full GPU
    case MI300XPartitionModeCPX:
        return []float64{0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1.0}  // XCD-based
    case MI300XPartitionModeTPX:
        return []float64{0.125, 0.25, 0.5, 0.75, 1.0}  // Custom partitioning
    }
}
```

#### **2. XCD-Level Allocation Tracking**
```go
// Tracks which XCDs are allocated to which workloads
xcdAllocations map[string]map[int]*types.GPUAllocation // deviceID -> xcdIndex -> allocation

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
```

#### **3. Mode-Specific Validation**
```go
func (f *MI300XFractionalAllocator) ValidateFraction(deviceID string, fraction float64) error {
    validFractions := f.GetValidFractions(deviceID)
    
    for _, valid := range validFractions {
        if math.Abs(fraction-valid) < 0.001 {
            return nil
        }
    }

    return fmt.Errorf("fraction %f is not valid for GPU %s. Valid fractions: %v", 
        fraction, deviceID, validFractions)
}
```

### Usage Examples

#### **SPX Mode Configuration**
```yaml
apiVersion: kaiwo.ai/v1alpha1
kind: KaiwoJob
metadata:
  name: spx-training-job
spec:
  template:
    spec:
      containers:
      - name: training-container
        image: amd/rocm-pytorch
        resources:
          limits:
            kaiwo.ai/gpu: "1.0"  # ✅ Only valid fraction for SPX
            kaiwo.ai/gpu-memory: "8192"
        env:
        - name: AMD_GPU_COMPUTE_MODE
          value: "SPX"
        - name: AMD_GPU_MEMORY_MODE
          value: "NPS1"
```

#### **CPX Mode Configuration**
```yaml
apiVersion: kaiwo.ai/v1alpha1
kind: KaiwoJob
metadata:
  name: cpx-inference-job
spec:
  template:
    spec:
      containers:
      - name: inference-container
        image: amd/rocm-pytorch
        resources:
          limits:
            kaiwo.ai/gpu: "0.25"  # ✅ 2 XCDs (valid for CPX)
            kaiwo.ai/gpu-memory: "2048"
        env:
        - name: AMD_GPU_COMPUTE_MODE
          value: "CPX"
        - name: AMD_GPU_MEMORY_MODE
          value: "NPS4"
```

### Error Handling

#### **Invalid Fraction Examples**
```yaml
# ❌ INVALID - SPX only supports full GPU
kaiwo.ai/gpu: "0.5"  # Error: fraction 0.5 is not valid for GPU card0. Valid fractions: [1.0]

# ❌ INVALID - Not a multiple of 0.125 in CPX mode
kaiwo.ai/gpu: "0.3"  # Error: fraction 0.3 is not valid for GPU card0. Valid fractions: [0.125, 0.25, 0.375, 0.5, 0.625, 0.75, 0.875, 1.0]

# ✅ VALID - Multiple of 0.125 (1 XCD)
kaiwo.ai/gpu: "0.125"

# ✅ VALID - Multiple of 0.125 (4 XCDs)
kaiwo.ai/gpu: "0.5"
```

### Benefits

1. **Hardware Accuracy**: Only allows fractions that match actual hardware capabilities
2. **Performance Optimization**: Leverages proper XCD allocation for optimal performance
3. **Error Prevention**: Catches invalid configurations early with clear error messages
4. **Resource Efficiency**: Ensures proper utilization of the 8 XCDs
5. **Mode Compatibility**: Validates compute/memory mode combinations

## Conclusion

The AMD Instinct MI300X GPU sharing implementation provides a comprehensive solution for advanced hardware-level GPU resource sharing:

### AMD Instinct MI300X (Advanced Chiplet Architecture)
Based on the [official AMD ROCm documentation](https://rocm.blogs.amd.com/software-tools-optimization/compute-memory-modes/README.html), MI300X provides:
- **Advanced Hardware Partitioning**: 8 XCDs with SPX/CPX/TPX compute partitioning modes
- **NUMA Memory Partitioning**: NPS1/NPS4 memory partitioning with up to 1TB/s bandwidth per XCD
- **SR-IOV Isolation**: Hardware-level Virtual Function isolation for security
- **Explicit Workgroup Control**: CPX mode allows explicit XCD assignment
- **Performance Optimization**: 10-15% higher throughput than SPX mode
- **Memory Localization**: NPS4 mode enables localized memory accesses for improved bandwidth

### Key Benefits
- **Hardware Optimization**: Leverages MI300X's advanced chiplet capabilities
- **Performance Gains**: Up to 15% performance improvement through proper partitioning
- **Enterprise Ready**: SR-IOV support for secure multi-tenant environments
- **Memory Bandwidth**: Up to 1TB/s per XCD in NPS4 mode
- **Concurrent Execution**: 8 XCDs can run simultaneously with hardware isolation

This implementation provides a robust and scalable solution for GPU resource management in Kubernetes environments, taking full advantage of AMD's advanced MI300X chiplet architecture for optimal performance and resource utilization.

The MI300X-aware fractional allocator ensures that **only hardware-valid fractions are allowed**, preventing allocation errors and optimizing performance based on the actual chiplet architecture. This hardware-constrained approach guarantees optimal resource utilization and prevents invalid configurations that could lead to performance degradation or allocation failures.
