#!/bin/bash

# Kaiwo Phase 2 Examples - Cleanup All
# This script removes all Phase 2 examples and associated resources

set -e

echo "🧹 Cleaning up Kaiwo Phase 2 Examples - Advanced Workload Management"
echo "====================================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to delete resources safely
delete_resources() {
    local file=$1
    local name=$2
    
    echo -e "${BLUE}🗑️  Deleting: ${name}${NC}"
    echo "   File: $file"
    
    if kubectl delete -f "$file" --ignore-not-found=true; then
        echo -e "${GREEN}✅ Successfully deleted: ${name}${NC}"
    else
        echo -e "${YELLOW}⚠️  Resource not found or already deleted: ${name}${NC}"
    fi
    
    echo ""
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl is not installed or not in PATH${NC}"
    exit 1
fi

# Check cluster connectivity
echo -e "${BLUE}🔍 Checking cluster connectivity...${NC}"
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Connected to cluster${NC}"
echo ""

echo -e "${YELLOW}==================== CLEANING ADVANCED FEATURES ==================${NC}"
echo ""

# Advanced Features (delete first to avoid dependencies)
delete_resources "advanced-features/01-gang-elastic-hybrid.yaml" "Gang + Elastic Hybrid Training"
delete_resources "advanced-features/02-multi-model-pipeline.yaml" "Multi-Model Inference Pipeline" 
delete_resources "advanced-features/03-research-federation.yaml" "Federated Research Cluster"

echo -e "${YELLOW}==================== CLEANING ELASTIC SCALING ====================${NC}"
echo ""

# Elastic Scaling Examples
delete_resources "elastic-scaling/01-web-service-elastic.yaml" "AI Web Service with Elastic Scaling"
delete_resources "elastic-scaling/02-batch-processing-elastic.yaml" "Batch Processing with Elastic Scaling"
delete_resources "elastic-scaling/03-training-elastic.yaml" "Adaptive Training with Elastic Scaling"

echo -e "${YELLOW}==================== CLEANING GANG SCHEDULING ====================${NC}"
echo ""

# Gang Scheduling Examples
delete_resources "gang-scheduling/01-distributed-training-gang.yaml" "Distributed PyTorch Training Gang"
delete_resources "gang-scheduling/02-multi-node-inference-gang.yaml" "Multi-Node LLM Inference Gang"
delete_resources "gang-scheduling/03-research-cluster-gang.yaml" "Research Experiment Cluster Gang"

echo ""
echo -e "${BLUE}🔍 Checking for remaining Phase 2 resources...${NC}"

# Check for any remaining Phase 2 resources
echo "Remaining Phase 2 Demo resources:"
kubectl get kaiwojobs -l phase=phase2-demo --all-namespaces || echo "No Phase 2 demo resources found"
echo ""

echo "Remaining Phase 2 Advanced resources:"
kubectl get kaiwojobs -l phase=phase2-advanced --all-namespaces || echo "No Phase 2 advanced resources found"
echo ""

# Clean up any lingering resources by labels
echo -e "${BLUE}🧹 Cleaning up by labels...${NC}"
kubectl delete kaiwojobs -l phase=phase2-demo --all-namespaces --ignore-not-found=true
kubectl delete kaiwojobs -l phase=phase2-advanced --all-namespaces --ignore-not-found=true
kubectl delete kaiwojobs -l kaiwo.ai/gang-scheduling=enabled --all-namespaces --ignore-not-found=true
kubectl delete kaiwojobs -l kaiwo.ai/elastic-scaling=enabled --all-namespaces --ignore-not-found=true

echo ""
echo -e "${BLUE}🔍 Final cleanup verification...${NC}"

# Final check
echo "All KaiwoJobs in cluster:"
kubectl get kaiwojobs --all-namespaces || echo "No KaiwoJobs found"
echo ""

# Clean up any completed/failed pods
echo -e "${BLUE}🧹 Cleaning up completed/failed pods...${NC}"
kubectl delete pods --field-selector=status.phase=Succeeded --all-namespaces --ignore-not-found=true
kubectl delete pods --field-selector=status.phase=Failed --all-namespaces --ignore-not-found=true

echo ""
echo -e "${GREEN}🎉 Phase 2 cleanup completed successfully!${NC}"
echo ""

echo -e "${YELLOW}💡 Cleanup summary:${NC}"
echo "✅ All Phase 2 gang scheduling examples removed"
echo "✅ All Phase 2 elastic scaling examples removed" 
echo "✅ All Phase 2 advanced features examples removed"
echo "✅ Associated pods and resources cleaned up"
echo "✅ Namespaces preserved (in case other workloads are running)"
echo ""

echo -e "${BLUE}📋 If you want to completely remove the namespaces:${NC}"
echo "kubectl delete namespace research --ignore-not-found=true"
echo "kubectl delete namespace advanced --ignore-not-found=true"
echo "kubectl delete namespace research-federation --ignore-not-found=true"
echo ""

echo -e "${GREEN}🚀 Ready for new Phase 2 examples or proceeding to Phase 3!${NC}"
