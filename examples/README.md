# Kaiwo Examples

This directory contains comprehensive examples demonstrating both **Phase 1** and **Phase 2** implementations of the Kaiwo project with advanced AMD GPU support and intelligent workload management.

## 🎯 **What's Available**

- **Phase 1**: Core Infrastructure Enhancement (Basic KaiwoJobs, GPU Management)
- **Phase 2**: Advanced Workload Management (Gang Scheduling, Elastic Scaling)

## 📁 Directory Structure

```
examples/
├── README.md                    # This file - comprehensive overview
├── your-kaiwojob.yaml          # Combined examples file (legacy)
├── kaiwojobs/                  # Phase 1: Basic KaiwoJob examples
│   ├── README.md               # Detailed kaiwojobs documentation
│   ├── apply-all-examples.sh   # Script to apply Phase 1 examples
│   ├── 01-simple-cpu-job.yaml
│   ├── 02-amd-gpu-fractional-job.yaml
│   ├── 03-multi-gpu-training-job.yaml
│   ├── 04-data-processing-job.yaml
│   ├── 05-ray-distributed-job.yaml
│   ├── 06-high-priority-job.yaml
│   └── 07-custom-labels-job.yaml
└── phase2/                     # Phase 2: Advanced Workload Management
    ├── README.md               # Comprehensive Phase 2 documentation
    ├── apply-all-examples.sh   # Deploy all Phase 2 examples
    ├── cleanup-all-examples.sh # Clean up all Phase 2 examples
    ├── gang-scheduling/        # Gang scheduling examples
    │   ├── 01-distributed-training-gang.yaml
    │   ├── 02-multi-node-inference-gang.yaml
    │   └── 03-research-cluster-gang.yaml
    ├── elastic-scaling/        # Elastic scaling examples
    │   ├── 01-web-service-elastic.yaml
    │   ├── 02-batch-processing-elastic.yaml
    │   └── 03-training-elastic.yaml
    └── advanced-features/      # Hybrid and advanced examples
        ├── 01-gang-elastic-hybrid.yaml
        ├── 02-multi-model-pipeline.yaml
        └── 03-research-federation.yaml
```

## 🚀 Quick Start

### Prerequisites
1. **Kaiwo scheduler** is running (`kaiwo-scheduler`)
2. **AMD GPU nodes** are available with ROCm support
3. **KaiwoJob CRD** is installed
4. **Kubernetes cluster** is accessible

### Phase 1: Basic Usage

```bash
# Navigate to Phase 1 examples
cd examples/kaiwojobs

# Apply a simple CPU job
kubectl apply -f 01-simple-cpu-job.yaml

# Apply all Phase 1 examples
./apply-all-examples.sh

# Check status
kubectl get kaiwojobs
```

### Phase 2: Advanced Workload Management

```bash
# Navigate to Phase 2 examples  
cd examples/phase2

# Apply all Phase 2 examples (gang scheduling + elastic scaling)
./apply-all-examples.sh

# Monitor gang scheduling
kubectl get kaiwojobs -l kaiwo.ai/gang-scheduling=enabled --all-namespaces

# Monitor elastic scaling
kubectl get kaiwojobs -l kaiwo.ai/elastic-scaling=enabled --all-namespaces

# Clean up when done
./cleanup-all-examples.sh
```

### Quick Feature Demo

```bash
# Demo gang scheduling for distributed training
kubectl apply -f phase2/gang-scheduling/01-distributed-training-gang.yaml

# Demo elastic scaling for web service
kubectl apply -f phase2/elastic-scaling/01-web-service-elastic.yaml

# Demo hybrid gang + elastic scaling
kubectl apply -f phase2/advanced-features/01-gang-elastic-hybrid.yaml
```

## 🧹 Cleanup Instructions

### Phase 1 Cleanup
```bash
cd examples/kaiwojobs
./cleanup-examples.sh
```

### Phase 2 Cleanup
```bash
cd examples/phase2
./cleanup-all-examples.sh
```

### Complete Cleanup (All Phases)

```bash
# Remove all KaiwoJobs at once
kubectl delete kaiwojobs --all

# Verify cleanup
kubectl get kaiwojobs
```

### Individual Cleanup

```bash
# Remove specific jobs
kubectl delete kaiwojob simple-cpu-job
kubectl delete kaiwojob amd-gpu-fractional-job
kubectl delete kaiwojob multi-gpu-training-job
kubectl delete kaiwojob data-processing-job
kubectl delete kaiwojob ray-distributed-job
kubectl delete kaiwojob high-priority-research-job
kubectl delete kaiwojob custom-labels-job
```

