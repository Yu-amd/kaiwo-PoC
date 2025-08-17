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

echo "📋 Monitoring test workload..."
kubectl logs -f job/test-gpu-job --tail=50

echo "🧹 Cleaning up test workload..."
kubectl delete -f test/manifests/test-gpu-job.yaml

echo "✅ GPU validation complete!"
