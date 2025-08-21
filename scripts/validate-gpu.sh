#!/bin/bash
set -e

echo "🔍 Validating GPU setup on remote cluster..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ Error: kubectl is not installed or not in PATH"
    exit 1
fi

echo "📊 Checking cluster nodes..."
kubectl get nodes

echo "🎯 Checking GPU detection..."
GPU_COUNT=$(kubectl get nodes -o json | jq -r '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))' | wc -l)

if [ "$GPU_COUNT" -gt 0 ]; then
    echo "✅ AMD GPUs detected!"
    kubectl get nodes -o json | jq '.items[].status.allocatable | keys | .[] | select(contains("amd.com/gpu"))'
else
    echo "❌ No AMD GPUs detected. Please check AMD GPU Operator installation."
    exit 1
fi

echo "🧪 Deploying test GPU workload..."
kubectl apply -f test/manifests/test-gpu-job.yaml

# Check if the job was created successfully
if kubectl get job amd-gpu-test &>/dev/null; then
    echo "✅ GPU test job created successfully"
else
    echo "❌ Failed to create GPU test job"
    exit 1
fi

echo "📋 Monitoring test workload..."
# Wait for the job to be ready
echo "⏳ Waiting for job to be ready..."
for i in {1..30}; do
    if kubectl get job amd-gpu-test | grep -q "1/1"; then
        echo "✅ Job is ready!"
        break
    fi
    echo "Waiting for job to be ready... (attempt $i/30)"
    kubectl get job amd-gpu-test
    sleep 5
done

# Check job status
echo "📊 Job status:"
kubectl get job amd-gpu-test
kubectl get pods | grep amd-gpu-test

# Get the logs
echo "📋 Job logs:"
kubectl logs job/amd-gpu-test --tail=50

echo "🧹 Cleaning up test workload..."
kubectl delete -f test/manifests/test-gpu-job.yaml

echo "✅ GPU validation complete!"