### Comprehensive Cleanup

```bash
# 1. Remove all KaiwoJobs
kubectl delete kaiwojobs --all

# 2. Remove any associated pods (if they exist)
kubectl delete pods -l kaiwo.silogen.ai/managed=true

# 3. Remove any associated jobs (if they exist)
kubectl delete jobs -l kaiwo.silogen.ai/managed=true

# 4. Remove any associated services (if they exist)
kubectl delete services -l kaiwo.silogen.ai/managed=true

# 5. Remove any associated configmaps (if they exist)
kubectl delete configmaps -l kaiwo.silogen.ai/managed=true

# 6. Verify all resources are cleaned up
kubectl get kaiwojobs
kubectl get pods -l kaiwo.silogen.ai/managed=true
kubectl get jobs -l kaiwo.silogen.ai/managed=true
```

### Cleanup Script

Create a cleanup script for convenience:

```bash
# Create cleanup script
cat > examples/kaiwojobs/cleanup-examples.sh << 'EOF'
#!/bin/bash

echo "🧹 Cleaning up Kaiwo Examples..."
echo "=================================="

# Remove all KaiwoJobs
echo "📋 Removing KaiwoJobs..."
kubectl delete kaiwojobs --all --ignore-not-found=true

# Remove associated resources
echo "🗑️  Removing associated resources..."
kubectl delete pods -l kaiwo.silogen.ai/managed=true --ignore-not-found=true
kubectl delete jobs -l kaiwo.silogen.ai/managed=true --ignore-not-found=true
kubectl delete services -l kaiwo.silogen.ai/managed=true --ignore-not-found=true
kubectl delete configmaps -l kaiwo.silogen.ai/managed=true --ignore-not-found=true

# Verify cleanup
echo "✅ Verifying cleanup..."
echo "KaiwoJobs:"
kubectl get kaiwojobs --no-headers | wc -l

echo "Associated pods:"
kubectl get pods -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

echo "Associated jobs:"
kubectl get jobs -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

echo "🎉 Cleanup completed!"
EOF

# Make it executable
chmod +x examples/kaiwojobs/cleanup-examples.sh
```

### Cleanup Verification

After cleanup, verify that all resources are removed:

```bash
# Check for remaining KaiwoJobs
kubectl get kaiwojobs

# Check for remaining managed resources
kubectl get pods -l kaiwo.silogen.ai/managed=true
kubectl get jobs -l kaiwo.silogen.ai/managed=true
kubectl get services -l kaiwo.silogen.ai/managed=true

# Check for any orphaned resources
kubectl get all --all-namespaces | grep -E "(simple-cpu|amd-gpu|multi-gpu|data-processing|ray-distributed|high-priority|custom-labels)"
```

### Troubleshooting Cleanup

If resources are stuck or not deleting:

```bash
# Force delete stuck resources
kubectl delete kaiwojob <job-name> --grace-period=0 --force

# Check finalizers
kubectl get kaiwojob <job-name> -o jsonpath='{.metadata.finalizers}'

# Remove finalizers if needed
kubectl patch kaiwojob <job-name> -p '{"metadata":{"finalizers":[]}}' --type=merge

# Check for stuck pods
kubectl get pods --field-selector=status.phase=Terminating

# Force delete stuck pods
kubectl delete pod <pod-name> --grace-period=0 --force
```

## 📋 Available Examples

### Individual Files (Recommended)

The `kaiwojobs/` directory contains **7 separate YAML files**, each demonstrating specific Phase 1 features:

1. **`01-simple-cpu-job.yaml`** - Basic CPU workload
2. **`02-amd-gpu-fractional-job.yaml`** - AMD GPU with fractional allocation
3. **`03-multi-gpu-training-job.yaml`** - Multi-GPU training
4. **`04-data-processing-job.yaml`** - Data processing workload
5. **`05-ray-distributed-job.yaml`** - Ray distributed computing
6. **`06-high-priority-job.yaml`** - High priority job
7. **`07-custom-labels-job.yaml`** - Custom labeling

### Combined File (Legacy)

- **`your-kaiwojob.yaml`** - Contains all examples in a single file (legacy format)

## 🔧 Phase 1 Features Demonstrated

### Enhanced GPU Management
- ✅ **AMD GPU allocation** (`amd.com/gpu`)
- ✅ **GPU annotations** for fractional allocation
- ✅ **AMD time-slicing** support
- ✅ **Multi-GPU support** for distributed training

### Enhanced Scheduling
- ✅ **Resource-aware allocation**
- ✅ **AMD GPU optimization**
- ✅ **Priority scheduling** (via labels)

