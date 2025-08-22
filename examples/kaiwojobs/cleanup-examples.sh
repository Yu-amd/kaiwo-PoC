#!/bin/bash

# =============================================================================
# Kaiwo Examples - Cleanup Script
# =============================================================================
# This script removes all KaiwoJob examples and associated resources
# =============================================================================

set -e

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

# Remove specific example resources by name
echo "🔍 Removing specific example resources..."
kubectl delete kaiwojob simple-cpu-job --ignore-not-found=true
kubectl delete kaiwojob amd-gpu-fractional-job --ignore-not-found=true
kubectl delete kaiwojob multi-gpu-training-job --ignore-not-found=true
kubectl delete kaiwojob data-processing-job --ignore-not-found=true
kubectl delete kaiwojob ray-distributed-job --ignore-not-found=true
kubectl delete kaiwojob high-priority-research-job --ignore-not-found=true
kubectl delete kaiwojob custom-labels-job --ignore-not-found=true

# Verify cleanup
echo "✅ Verifying cleanup..."
echo "KaiwoJobs remaining:"
kubectl get kaiwojobs --no-headers | wc -l

echo "Associated pods remaining:"
kubectl get pods -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

echo "Associated jobs remaining:"
kubectl get jobs -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

echo "Associated services remaining:"
kubectl get services -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

echo "Associated configmaps remaining:"
kubectl get configmaps -l kaiwo.silogen.ai/managed=true --no-headers | wc -l

# Final verification
echo ""
echo "🔍 Final verification..."
if kubectl get kaiwojobs --no-headers | wc -l | grep -q "0"; then
    echo "✅ All KaiwoJobs successfully removed!"
else
    echo "⚠️  Some KaiwoJobs may still exist. Check with: kubectl get kaiwojobs"
fi

echo "🎉 Cleanup completed!"
echo ""
echo "📝 To reapply examples, run: ./apply-all-examples.sh"
