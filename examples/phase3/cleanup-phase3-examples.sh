#!/bin/bash

# Cleanup Phase 3 Examples Script
# This script removes all Phase 3 ML-driven intelligence examples and associated resources

set -e

echo "🧹 Cleaning Up Phase 3 Examples - ML-Driven Intelligence"
echo "========================================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if connected to cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Not connected to a Kubernetes cluster"
    print_warning "Please configure kubectl to connect to your cluster"
    exit 1
fi

print_status "Connected to cluster: $(kubectl config current-context)"
echo ""

# Confirmation prompt
read -p "⚠️  This will delete all Phase 3 examples and associated resources. Continue? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_warning "Cleanup cancelled by user"
    exit 0
fi

echo ""

# Array of Phase 3 examples
examples=(
    "01-ml-performance-prediction.yaml:ML Performance Prediction"
    "02-workload-analytics.yaml:Advanced Workload Analytics"
    "03-mlflow-integration.yaml:MLflow Integration"
    "04-kubeflow-optimization.yaml:Kubeflow Pipeline Optimization"
    "05-hyperparameter-tuning.yaml:Automated Hyperparameter Tuning"
    "06-resource-prediction.yaml:Intelligent Resource Prediction"
    "07-cost-optimization.yaml:Cost Optimization"
)

# Delete each example
success_count=0
total_count=${#examples[@]}

echo "🗑️  Deleting Phase 3 Examples:"
echo ""

for example in "${examples[@]}"; do
    IFS=':' read -r filename description <<< "$example"
    
    print_status "Deleting $description ($filename)..."
    
    if kubectl delete -f "$filename" --ignore-not-found=true; then
        print_success "$description deleted successfully"
        ((success_count++))
    else
        print_error "Failed to delete $description"
    fi
    sleep 1
done

echo ""
echo "🧹 Cleaning up additional resources..."

# Clean up Phase 3 specific resources
print_status "Cleaning up KaiwoJobs..."
kubectl delete kaiwojobs -l phase=3 --ignore-not-found=true

print_status "Cleaning up ConfigMaps..."
kubectl delete configmap ml-prediction-config workload-analytics-config kaiwo-mlflow-config kubeflow-optimization-config hyperparameter-tuning-config resource-prediction-config cost-optimization-config --ignore-not-found=true

print_status "Cleaning up Services..."
kubectl delete service analytics-dashboard mlflow-service pipeline-monitor tuning-dashboard capacity-dashboard cost-dashboard --ignore-not-found=true

print_status "Cleaning up Deployments..."
kubectl delete deployment mlflow-server capacity-dashboard --ignore-not-found=true

print_status "Cleaning up PersistentVolumeClaims..."
kubectl delete pvc mlflow-pvc --ignore-not-found=true

print_status "Cleaning up CronJobs..."
kubectl delete cronjob daily-cost-optimization --ignore-not-found=true

echo ""
echo "🔍 Checking for remaining Phase 3 resources..."

# Check for any remaining Phase 3 resources
remaining_jobs=$(kubectl get kaiwojobs -l phase=3 2>/dev/null | wc -l)
remaining_configmaps=$(kubectl get configmap -l phase=3 2>/dev/null | wc -l)

if [ "$remaining_jobs" -gt 1 ] || [ "$remaining_configmaps" -gt 1 ]; then
    print_warning "Some Phase 3 resources may still exist:"
    echo ""
    echo "Remaining KaiwoJobs:"
    kubectl get kaiwojobs -l phase=3 2>/dev/null || echo "  None found"
    echo ""
    echo "Remaining ConfigMaps:"
    kubectl get configmap -l phase=3 2>/dev/null || echo "  None found"
else
    print_success "All Phase 3 resources have been cleaned up"
fi

echo ""
echo "📊 Cleanup Summary:"
echo "  Successfully deleted: $success_count/$total_count examples"

if [ $success_count -eq $total_count ]; then
    print_success "All Phase 3 examples cleaned up successfully! ✨"
else
    print_warning "Some examples failed to delete. Check the errors above."
fi

echo ""
echo "🔍 Verification Commands:"
echo "  # Check for remaining KaiwoJobs"
echo "  kubectl get kaiwojobs"
echo ""
echo "  # Check for remaining Phase 3 resources"
echo "  kubectl get all -l phase=3"
echo ""
echo "  # Check for remaining ConfigMaps"
echo "  kubectl get configmaps | grep -E 'ml-prediction|analytics|mlflow|kubeflow|tuning|resource|cost'"
echo ""

echo "🗂️  Phase 3 Cleanup Actions Performed:"
echo "  ✅ Deleted all Phase 3 example KaiwoJobs"
echo "  ✅ Removed ML prediction and analytics configurations"
echo "  ✅ Cleaned up MLflow and Kubeflow integration resources"
echo "  ✅ Removed hyperparameter tuning experiments"
echo "  ✅ Deleted resource prediction and cost optimization services"
echo "  ✅ Cleaned up associated ConfigMaps, Services, and Deployments"
echo "  ✅ Removed persistent storage claims"
echo "  ✅ Deleted scheduled cost optimization jobs"
echo ""

echo "🎯 Next Steps:"
echo "  1. Verify cleanup: kubectl get kaiwojobs"
echo "  2. Check cluster resources: kubectl get all"
echo "  3. Review any remaining resources manually if needed"
echo "  4. Re-run examples anytime with: ./apply-all-phase3-examples.sh"
echo ""

print_success "Phase 3 ML-driven intelligence examples cleanup completed! 🧹✨"

# Optional: Show cluster resource usage after cleanup
echo ""
print_status "Current cluster resource usage:"
kubectl top nodes 2>/dev/null || print_warning "Metrics server not available for resource usage display"
