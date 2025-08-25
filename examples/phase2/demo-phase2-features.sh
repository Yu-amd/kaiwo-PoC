#!/bin/bash

# Kaiwo Phase 2 Features Demo
# Interactive demonstration of gang scheduling and elastic scaling capabilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Demo configuration
DEMO_PAUSE=${DEMO_PAUSE:-3}  # Pause between steps (seconds)

echo -e "${PURPLE}🎯 Kaiwo Phase 2 Features Demo${NC}"
echo -e "${PURPLE}================================${NC}"
echo ""
echo -e "${CYAN}This demo showcases the advanced workload management capabilities:${NC}"
echo -e "${YELLOW}✅ Gang Scheduling${NC} - All-or-nothing scheduling for distributed workloads"
echo -e "${YELLOW}✅ Elastic Scaling${NC} - Dynamic auto-scaling based on metrics"
echo -e "${YELLOW}✅ Hybrid Features${NC} - Combined gang + elastic capabilities"
echo ""

# Function to pause and wait for user
pause_demo() {
    echo -e "${BLUE}⏱️  Press ENTER to continue (or Ctrl+C to exit)...${NC}"
    read -r
}

# Function to show a command before running it
run_demo_command() {
    local cmd=$1
    local description=$2
    
    echo -e "${YELLOW}📋 ${description}${NC}"
    echo -e "${BLUE}💻 Command: ${cmd}${NC}"
    echo ""
    
    eval "$cmd"
    echo ""
    sleep "$DEMO_PAUSE"
}

# Check prerequisites
echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found. Please install kubectl first.${NC}"
    exit 1
fi

if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Cannot connect to Kubernetes cluster.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites met${NC}"
echo ""

pause_demo

# Demo 1: Gang Scheduling
echo -e "${PURPLE}🎭 Demo 1: Gang Scheduling${NC}"
echo -e "${PURPLE}=========================${NC}"
echo ""
echo -e "${CYAN}Gang scheduling ensures all replicas of a distributed workload${NC}"
echo -e "${CYAN}are scheduled together atomically - essential for distributed training.${NC}"
echo ""

pause_demo

run_demo_command \
    "kubectl apply -f gang-scheduling/01-distributed-training-gang.yaml" \
    "Apply distributed training job with gang scheduling (4 workers)"

run_demo_command \
    "kubectl get kaiwojobs distributed-pytorch-training -o wide" \
    "Check the gang-scheduled job status"

run_demo_command \
    "kubectl describe kaiwojobs distributed-pytorch-training | grep -A 10 'Gang Scheduling'" \
    "View gang scheduling configuration details"

echo -e "${GREEN}✅ Gang scheduling demo complete!${NC}"
echo -e "${CYAN}The job requires all 4 workers to be scheduled together atomically.${NC}"
echo ""

pause_demo

# Demo 2: Elastic Scaling
echo -e "${PURPLE}📈 Demo 2: Elastic Scaling${NC}"
echo -e "${PURPLE}==========================${NC}"
echo ""
echo -e "${CYAN}Elastic scaling automatically adjusts replicas based on real-time${NC}"
echo -e "${CYAN}metrics like CPU, memory, and GPU utilization.${NC}"
echo ""

pause_demo

run_demo_command \
    "kubectl apply -f elastic-scaling/01-web-service-elastic.yaml" \
    "Apply web service with elastic scaling (2-20 replicas)"

run_demo_command \
    "kubectl get kaiwojobs ai-web-service-elastic -o wide" \
    "Check the elastic scaling job status"

run_demo_command \
    "kubectl describe kaiwojobs ai-web-service-elastic | grep -A 15 'Elastic Scaling'" \
    "View elastic scaling configuration details"

echo -e "${GREEN}✅ Elastic scaling demo complete!${NC}"
echo -e "${CYAN}The service will scale from 2 to 20 replicas based on CPU/GPU utilization.${NC}"
echo ""

pause_demo

# Demo 3: Hybrid Gang + Elastic
echo -e "${PURPLE}🎨 Demo 3: Hybrid Gang + Elastic Scaling${NC}"
echo -e "${PURPLE}=======================================${NC}"
echo ""
echo -e "${CYAN}Hybrid workloads combine gang scheduling for initial coordination${NC}"
echo -e "${CYAN}with elastic scaling for dynamic adaptation.${NC}"
echo ""

pause_demo

run_demo_command \
    "kubectl apply -f advanced-features/01-gang-elastic-hybrid.yaml" \
    "Apply hybrid training job (gang formation + elastic scaling)"

run_demo_command \
    "kubectl get kaiwojobs hybrid-gang-elastic-training -o wide" \
    "Check the hybrid workload status"

run_demo_command \
    "kubectl describe kaiwojobs hybrid-gang-elastic-training | grep -A 5 'Gang\\|Elastic'" \
    "View both gang and elastic configurations"

echo -e "${GREEN}✅ Hybrid features demo complete!${NC}"
echo -e "${CYAN}The job starts with 8 coordinated workers, then scales up to 32 based on performance.${NC}"
echo ""

pause_demo

# Show all workloads
echo -e "${PURPLE}📊 Demo Summary: All Phase 2 Workloads${NC}"
echo -e "${PURPLE}=====================================${NC}"
echo ""

run_demo_command \
    "kubectl get kaiwojobs -l phase=phase2-demo --all-namespaces -o wide" \
    "View all Phase 2 demo workloads"

run_demo_command \
    "kubectl get kaiwojobs -l kaiwo.ai/gang-scheduling=enabled --all-namespaces" \
    "View gang-scheduled workloads"

run_demo_command \
    "kubectl get kaiwojobs -l kaiwo.ai/elastic-scaling=enabled --all-namespaces" \
    "View elastic-scaling workloads"

echo ""
echo -e "${GREEN}🎉 Phase 2 Features Demo Complete!${NC}"
echo ""

# Show monitoring commands
echo -e "${PURPLE}📊 Monitoring Commands${NC}"
echo -e "${PURPLE}======================${NC}"
echo ""
echo -e "${YELLOW}Monitor gang scheduling:${NC}"
echo -e "${BLUE}kubectl get events --field-selector reason=GangScheduling${NC}"
echo ""
echo -e "${YELLOW}Monitor elastic scaling:${NC}"
echo -e "${BLUE}kubectl get events --field-selector reason=ScalingUp${NC}"
echo -e "${BLUE}kubectl get events --field-selector reason=ScalingDown${NC}"
echo ""
echo -e "${YELLOW}View detailed workload status:${NC}"
echo -e "${BLUE}kubectl describe kaiwojobs <job-name>${NC}"
echo ""
echo -e "${YELLOW}Monitor resource usage:${NC}"
echo -e "${BLUE}kubectl top pods${NC}"
echo -e "${BLUE}kubectl top nodes${NC}"
echo ""

# Cleanup option
echo -e "${YELLOW}🧹 Cleanup${NC}"
echo -e "${YELLOW}==========${NC}"
echo ""
echo -e "${CYAN}To clean up all demo resources:${NC}"
echo -e "${BLUE}./cleanup-all-examples.sh${NC}"
echo ""
echo -e "${CYAN}Or remove individual jobs:${NC}"
echo -e "${BLUE}kubectl delete kaiwojobs distributed-pytorch-training${NC}"
echo -e "${BLUE}kubectl delete kaiwojobs ai-web-service-elastic${NC}"
echo -e "${BLUE}kubectl delete kaiwojobs hybrid-gang-elastic-training${NC}"
echo ""

echo -e "${GREEN}✨ Thanks for exploring Kaiwo Phase 2 Advanced Workload Management! ✨${NC}"
