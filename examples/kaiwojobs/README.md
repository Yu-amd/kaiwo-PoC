# KaiwoJob Examples

This directory contains individual example YAML files for different KaiwoJob use cases. Each example demonstrates specific Phase 1 features of the Kaiwo project.

**Note**: These examples use a simplified CRD structure due to the current installation. The examples focus on basic resource management and GPU allocation.

## 📁 Available Examples

### 1. **01-simple-cpu-job.yaml**
- **Purpose**: Basic CPU-only workload
- **Use Case**: Data processing, web services, batch jobs
- **Features**: Simple resource management
- **Apply**: `kubectl apply -f 01-simple-cpu-job.yaml`

### 2. **02-amd-gpu-fractional-job.yaml**
- **Purpose**: AMD GPU with fractional allocation
- **Use Case**: GPU sharing, research workloads
- **Features**: 
  - AMD GPU allocation (`amd.com/gpu: 1`)
  - GPU annotations for fractional allocation
  - AMD time-slicing support
- **Apply**: `kubectl apply -f 02-amd-gpu-fractional-job.yaml`

### 3. **03-multi-gpu-training-job.yaml**
- **Purpose**: Multi-GPU training workloads
- **Use Case**: Distributed training, large model training
- **Features**: 
  - Multi-GPU support (`amd.com/gpu: 2`)
  - Distributed training resources
- **Apply**: `kubectl apply -f 03-multi-gpu-training-job.yaml`

### 4. **04-data-processing-job.yaml**
- **Purpose**: Data processing workload
- **Use Case**: ML data pipelines, model training
- **Features**:
  - GPU-accelerated data processing
  - Resource management
- **Apply**: `kubectl apply -f 04-data-processing-job.yaml`

### 5. **05-ray-distributed-job.yaml**
- **Purpose**: Ray distributed computing
- **Use Case**: Distributed ML workloads, parallel processing
- **Features**:
  - Ray container with GPU support
  - Distributed computing resources
- **Apply**: `kubectl apply -f 05-ray-distributed-job.yaml`

### 6. **06-high-priority-job.yaml**
- **Purpose**: High priority jobs
- **Use Case**: Research workloads, urgent processing
- **Features**:
  - Priority labels
  - Queue management labels
- **Apply**: `kubectl apply -f 06-high-priority-job.yaml`

### 7. **07-custom-labels-job.yaml**
- **Purpose**: Jobs with custom labeling
- **Use Case**: Production workloads, team management
- **Features**:
  - Custom labels for organization
  - Production-ready configuration
- **Apply**: `kubectl apply -f 07-custom-labels-job.yaml`

## 🚀 Quick Start

### Prerequisites
1. Kaiwo scheduler is running
2. AMD GPU nodes are available
3. KaiwoJob CRD is installed

### Basic Usage
```bash
# Apply a simple CPU job
kubectl apply -f 01-simple-cpu-job.yaml

# Check job status
kubectl get kaiwojobs

# View job details
kubectl describe kaiwojob simple-cpu-job
```

### AMD GPU Testing
```bash
# Apply AMD GPU job
kubectl apply -f 02-amd-gpu-fractional-job.yaml

# Check GPU allocation
kubectl get kaiwojob amd-gpu-fractional-job -o jsonpath='{.spec.template.spec.containers[0].resources}'

# View job status
kubectl describe kaiwojob amd-gpu-fractional-job
```

## 🔧 Phase 1 Features Demonstrated

### Enhanced GPU Management
- ✅ AMD GPU allocation (`amd.com/gpu`)
- ✅ GPU annotations for fractional allocation
- ✅ AMD time-slicing support
- ✅ Multi-GPU support

### Enhanced Scheduling
- ✅ Resource-aware allocation
- ✅ AMD GPU optimization
- ✅ Priority scheduling (via labels)

### Enhanced Monitoring
- ✅ Resource monitoring
- ✅ Job status tracking

### Plugin Architecture
- ✅ Basic plugin system demonstration
- ✅ GPU management integration

## 📊 Monitoring Commands

```bash
# Check all KaiwoJobs
kubectl get kaiwojobs

# Check job details
kubectl describe kaiwojob <job-name>

# Check AMD GPU resources
kubectl get nodes -o jsonpath='{.items[0].status.capacity["amd.com/gpu"]}'

# Check scheduler usage
kubectl get pods --all-namespaces -o jsonpath='{range .items[*]}{.spec.schedulerName}{"\n"}{end}' | sort | uniq -c
```

## 🎯 Customization

Each example can be customized by modifying:

- **Resources**: Adjust CPU/memory/GPU requirements
- **Image**: Change container image
- **Labels**: Add custom labels for organization
- **Annotations**: Add GPU-specific annotations

## 🔍 Troubleshooting

### Common Issues
1. **Job not scheduled**: Check if Kaiwo scheduler is running
2. **GPU not available**: Verify AMD GPU nodes and resources
3. **CRD issues**: Ensure KaiwoJob CRD is properly installed

### Debug Commands
```bash
# Check scheduler logs
kubectl logs -n kube-system deployment/kaiwo-scheduler

# Check job events
kubectl describe kaiwojob <job-name>

# Check resource availability
kubectl describe node <node-name>
```

## 📝 Notes

- All examples use the **simplified CRD structure** with `spec.template`
- AMD GPU examples require **AMD GPU nodes** with ROCm support
- The current CRD supports basic resource management
- GPU annotations demonstrate Phase 1 features
- These are **simplified examples** due to CRD limitations

## 🔄 CRD Limitations

The current KaiwoJob CRD installation supports:
- ✅ Basic container specification
- ✅ Resource requests and limits
- ✅ AMD GPU allocation
- ✅ Custom labels and annotations

**Not supported in current CRD:**
- ❌ Advanced job specifications
- ❌ Ray job integration
- ❌ Storage management
- ❌ Complex command/args
- ❌ Environment variables

---

**These examples demonstrate the basic Phase 1 implementation with the current CRD structure!** 🎉
