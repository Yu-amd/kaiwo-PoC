#!/bin/bash

# Kaiwo Phase 2 Examples - Apply All
# This script applies all Phase 2 examples to demonstrate advanced workload management features

set -e

echo "🚀 Applying Kaiwo Phase 2 Examples - Advanced Workload Management"
echo "=================================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to apply and wait for readiness
apply_and_wait() {
    local file=$1
    local name=$2
    
    echo -e "${BLUE}📋 Applying: ${name}${NC}"
    echo "   File: $file"
    
    if kubectl apply -f "$file"; then
        echo -e "${GREEN}✅ Successfully applied: ${name}${NC}"
    else
        echo -e "${RED}❌ Failed to apply: ${name}${NC}"
        return 1
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

# Create namespaces if they don't exist
echo -e "${BLUE}📁 Ensuring namespaces exist...${NC}"
kubectl create namespace research --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace advanced --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace research-federation --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✅ Namespaces ready${NC}"
echo ""

echo -e "${YELLOW}==================== GANG SCHEDULING EXAMPLES ====================${NC}"
echo ""

# Gang Scheduling Examples
apply_and_wait "gang-scheduling/01-distributed-training-gang.yaml" "Distributed PyTorch Training Gang"
sleep 2

apply_and_wait "gang-scheduling/02-multi-node-inference-gang.yaml" "Multi-Node LLM Inference Gang"
sleep 2

apply_and_wait "gang-scheduling/03-research-cluster-gang.yaml" "Research Experiment Cluster Gang"
sleep 2

echo -e "${YELLOW}==================== ELASTIC SCALING EXAMPLES ====================${NC}"
echo ""

# Elastic Scaling Examples
apply_and_wait "elastic-scaling/01-web-service-elastic.yaml" "AI Web Service with Elastic Scaling"
sleep 2

apply_and_wait "elastic-scaling/02-batch-processing-elastic.yaml" "Batch Processing with Elastic Scaling"
sleep 2

apply_and_wait "elastic-scaling/03-training-elastic.yaml" "Adaptive Training with Elastic Scaling"
sleep 2

echo -e "${YELLOW}==================== ADVANCED FEATURES EXAMPLES ==================${NC}"
echo ""

# Advanced Features Examples
apply_and_wait "advanced-features/01-gang-elastic-hybrid.yaml" "Gang + Elastic Hybrid Training"
sleep 2

apply_and_wait "advanced-features/02-multi-model-pipeline.yaml" "Multi-Model Inference Pipeline"
sleep 2

apply_and_wait "advanced-features/03-research-federation.yaml" "Federated Research Cluster"
sleep 2

echo ""
echo -e "${GREEN}🎉 All Phase 2 examples applied successfully!${NC}"
echo ""

echo -e "${BLUE}📊 Checking deployment status...${NC}"
echo ""

# Check status of all deployments
echo "Gang Scheduling Examples:"
kubectl get kaiwojobs -l phase=phase2-demo --all-namespaces -o wide || echo "No gang scheduling jobs found yet"
echo ""

echo "Advanced Features Examples:"
kubectl get kaiwojobs -l phase=phase2-advanced --all-namespaces -o wide || echo "No advanced features jobs found yet"
echo ""

echo -e "${YELLOW}💡 Next steps:${NC}"
echo "1. Monitor workloads: kubectl get kaiwojobs --all-namespaces -w"
echo "2. Check gang scheduling: kubectl describe kaiwojobs -l kaiwo.ai/gang-scheduling=enabled"
echo "3. Monitor elastic scaling: kubectl describe kaiwojobs -l kaiwo.ai/elastic-scaling=enabled"
echo "4. View logs: kubectl logs -l app=<app-name> -f"
echo "5. Clean up when done: ./cleanup-all-examples.sh"
echo ""

echo -e "${GREEN}🚀 Phase 2 Advanced Workload Management Examples are now running!${NC}"
