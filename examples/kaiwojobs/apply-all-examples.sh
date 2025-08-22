#!/bin/bash

# =============================================================================
# KaiwoJob Examples - Apply All Script
# =============================================================================
# This script applies all KaiwoJob examples to demonstrate Phase 1 features
# =============================================================================

set -e

echo "🚀 Applying KaiwoJob Examples..."
echo "=================================="

# Function to apply a job and show status
apply_job() {
    local file=$1
    local name=$2
    echo "📋 Applying $name..."
    kubectl apply -f "$file"
    echo "✅ $name applied successfully"
    echo ""
}

# Apply all examples
echo "1️⃣  Simple CPU Job"
apply_job "01-simple-cpu-job.yaml" "Simple CPU Job"

echo "2️⃣  AMD GPU Fractional Job"
apply_job "02-amd-gpu-fractional-job.yaml" "AMD GPU Fractional Job"

echo "3️⃣  Multi-GPU Training Job"
apply_job "03-multi-gpu-training-job.yaml" "Multi-GPU Training Job"

echo "4️⃣  Data Processing Job"
apply_job "04-data-processing-job.yaml" "Data Processing Job"

echo "5️⃣  Ray Distributed Job"
apply_job "05-ray-distributed-job.yaml" "Ray Distributed Job"

echo "6️⃣  High Priority Job"
apply_job "06-high-priority-job.yaml" "High Priority Job"

echo "7️⃣  Custom Labels Job"
apply_job "07-custom-labels-job.yaml" "Custom Labels Job"

echo "🎉 All examples applied successfully!"
echo ""
echo "📊 Check status with:"
echo "   kubectl get kaiwojobs"
echo "   kubectl get jobs"
echo ""
echo "🔍 Monitor scheduler usage:"
echo "   kubectl get pods --all-namespaces -o jsonpath='{range .items[*]}{.spec.schedulerName}{\"\n\"}{end}' | sort | uniq -c"
echo ""
echo "📝 View individual job details:"
echo "   kubectl describe kaiwojob <job-name>"