### Enhanced Monitoring
- ✅ **Resource monitoring**
- ✅ **Job status tracking**

### Plugin Architecture
- ✅ **Basic plugin system** demonstration
- ✅ **GPU management integration**

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

# Check scheduler logs
kubectl logs -n kube-system deployment/kaiwo-scheduler
```

## 🎯 Example Details

### 1. Simple CPU Job
```yaml
# Demonstrates basic resource management
spec:
  template:
    spec:
      containers:
      - name: cpu-worker
        image: busybox:latest
        resources:
          requests:
            cpu: 1
            memory: 2Gi
          limits:
            cpu: 2
            memory: 4Gi
```

### 2. AMD GPU Fractional Job
```yaml
# Demonstrates AMD GPU allocation with annotations
metadata:
  annotations:
    kaiwo.ai/gpu-fraction: "0.5"
    kaiwo.ai/gpu-isolation: "time-slicing"
spec:
  template:
    spec:
      containers:
      - name: gpu-worker
        image: amd/pytorch:rocm5.6
        resources:
          requests:
            amd.com/gpu: 1
```

### 3. Multi-GPU Training Job
```yaml
# Demonstrates multi-GPU support
spec:
  template:
    spec:
      containers:
      - name: training-worker
        image: amd/pytorch:rocm5.6
        resources:
          requests:
            amd.com/gpu: 2
```

## 🔍 Troubleshooting

### Common Issues

1. **Job not scheduled**
   ```bash
   # Check if Kaiwo scheduler is running
   kubectl get deployment -n kube-system kaiwo-scheduler
   kubectl logs -n kube-system deployment/kaiwo-scheduler
   ```

2. **GPU not available**
   ```bash
   # Check AMD GPU nodes
   kubectl get nodes -l amd.com/gpu
   kubectl describe node <node-name>
   ```

3. **CRD issues**
   ```bash
   # Check CRD installation
   kubectl get crd kaiwojobs.kaiwo.silogen.ai
   ```

### Debug Commands

```bash
# Check job events
kubectl describe kaiwojob <job-name>

# Check pod events
kubectl get pods -l job-name=<job-name>

# Check resource availability
kubectl describe node <node-name>

# Check scheduler configuration
kubectl get configmap -n kube-system kaiwo-scheduler-config -o yaml
```

## 🔄 CRD Limitations

The current KaiwoJob CRD installation supports:
- ✅ **Basic container specification**
- ✅ **Resource requests and limits**
- ✅ **AMD GPU allocation**
- ✅ **Custom labels and annotations**

**Not supported in current CRD:**
- ❌ Advanced job specifications
- ❌ Ray job integration
- ❌ Storage management
- ❌ Complex command/args
- ❌ Environment variables

## 📝 Notes

- All examples use the **simplified CRD structure** with `spec.template`
- AMD GPU examples require **AMD GPU nodes** with ROCm support
- The current CRD supports basic resource management
- GPU annotations demonstrate Phase 1 features
- These are **simplified examples** due to CRD limitations

## 🎯 Customization

Each example can be customized by modifying:

- **Resources**: Adjust CPU/memory/GPU requirements
- **Image**: Change container image
- **Labels**: Add custom labels for organization
- **Annotations**: Add GPU-specific annotations

## 📚 Additional Resources

- **Detailed Documentation**: See `kaiwojobs/README.md` for comprehensive examples documentation
- **Phase 1 Summary**: See `PHASE1-IMPLEMENTATION-SUMMARY.md` for complete implementation details
- **Demo Scripts**: See `scripts/demo-phase1.sh` for interactive demonstrations
- **Command Line Demo**: See `COMMAND-LINE-DEMO.md` for step-by-step instructions

## 🏆 Success Verification

To verify that everything is working:

```bash
# 1. Apply all examples
cd examples/kaiwojobs
./apply-all-examples.sh

# 2. Check all jobs are created
kubectl get kaiwojobs

# 3. Verify GPU allocation
kubectl describe kaiwojob amd-gpu-fractional-job

# 4. Check scheduler is working
kubectl get deployment -n kube-system kaiwo-scheduler
```

**Expected Output:**
```
NAME                         AGE
amd-gpu-fractional-job       45s
custom-labels-job            41s
data-processing-job          5s
high-priority-research-job   5s
multi-gpu-training-job       5s
ray-distributed-job          5s
simple-cpu-job               2m24s
```

---

**These examples demonstrate the basic Phase 1 implementation with the current CRD structure!** 🎉

For more detailed information, see the individual README files in each subdirectory.
